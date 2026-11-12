package practice

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/quiz"
	"github.com/edupilot/backend-go/internal/store"
)

// AnswerIn là thân `PUT …/answers/{itemId}`. `Answer` theo loại câu: MCQ `{"option_ids":[…]}`, đúng–sai `{"value":true}`, trả lời ngắn / tự luận `{"text":"…"}`.
type AnswerIn struct {
	Answer  json.RawMessage `json:"answer"`
	Version int             `json:"version"`
	// ClientTS: dấu thời gian cục bộ của lựa chọn ghi ngoại tuyến; chỉ có tác dụng khi gửi muộn quá hạn trong dung sai (FR-17).
	ClientTS *time.Time `json:"client_ts"`
}

// answerDoc là dạng chuẩn của câu trả lời lưu trong DB.
type answerDoc struct {
	OptionIDs []string `json:"option_ids,omitempty"`
	Value     *bool    `json:"value,omitempty"`
	Text      *string  `json:"text,omitempty"`
}

// MetaView là phản hồi của `PATCH …/attempts/{aid}`.
type MetaView struct {
	CurrentIndex int `json:"current_index"`
	Version      int `json:"version"`
}

// normalize kiểm và chuẩn hoá một câu trả lời theo loại câu; sai → 422.
func normalize(sn snapshot, raw json.RawMessage, topicMode bool) (answerDoc, []byte, error) {
	var d answerDoc
	if err := json.Unmarshal(raw, &d); err != nil {
		return d, nil, fieldErr("answer", "INVALID_ANSWER", "Câu trả lời không đúng định dạng.")
	}
	switch sn.Type {
	case "MCQ_SINGLE", "MCQ_MULTI":
		valid := map[string]bool{}
		for _, o := range sn.Options {
			valid[o.ID.String()] = true
		}
		seen := map[string]bool{}
		ids := []string{}
		for _, id := range d.OptionIDs {
			if !valid[id] {
				return d, nil, apierr.New(http.StatusUnprocessableEntity, apierr.InvalidOptionID)
			}
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
		if sn.Type == "MCQ_SINGLE" && len(ids) > 1 {
			return d, nil, fieldErr("answer", "SINGLE_MULTIPLE", "Câu này chỉ chọn một đáp án.")
		}
		if topicMode && len(ids) == 0 {
			return d, nil, fieldErr("answer", "VALUE_REQUIRED", "Hãy chọn một đáp án.")
		}
		d = answerDoc{OptionIDs: ids}
	case "TRUE_FALSE":
		if d.Value == nil {
			return d, nil, fieldErr("answer", "VALUE_REQUIRED", "Hãy chọn Đúng hoặc Sai.")
		}
		d = answerDoc{Value: d.Value}
	case "SHORT", "ESSAY":
		if d.Text == nil {
			return d, nil, fieldErr("answer", "VALUE_REQUIRED", "Hãy nhập câu trả lời.")
		}
		limit := MaxShortChars
		if sn.Type == "ESSAY" {
			limit = MaxEssayChars
		}
		if utf8.RuneCountInString(*d.Text) > limit {
			return d, nil, apierr.New(http.StatusUnprocessableEntity, apierr.TooLong)
		}
		if topicMode && quiz.NormalizeShort(*d.Text) == "" && sn.Type == "ESSAY" {
			return d, nil, fieldErr("answer", "VALUE_REQUIRED", "Hãy nhập câu trả lời.")
		}
		d = answerDoc{Text: d.Text}
	default:
		return d, nil, apierr.New(http.StatusUnprocessableEntity, apierr.InvalidQuestion)
	}
	b, _ := json.Marshal(d)
	return d, b, nil
}

// grade chấm một câu bằng Quiz Engine (code thuần, không LLM). nil = không chấm (tự luận). Id lạ / khoá hỏng → sai.
func grade(sn snapshot, key json.RawMessage, rawAnswer []byte) *bool {
	var d answerDoc
	if len(rawAnswer) > 0 {
		_ = json.Unmarshal(rawAnswer, &d)
	}
	one := decimal.NewFromInt(1)
	var k struct {
		OptionIDs []string `json:"option_ids"`
		Value     *bool    `json:"value"`
		Accepted  []string `json:"accepted"`
	}
	_ = json.Unmarshal(key, &k)
	var (
		t   quiz.Type
		ans *quiz.Answer
	)
	opts := make([]string, len(sn.Options))
	for i, o := range sn.Options {
		opts[i] = o.ID.String()
	}
	switch sn.Type {
	case "MCQ_SINGLE":
		t, ans = quiz.MCQSingle, &quiz.Answer{OptionIDs: d.OptionIDs}
	case "MCQ_MULTI":
		t, ans = quiz.MCQMulti, &quiz.Answer{OptionIDs: d.OptionIDs}
	case "TRUE_FALSE":
		t, ans = quiz.TrueFalse, &quiz.Answer{Value: d.Value}
	case "SHORT":
		t, ans = quiz.Short, &quiz.Answer{Text: d.Text}
	default:
		return nil
	}
	got, err := quiz.Grade(t, one, quiz.Key{OptionIDs: k.OptionIDs, Value: k.Value, Accepted: k.Accepted}, ans, quiz.AllOrNothing, opts...)
	ok := err == nil && got.Equal(one)
	return &ok
}

// revealed: khi nào câu được lộ đáp án / giải thích / trích dẫn (US-P9-01 AC5, US-P9-10 AC2).
func revealed(att store.PracticeAttempt, locked *bool) bool {
	if att.Mode == store.PracticeModeMOCK {
		return att.Status != store.PracticeStatusINPROGRESS
	}
	return locked != nil && *locked
}

func feedbackOf(sn snapshot, key json.RawMessage, correct *bool) *FeedbackView {
	c := sn.Citations
	if len(c) == 0 {
		c = json.RawMessage(`[]`)
	}
	return &FeedbackView{Correct: correct, Key: key, Explanation: sn.Explanation, Citations: c}
}

func itemView(att store.PracticeAttempt, id uuid.UUID, pos int32, optionOrder, key, question json.RawMessage, answer json.RawMessage, correct, locked *bool, ver *int32) (ItemView, error) {
	sn, err := parseSnapshot(question)
	if err != nil {
		return ItemView{}, err
	}
	var order []uuid.UUID
	_ = json.Unmarshal(optionOrder, &order)
	byID := map[uuid.UUID]snapOption{}
	for _, o := range sn.Options {
		byID[o.ID] = o
	}
	opts := make([]OptionView, 0, len(order))
	for _, id := range order {
		if o, ok := byID[id]; ok {
			opts = append(opts, OptionView{ID: o.ID, Body: o.Body})
		}
	}
	v := ItemView{ID: id, Position: int(pos), Question: QuestionView{Type: sn.Type, Topic: sn.Topic, Stem: sn.Stem, Options: opts}}
	if ver != nil {
		v.Answer = &AnswerView{Answer: answer, Version: int(*ver), Locked: locked != nil && *locked}
	}
	if revealed(att, locked) {
		v.Feedback = feedbackOf(sn, key, correct)
	}
	return v, nil
}

// view dựng AttemptView; nếu `tab` ≠ nil và lượt còn mở thì tab này tiếp quản (tab cũ nhận 409 TAB_TAKEN_OVER ở lần ghi kế).
func (s *Service) view(ctx context.Context, q *store.Queries, att store.PracticeAttempt, tab uuid.UUID) (AttemptView, error) {
	if tab != uuid.Nil && att.Status == store.PracticeStatusINPROGRESS && (att.TabID == nil || *att.TabID != tab.String()) {
		t := tab.String()
		if err := q.PracticeAttemptTakeTab(ctx, store.PracticeAttemptTakeTabParams{TabID: &t, CourseID: att.CourseID, ID: att.ID}); err != nil {
			return AttemptView{}, fmt.Errorf("practice: tiếp quản tab: %w", err)
		}
	}
	rows, err := q.PracticeItemsWithAnswers(ctx, store.PracticeItemsWithAnswersParams{CourseID: att.CourseID, AttemptID: att.ID})
	if err != nil {
		return AttemptView{}, fmt.Errorf("practice: đọc các câu của lượt: %w", err)
	}
	notes := []string{}
	_ = json.Unmarshal(att.Notes, &notes)
	v := AttemptView{ID: att.ID, Mode: string(att.Mode), Topic: att.Topic, Status: string(att.Status), StartedAt: att.StartedAt, DeadlineAt: att.DeadlineAt, ServerNow: s.now(),
		ItemCount: int(att.ItemCount), CurrentIndex: int(att.CurrentIndex), Notes: notes, Version: int(att.Version), Items: make([]ItemView, 0, len(rows))}
	var corrects []*bool
	var topics []string
	for _, r := range rows {
		iv, err := itemView(att, r.ID, r.Position, r.OptionOrder, r.AnswerKey, r.Question, r.Answer, r.IsCorrect, r.Locked, r.AnswerVersion)
		if err != nil {
			return v, err
		}
		v.Items = append(v.Items, iv)
		corrects = append(corrects, r.IsCorrect)
		topics = append(topics, iv.Question.Topic)
	}
	if att.Status != store.PracticeStatusINPROGRESS {
		res := resultOf(topics, corrects)
		res.Expired = att.Status == store.PracticeStatusEXPIRED
		v.Result = &res
	}
	return v, nil
}

// resultOf tổng hợp kết quả: câu không chấm (tự luận, is_correct nil) không tính vào tổng. Điểm thang 10 một chữ số bằng decimal (phụ).
func resultOf(topics []string, corrects []*bool) ResultView {
	byTopic := map[string]*TopicResult{}
	var names []string
	res := ResultView{Topics: []TopicResult{}}
	for i, c := range corrects {
		if c == nil {
			continue
		}
		t := byTopic[topics[i]]
		if t == nil {
			t = &TopicResult{Topic: topics[i]}
			byTopic[topics[i]] = t
			names = append(names, topics[i])
		}
		t.Total++
		res.Total++
		if *c {
			t.Correct++
			res.Correct++
		}
	}
	sort.Strings(names)
	var worst *TopicResult
	for _, n := range names {
		t := byTopic[n]
		res.Topics = append(res.Topics, *t)
		if t.Correct < t.Total && (worst == nil || t.Correct*worst.Total < worst.Correct*t.Total) { // tỉ lệ đúng thấp hơn (so chéo, không chia)
			worst = t
		}
	}
	if worst != nil {
		w := worst.Topic
		res.Weakest = &w
	}
	res.Score = scoreOf(res.Correct, res.Total)
	return res
}

// scoreOf: đúng × 10 ÷ tổng, làm tròn 1 chữ số (decimal). Tổng 0 → "0.0".
func scoreOf(correct, total int) string {
	if total == 0 {
		return "0.0"
	}
	return decimal.NewFromInt(int64(correct)).Mul(decimal.NewFromInt(10)).Div(decimal.NewFromInt(int64(total))).Round(1).StringFixed(1)
}

// Get trả lượt của chính người gọi (DTO sinh viên). Người khác → 404. Truyền `tab` để mở ở tab này (tiếp quản).
func (s *Service) Get(ctx context.Context, userID, courseID, attemptID, tab uuid.UUID) (AttemptView, error) {
	q := store.New(s.Pool)
	att, err := owned(ctx, q, userID, courseID, attemptID, false)
	if err != nil {
		return AttemptView{}, err
	}
	return s.view(ctx, q, att, tab)
}

func tabCheck(att store.PracticeAttempt, tab uuid.UUID) error {
	if att.Mode == store.PracticeModeMOCK && tab == uuid.Nil {
		return fieldErr("X-Practice-Tab", "VALUE_REQUIRED", "Thiếu mã tab (X-Practice-Tab, uuid).")
	}
	if tab != uuid.Nil && att.TabID != nil && *att.TabID != tab.String() {
		return apierr.New(http.StatusConflict, apierr.TabTakenOver)
	}
	return nil
}

func closedErr() *apierr.Error { return apierr.New(http.StatusConflict, apierr.AttemptClosed) }

// SaveAnswer ghi / chấm một câu (FR-13). TOPIC: chấm ngay, trả phản hồi trong cùng response và khoá đáp án. MOCK: chỉ lưu (version + tab); không phản hồi.
func (s *Service) SaveAnswer(ctx context.Context, userID, courseID, attemptID, itemID, tab uuid.UUID, in AnswerIn) (SaveView, error) {
	var out SaveView
	err := s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := writable(ctx, q, courseID); err != nil {
			return err
		}
		att, err := owned(ctx, q, userID, courseID, attemptID, true)
		if err != nil {
			return err
		}
		if att.Status != store.PracticeStatusINPROGRESS {
			return closedErr()
		}
		if err := tabCheck(att, tab); err != nil {
			return err
		}
		at := s.now()
		if att.Mode == store.PracticeModeMOCK && att.DeadlineAt != nil && at.After(*att.DeadlineAt) {
			// Quá hạn trên máy chủ: chỉ nhận lựa chọn ghi ngoại tuyến có dấu thời gian cục bộ ≤ hạn + dung sai, và chỉ tới khi worker đóng lượt (FR-17).
			if in.ClientTS == nil || in.ClientTS.After(att.DeadlineAt.Add(s.grace())) || at.After(att.DeadlineAt.Add(s.grace())) {
				return closedErr()
			}
		}
		it, err := q.PracticeItemGet(ctx, store.PracticeItemGetParams{CourseID: courseID, AttemptID: attemptID, ItemID: itemID})
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound()
		}
		if err != nil {
			return fmt.Errorf("practice: đọc câu của lượt: %w", err)
		}
		sn, err := parseSnapshot(it.Question)
		if err != nil {
			return err
		}
		topicMode := att.Mode == store.PracticeModeTOPIC
		_, canon, err := normalize(sn, in.Answer, topicMode)
		if err != nil {
			return err
		}
		same := it.Answer != nil && bytes.Equal(compact(it.Answer), canon)
		if it.AnswerVersion != nil { // đã có dòng
			locked := it.Locked != nil && *it.Locked
			switch {
			case same: // gửi lại sau khi mất phản hồi: trả đúng kết quả cũ (idempotent)
				out = SaveView{Version: int(*it.AnswerVersion), Locked: locked}
				if topicMode {
					out.Feedback = feedbackOf(sn, it.AnswerKey, it.IsCorrect)
				}
				return nil
			case locked:
				return apierr.New(http.StatusConflict, apierr.AnswerLocked)
			case in.Version != int(*it.AnswerVersion):
				return apierr.New(http.StatusConflict, apierr.VersionConflict).WithDetails(map[string]int{"version": int(*it.AnswerVersion)})
			}
		}
		var correct *bool
		if topicMode {
			correct = grade(sn, it.AnswerKey, canon)
		}
		answeredAt := at
		if in.ClientTS != nil && in.ClientTS.Before(at) {
			answeredAt = *in.ClientTS
		}
		exp := int32(0)
		if it.AnswerVersion != nil {
			exp = *it.AnswerVersion
		}
		ver, err := q.PracticeAnswerUpsert(ctx, store.PracticeAnswerUpsertParams{CourseID: courseID, AttemptID: attemptID, ItemID: itemID, Answer: canon, IsCorrect: correct, AnsweredAt: answeredAt, Locked: topicMode, ExpectedVersion: exp})
		if errors.Is(err, pgx.ErrNoRows) {
			return apierr.New(http.StatusConflict, apierr.VersionConflict)
		}
		if err != nil {
			return fmt.Errorf("practice: ghi đáp án: %w", err)
		}
		out = SaveView{Version: int(ver), Locked: topicMode}
		if topicMode {
			out.Feedback = feedbackOf(sn, it.AnswerKey, correct)
		}
		return nil
	})
	return out, err
}

func compact(raw []byte) []byte {
	var b bytes.Buffer
	if err := json.Compact(&b, raw); err != nil {
		return raw
	}
	return b.Bytes()
}

// SetIndex lưu vị trí câu đang làm (FR-14).
func (s *Service) SetIndex(ctx context.Context, userID, courseID, attemptID, tab uuid.UUID, idx int) (MetaView, error) {
	var out MetaView
	err := s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		att, err := owned(ctx, q, userID, courseID, attemptID, true)
		if err != nil {
			return err
		}
		if att.Status != store.PracticeStatusINPROGRESS {
			return closedErr()
		}
		if err := tabCheck(att, tab); err != nil {
			return err
		}
		if idx < 0 || idx >= int(att.ItemCount) {
			return fieldErr("current_index", "LIMIT_OUT_OF_RANGE", "Vị trí câu không hợp lệ.")
		}
		upd, err := q.PracticeAttemptSetIndex(ctx, store.PracticeAttemptSetIndexParams{CurrentIndex: int32(idx), CourseID: courseID, ID: attemptID}) //nolint:gosec // < 50
		if err != nil {
			return fmt.Errorf("practice: lưu vị trí: %w", err)
		}
		out = MetaView{CurrentIndex: int(upd.CurrentIndex), Version: int(upd.Version)}
		return nil
	})
	return out, err
}

// Submit nộp lượt (FR-15): chấm câu chưa chấm, đóng lượt; gọi lại → cùng kết quả. Lượt MOCK nộp quá hạn + dung sai tính là hết giờ (EXPIRED).
func (s *Service) Submit(ctx context.Context, userID, courseID, attemptID uuid.UUID) (AttemptView, error) {
	var out AttemptView
	err := s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		att, err := owned(ctx, q, userID, courseID, attemptID, true)
		if err != nil {
			return err
		}
		if att.Status == store.PracticeStatusINPROGRESS {
			status := store.PracticeStatusSUBMITTED
			if att.DeadlineAt != nil && s.now().After(att.DeadlineAt.Add(s.grace())) {
				status = store.PracticeStatusEXPIRED
			}
			if att, err = s.closeLocked(ctx, q, att, status); err != nil {
				return err
			}
		}
		out, err = s.view(ctx, q, att, uuid.Nil)
		return err
	})
	return out, err
}

// closeLocked chấm mọi câu của lượt đã khoá FOR UPDATE rồi đóng nó.
func (s *Service) closeLocked(ctx context.Context, q *store.Queries, att store.PracticeAttempt, status store.PracticeStatus) (store.PracticeAttempt, error) {
	rows, err := q.PracticeItemsWithAnswers(ctx, store.PracticeItemsWithAnswersParams{CourseID: att.CourseID, AttemptID: att.ID})
	if err != nil {
		return att, fmt.Errorf("practice: đọc các câu để chấm: %w", err)
	}
	correct, total := 0, 0
	for _, r := range rows {
		sn, err := parseSnapshot(r.Question)
		if err != nil {
			return att, err
		}
		res := r.IsCorrect
		if res == nil || r.Answer == nil {
			res = grade(sn, r.AnswerKey, r.Answer) // câu bỏ trống: Answer nil → sai (tự luận: nil, không tính)
		}
		if r.Answer == nil || r.Locked == nil || !*r.Locked { // chưa có dòng (bỏ trống) hoặc MOCK chưa khoá: ghi kết quả
			if err := q.PracticeAnswerSetResult(ctx, store.PracticeAnswerSetResultParams{CourseID: att.CourseID, AttemptID: att.ID, ItemID: r.ID, IsCorrect: res}); err != nil {
				return att, fmt.Errorf("practice: ghi kết quả câu: %w", err)
			}
		}
		if res != nil {
			total++
			if *res {
				correct++
			}
		}
	}
	c := int32(correct) //nolint:gosec // ≤ 50
	score := decimal.NullDecimal{Decimal: decimal.RequireFromString(scoreOf(correct, total)), Valid: true}
	closed, err := q.PracticeAttemptClose(ctx, store.PracticeAttemptCloseParams{Status: status, CorrectCount: &c, Score: score, CourseID: att.CourseID, ID: att.ID})
	if err != nil {
		return att, fmt.Errorf("practice: đóng lượt: %w", err)
	}
	return closed, nil
}

// CloseDue đóng các lượt MOCK quá hạn + dung sai (worker `practice.close`, mỗi phút; FR-16). Idempotent: lượt đã đóng không còn trong truy vấn.
func (s *Service) CloseDue(ctx context.Context) (int, error) {
	n := 0
	for {
		var batch int
		err := s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
			due, err := q.PracticeDue(ctx, store.PracticeDueParams{GraceSeconds: int32(s.grace().Seconds()), MaxRows: 100})
			if err != nil {
				return fmt.Errorf("practice: tìm lượt hết giờ: %w", err)
			}
			for _, d := range due {
				att, err := q.PracticeAttemptGet(ctx, store.PracticeAttemptGetParams{CourseID: d.CourseID, ID: d.ID})
				if err != nil {
					return fmt.Errorf("practice: đọc lượt hết giờ: %w", err)
				}
				if _, err := s.closeLocked(ctx, q, att, store.PracticeStatusEXPIRED); err != nil {
					return err
				}
			}
			batch = len(due)
			return nil
		})
		if err != nil {
			return n, err
		}
		n += batch
		if batch < 100 {
			return n, nil
		}
	}
}

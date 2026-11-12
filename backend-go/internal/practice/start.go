package practice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/quiz"
	"github.com/edupilot/backend-go/internal/store"
)

// Loại câu đưa vào luyện đề (Q-P9-6): CODE không bao giờ; ESSAY chỉ ở lượt TOPIC.
var (
	topicTypes = []string{"MCQ_SINGLE", "MCQ_MULTI", "TRUE_FALSE", "SHORT", "ESSAY"}
	mockTypes  = []string{"MCQ_SINGLE", "MCQ_MULTI", "TRUE_FALSE", "SHORT"}
)

// StartIn là thân `POST …/practice/start`.
type StartIn struct {
	Mode            string      `json:"mode"`
	Topic           string      `json:"topic"`
	Count           int         `json:"count"`
	DurationMinutes int         `json:"duration_minutes"`
	Matrix          []quiz.Cell `json:"matrix"`
}

// Overview là `GET …/practice/overview` (FR-11).
func (s *Service) Overview(ctx context.Context, userID, courseID uuid.UUID) (OverviewView, error) {
	q := store.New(s.Pool)
	out := OverviewView{Topics: []TopicInfo{}, InProgress: []InProgressView{}}
	topics, err := q.PracticeTopics(ctx, store.PracticeTopicsParams{CourseID: courseID, Types: topicTypes})
	if err != nil {
		return out, fmt.Errorf("practice: đọc chủ đề: %w", err)
	}
	for _, t := range topics {
		out.Topics = append(out.Topics, TopicInfo{Topic: t.Topic, Available: int(t.Available)})
	}
	stats, err := q.PracticeTopicStats(ctx, store.PracticeTopicStatsParams{CourseID: courseID, StudentID: userID})
	if err != nil {
		return out, fmt.Errorf("practice: đọc thống kê chủ đề: %w", err)
	}
	// Khuyến nghị: chủ đề sai nhiều nhất trong 30 ngày (≥ 3 câu đã làm, còn ≥ MinCount câu); không có thì chủ đề nhiều câu nhất.
	avail := map[string]int{}
	for _, t := range out.Topics {
		avail[t.Topic] = t.Available
	}
	for _, st := range stats { // đã sắp wrong giảm dần
		if st.Answered >= 3 && st.Wrong > 0 && avail[st.Topic] >= MinCount {
			out.Recommended = &Recommended{Topic: st.Topic, Count: min(DefaultTopicN, avail[st.Topic])}
			break
		}
	}
	if out.Recommended == nil {
		for _, t := range out.Topics { // đã sắp số câu giảm dần
			if t.Available >= MinCount {
				out.Recommended = &Recommended{Topic: t.Topic, Count: min(DefaultTopicN, t.Available)}
				break
			}
		}
	}
	open, err := q.PracticeInProgress(ctx, store.PracticeInProgressParams{CourseID: courseID, StudentID: userID})
	if err != nil {
		return out, fmt.Errorf("practice: đọc lượt dở: %w", err)
	}
	for _, o := range open {
		out.InProgress = append(out.InProgress, InProgressView{ID: o.ID, Mode: string(o.Mode), Topic: o.Topic, DeadlineAt: o.DeadlineAt, ItemCount: int(o.ItemCount), CurrentIndex: int(o.CurrentIndex)})
	}
	return out, nil
}

// checkStart kiểm thân `start` (US-P9-05 AC7) và điền mặc định.
func checkStart(in *StartIn) error {
	var errs []apierr.FieldError
	bad := func(f, c, m string) { errs = append(errs, apierr.FieldError{Field: f, Code: c, Message: m}) }
	switch in.Mode {
	case "TOPIC":
		in.Topic = strings.TrimSpace(in.Topic)
		if in.Topic == "" {
			bad("topic", "VALUE_REQUIRED", "Chọn một chủ đề.")
		}
		if in.Count == 0 {
			in.Count = DefaultTopicN
		}
	case "MOCK":
		if in.Count == 0 {
			in.Count = DefaultMockN
		}
		if in.DurationMinutes == 0 {
			in.DurationMinutes = DefaultDuration
		}
		if in.DurationMinutes < MinDuration || in.DurationMinutes > MaxDuration {
			bad("duration_minutes", "LIMIT_OUT_OF_RANGE", "Thời gian thi thử từ 10 đến 180 phút.")
		}
	default:
		bad("mode", "INVALID_MODE", "Chế độ phải là TOPIC hoặc MOCK.")
	}
	if in.Count < MinCount || in.Count > MaxCount {
		bad("count", "LIMIT_OUT_OF_RANGE", "Số câu từ 5 đến 50.")
	}
	if in.Mode == "MOCK" && len(in.Matrix) > 0 {
		sum := 0
		for i, c := range in.Matrix {
			if c.Count < 1 || (c.Difficulty != "" && c.Difficulty != "EASY" && c.Difficulty != "MEDIUM" && c.Difficulty != "HARD") {
				bad(fmt.Sprintf("matrix[%d]", i), "INVALID_CELL", "Ô ma trận không hợp lệ.")
			}
			sum += c.Count
		}
		if sum < MinCount || sum > MaxCount {
			bad("matrix", "LIMIT_OUT_OF_RANGE", "Tổng số câu của ma trận từ 5 đến 50.")
		}
		in.Count = sum
	}
	if len(errs) > 0 {
		return apierr.Validation(errs...)
	}
	return nil
}

// Start dựng một lượt (FR-12): `201` lượt mới, hoặc `200` (created=false) lượt dở cùng (sinh viên, lớp, chế độ, chủ đề). `Idempotency-Key` do middleware xử lý.
func (s *Service) Start(ctx context.Context, userID, courseID uuid.UUID, in StartIn, tab uuid.UUID) (AttemptView, bool, error) {
	if err := checkStart(&in); err != nil {
		return AttemptView{}, false, err
	}
	for try := 0; ; try++ {
		var v AttemptView
		created := false
		err := s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
			if err := writable(ctx, q, courseID); err != nil {
				return err
			}
			topicKey := ""
			if in.Mode == "TOPIC" {
				topicKey = in.Topic
			}
			open, err := q.PracticeAttemptOpen(ctx, store.PracticeAttemptOpenParams{CourseID: courseID, StudentID: userID, Mode: store.PracticeMode(in.Mode), Topic: topicKey})
			if err == nil {
				v, err = s.view(ctx, q, open, tab)
				return err
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("practice: đọc lượt dở: %w", err)
			}
			a, err := s.build(ctx, q, userID, courseID, in, tab)
			if err != nil {
				return err
			}
			created = true
			v, err = s.view(ctx, q, a, uuid.Nil)
			return err
		})
		if isUnique(err) && try == 0 { // hai yêu cầu cùng lúc: yêu cầu kia thắng, đọc lại lượt dở của nó
			continue
		}
		return v, created, err
	}
}

// build chọn câu bằng BuildMatrix (thuần, tất định theo id lượt), chụp câu và ghi lượt + mục. Gọi trong transaction.
func (s *Service) build(ctx context.Context, q *store.Queries, userID, courseID uuid.UUID, in StartIn, tab uuid.UUID) (store.PracticeAttempt, error) {
	types := mockTypes
	if in.Mode == "TOPIC" {
		types = topicTypes
	}
	topic := ""
	if in.Mode == "TOPIC" {
		topic = in.Topic
	}
	pool, err := q.PracticePool(ctx, store.PracticePoolParams{CourseID: courseID, Types: types, Topic: topic})
	if err != nil {
		return store.PracticeAttempt{}, fmt.Errorf("practice: đọc ngân hàng câu: %w", err)
	}
	if in.Mode == "TOPIC" && len(pool) == 0 {
		return store.PracticeAttempt{}, fieldErr("topic", "UNKNOWN_TOPIC", "Chủ đề này chưa có câu hỏi.")
	}
	if len(pool) < MinCount {
		return store.PracticeAttempt{}, apierr.New(http.StatusConflict, apierr.NotEnoughQuestions).WithDetails(map[string]int{"available": len(pool)})
	}
	cands := make([]quiz.Candidate, len(pool))
	for i, p := range pool {
		cands[i] = quiz.Candidate{ID: p.ID.String(), Topic: p.Topic, Difficulty: string(p.Difficulty)}
	}
	hist := quiz.History{Right: map[string]bool{}, Wrong: map[string]bool{}}
	recent, err := q.PracticeRecent(ctx, store.PracticeRecentParams{CourseID: courseID, StudentID: userID})
	if err != nil {
		return store.PracticeAttempt{}, fmt.Errorf("practice: đọc lịch sử 30 ngày: %w", err)
	}
	for _, r := range recent {
		if r.IsCorrect != nil && *r.IsCorrect {
			hist.Right[r.QuestionID.String()] = true
		} else if r.IsCorrect != nil {
			hist.Wrong[r.QuestionID.String()] = true
		}
	}
	matrix := in.Matrix
	if len(matrix) == 0 {
		matrix = []quiz.Cell{{Topic: topic, Count: in.Count}}
	}
	// Seed tạm cho BuildMatrix chỉ để chọn câu; seed xáo dùng id lượt (ổn định khi tải lại). Dùng thời điểm + người: hai sinh viên không cùng đề.
	seed := quiz.SeedOf(userID.String() + s.now().Format(time.RFC3339Nano))
	ids, notes := quiz.BuildMatrix(cands, matrix, hist, seed)
	if len(ids) < MinCount {
		return store.PracticeAttempt{}, apierr.New(http.StatusConflict, apierr.NotEnoughQuestions).WithDetails(map[string]int{"available": len(pool)})
	}
	if notes == nil {
		notes = []string{}
	}
	notesJSON, _ := json.Marshal(notes)
	mx, _ := json.Marshal(matrix)
	arg := store.PracticeAttemptInsertParams{CourseID: courseID, StudentID: userID, Mode: store.PracticeMode(in.Mode), ItemCount: int32(len(ids)), Notes: notesJSON} //nolint:gosec // ≤ 50
	if in.Mode == "TOPIC" {
		arg.Topic = &in.Topic
	} else {
		d := s.now().Add(time.Duration(in.DurationMinutes) * time.Minute)
		arg.DeadlineAt = &d
		arg.Matrix = mx
	}
	if tab != uuid.Nil {
		t := tab.String()
		arg.TabID = &t
	}
	att, err := q.PracticeAttemptInsert(ctx, arg)
	if err != nil {
		return att, err
	}
	// chụp câu
	qs, err := q.PracticeQuestionsByIDs(ctx, store.PracticeQuestionsByIDsParams{CourseID: courseID, Ids: ids})
	if err != nil {
		return att, fmt.Errorf("practice: đọc câu hỏi: %w", err)
	}
	opts, err := q.PracticeOptionsByQuestions(ctx, store.PracticeOptionsByQuestionsParams{CourseID: courseID, Ids: ids})
	if err != nil {
		return att, fmt.Errorf("practice: đọc đáp án: %w", err)
	}
	byQ := make(map[uuid.UUID]store.PracticeQuestionsByIDsRow, len(qs))
	for _, r := range qs {
		byQ[r.ID] = r
	}
	optsOf := map[uuid.UUID][]store.PracticeOptionsByQuestionsRow{}
	for _, o := range opts {
		optsOf[o.QuestionID] = append(optsOf[o.QuestionID], o)
	}
	rngSeed := quiz.SeedOf(att.ID.String())
	order := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		order[i] = uuid.MustParse(id)
	}
	quiz.Shuffle(order, rngSeed) // thứ tự câu xáo riêng từng lượt, ổn định theo id lượt
	for pos, qid := range order {
		row, ok := byQ[qid]
		if !ok {
			return att, errors.New("practice: câu hỏi biến mất giữa chừng")
		}
		sn := snapshot{Type: string(row.Type), Topic: row.Topic, Difficulty: string(row.Difficulty), Stem: row.Stem, Explanation: row.Explanation, Citations: row.Citations, Options: []snapOption{}}
		free, pinned := []uuid.UUID{}, []uuid.UUID{}
		for _, o := range optsOf[qid] {
			sn.Options = append(sn.Options, snapOption{ID: o.ID, Body: o.Body})
			if o.PinnedLast {
				pinned = append(pinned, o.ID)
			} else {
				free = append(free, o.ID)
			}
		}
		quiz.Shuffle(free, rngSeed^uint64(pos+1)*0x9e3779b97f4a7c15) // phương án xáo riêng từng câu; ghim cuối giữ cuối
		optionOrder, _ := json.Marshal(append(free, pinned...))
		snap, _ := json.Marshal(sn)
		key := row.AnswerKey
		if len(key) == 0 {
			key = json.RawMessage(`{}`)
		}
		if err := q.PracticeItemInsert(ctx, store.PracticeItemInsertParams{CourseID: courseID, AttemptID: att.ID, QuestionID: qid, Position: int32(pos), OptionOrder: optionOrder, AnswerKey: key, Question: snap, QuestionVersion: row.Version}); err != nil { //nolint:gosec // ≤ 50
			return att, fmt.Errorf("practice: ghi mục: %w", err)
		}
	}
	return att, nil
}

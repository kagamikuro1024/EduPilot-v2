package exam

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/quiz"
	"github.com/edupilot/backend-go/internal/store"
)

// Chủ đề outbox của lượt làm (SRS 4.10).
const (
	TopicAttemptStarted   = "exam.attempt_started"
	TopicAttemptSubmitted = "exam.attempt_submitted"
)

// Lý do nộp.
const (
	ReasonManual  = "MANUAL"
	ReasonTimeout = "TIMEOUT"
	ReasonClosed  = "CLOSED"
)

// Hằng số của lượt làm (SRS 4.3): mốc "sắp đóng", nhịp cập nhật `writer_seen_at`.
const (
	closingLead     = 60 * time.Second
	writerSeenEvery = 10 * time.Second
)

// Config của lượt làm (US-PE-05). Số 0 = mặc định.
type AttemptConfig struct {
	Grace    time.Duration // EXAM_GRACE_SECONDS (10 s)
	TabStale time.Duration // EXAM_TAB_STALE (20 s)
	SaveRate int           // EXAM_SAVE_RATE_PER_MIN (240)
}

func (c AttemptConfig) grace() time.Duration {
	if c.Grace <= 0 {
		return 10 * time.Second
	}
	return c.Grace
}

func (c AttemptConfig) tabStale() time.Duration {
	if c.TabStale <= 0 {
		return 20 * time.Second
	}
	return c.TabStale
}

func (c AttemptConfig) saveRate() int {
	if c.SaveRate <= 0 {
		return 240
	}
	return c.SaveRate
}

// ---- DTO (sinh viên) -----------------------------------------------------------------------------------------------------------

// WriterView cho biết tab gọi có phải người ghi hiện tại không.
type WriterView struct {
	IsYou bool `json:"is_you"`
}

// AttemptView là lượt làm đang chạy.
type AttemptView struct {
	ID         uuid.UUID  `json:"id"`
	ExamID     uuid.UUID  `json:"exam_id"`
	Status     string     `json:"status"`
	StartedAt  time.Time  `json:"started_at"`
	DeadlineAt time.Time  `json:"deadline_at"`
	ServerTime time.Time  `json:"server_time"`
	Writer     WriterView `json:"writer"`
}

// ExamBlockView là khối `exam` của lượt đang làm.
type ExamBlockView struct {
	ID              uuid.UUID `json:"id"`
	Title           string    `json:"title"`
	Instructions    *string   `json:"instructions"`
	Kind            string    `json:"kind"`
	DurationMinutes int       `json:"duration_minutes"`
	ClosesAt        time.Time `json:"closes_at"`
	MultiScoring    string    `json:"multi_scoring"`
}

// AttemptStartView là `POST …/attempts` và `GET …/attempts/mine` khi IN_PROGRESS.
type AttemptStartView struct {
	Attempt AttemptView   `json:"attempt"`
	Exam    ExamBlockView `json:"exam"`
	Items   []ItemView    `json:"items"`
}

// SubmittedAttemptView là lượt đã nộp (không điểm, không đúng / sai).
type SubmittedAttemptView struct {
	ID           uuid.UUID `json:"id"`
	Status       string    `json:"status"`
	SubmittedAt  time.Time `json:"submitted_at"`
	SubmitReason string    `json:"submit_reason"`
}

// SummaryExamView là khối `exam` sau khi nộp.
type SummaryExamView struct {
	ID       uuid.UUID  `json:"id"`
	Title    string     `json:"title"`
	ClosesAt *time.Time `json:"closes_at"`
	Status   string     `json:"status"`
}

// AttemptSummaryView là `GET …/attempts/mine` sau khi nộp: CHỈ tóm tắt — không `items` (không đọc lại đề khi cả lớp còn làm), không điểm.
type AttemptSummaryView struct {
	Attempt SubmittedAttemptView `json:"attempt"`
	Exam    SummaryExamView      `json:"exam"`
}

// NoAttemptView là `GET …/attempts/mine` khi chưa bắt đầu: `attempt` null kèm khối bài (màn giới thiệu).
type NoAttemptView struct {
	Attempt *struct{}       `json:"attempt"`
	Exam    ExamStudentView `json:"exam"`
}

// SubmitView là phản hồi nộp tay: chỉ tóm tắt, KHÔNG điểm / đúng sai.
type SubmitView struct {
	Status      string    `json:"status"`
	SubmittedAt time.Time `json:"submitted_at"`
	Answered    int       `json:"answered"`
	Total       int       `json:"total"`
}

// SaveView là phản hồi lưu: không bao giờ có đúng / sai.
type SaveView struct {
	SavedAt    time.Time `json:"saved_at"`
	ServerTime time.Time `json:"server_time"`
	DeadlineAt time.Time `json:"deadline_at"`
}

// AnswerIn là một câu trả lời gửi lên.
type AnswerIn struct {
	ItemID uuid.UUID       `json:"item_id"`
	Answer json.RawMessage `json:"answer"`
}

// AnswersIn là thân `PUT …/attempts/{aid}/answers`.
type AnswersIn struct {
	Items []AnswerIn `json:"items"`
}

// ---- bắt đầu / làm tiếp --------------------------------------------------------------------------------------------------------

func examNotOpen(reason string, kv ...any) *apierr.Error {
	d := map[string]any{"reason": reason}
	for i := 0; i+1 < len(kv); i += 2 {
		d[kv[i].(string)] = kv[i+1]
	}
	return apierr.New(http.StatusConflict, apierr.ExamNotOpen).WithDetails(d)
}

func alreadySubmitted(at *time.Time) *apierr.Error {
	return apierr.New(http.StatusConflict, apierr.AttemptAlreadySubmit).WithDetails(map[string]any{"submitted_at": at})
}

// StartAttempt: `POST …/attempts`. Chưa có lượt → tạo (201); đã có lượt đang làm → làm tiếp CÙNG lượt (200, đồng hồ không reset); đã nộp → 409.
// Trong MỘT transaction: khoá chia sẻ bài, kiểm trạng thái hiệu lực, INSERT … ON CONFLICT DO NOTHING (SRS 4.3.1).
func (s *Service) StartAttempt(ctx context.Context, userID, courseID, examID uuid.UUID, tab uuid.UUID) (AttemptStartView, bool, error) {
	var out AttemptStartView
	created := false
	err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if err := writable(ctx, q, courseID); err != nil {
			return err
		}
		e, err := q.ExamShare(ctx, store.ExamShareParams{CourseID: courseID, ID: examID})
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound()
		}
		if err != nil {
			return fmt.Errorf("exam: khoá bài thi: %w", err)
		}
		now := s.now()
		if a, err := q.AttemptByStudent(ctx, store.AttemptByStudentParams{CourseID: courseID, ExamID: examID, StudentID: userID}); err == nil {
			if a.Status != store.AttemptStatusINPROGRESS {
				return alreadySubmitted(a.SubmittedAt)
			}
			if now.After(a.DeadlineAt.Add(s.Attempt.grace())) {
				return apierr.New(http.StatusConflict, apierr.AttemptClosed).WithDetails(map[string]any{"deadline_at": a.DeadlineAt})
			}
			out, err = s.attemptStartView(ctx, q, e, a, tab, now)
			return err
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("exam: đọc lượt làm: %w", err)
		}
		switch eff := EffectiveStatus(string(e.Status), e.OpensAt, e.ClosesAt, now); eff {
		case StatusDraft:
			return examNotOpen("not_scheduled")
		case StatusScheduled:
			return examNotOpen("not_yet", "opens_at", e.OpensAt)
		case StatusClosed, StatusPublished:
			return examNotOpen("closed", "closes_at", e.ClosesAt)
		}
		if e.ClosesAt == nil || e.DurationMinutes == nil || !now.Add(closingLead).Before(*e.ClosesAt) {
			return examNotOpen("closing", "closes_at", e.ClosesAt)
		}
		deadline := now.Add(time.Duration(*e.DurationMinutes) * time.Minute)
		if deadline.After(*e.ClosesAt) {
			deadline = *e.ClosesAt
		}
		a, err := q.AttemptInsert(ctx, store.AttemptInsertParams{CourseID: courseID, ExamID: examID, StudentID: userID, StartedAt: now, DeadlineAt: deadline, WriterTab: &tab})
		if errors.Is(err, pgx.ErrNoRows) { // đã có lượt chen vào giữa: trả lượt đó (cùng quy tắc trên)
			a, err = q.AttemptByStudent(ctx, store.AttemptByStudentParams{CourseID: courseID, ExamID: examID, StudentID: userID})
			if err != nil {
				return fmt.Errorf("exam: đọc lượt làm: %w", err)
			}
			if a.Status != store.AttemptStatusINPROGRESS {
				return alreadySubmitted(a.SubmittedAt)
			}
		} else if err != nil {
			return fmt.Errorf("exam: tạo lượt làm: %w", err)
		} else {
			created = true
			if _, err := outbox.Write(ctx, tx, TopicAttemptStarted, map[string]any{"exam_id": examID, "course_id": courseID, "attempt_id": a.ID, "user_id": userID}); err != nil {
				return fmt.Errorf("exam: outbox: %w", err)
			}
		}
		out, err = s.attemptStartView(ctx, q, e, a, tab, now)
		return err
	})
	if err == nil {
		s.refreshLock(ctx, userID) // bắt đầu / làm tiếp: chat AI khoá tới hạn + grace (US-PE-07 AC1)
	}
	return out, created, err
}

// attemptStartView dựng khối đang làm: đề xáo theo `attempt_id` + câu trả lời đã lưu của chính người này.
func (s *Service) attemptStartView(ctx context.Context, q *store.Queries, e store.Exam, a store.ExamAttempt, tab uuid.UUID, now time.Time) (AttemptStartView, error) {
	src, err := s.loadViewSource(ctx, q, a.CourseID, a.ExamID)
	if err != nil {
		return AttemptStartView{}, err
	}
	saved, err := q.AnswersList(ctx, store.AnswersListParams{CourseID: a.CourseID, AttemptID: a.ID})
	if err != nil {
		return AttemptStartView{}, fmt.Errorf("exam: câu trả lời đã lưu: %w", err)
	}
	byItem := make(map[uuid.UUID]json.RawMessage, len(saved))
	for _, r := range saved {
		byItem[r.ItemID] = r.Answer
	}
	items, err := BuildStudentView(src, e.ShuffleQuestions, e.ShuffleOptions, AttemptSeed(a.ID), byItem)
	if err != nil {
		return AttemptStartView{}, err
	}
	if err := s.attachDrafts(ctx, q, a, items); err != nil {
		return AttemptStartView{}, err
	}
	dur := 0
	if e.DurationMinutes != nil {
		dur = int(*e.DurationMinutes)
	}
	closes := a.DeadlineAt
	if e.ClosesAt != nil {
		closes = *e.ClosesAt
	}
	return AttemptStartView{
		Attempt: attemptView(a, tab, now),
		Exam:    ExamBlockView{ID: e.ID, Title: e.Title, Instructions: e.Instructions, Kind: string(e.Kind), DurationMinutes: dur, ClosesAt: closes, MultiScoring: string(e.MultiScoring)},
		Items:   items,
	}, nil
}

func attemptView(a store.ExamAttempt, tab uuid.UUID, now time.Time) AttemptView {
	return AttemptView{ID: a.ID, ExamID: a.ExamID, Status: string(a.Status), StartedAt: a.StartedAt, DeadlineAt: a.DeadlineAt, ServerTime: now,
		Writer: WriterView{IsYou: a.WriterTab != nil && tab != uuid.Nil && *a.WriterTab == tab}}
}

// MyAttempt: `GET …/attempts/mine` — một trong ba dạng theo trạng thái: chưa bắt đầu (`NoAttemptView`), đang làm (`AttemptStartView`, đủ đề), đã nộp (`AttemptSummaryView`, KHÔNG đề).
func (s *Service) MyAttempt(ctx context.Context, userID, courseID, examID uuid.UUID, tab uuid.UUID) (any, error) {
	q := store.New(s.Pool)
	e, err := q.ExamGet(ctx, store.ExamGetParams{CourseID: courseID, ID: examID})
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && e.Status == store.ExamStatusDRAFT) {
		return nil, notFound()
	}
	if err != nil {
		return nil, fmt.Errorf("exam: đọc bài thi: %w", err)
	}
	now := s.now()
	a, err := q.AttemptByStudent(ctx, store.AttemptByStudentParams{CourseID: courseID, ExamID: examID, StudentID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		v, err := s.GetExamForStudent(ctx, courseID, userID, examID)
		if err != nil {
			return nil, err
		}
		return NoAttemptView{Exam: v}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("exam: đọc lượt làm: %w", err)
	}
	if a.Status == store.AttemptStatusINPROGRESS && !now.After(a.DeadlineAt.Add(s.Attempt.grace())) {
		return s.attemptStartView(ctx, q, e, a, tab, now)
	}
	reason, submitted := "", a.DeadlineAt
	if a.SubmitReason != nil {
		reason = string(*a.SubmitReason)
	}
	if a.SubmittedAt != nil {
		submitted = *a.SubmittedAt
	}
	status := string(a.Status)
	if a.Status == store.AttemptStatusINPROGRESS { // hết hạn mà tick chưa kịp nộp: với sinh viên bài đã là của hạn chót (không còn ghi được)
		status, reason = string(store.AttemptStatusGRADING), ReasonTimeout
	}
	return AttemptSummaryView{
		Attempt: SubmittedAttemptView{ID: a.ID, Status: status, SubmittedAt: submitted, SubmitReason: reason},
		Exam:    SummaryExamView{ID: e.ID, Title: e.Title, ClosesAt: e.ClosesAt, Status: EffectiveStatus(string(e.Status), e.OpensAt, e.ClosesAt, now)},
	}, nil
}

// ---- một nơi được ghi ----------------------------------------------------------------------------------------------------------

// claimWriter kiểm / giành quyền ghi (SRS 4.3.5): tab gọi là người ghi nếu `writer_tab` rỗng, trùng, hoặc người ghi cũ im quá `EXAM_TAB_STALE`; `writer_seen_at`
// chỉ cập nhật tối đa một lần / 10 s. Ngược lại 409 ATTEMPT_OTHER_TAB.
func (s *Service) claimWriter(ctx context.Context, q *store.Queries, a store.ExamAttempt, tab uuid.UUID, now time.Time) error {
	switch {
	case a.WriterTab != nil && *a.WriterTab == tab:
		if a.WriterSeenAt == nil || now.Sub(*a.WriterSeenAt) >= writerSeenEvery {
			return q.AttemptSetWriter(ctx, store.AttemptSetWriterParams{CourseID: a.CourseID, ID: a.ID, WriterTab: &tab, SeenAt: &now})
		}
		return nil
	case a.WriterTab == nil || a.WriterSeenAt == nil || now.Sub(*a.WriterSeenAt) > s.Attempt.tabStale():
		return q.AttemptSetWriter(ctx, store.AttemptSetWriterParams{CourseID: a.CourseID, ID: a.ID, WriterTab: &tab, SeenAt: &now})
	}
	return apierr.New(http.StatusConflict, apierr.AttemptOtherTab).WithDetails(map[string]any{"writer_seen_at": a.WriterSeenAt})
}

// openAttempt khoá lượt của CHÍNH sinh viên và kiểm điều kiện ghi: đang làm, còn trong hạn + grace.
func (s *Service) openAttempt(ctx context.Context, q *store.Queries, courseID, examID, attemptID, userID uuid.UUID, now time.Time) (store.ExamAttempt, error) {
	a, err := q.AttemptLock(ctx, store.AttemptLockParams{CourseID: courseID, ExamID: examID, ID: attemptID, StudentID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return a, notFound() // không có, hoặc là lượt của người khác
	}
	if err != nil {
		return a, fmt.Errorf("exam: khoá lượt làm: %w", err)
	}
	if a.Status != store.AttemptStatusINPROGRESS {
		return a, alreadySubmitted(a.SubmittedAt)
	}
	if now.After(a.DeadlineAt.Add(s.Attempt.grace())) {
		return a, apierr.New(http.StatusConflict, apierr.AttemptClosed).WithDetails(map[string]any{"deadline_at": a.DeadlineAt})
	}
	return a, nil
}

// Takeover: `POST …/takeover` — chuyển quyền ghi cho tab gọi; ghi `TAB_TAKEOVER` (giảng viên xem). `reload` đánh dấu tự takeover sau khi tải lại trang.
func (s *Service) Takeover(ctx context.Context, userID, courseID, examID, attemptID, tab uuid.UUID, reload bool) (AttemptView, error) {
	var out AttemptView
	err := s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		now := s.now()
		a, err := s.openAttempt(ctx, q, courseID, examID, attemptID, userID, now)
		if err != nil {
			return err
		}
		if err := q.AttemptSetWriter(ctx, store.AttemptSetWriterParams{CourseID: courseID, ID: attemptID, WriterTab: &tab, SeenAt: &now}); err != nil {
			return fmt.Errorf("exam: đổi người ghi: %w", err)
		}
		meta, _ := json.Marshal(map[string]any{"reload": reload})
		if err := q.ExamEventInsert(ctx, store.ExamEventInsertParams{CourseID: courseID, ExamID: examID, AttemptID: attemptID, StudentID: userID, Type: store.ExamEventTypeTABTAKEOVER, Meta: meta}); err != nil {
			return fmt.Errorf("exam: ghi sự kiện: %w", err)
		}
		a.WriterTab = &tab
		out = attemptView(a, tab, now)
		return nil
	})
	return out, err
}

// ---- lưu câu trả lời -----------------------------------------------------------------------------------------------------------

type answerShape struct {
	OptionIDs *[]string `json:"option_ids"`
	Value     *bool     `json:"value"`
}

func invalidAnswer(i int, code, msg string) apierr.FieldError {
	return apierr.FieldError{Field: fmt.Sprintf("items[%d].answer", i), Code: code, Message: msg}
}

// rateLimit: tối đa `EXAM_SAVE_RATE_PER_MIN` lần lưu mỗi lượt mỗi phút (`ep:rl:exam:{attempt_id}:{phút}`). Redis nil / lỗi → bỏ qua (fail-open: không chặn sinh viên đang thi vì Redis).
func (s *Service) rateLimit(ctx context.Context, attemptID uuid.UUID, now time.Time) error {
	if s.Redis == nil {
		return nil
	}
	key := appredis.Key("rl", "exam", attemptID.String(), fmt.Sprint(now.Unix()/60))
	n, err := s.Redis.Incr(ctx, key).Result()
	if err != nil {
		return nil
	}
	if n == 1 {
		_ = s.Redis.Expire(ctx, key, 120*time.Second).Err()
	}
	if int(n) > s.Attempt.saveRate() {
		return apierr.New(http.StatusTooManyRequests, apierr.RateLimited).WithRetryAfter(int(60 - now.Unix()%60))
	}
	return nil
}

// SaveAnswers: `PUT …/answers` — ghi CHỈ các câu gửi lên (diff); mảng rỗng xoá lựa chọn; một lệnh SQL cho cả lô. Không bao giờ trả đúng / sai.
func (s *Service) SaveAnswers(ctx context.Context, userID, courseID, examID, attemptID, tab uuid.UUID, in AnswersIn) (SaveView, error) {
	var out SaveView
	err := s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		now := s.now()
		a, err := s.openAttempt(ctx, q, courseID, examID, attemptID, userID, now)
		if err != nil {
			return err
		}
		if err := s.claimWriter(ctx, q, a, tab, now); err != nil {
			return err
		}
		if err := s.rateLimit(ctx, attemptID, now); err != nil {
			return err
		}
		if len(in.Items) == 0 {
			return apierr.Validation(apierr.FieldError{Field: "items", Code: "VALUE_REQUIRED", Message: "Chưa có câu trả lời nào để lưu."})
		}
		src, err := s.loadViewSource(ctx, q, courseID, examID)
		if err != nil {
			return err
		}
		kinds := make(map[uuid.UUID]store.ExamPreviewItemsRow, len(src.Items))
		for _, it := range src.Items {
			kinds[it.ItemID] = it
		}
		var errs []apierr.FieldError
		var clear []string
		rows := make([]map[string]any, 0, len(in.Items))
		seen := map[uuid.UUID]bool{}
		for i, it := range in.Items {
			item, ok := kinds[it.ItemID]
			switch {
			case !ok:
				errs = append(errs, invalidAnswer(i, "INVALID_ITEM", "Câu này không thuộc bài thi."))
				continue
			case seen[it.ItemID]:
				errs = append(errs, invalidAnswer(i, "DUPLICATE_ITEM", "Mỗi câu chỉ gửi một lần trong một lần lưu."))
				continue
			case item.Type == store.QuestionTypeCODE:
				errs = append(errs, invalidAnswer(i, "INVALID_ITEM", "Bài lập trình lưu bằng bản nháp mã, không bằng câu trả lời."))
				continue
			}
			seen[it.ItemID] = true
			var sh answerShape
			dec := json.NewDecoder(bytes.NewReader(it.Answer))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&sh); err != nil {
				errs = append(errs, invalidAnswer(i, "INVALID_ANSWER", "Câu trả lời sai dạng."))
				continue
			}
			if item.Type == store.QuestionTypeTRUEFALSE {
				if sh.Value == nil || sh.OptionIDs != nil {
					errs = append(errs, invalidAnswer(i, "INVALID_ANSWER", "Câu đúng / sai chỉ nhận {\"value\": true | false}."))
					continue
				}
				rows = append(rows, map[string]any{"item_id": it.ItemID, "answer": map[string]any{"value": *sh.Value}})
				continue
			}
			if sh.OptionIDs == nil || sh.Value != nil {
				errs = append(errs, invalidAnswer(i, "INVALID_ANSWER", "Câu trắc nghiệm chỉ nhận {\"option_ids\": […]}."))
				continue
			}
			valid := map[string]bool{}
			for _, o := range src.Options[item.QuestionID] {
				valid[o.ID.String()] = true
			}
			ids := *sh.OptionIDs
			bad := false
			for _, id := range ids {
				if !valid[id] {
					errs = append(errs, invalidAnswer(i, "INVALID_OPTION_ID", "Đáp án không thuộc câu này."))
					bad = true
					break
				}
			}
			switch {
			case bad:
			case item.Type == store.QuestionTypeMCQSINGLE && len(ids) > 1:
				errs = append(errs, invalidAnswer(i, "INVALID_OPTION_ID", "Câu một đáp án chỉ chọn được một lựa chọn."))
			case len(ids) == 0:
				clear = append(clear, it.ItemID.String())
			default:
				rows = append(rows, map[string]any{"item_id": it.ItemID, "answer": map[string]any{"option_ids": dedupeIDs(ids)}})
			}
		}
		if len(errs) > 0 {
			return apierr.Validation(errs...)
		}
		if len(clear) > 0 {
			if err := q.AnswersDelete(ctx, store.AnswersDeleteParams{CourseID: courseID, AttemptID: attemptID, ItemIds: clear}); err != nil {
				return fmt.Errorf("exam: xoá lựa chọn: %w", err)
			}
		}
		if len(rows) > 0 {
			raw, err := json.Marshal(rows)
			if err != nil {
				return fmt.Errorf("exam: mã hoá lô câu trả lời: %w", err)
			}
			if err := q.AnswersUpsert(ctx, store.AnswersUpsertParams{AttemptID: attemptID, CourseID: courseID, SavedAt: now, Rows: raw}); err != nil {
				return fmt.Errorf("exam: lưu câu trả lời: %w", err)
			}
		}
		out = SaveView{SavedAt: now, ServerTime: now, DeadlineAt: a.DeadlineAt}
		return nil
	})
	return out, err
}

func dedupeIDs(ids []string) []string {
	seen := make(map[string]bool, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// ---- nộp / tự nộp / chấm trắc nghiệm ------------------------------------------------------------------------------------------

// SubmitAttempt: `POST …/submit` — nộp tay (MANUAL): trắc nghiệm chấm ngay; trả CHỈ tóm tắt. Lượt đã nộp → 409 (cùng khoá gửi lại được middleware trả đúng phản hồi cũ).
func (s *Service) SubmitAttempt(ctx context.Context, userID, courseID, examID, attemptID, tab uuid.UUID) (SubmitView, error) {
	var out SubmitView
	err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		now := s.now()
		a, err := s.openAttempt(ctx, q, courseID, examID, attemptID, userID, now)
		if err != nil {
			return err
		}
		if err := s.claimWriter(ctx, q, a, tab, now); err != nil {
			return err
		}
		e, err := q.ExamGet(ctx, store.ExamGetParams{CourseID: courseID, ID: examID})
		if err != nil {
			return fmt.Errorf("exam: đọc bài thi: %w", err)
		}
		at := now
		if a.DeadlineAt.Before(at) {
			at = a.DeadlineAt
		}
		fin, err := s.finish(ctx, q, tx, e, a, ReasonManual, at)
		if err != nil {
			return err
		}
		out = fin
		return nil
	})
	if err == nil {
		s.refreshLock(ctx, userID) // không còn lượt IN_PROGRESS nào → gỡ khoá chat
	}
	return out, err
}

// finish nộp một lượt IN_PROGRESS (dùng chung cho nộp tay và tick): chấm trắc nghiệm ngay bằng các câu đã lưu; bài có câu code → GRADING (hoàn tất ở US-PE-06);
// bài chỉ trắc nghiệm → GRADED kèm điểm. Câu chưa trả lời tính 0. Phát `exam.attempt_submitted`.
func (s *Service) finish(ctx context.Context, q *store.Queries, tx pgx.Tx, e store.Exam, a store.ExamAttempt, reason string, at time.Time) (SubmitView, error) {
	answered, err := q.AttemptCountsByAnswer(ctx, store.AttemptCountsByAnswerParams{CourseID: a.CourseID, AttemptID: a.ID})
	if err != nil {
		return SubmitView{}, fmt.Errorf("exam: đếm câu đã trả lời: %w", err)
	}
	items, err := q.ExamGradeItems(ctx, store.ExamGradeItemsParams{CourseID: a.CourseID, ExamID: a.ExamID})
	if err != nil {
		return SubmitView{}, fmt.Errorf("exam: mục để chấm: %w", err)
	}
	if err := s.autoSubmitDrafts(ctx, q, tx, a, items); err != nil {
		return SubmitView{}, err
	}
	arg := store.AttemptFinishParams{CourseID: a.CourseID, ID: a.ID, Status: store.AttemptStatusGRADING, SubmittedAt: &at, SubmitReason: ptrOf[store.AttemptSubmitReason](reason)}
	if gr, err := s.computeGrade(ctx, q, e, a, items); err != nil {
		return SubmitView{}, err
	} else if gr.Complete { // MCQ-only, hoặc câu code không có bản nộp: chấm xong ngay; còn bản nộp QUEUED → GRADING, hoàn tất ở TryFinishGrading (SRS 4.8.1)
		now := s.now()
		arg.Status, arg.AutoScore, arg.GradedAt, arg.Breakdown = store.AttemptStatusGRADED, decimal.NullDecimal{Decimal: gr.Score, Valid: true}, &now, gr.Breakdown
	}
	row, err := q.AttemptFinish(ctx, arg)
	if errors.Is(err, pgx.ErrNoRows) { // đã nộp (tick và nộp tay chạm nhau): idempotent
		return SubmitView{}, alreadySubmitted(a.SubmittedAt)
	}
	if err != nil {
		return SubmitView{}, fmt.Errorf("exam: nộp bài: %w", err)
	}
	if _, err := outbox.Write(ctx, tx, TopicAttemptSubmitted, map[string]any{"exam_id": a.ExamID, "course_id": a.CourseID, "attempt_id": a.ID, "user_id": a.StudentID}); err != nil {
		return SubmitView{}, fmt.Errorf("exam: outbox: %w", err)
	}
	return SubmitView{Status: string(row.Status), SubmittedAt: at, Answered: int(answered), Total: len(items)}, nil
}

// gradeOne chấm một câu từ đáp án đúng và câu trả lời ở dạng JSON thô; câu trả lời chọn id lạ hoặc sai dạng coi như chưa trả lời.
func gradeOne(t quiz.Type, points decimal.Decimal, key json.RawMessage, answer json.RawMessage, mode quiz.Mode, options []string) (decimal.Decimal, error) {
	var k struct {
		OptionIDs []string `json:"option_ids"`
		Value     *bool    `json:"value"`
	}
	if err := json.Unmarshal(key, &k); err != nil {
		return decimal.Zero, fmt.Errorf("đáp án đúng không đọc được: %w", err)
	}
	var a *quiz.Answer
	if len(answer) > 0 {
		var ans struct {
			OptionIDs []string `json:"option_ids"`
			Value     *bool    `json:"value"`
		}
		if err := json.Unmarshal(answer, &ans); err == nil {
			a = &quiz.Answer{OptionIDs: ans.OptionIDs, Value: ans.Value}
		}
	}
	if t == "CODE" { // câu code do máy chấm tính (US-PE-06): ở đây chưa có điểm
		return decimal.Zero, nil
	}
	got, err := quiz.Grade(t, points, quiz.Key{OptionIDs: k.OptionIDs, Value: k.Value}, a, mode, options...)
	if errors.Is(err, quiz.ErrInvalidOption) {
		return decimal.Zero, nil
	}
	return got, err
}

// AutoSubmitDue (bước 3 của exam.tick): nộp các lượt IN_PROGRESS đã quá `deadline_at + EXAM_GRACE_SECONDS` (lô 200). `submitted_at = deadline_at`;
// `CLOSED` nếu hạn là giờ đóng của lớp, ngược lại `TIMEOUT`. Mỗi lượt một transaction nên một lượt lỗi không chặn các lượt khác.
func (s *Service) AutoSubmitDue(ctx context.Context) (int, error) {
	due, err := store.New(s.Pool).AttemptsDue(ctx, store.AttemptsDueParams{GraceSeconds: int32(s.Attempt.grace() / time.Second), Now: s.now()}) //nolint:gosec // ≤ 300
	if err != nil {
		return 0, fmt.Errorf("exam: lượt quá hạn: %w", err)
	}
	var firstErr error
	n := 0
	for _, d := range due {
		reason := ReasonTimeout
		if d.ByClose {
			reason = ReasonClosed
		}
		err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
			a, err := q.AttemptLockByID(ctx, store.AttemptLockByIDParams{CourseID: d.CourseID, ID: d.ID})
			if err != nil || a.Status != store.AttemptStatusINPROGRESS {
				return err // đã nộp (nộp tay chen vào): bỏ qua
			}
			e, err := q.ExamGet(ctx, store.ExamGetParams{CourseID: d.CourseID, ID: d.ExamID})
			if err != nil {
				return fmt.Errorf("đọc bài thi: %w", err)
			}
			_, err = s.finish(ctx, q, tx, e, a, reason, a.DeadlineAt)
			return err
		})
		if err != nil {
			var ae *apierr.Error
			if errors.As(err, &ae) { // nộp tay chạm nhau: không phải lỗi
				continue
			}
			if firstErr == nil {
				firstErr = fmt.Errorf("lượt %s: %w", d.ID, err)
			}
			continue
		}
		s.refreshLock(ctx, d.StudentID)
		n++
	}
	return n, firstErr
}

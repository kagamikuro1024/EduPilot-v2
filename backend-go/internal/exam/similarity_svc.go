package exam

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/exam/similarity"
	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// KindSimilarity là loại việc nền so độ giống mã của một bài thi đã đóng (US-PE-07 AC8).
const KindSimilarity = "exam.similarity"

// Topic outbox: so xong / giảng viên đã xem một cặp → xoá cache "Hôm nay" của Giảng viên.
const (
	TopicSimilarityDone     = "exam.similarity_done"
	TopicSimilarityReviewed = "exam.similarity_reviewed"
)

// Tham số so (research mục 4): giữ cặp ≥ 0,40 và ≥ 10 dấu vân tay chung; tối đa 200 cặp; cờ = max(SIMILARITY_MIN, mean + 3 × stddev).
const (
	simMinScore  = 400
	simMinShared = 10
	simTop       = 200
	simSigmas    = 3
	noteMax      = 500
)

type similarityPayload struct {
	CourseID uuid.UUID `json:"course_id"`
	ExamID   uuid.UUID `json:"exam_id"`
}

func (c IntegrityConfig) similarityMin() int {
	if c.SimilarityMinPermille <= 0 {
		return 600
	}
	return c.SimilarityMinPermille
}

// EnqueueSimilarity: `POST …/similarity/run` — Giảng viên chạy lại. Bài phải đã đóng (CLOSED / PUBLISHED) và có câu code.
func (s *Service) EnqueueSimilarity(ctx context.Context, owner, courseID, examID uuid.UUID) (uuid.UUID, error) {
	q := store.New(s.Pool)
	e, err := q.ExamGet(ctx, store.ExamGetParams{CourseID: courseID, ID: examID})
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, notFound()
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("exam: đọc bài thi: %w", err)
	}
	if e.Kind == store.ExamKindMCQ {
		return uuid.Nil, fieldErr("exam", "NO_CODE_ITEMS", "Bài thi này không có câu lập trình để so độ giống.")
	}
	if eff := EffectiveStatus(string(e.Status), e.OpensAt, e.ClosesAt, s.now()); eff != StatusClosed && eff != StatusPublished {
		return uuid.Nil, apierr.New(http.StatusConflict, apierr.Conflict).WithDetails(map[string]any{"reason": "exam_not_closed"})
	}
	j, err := s.Jobs.Enqueue(ctx, owner, KindSimilarity, similarityPayload{CourseID: courseID, ExamID: examID})
	if err != nil {
		return uuid.Nil, fmt.Errorf("exam: xếp việc so độ giống: %w", err)
	}
	return j.ID, nil
}

// EnqueueDueSimilarity (bộ lập lịch, mỗi nhịp): xếp việc cho các bài code đã đóng và đã nộp hết mà chưa từng so. Dấu vết `exam.similarity.queued` ở `audit_log` giữ cho một bài chỉ được xếp một lần tự động.
func (s *Service) EnqueueDueSimilarity(ctx context.Context) (int, error) {
	q := store.New(s.Pool)
	due, err := q.SimilarityDueExams(ctx)
	if err != nil {
		return 0, fmt.Errorf("exam: bài cần so độ giống: %w", err)
	}
	n := 0
	for _, d := range due {
		if err := audit(ctx, q, d.CourseID, d.CreatedBy, "exam", d.ID, "exam.similarity.queued", nil, nil); err != nil {
			return n, err
		}
		if _, err := s.Jobs.Enqueue(ctx, d.CreatedBy, KindSimilarity, similarityPayload{CourseID: d.CourseID, ExamID: d.ID}); err != nil {
			return n, fmt.Errorf("exam: xếp việc so độ giống: %w", err)
		}
		n++
	}
	return n, nil
}

// SimilarityResult là `jobs.result` của `exam.similarity`.
type SimilarityResult struct {
	RunID   uuid.UUID `json:"run_id"`
	Docs    int       `json:"docs"`
	Pairs   int       `json:"pairs"`
	Flagged int       `json:"flagged"`
}

// similarityJob đọc bản nộp cuối của mỗi (lượt, câu), so từng bài bằng chỉ mục ngược và lưu tối đa 200 cặp / bài vào `similarity_reports` cùng một `run_id` mới.
// Mã nguồn chỉ nằm trong bộ nhớ của tiến trình worker trong lúc tính; không ra khỏi hệ thống.
func (w *Worker) similarityJob(ctx context.Context, j jobs.JobCtx) (any, error) {
	var p similarityPayload
	if err := json.Unmarshal(j.Payload, &p); err != nil {
		return nil, fmt.Errorf("exam: payload similarity: %w", err)
	}
	q := store.New(w.Pool)
	subs, err := q.SimilarityFinalSubmissions(ctx, store.SimilarityFinalSubmissionsParams{CourseID: p.CourseID, ExamID: p.ExamID})
	if err != nil {
		return nil, fmt.Errorf("exam: bản nộp cuối: %w", err)
	}
	j.Progress(20)
	byProblem := map[uuid.UUID][]store.SimilarityFinalSubmissionsRow{}
	var probs []string
	for _, s := range subs {
		if _, ok := byProblem[s.ProblemID]; !ok {
			probs = append(probs, s.ProblemID.String())
		}
		byProblem[s.ProblemID] = append(byProblem[s.ProblemID], s)
	}
	starters := map[uuid.UUID]map[string]string{}
	if len(probs) > 0 {
		rows, err := q.SimilarityStarters(ctx, store.SimilarityStartersParams{CourseID: p.CourseID, Ids: probs})
		if err != nil {
			return nil, fmt.Errorf("exam: mã khởi đầu: %w", err)
		}
		for _, r := range rows {
			m := map[string]string{}
			_ = json.Unmarshal(r.StarterCode, &m)
			starters[r.QuestionID] = m
		}
	}
	run := uuid.New()
	min := w.Svc.Integrity.similarityMin()
	type rowOut struct {
		ProblemID uuid.UUID `json:"problem_id"`
		SA        uuid.UUID `json:"sa"`
		SB        uuid.UUID `json:"sb"`
		AA        uuid.UUID `json:"aa"`
		AB        uuid.UUID `json:"ab"`
		Score     int       `json:"score"`
		Shared    int       `json:"shared"`
		Flagged   bool      `json:"flagged"`
	}
	var out []rowOut
	res := SimilarityResult{RunID: run, Docs: len(subs)}
	for prob, list := range byProblem {
		docs := make([]similarity.Doc, 0, len(list))
		meta := map[string]store.SimilarityFinalSubmissionsRow{}
		for _, s := range list {
			docs = append(docs, similarity.Build(s.ID.String(), s.StudentID.String(), s.Source, starters[prob][s.Language]))
			meta[s.ID.String()] = s
		}
		r := similarity.Run(docs, similarity.Options{MinScore: simMinScore, MinShared: simMinShared, Top: simTop, FlagMin: min, FlagSigmas: simSigmas})
		for _, pr := range r.Pairs {
			a, b := meta[pr.A], meta[pr.B]
			fl := r.Flagged[[2]string{pr.A, pr.B}]
			if bytes.Compare(a.AttemptID[:], b.AttemptID[:]) > 0 { // ràng buộc `attempt_a < attempt_b` (mỗi cặp một dòng)
				a, b = b, a
			}
			out = append(out, rowOut{ProblemID: prob, SA: a.ID, SB: b.ID, AA: a.AttemptID, AB: b.AttemptID, Score: pr.Score, Shared: pr.Shared, Flagged: fl})
			if fl {
				res.Flagged++
			}
		}
	}
	res.Pairs = len(out)
	j.Progress(80)
	// ghi theo thứ tự điểm TĂNG dần không cần thiết: danh sách sắp theo (score, id) khi đọc
	raw, _ := json.Marshal(out)
	tx, err := w.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("exam: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	tq := store.New(tx)
	if len(out) > 0 {
		if err := tq.SimilarityInsert(ctx, store.SimilarityInsertParams{CourseID: p.CourseID, ExamID: p.ExamID, RunID: run, Rows: raw}); err != nil {
			return nil, fmt.Errorf("exam: ghi báo cáo độ giống: %w", err)
		}
	}
	creator, _ := tq.ExamGet(ctx, store.ExamGetParams{CourseID: p.CourseID, ID: p.ExamID})
	if err := audit(ctx, tq, p.CourseID, creator.CreatedBy, "exam", p.ExamID, "exam.similarity.run", nil, map[string]any{"run_id": run, "pairs": res.Pairs, "flagged": res.Flagged}); err != nil {
		return nil, err
	}
	if _, err := outbox.Write(ctx, tx, TopicSimilarityDone, map[string]any{"course_id": p.CourseID, "exam_id": p.ExamID}); err != nil {
		return nil, fmt.Errorf("exam: outbox: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("exam: commit: %w", err)
	}
	return res, nil
}

// ---- đọc / xử lý (Giảng viên) ------------------------------------------------------------------------------------------------

// SimilarityStudent là một bên của cặp (Giảng viên thấy tên).
type SimilarityStudent struct {
	AttemptID uuid.UUID `json:"attempt_id"`
	Name      string    `json:"name"`
}

// SimilarityView là một cặp nghi giống nhau — tín hiệu để tham khảo, không phải kết luận.
type SimilarityView struct {
	ID                 uuid.UUID         `json:"id"`
	ProblemID          uuid.UUID         `json:"problem_id"`
	ProblemTitle       string            `json:"problem_title"`
	RunID              uuid.UUID         `json:"run_id"`
	A                  SimilarityStudent `json:"a"`
	B                  SimilarityStudent `json:"b"`
	Score              decimal.Decimal   `json:"score"`
	SharedFingerprints int               `json:"shared_fingerprints"`
	Flagged            bool              `json:"flagged"`
	ReviewState        string            `json:"review_state"`
	Note               *string           `json:"note"`
	ReviewedAt         *time.Time        `json:"reviewed_at"`
	CreatedAt          time.Time         `json:"created_at"`
}

// SimilaritySide là mã của một bên kèm các dòng khớp (tô sáng).
type SimilaritySide struct {
	Language   string `json:"language"`
	Source     string `json:"source"`
	MatchLines []int  `json:"match_lines"`
}

// SimilarityDetail là một cặp kèm hai mã cạnh nhau.
type SimilarityDetail struct {
	Pair SimilarityView `json:"pair"`
	A    SimilaritySide `json:"a"`
	B    SimilaritySide `json:"b"`
}

// SimilarityCursor là con trỏ danh sách: (điểm, id) — danh sách sắp điểm cao trước.
type SimilarityCursor struct {
	Score decimal.Decimal
	ID    uuid.UUID
}

// ListSimilarity: `GET …/similarity?run&flagged` — mặc định bản chạy mới nhất; chưa từng chạy → rỗng.
func (s *Service) ListSimilarity(ctx context.Context, courseID, examID uuid.UUID, run *uuid.UUID, flaggedOnly bool, cur *SimilarityCursor, fetch int) ([]SimilarityView, error) {
	q := store.New(s.Pool)
	if _, err := q.ExamGet(ctx, store.ExamGetParams{CourseID: courseID, ID: examID}); errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound()
	} else if err != nil {
		return nil, fmt.Errorf("exam: đọc bài thi: %w", err)
	}
	var runID uuid.UUID
	if run != nil {
		runID = *run
	} else {
		r, err := q.SimilarityLatestRun(ctx, store.SimilarityLatestRunParams{CourseID: courseID, ExamID: examID})
		if errors.Is(err, pgx.ErrNoRows) {
			return []SimilarityView{}, nil
		}
		if err != nil {
			return nil, fmt.Errorf("exam: bản chạy mới nhất: %w", err)
		}
		runID = r
	}
	arg := store.SimilarityListParams{CourseID: courseID, ExamID: examID, RunID: runID, FlaggedOnly: flaggedOnly, RowLimit: int32(fetch)} //nolint:gosec // ≤ 101
	if cur != nil {
		arg.CursorScore = decimal.NullDecimal{Decimal: cur.Score, Valid: true}
		arg.CursorID = &cur.ID
	}
	rows, err := q.SimilarityList(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("exam: danh sách độ giống: %w", err)
	}
	out := make([]SimilarityView, 0, len(rows))
	for _, r := range rows {
		out = append(out, SimilarityView{ID: r.ID, ProblemID: r.ProblemID, ProblemTitle: r.ProblemTitle, RunID: r.RunID,
			A: SimilarityStudent{AttemptID: r.AttemptA, Name: r.NameA}, B: SimilarityStudent{AttemptID: r.AttemptB, Name: r.NameB},
			Score: r.Score, SharedFingerprints: int(r.SharedFingerprints), Flagged: r.Flagged, ReviewState: string(r.ReviewState), Note: r.Note, ReviewedAt: r.ReviewedAt, CreatedAt: r.CreatedAt})
	}
	return out, nil
}

// GetSimilarity: `GET …/similarity/{id}` — một cặp kèm hai mã và các dòng khớp. Không có (hoặc thuộc bài khác) → 404.
func (s *Service) GetSimilarity(ctx context.Context, courseID, examID, id uuid.UUID) (SimilarityDetail, error) {
	q := store.New(s.Pool)
	r, err := q.SimilarityGet(ctx, store.SimilarityGetParams{CourseID: courseID, ExamID: examID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return SimilarityDetail{}, notFound()
	}
	if err != nil {
		return SimilarityDetail{}, fmt.Errorf("exam: đọc cặp: %w", err)
	}
	sa, err := q.SimilaritySource(ctx, store.SimilaritySourceParams{CourseID: courseID, ID: r.SubmissionA})
	if err != nil {
		return SimilarityDetail{}, fmt.Errorf("exam: đọc mã A: %w", err)
	}
	sb, err := q.SimilaritySource(ctx, store.SimilaritySourceParams{CourseID: courseID, ID: r.SubmissionB})
	if err != nil {
		return SimilarityDetail{}, fmt.Errorf("exam: đọc mã B: %w", err)
	}
	starter := ""
	if st, err := q.SimilarityStarters(ctx, store.SimilarityStartersParams{CourseID: courseID, Ids: []string{r.ProblemID.String()}}); err == nil && len(st) == 1 {
		m := map[string]string{}
		_ = json.Unmarshal(st[0].StarterCode, &m)
		starter = m[sa.Language]
	}
	la, lb := similarity.MatchLines(sa.Source, sb.Source, starter)
	v := SimilarityView{ID: r.ID, ProblemID: r.ProblemID, ProblemTitle: r.ProblemTitle, RunID: r.RunID,
		A: SimilarityStudent{AttemptID: r.AttemptA, Name: r.NameA}, B: SimilarityStudent{AttemptID: r.AttemptB, Name: r.NameB},
		Score: r.Score, SharedFingerprints: int(r.SharedFingerprints), Flagged: r.Flagged, ReviewState: string(r.ReviewState), Note: r.Note, ReviewedAt: r.ReviewedAt, CreatedAt: r.CreatedAt}
	return SimilarityDetail{Pair: v, A: SimilaritySide{Language: sa.Language, Source: sa.Source, MatchLines: la}, B: SimilaritySide{Language: sb.Language, Source: sb.Source, MatchLines: lb}}, nil
}

// SimilarityReviewIn là thân `PUT …/similarity/{id}/review`.
type SimilarityReviewIn struct {
	State string  `json:"state" validate:"required"`
	Note  *string `json:"note"`
}

// ReviewSimilarity: Giảng viên đánh dấu `CLEARED` (đã xem — không có vấn đề) hoặc `FOLLOW_UP` (cần trao đổi), ghi chú ≤ 500 ký tự. Không có hành động trừ điểm.
func (s *Service) ReviewSimilarity(ctx context.Context, actor, courseID, examID, id uuid.UUID, in SimilarityReviewIn) (SimilarityView, error) {
	if in.State != "CLEARED" && in.State != "FOLLOW_UP" {
		return SimilarityView{}, fieldErr("state", "enum", "Trạng thái phải là CLEARED hoặc FOLLOW_UP.")
	}
	var note *string
	if in.Note != nil {
		n := strings.TrimSpace(*in.Note)
		if utf8.RuneCountInString(n) > noteMax {
			return SimilarityView{}, fieldErr("note", "max", "Ghi chú tối đa 500 ký tự.")
		}
		if n != "" {
			note = &n
		}
	}
	err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if _, err := q.SimilarityReview(ctx, store.SimilarityReviewParams{CourseID: courseID, ExamID: examID, ID: id, State: store.SimilarityReviewState(in.State), Note: note, Reviewer: &actor, At: new(s.now())}); errors.Is(err, pgx.ErrNoRows) {
			return notFound()
		} else if err != nil {
			return fmt.Errorf("exam: ghi đánh dấu: %w", err)
		}
		if err := audit(ctx, q, courseID, actor, "similarity_report", id, "exam.similarity.review", nil, map[string]any{"state": in.State}); err != nil {
			return err
		}
		_, err := outbox.Write(ctx, tx, TopicSimilarityReviewed, map[string]any{"course_id": courseID, "exam_id": examID})
		return err
	})
	if err != nil {
		return SimilarityView{}, err
	}
	rows, err := store.New(s.Pool).SimilarityGet(ctx, store.SimilarityGetParams{CourseID: courseID, ExamID: examID, ID: id})
	if err != nil {
		return SimilarityView{}, fmt.Errorf("exam: đọc cặp: %w", err)
	}
	return SimilarityView{ID: rows.ID, ProblemID: rows.ProblemID, ProblemTitle: rows.ProblemTitle, RunID: rows.RunID,
		A: SimilarityStudent{AttemptID: rows.AttemptA, Name: rows.NameA}, B: SimilarityStudent{AttemptID: rows.AttemptB, Name: rows.NameB},
		Score: rows.Score, SharedFingerprints: int(rows.SharedFingerprints), Flagged: rows.Flagged, ReviewState: string(rows.ReviewState), Note: rows.Note, ReviewedAt: rows.ReviewedAt, CreatedAt: rows.CreatedAt}, nil
}

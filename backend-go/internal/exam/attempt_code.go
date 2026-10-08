package exam

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	goredis "github.com/redis/go-redis/v9"

	"github.com/edupilot/backend-go/internal/httpapi/apierr"
	"github.com/edupilot/backend-go/internal/judge"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/store"
)

// Giới hạn mã nguồn (SRS 4.4): `code_drafts.source` / `code_submissions.source` ≤ 65.536 byte.
const (
	maxSourceBytes = 65536
	// excerptMax: khối đầu vào / mong đợi / kết quả của WA ở `Chạy thử` cắt ≤ 2 KiB (AC7).
	excerptMax = 2048
)

// CodeConfig là cấu hình bài code (US-PE-06). Số 0 = mặc định.
type CodeConfig struct {
	RunLimit       int           // EXAM_RUN_LIMIT (10)
	RunWindow      time.Duration // EXAM_RUN_WINDOW (10 phút)
	SubmitCooldown time.Duration // EXAM_SUBMIT_COOLDOWN (15 s)
	SubmissionCap  int           // EXAM_SUBMISSION_CAP (30)
}

func (c CodeConfig) runLimit() int {
	if c.RunLimit <= 0 {
		return 10
	}
	return c.RunLimit
}

func (c CodeConfig) runWindow() time.Duration {
	if c.RunWindow <= 0 {
		return 10 * time.Minute
	}
	return c.RunWindow
}

func (c CodeConfig) submitCooldown() time.Duration {
	if c.SubmitCooldown <= 0 {
		return 15 * time.Second
	}
	return c.SubmitCooldown
}

func (c CodeConfig) submissionCap() int {
	if c.SubmissionCap <= 0 {
		return 30
	}
	return c.SubmissionCap
}

// ---- DTO (sinh viên) -----------------------------------------------------------------------------------------------------------

// DraftIn là thân `PUT …/code/{itemId}/draft`.
type DraftIn struct {
	Language string `json:"language" validate:"required"`
	Source   string `json:"source"`
	BaseRev  int    `json:"base_rev" validate:"gte=0"`
}

// DraftSaved là phản hồi lưu nháp.
type DraftSaved struct {
	Rev        int       `json:"rev"`
	SavedAt    time.Time `json:"saved_at"`
	ServerTime time.Time `json:"server_time"`
	DeadlineAt time.Time `json:"deadline_at"`
}

// CodeRunIn là thân `Chạy thử` / `Nộp lời giải`.
type CodeRunIn struct {
	Language string `json:"language" validate:"required"`
	Source   string `json:"source"`
}

// RunQueued / SubmitQueued: phản hồi 202.
type RunQueued struct {
	RunID uuid.UUID `json:"run_id"`
}

type SubmitQueued struct {
	SubmissionID uuid.UUID `json:"submission_id"`
}

// SampleResultView là kết quả MỘT test mẫu. `Input` / `Expected` / `Got` chỉ có ở `Chạy thử` khi sai kết quả (test mẫu vốn công khai).
type SampleResultView struct {
	Name     string `json:"name"`
	Verdict  string `json:"verdict"`
	TimeMS   int    `json:"time_ms"`
	MemoryKB int    `json:"memory_kb"`
	Input    string `json:"input,omitempty"`
	Expected string `json:"expected,omitempty"`
	Got      string `json:"got,omitempty"`
}

// SubmissionView là bản nộp như SINH VIÊN thấy trong giờ thi (AC9): chỉ biên dịch + test mẫu. Không có `results` của test ẩn, trọng số, verdict chung, điểm.
// `Source` chỉ có ở `GET …/submissions/{sid}` (xem lại mã của chính mình — AC10).
type SubmissionView struct {
	ID         uuid.UUID          `json:"id"`
	Status     string             `json:"status"`
	Language   string             `json:"language"`
	CreatedAt  time.Time          `json:"created_at"`
	CompileOK  *bool              `json:"compile_ok"`
	CompileLog *string            `json:"compile_log,omitempty"`
	Samples    []SampleResultView `json:"samples"`
	IsFinal    bool               `json:"is_final"`
	Source     *string            `json:"source,omitempty"`
}

// RunView là kết quả một lần `Chạy thử` (không có `is_final`, không ảnh hưởng điểm).
type RunView struct {
	ID         uuid.UUID          `json:"id"`
	Status     string             `json:"status"`
	Language   string             `json:"language"`
	CreatedAt  time.Time          `json:"created_at"`
	CompileOK  *bool              `json:"compile_ok"`
	CompileLog *string            `json:"compile_log,omitempty"`
	Samples    []SampleResultView `json:"samples"`
}

// DraftView là bản nháp CHÍNH sinh viên đã lưu, trả lại khi mở / tải lại trang.
type DraftView struct {
	Source  string    `json:"source"`
	Rev     int       `json:"rev"`
	SavedAt time.Time `json:"saved_at"`
}

// ---- kiểm điều kiện chung ------------------------------------------------------------------------------------------------------

func languageNotAllowed() *apierr.Error {
	return apierr.Validation(apierr.FieldError{Field: "language", Code: "LANGUAGE_NOT_ALLOWED", Message: "Bài này không cho phép ngôn ngữ đã chọn."})
}

func sourceTooLarge() *apierr.Error {
	return apierr.Validation(apierr.FieldError{Field: "source", Code: "max", Message: "Mã nguồn tối đa 64 KiB (65.536 byte)."})
}

// codeWrite khoá lượt, kiểm đang làm + còn trong hạn + là nơi được ghi, rồi kiểm mục là câu CODE của bài và ngôn ngữ được phép. Trả mục để dùng tiếp.
func (s *Service) codeWrite(ctx context.Context, q *store.Queries, userID, courseID, examID, attemptID, itemID, tab uuid.UUID, language string) (store.ExamAttempt, store.CodeItemOfExamRow, error) {
	now := s.now()
	a, err := s.openAttempt(ctx, q, courseID, examID, attemptID, userID, now)
	if err != nil {
		return a, store.CodeItemOfExamRow{}, err
	}
	if err := s.claimWriter(ctx, q, a, tab, now); err != nil {
		return a, store.CodeItemOfExamRow{}, err
	}
	it, err := q.CodeItemOfExam(ctx, store.CodeItemOfExamParams{CourseID: courseID, ExamID: examID, ItemID: itemID})
	if errors.Is(err, pgx.ErrNoRows) {
		return a, it, notFound()
	}
	if err != nil {
		return a, it, fmt.Errorf("exam: đọc mục code: %w", err)
	}
	if !slices.Contains(it.Languages, language) {
		return a, it, languageNotAllowed()
	}
	return a, it, nil
}

func checkSource(src string, required bool) error {
	if len(src) > maxSourceBytes {
		return sourceTooLarge()
	}
	if required && strings.TrimSpace(src) == "" {
		return apierr.Validation(apierr.FieldError{Field: "source", Code: "required", Message: "Chưa có mã nguồn để chạy."})
	}
	return nil
}

// ---- bản nháp ------------------------------------------------------------------------------------------------------------------

// SaveDraft: `PUT …/code/{itemId}/draft` — ghi bản nháp theo (lượt, câu, ngôn ngữ); `base_rev` phải bằng `rev` hiện tại (lần đầu 0), khác → 409 DRAFT_CONFLICT (không ghi đè).
func (s *Service) SaveDraft(ctx context.Context, userID, courseID, examID, attemptID, itemID, tab uuid.UUID, in DraftIn) (DraftSaved, error) {
	var out DraftSaved
	err := s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		a, _, err := s.codeWrite(ctx, q, userID, courseID, examID, attemptID, itemID, tab, in.Language)
		if err != nil {
			return err
		}
		if err := checkSource(in.Source, false); err != nil {
			return err
		}
		// ponytail: bản nháp dùng chung hạn mức lưu theo lượt với câu trắc nghiệm (240 / phút); tách hạn mức riêng nếu đo thấy lẫn.
		if err := s.rateLimit(ctx, attemptID, s.now()); err != nil {
			return err
		}
		cur := 0
		var curAt time.Time
		d, err := q.CodeDraftGet(ctx, store.CodeDraftGetParams{CourseID: courseID, AttemptID: attemptID, ItemID: itemID, Language: in.Language})
		switch {
		case err == nil:
			cur, curAt = int(d.Rev), d.UpdatedAt
		case !errors.Is(err, pgx.ErrNoRows):
			return fmt.Errorf("exam: đọc bản nháp: %w", err)
		}
		if in.BaseRev != cur {
			return apierr.New(http.StatusConflict, apierr.DraftConflict).WithDetails(map[string]any{"current_rev": cur, "updated_at": curAt})
		}
		now := s.now()
		row, err := q.CodeDraftUpsert(ctx, store.CodeDraftUpsertParams{AttemptID: attemptID, ItemID: itemID, CourseID: courseID, Language: in.Language, Source: in.Source, At: now})
		if err != nil {
			return fmt.Errorf("exam: ghi bản nháp: %w", err)
		}
		out = DraftSaved{Rev: int(row.Rev), SavedAt: row.UpdatedAt, ServerTime: now, DeadlineAt: a.DeadlineAt}
		return nil
	})
	return out, err
}

// ---- chạy thử / nộp -----------------------------------------------------------------------------------------------------------

// judgeUp: gateway chỉ ĐỌC `ep:judge:up` do worker đặt (SRS 4.4.2, góp ý #6). Redis lỗi / không có → coi như lên (không chặn sinh viên vì lỗi giám sát).
func (s *Service) judgeUp(ctx context.Context) bool {
	if s.Redis == nil {
		return true
	}
	n, err := s.Redis.Exists(ctx, appredis.Key("judge", "up")).Result()
	return err != nil || n > 0
}

func judgeUnavailable(now time.Time) *apierr.Error {
	return apierr.New(http.StatusServiceUnavailable, apierr.JudgeUnavailable).WithRetryAfter(10 + int(now.UnixNano()%21)) // 10…30 s: tránh cả lớp thử lại cùng lúc
}

// runWindowScript là cửa sổ trượt nguyên tử: xoá phần tử hết hạn; còn chỗ → thêm và trả 0; hết chỗ → trả mili-giây tới khi phần tử cũ nhất hết hạn.
const runWindowScript = `
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', ARGV[1] - ARGV[2])
if redis.call('ZCARD', KEYS[1]) >= tonumber(ARGV[3]) then
  local oldest = redis.call('ZRANGE', KEYS[1], 0, 0, 'WITHSCORES')
  return tonumber(oldest[2]) + tonumber(ARGV[2]) - tonumber(ARGV[1])
end
redis.call('ZADD', KEYS[1], ARGV[1], ARGV[4])
redis.call('PEXPIRE', KEYS[1], ARGV[2])
return 0
`

// takeRunSlot dùng một lượt trong hạn mức `Chạy thử` của sinh viên (mọi bài thi đang mở của người đó); hết lượt → 429 kèm `retry_after`.
func (s *Service) takeRunSlot(ctx context.Context, userID uuid.UUID, now time.Time) error {
	if s.Redis == nil {
		return nil
	}
	win := s.Code.runWindow()
	left, err := goredis.NewScript(runWindowScript).Run(ctx, s.Redis, []string{appredis.Key("exam", "run", userID.String())},
		now.UnixMilli(), win.Milliseconds(), s.Code.runLimit(), uuid.NewString()).Int64()
	if err != nil || left <= 0 {
		return nil // lỗi Redis: không chặn
	}
	return apierr.New(http.StatusTooManyRequests, apierr.RateLimited).WithRetryAfter(int((left + 999) / 1000))
}

func sha256Hex(src string) string {
	h := sha256.Sum256([]byte(src))
	return hex.EncodeToString(h[:])
}

// enqueue ghi bản nộp + outbox `judge.enqueue` CÙNG giao dịch (SRS 4.4.3); consumer đọc tín hiệu rồi nhận việc.
func (s *Service) enqueue(ctx context.Context, q *store.Queries, tx pgx.Tx, a store.ExamAttempt, it store.CodeItemOfExamRow, kind store.SubmissionKind, language, source string, auto bool) (uuid.UUID, error) {
	row, err := q.CodeSubmissionInsert(ctx, store.CodeSubmissionInsertParams{CourseID: a.CourseID, ExamID: a.ExamID, AttemptID: a.ID, ItemID: it.ItemID, ProblemID: it.ProblemID,
		StudentID: a.StudentID, Kind: kind, Language: language, Source: source, SourceSha256: sha256Hex(source), Auto: auto})
	if err != nil {
		return uuid.Nil, fmt.Errorf("exam: ghi bản nộp: %w", err)
	}
	if _, err := outbox.Write(ctx, tx, judge.TopicEnqueue, judge.Enqueue{SubmissionID: row.ID, Kind: string(kind)}); err != nil {
		return uuid.Nil, fmt.Errorf("exam: outbox judge: %w", err)
	}
	return row.ID, nil
}

// RunCode: `POST …/code/{itemId}/run` — chạy thử với test MẪU; không ảnh hưởng điểm. Hạn mức 10 lần / 10 phút / sinh viên.
func (s *Service) RunCode(ctx context.Context, userID, courseID, examID, attemptID, itemID, tab uuid.UUID, in CodeRunIn) (RunQueued, error) {
	var out RunQueued
	err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		a, it, err := s.codeWrite(ctx, q, userID, courseID, examID, attemptID, itemID, tab, in.Language)
		if err != nil {
			return err
		}
		if err := checkSource(in.Source, true); err != nil {
			return err
		}
		now := s.now()
		if !s.judgeUp(ctx) {
			return judgeUnavailable(now)
		}
		if err := s.takeRunSlot(ctx, userID, now); err != nil {
			return err
		}
		id, err := s.enqueue(ctx, q, tx, a, it, store.SubmissionKindRUN, in.Language, in.Source, false)
		out = RunQueued{RunID: id}
		return err
	})
	return out, err
}

// SubmitCode: `POST …/code/{itemId}/submit` — nộp lời giải; nhiều lần, tối đa 30 / bài, cách nhau ≥ 15 s. Bản cuối được tính điểm (AC8, AC11).
func (s *Service) SubmitCode(ctx context.Context, userID, courseID, examID, attemptID, itemID, tab uuid.UUID, in CodeRunIn) (SubmitQueued, error) {
	var out SubmitQueued
	err := s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		a, it, err := s.codeWrite(ctx, q, userID, courseID, examID, attemptID, itemID, tab, in.Language)
		if err != nil {
			return err
		}
		if err := checkSource(in.Source, true); err != nil {
			return err
		}
		n, err := q.CodeSubmitCount(ctx, store.CodeSubmitCountParams{CourseID: courseID, AttemptID: attemptID, ItemID: itemID})
		if err != nil {
			return fmt.Errorf("exam: đếm bản nộp: %w", err)
		}
		if int(n) >= s.Code.submissionCap() {
			return apierr.New(http.StatusConflict, apierr.SubmissionLimitReached).WithDetails(map[string]any{"cap": s.Code.submissionCap()})
		}
		if s.Redis != nil {
			key := appredis.Key("exam", "submitcd", attemptID.String(), itemID.String())
			ok, err := s.Redis.SetNX(ctx, key, "1", s.Code.submitCooldown()).Result()
			if err == nil && !ok {
				ttl, _ := s.Redis.PTTL(ctx, key).Result()
				return apierr.New(http.StatusTooManyRequests, apierr.RateLimited).WithRetryAfter(max(1, int((ttl+time.Second-1)/time.Second)))
			}
		}
		id, err := s.enqueue(ctx, q, tx, a, it, store.SubmissionKindSUBMIT, in.Language, in.Source, false)
		out = SubmitQueued{SubmissionID: id}
		return err
	})
	return out, err
}

// autoSubmitDrafts (AC11): với mỗi câu code CHƯA có `SUBMIT` nào mà bản nháp mới nhất không rỗng → tạo `SUBMIT` tự động từ đúng bản nháp đó (không quay sang ngôn ngữ khác);
// đã có `SUBMIT` thì không dùng nháp. Gọi trong giao dịch chốt lượt.
func (s *Service) autoSubmitDrafts(ctx context.Context, q *store.Queries, tx pgx.Tx, a store.ExamAttempt, items []store.ExamGradeItemsRow) error {
	for _, g := range items {
		if g.Type != store.QuestionTypeCODE {
			continue
		}
		n, err := q.CodeSubmitCount(ctx, store.CodeSubmitCountParams{CourseID: a.CourseID, AttemptID: a.ID, ItemID: g.ItemID})
		if err != nil {
			return fmt.Errorf("exam: đếm bản nộp: %w", err)
		}
		if n > 0 {
			continue
		}
		d, err := q.CodeDraftLatest(ctx, store.CodeDraftLatestParams{CourseID: a.CourseID, AttemptID: a.ID, ItemID: g.ItemID})
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return fmt.Errorf("exam: đọc bản nháp mới nhất: %w", err)
		}
		if strings.TrimSpace(d.Source) == "" {
			continue
		}
		if _, err := s.enqueue(ctx, q, tx, a, store.CodeItemOfExamRow{ItemID: g.ItemID, ProblemID: g.QuestionID}, store.SubmissionKindSUBMIT, d.Language, d.Source, true); err != nil {
			return err
		}
	}
	return nil
}

// ---- đọc kết quả -----------------------------------------------------------------------------------------------------------------

// storedResult là một phần tử của `code_submissions.results` (chỉ các trường cần cho test MẪU; không có trọng số).
type storedResult struct {
	TestID        uuid.UUID `json:"test_id"`
	Position      int       `json:"position"`
	IsSample      bool      `json:"is_sample"`
	Verdict       string    `json:"verdict"`
	TimeMS        int       `json:"time_ms"`
	MemoryKB      int       `json:"memory_kb"`
	StdoutExcerpt string    `json:"stdout_excerpt"`
}

func cutExcerpt(s string) string {
	if len(s) <= excerptMax {
		return s
	}
	s = s[:excerptMax]
	for !utf8.ValidString(s) && len(s) > 0 {
		s = s[:len(s)-1]
	}
	return s
}

// sampleResults lọc CHỈ test mẫu (an toàn thêm: test ẩn không bao giờ lọt qua đây dù `results` có chứa) và gắn tên. `detail`: kèm đầu vào / mong đợi / kết quả cho test mẫu trượt (chỉ `Chạy thử`).
func (s *Service) sampleResults(ctx context.Context, q *store.Queries, courseID, problemID uuid.UUID, raw []byte, detail bool) ([]SampleResultView, error) {
	out := []SampleResultView{}
	if len(raw) == 0 {
		return out, nil
	}
	var rs []storedResult
	if err := json.Unmarshal(raw, &rs); err != nil {
		return nil, fmt.Errorf("exam: đọc kết quả chấm: %w", err)
	}
	type sample struct{ name, input, expected string }
	byID := map[uuid.UUID]sample{}
	if len(rs) > 0 {
		rows, err := q.CodeSamplesOfProblem(ctx, store.CodeSamplesOfProblemParams{CourseID: courseID, ProblemID: problemID})
		if err != nil {
			return nil, fmt.Errorf("exam: đọc test mẫu: %w", err)
		}
		for _, r := range rows {
			byID[r.ID] = sample{name: r.Name, input: deref(r.Input), expected: deref(r.Expected)}
		}
	}
	for _, r := range rs {
		if !r.IsSample {
			continue
		}
		sm, ok := byID[r.TestID]
		v := SampleResultView{Name: sm.name, Verdict: r.Verdict, TimeMS: r.TimeMS, MemoryKB: r.MemoryKB}
		if !ok {
			v.Name = fmt.Sprintf("Test mẫu %d", r.Position)
		}
		if detail && ok && r.Verdict == "WA" {
			v.Input, v.Expected, v.Got = cutExcerpt(sm.input), cutExcerpt(sm.expected), cutExcerpt(r.StdoutExcerpt)
		}
		out = append(out, v)
	}
	return out, nil
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// GetRun: `GET …/runs/{runId}` — kết quả `Chạy thử` của CHÍNH sinh viên trong lượt này.
func (s *Service) GetRun(ctx context.Context, userID, courseID, examID, attemptID, runID uuid.UUID) (RunView, error) {
	q := store.New(s.Pool)
	row, err := q.CodeSubmissionGetOwn(ctx, store.CodeSubmissionGetOwnParams{CourseID: courseID, ExamID: examID, AttemptID: attemptID, StudentID: userID, ID: runID, Kind: store.SubmissionKindRUN})
	if errors.Is(err, pgx.ErrNoRows) {
		return RunView{}, notFound()
	}
	if err != nil {
		return RunView{}, fmt.Errorf("exam: đọc lần chạy thử: %w", err)
	}
	samples, err := s.sampleResults(ctx, q, courseID, row.ProblemID, row.Results, true)
	if err != nil {
		return RunView{}, err
	}
	return RunView{ID: row.ID, Status: string(row.Status), Language: row.Language, CreatedAt: row.CreatedAt, CompileOK: row.CompileOk, CompileLog: row.CompileLog, Samples: samples}, nil
}

// GetSubmission: `GET …/submissions/{sid}` — bản nộp của CHÍNH sinh viên kèm mã (xem lại / `Dùng lại mã này`).
func (s *Service) GetSubmission(ctx context.Context, userID, courseID, examID, attemptID, id uuid.UUID) (SubmissionView, error) {
	q := store.New(s.Pool)
	row, err := q.CodeSubmissionGetOwn(ctx, store.CodeSubmissionGetOwnParams{CourseID: courseID, ExamID: examID, AttemptID: attemptID, StudentID: userID, ID: id, Kind: store.SubmissionKindSUBMIT})
	if errors.Is(err, pgx.ErrNoRows) {
		return SubmissionView{}, notFound()
	}
	if err != nil {
		return SubmissionView{}, fmt.Errorf("exam: đọc bản nộp: %w", err)
	}
	samples, err := s.sampleResults(ctx, q, courseID, row.ProblemID, row.Results, false)
	if err != nil {
		return SubmissionView{}, err
	}
	return SubmissionView{ID: row.ID, Status: string(row.Status), Language: row.Language, CreatedAt: row.CreatedAt, CompileOK: row.CompileOk, CompileLog: row.CompileLog,
		Samples: samples, IsFinal: row.IsFinal, Source: &row.Source}, nil
}

// ListSubmissions: `GET …/code/{itemId}/submissions` — lịch sử nộp của chính mình ở một câu, mới nhất trước (cursor).
func (s *Service) ListSubmissions(ctx context.Context, userID, courseID, examID, attemptID, itemID uuid.UUID, cur *Cursor, fetch int) ([]SubmissionView, error) {
	q := store.New(s.Pool)
	if _, err := q.AttemptOwn(ctx, store.AttemptOwnParams{CourseID: courseID, ExamID: examID, ID: attemptID, StudentID: userID}); errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound()
	} else if err != nil {
		return nil, fmt.Errorf("exam: đọc lượt làm: %w", err)
	}
	arg := store.CodeSubmissionsListOwnParams{CourseID: courseID, AttemptID: attemptID, StudentID: userID, ItemID: itemID, RowLimit: int32(fetch)}
	if cur != nil {
		arg.CursorAt, arg.CursorID = &cur.At, &cur.ID
	}
	rows, err := q.CodeSubmissionsListOwn(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("exam: danh sách bản nộp: %w", err)
	}
	out := make([]SubmissionView, 0, len(rows))
	for _, r := range rows {
		samples, err := s.sampleResults(ctx, q, courseID, r.ProblemID, r.Results, false)
		if err != nil {
			return nil, err
		}
		out = append(out, SubmissionView{ID: r.ID, Status: string(r.Status), Language: r.Language, CreatedAt: r.CreatedAt, CompileOK: r.CompileOk, CompileLog: r.CompileLog, Samples: samples, IsFinal: r.IsFinal})
	}
	return out, nil
}

// attachDrafts gắn bản nháp đã lưu vào các mục code của đề (trả lại khi mở / tải lại trang): `drafts[ngôn ngữ]` + `language` = ngôn ngữ của bản lưu gần nhất (AC4).
func (s *Service) attachDrafts(ctx context.Context, q *store.Queries, a store.ExamAttempt, items []ItemView) error {
	has := false
	for _, it := range items {
		has = has || it.Code != nil
	}
	if !has {
		return nil
	}
	rows, err := q.CodeDraftsOfAttempt(ctx, store.CodeDraftsOfAttemptParams{CourseID: a.CourseID, AttemptID: a.ID})
	if err != nil {
		return fmt.Errorf("exam: bản nháp của lượt: %w", err)
	}
	for _, r := range rows { // đã sắp `updated_at` giảm dần trong từng câu
		for i := range items {
			c := items[i].Code
			if c == nil || items[i].ItemID != r.ItemID {
				continue
			}
			if c.Language == nil {
				lang := r.Language
				c.Language = &lang
			}
			c.Drafts[r.Language] = DraftView{Source: r.Source, Rev: int(r.Rev), SavedAt: r.UpdatedAt}
		}
	}
	return nil
}

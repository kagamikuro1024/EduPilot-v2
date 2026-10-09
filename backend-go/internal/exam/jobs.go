package exam

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/judge"
	"github.com/edupilot/backend-go/internal/store"
)

// Sandbox là phần của `judge.Client` mà việc nền dùng (test thay bằng bản giả).
type Sandbox interface {
	Judge(ctx context.Context, lang judge.Language, source string, p judge.Problem, tests []judge.Test) (judge.Result, error)
	Compile(ctx context.Context, lang judge.Language, source string) (judge.Compiled, error)
	Run(ctx context.Context, fileID, input string, lim judge.Limits) (judge.RunResult, error)
	DeleteFile(ctx context.Context, id string) error
}

// Worker là phía worker của gói: các việc `code.verify_reference`, `question.suggest`, `exam.similarity`, `exam.regrade`.
type Worker struct {
	Pool    *pgxpool.Pool
	Svc     *Service   // đọc nội dung test lớn
	Sandbox Sandbox    // nil ⇒ chưa có máy chấm (việc FAILED với câu tiếng Việt)
	LLM     Structurer // nil ⇒ chưa cấu hình AI
	Log     *slog.Logger
	// RetryWait: chờ giữa các lần thử lại khi cổng AI quá tải (mặc định 2 s, 5 s).
	RetryWait []time.Duration
}

// Register gắn các loại việc vào `jobs.Runner`.
func (w *Worker) Register(r *jobs.Runner) {
	r.Register(KindVerifyReference, w.verify)
	r.Register(KindSuggest, w.suggest)
	r.Register(KindSimilarity, w.similarityJob)
	r.Register(KindRegrade, w.regradeJob)
}

func errNoJudge() error {
	return &jobs.UserError{Code: "JUDGE_UNAVAILABLE", Message: "Chưa chạy được lời giải mẫu, thử lại sau."}
}

// PerTest là kết quả của một test khi chạy lời giải mẫu.
type PerTest struct {
	TestID  uuid.UUID     `json:"test_id"`
	Name    string        `json:"name"`
	Verdict judge.Verdict `json:"verdict"`
	TimeMS  int           `json:"time_ms"`
}

// VerifyResult là `jobs.result` của `code.verify_reference`.
type VerifyResult struct {
	OK           bool      `json:"ok"`
	CompileOK    bool      `json:"compile_ok"`
	CompileLog   string    `json:"compile_log,omitempty"`
	TestsVersion int       `json:"tests_version"`
	PerTest      []PerTest `json:"per_test"`
	Message      string    `json:"message,omitempty"`
}

// approvedTests đọc các test ĐÃ DUYỆT của bài theo `position` (kèm nội dung từ DB hoặc kho đối tượng).
func (w *Worker) approvedTests(ctx context.Context, courseID, qid uuid.UUID) ([]judge.Test, error) {
	rows, err := store.New(w.Pool).TestcaseList(ctx, store.TestcaseListParams{CourseID: courseID, ProblemID: qid, MaxRows: MaxTests})
	if err != nil {
		return nil, fmt.Errorf("exam: đọc test: %w", err)
	}
	var out []judge.Test
	for _, r := range rows {
		if !r.Approved {
			continue
		}
		in, err := w.Svc.ReadText(ctx, r.Input, r.InputBlobKey)
		if err != nil {
			return nil, err
		}
		ex, err := w.Svc.ReadText(ctx, r.Expected, r.ExpectedBlobKey)
		if err != nil {
			return nil, err
		}
		out = append(out, judge.Test{ID: r.ID.String(), Position: int(r.Position), IsSample: r.IsSample, Weight: int(r.Weight), Input: in, Expected: ex, Approved: true})
	}
	return out, nil
}

func toProblem(cp store.CodeProblem) judge.Problem {
	p := judge.Problem{Limits: judge.Limits{TimeMS: int(cp.TimeLimitMs), MemoryMB: int(cp.MemoryLimitMb), OutputLimitKB: int(cp.OutputLimitKb)},
		Checker: judge.CheckerKind(cp.Checker), TestsVersion: int(cp.TestsVersion)}
	if cp.FloatEps.Valid {
		p.FloatEps = cp.FloatEps.Decimal.InexactFloat64()
	}
	return p
}

// verify biên dịch lời giải mẫu rồi chạy MỌI test đã duyệt qua sandbox (không qua LLM). Mọi test AC → ghi `reference_verified_version = tests_version`
// (chỉ khi `tests_version` chưa đổi giữa chừng); ngược lại trả danh sách test lệch. Sandbox chết → việc FAILED.
func (w *Worker) verify(ctx context.Context, j jobs.JobCtx) (any, error) {
	var p verifyPayload
	if err := json.Unmarshal(j.Payload, &p); err != nil {
		return nil, fmt.Errorf("exam: payload verify: %w", err)
	}
	if w.Sandbox == nil {
		return nil, errNoJudge()
	}
	q := store.New(w.Pool)
	cp, err := q.CodeProblemGet(ctx, store.CodeProblemGetParams{CourseID: p.CourseID, QuestionID: p.QuestionID})
	if err != nil {
		return nil, fmt.Errorf("exam: đọc bài code: %w", err)
	}
	if cp.ReferenceLanguage == nil || cp.ReferenceSource == nil {
		return nil, &jobs.UserError{Code: "REFERENCE_REQUIRED", Message: "Cần nhập lời giải mẫu trước khi chạy."}
	}
	j.Progress(10)
	tests, err := w.approvedTests(ctx, p.CourseID, p.QuestionID)
	if err != nil {
		return nil, err
	}
	names := map[string]string{}
	rows, err := q.TestcaseList(ctx, store.TestcaseListParams{CourseID: p.CourseID, ProblemID: p.QuestionID, MaxRows: MaxTests})
	if err != nil {
		return nil, fmt.Errorf("exam: đọc test: %w", err)
	}
	for _, r := range rows {
		names[r.ID.String()] = r.Name
	}
	res, err := w.Sandbox.Judge(ctx, judge.Language(*cp.ReferenceLanguage), *cp.ReferenceSource, toProblem(cp), tests)
	if err != nil {
		if judge.IsSystemError(err) {
			return nil, errNoJudge()
		}
		return nil, fmt.Errorf("exam: chạy lời giải mẫu: %w", err)
	}
	if res.ConfigProblem != "" {
		return VerifyResult{TestsVersion: int(cp.TestsVersion), PerTest: []PerTest{}, Message: "Tổng trọng số của các test phải lớn hơn 0."}, nil
	}
	out := VerifyResult{OK: res.Verdict == judge.AC && res.CompileOK, CompileOK: res.CompileOK, CompileLog: res.CompileLog, TestsVersion: int(cp.TestsVersion), PerTest: []PerTest{}}
	for _, r := range res.Results {
		id, _ := uuid.Parse(r.TestID)
		out.PerTest = append(out.PerTest, PerTest{TestID: id, Name: names[r.TestID], Verdict: r.Verdict, TimeMS: r.TimeMS})
	}
	j.Progress(90)
	if out.OK {
		if _, err := q.CodeProblemMarkVerified(ctx, store.CodeProblemMarkVerifiedParams{CourseID: p.CourseID, QuestionID: p.QuestionID, TestsVersion: cp.TestsVersion}); err != nil {
			return nil, fmt.Errorf("exam: ghi cờ lời giải mẫu: %w", err)
		}
	}
	return out, nil
}

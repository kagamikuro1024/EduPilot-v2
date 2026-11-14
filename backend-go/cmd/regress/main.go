// Command regress: chạy lại ca hồi quy chấm bài (FEAT-grading-guidance M2) — CÔNG CỤ VẬN HÀNH ở môi trường đo, không phải service thường trực, không nằm trong image.
//
//	regress run    --course <id> [--limit N] [--baseline benchmarks/guidance/baseline.json] [--write-baseline path] [--e2]
//	regress export --course <id> --out /duong/dan/ngoai/repo.jsonl      (mặc định TỪ CHỐI: REGRESSION_EXPORT_ALLOWED=false)
//
// `run` đọc ca + văn bản bài NGAY TRONG DB, chấm lại qua cổng LLM (9router ở môi trường đo) với lưu ý ở phiên bản HIỆN HÀNH của bài tập, in MAE / lệch / số ca xấu đi,
// rồi so với baseline: «Có lùi» → thoát 1. CI không chạy lệnh này (cần stack test + LLM).
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/grading"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llm/llmrt"
	"github.com/edupilot/backend-go/internal/platform/config"
	appredis "github.com/edupilot/backend-go/internal/platform/redis"
	"github.com/edupilot/backend-go/internal/privacy"
	"github.com/edupilot/backend-go/internal/regress"
	"github.com/edupilot/backend-go/internal/store"
)

// say / sayf: ghi ra stdout / stderr; lỗi ghi không có cách xử lý nào hợp lý trong CLI.
func say(w io.Writer, a ...any)            { _, _ = fmt.Fprintln(w, a...) }
func sayf(w io.Writer, f string, a ...any) { _, _ = fmt.Fprintf(w, f, a...) }

func main() { os.Exit(run(os.Args[1:], os.Getenv, os.Stdout, os.Stderr)) }

func run(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		say(stderr, "dùng: regress run|export --course <id> …")
		return 2
	}
	switch args[0] {
	case "export":
		return runExport(args[1:], getenv, stdout, stderr)
	case "run":
		return runRun(args[1:], getenv, stdout, stderr)
	}
	sayf(stderr, "lệnh lạ %q\n", args[0])
	return 2
}

// ---- export ---------------------------------------------------------------------------------------------------------------------------------

// exportAllowed: REGRESSION_EXPORT_ALLOWED phải là "true" tường minh (mặc định false, kể cả khi thiếu / sai kiểu).
func exportAllowed(getenv func(string) string) bool {
	return strings.EqualFold(strings.TrimSpace(getenv("REGRESSION_EXPORT_ALLOWED")), "true")
}

func runExport(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	if !exportAllowed(getenv) {
		say(stderr, "export bị từ chối: REGRESSION_EXPORT_ALLOWED=false (ca chứa văn bản bài của sinh viên; chỉ bật khi có quyết định của quản trị viên)")
		return 3
	}
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(stderr)
	course := fs.String("course", "", "id lớp")
	out := fs.String("out", "", "tệp JSONL đầu ra (đường dẫn tuyệt đối, NGOÀI repo)")
	if fs.Parse(args) != nil {
		return 2
	}
	cid, err := uuid.Parse(*course)
	if err != nil || !filepath.IsAbs(*out) {
		say(stderr, "cần --course <uuid> và --out là đường dẫn tuyệt đối")
		return 2
	}
	if wd, _ := os.Getwd(); wd != "" && strings.HasPrefix(*out, wd+string(filepath.Separator)) {
		say(stderr, "--out phải nằm NGOÀI thư mục repo đang chạy")
		return 2
	}
	ctx := context.Background()
	d, err := connect(ctx, getenv, false)
	if err != nil {
		say(stderr, err)
		return 1
	}
	defer d.close(ctx)
	n, err := exportCases(ctx, d, cid, *out)
	if err != nil {
		say(stderr, err)
		return 1
	}
	sayf(stdout, "đã ghi %d ca (đã che PII, id băm) → %s\n", n, *out)
	return 0
}

type exportLine struct {
	ID        string         `json:"id"` // băm: không lộ id gốc
	Kind      string         `json:"kind"`
	Criterion string         `json:"criterion_id,omitempty"`
	Expected  map[string]any `json:"expected"`
	Text      string         `json:"text"` // đã qua privacy.Detect + Redact
}

func exportCases(ctx context.Context, d *deps, course uuid.UUID, out string) (int, error) {
	q := store.New(d.pool)
	rows, err := q.RegressionRunCases(ctx, store.RegressionRunCasesParams{CourseID: course, PageLimit: 5000})
	if err != nil {
		return 0, fmt.Errorf("đọc ca: %w", err)
	}
	det := &privacy.Detector{Roster: &privacy.Roster{Src: privacy.StoreRoster{Pool: d.pool}, Redis: d.rdb, Log: d.log}}
	f, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) //nolint:gosec // đường dẫn do người vận hành chỉ định (đã kiểm tuyệt đối, ngoài repo)
	if err != nil {
		return 0, fmt.Errorf("mở %s: %w", out, err)
	}
	defer func() { _ = f.Close() }()
	enc := json.NewEncoder(f)
	for _, r := range rows {
		text := ""
		if r.ExtractedText != nil {
			text = *r.ExtractedText
		}
		fs, err := det.Detect(ctx, course, text)
		if err != nil {
			return 0, fmt.Errorf("phát hiện PII: %w", err)
		}
		var exp map[string]any
		_ = json.Unmarshal(r.Expected, &exp)
		h := sha256.Sum256([]byte(r.ID.String()))
		line := exportLine{ID: hex.EncodeToString(h[:8]), Kind: r.Kind, Expected: exp, Text: privacy.Redact(text, fs)}
		if r.CriterionID != nil {
			line.Criterion = *r.CriterionID
		}
		if err := enc.Encode(line); err != nil {
			return 0, fmt.Errorf("ghi: %w", err)
		}
	}
	if _, err := q.InsertAuditLog(ctx, store.InsertAuditLogParams{CourseID: &course, Entity: "grading_regression_cases", EntityID: course.String(), Action: "regress.export", After: mustJSON(map[string]any{"cases": len(rows)})}); err != nil {
		return 0, fmt.Errorf("audit: %w", err)
	}
	return len(rows), nil
}

func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }

// ---- run ------------------------------------------------------------------------------------------------------------------------------------

type deps struct {
	pool *pgxpool.Pool
	rdb  *appredis.Client
	log  *slog.Logger
	cfg  config.Config
	rt   *llmrt.Runtime
}

func (d *deps) close(ctx context.Context) {
	if d.rt != nil {
		d.rt.Close(context.WithoutCancel(ctx))
	}
	if d.rdb != nil {
		_ = d.rdb.Close()
	}
	if d.pool != nil {
		d.pool.Close()
	}
}

func connect(ctx context.Context, getenv func(string) string, withLLM bool) (*deps, error) {
	cfg, err := config.Load(getenv, config.Worker)
	if err != nil {
		return nil, fmt.Errorf("cấu hình: %w", err)
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}
	rdb, err := appredis.New(ctx, cfg.RedisURL)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("redis: %w", err)
	}
	d := &deps{pool: pool, rdb: rdb, log: log, cfg: cfg}
	if withLLM {
		if d.rt, err = llmrt.New(ctx, cfg, pool, rdb.Client, log); err != nil {
			d.close(ctx)
			return nil, fmt.Errorf("cổng LLM: %w", err)
		}
	}
	return d, nil
}

// Report là số đo của MỘT lần chạy (chuỗi decimal; không float cho điểm).
type Report struct {
	N       int    `json:"n"`
	MAE     string `json:"mae"`     // trung bình |điểm AI mới − điểm giảng viên|
	Bias    string `json:"bias"`    // trung bình có dấu (AI mới − giảng viên)
	OldMAE  string `json:"old_mae"` // trung bình |điểm AI lúc đó − điểm giảng viên|
	Worse   int    `json:"worse"`   // số ca AI mới xa giảng viên hơn lúc đó
	Skipped int    `json:"skipped"` // ca bỏ qua (rubric đã đổi / chấm lỗi)
}

type pair struct{ old, now, teacher decimal.Decimal }

// compare gom các cặp (điểm AI lúc đó, điểm AI mới, điểm giảng viên) thành báo cáo.
func compare(ps []pair, skipped int) Report {
	r := Report{N: len(ps), Skipped: skipped, MAE: "0.00", Bias: "0.00", OldMAE: "0.00"}
	if len(ps) == 0 {
		return r
	}
	var mae, bias, old decimal.Decimal
	for _, p := range ps {
		mae = mae.Add(p.now.Sub(p.teacher).Abs())
		bias = bias.Add(p.now.Sub(p.teacher))
		old = old.Add(p.old.Sub(p.teacher).Abs())
		if p.now.Sub(p.teacher).Abs().GreaterThan(p.old.Sub(p.teacher).Abs()) {
			r.Worse++
		}
	}
	n := decimal.NewFromInt(int64(len(ps)))
	r.MAE, r.Bias, r.OldMAE = mae.Div(n).StringFixed(2), bias.Div(n).StringFixed(2), old.Div(n).StringFixed(2)
	return r
}

// regressed: «Có lùi» khi MAE tăng quá maxRel so với baseline, hoặc |lệch TB| xấu thêm quá 0,20 (SRS §9).
func regressed(now, base Report, maxRel float64) (bool, string) {
	m, b := decimal.RequireFromString(now.MAE), decimal.RequireFromString(base.MAE)
	limit := b.Mul(decimal.NewFromFloat(1 + maxRel))
	if m.GreaterThan(limit) {
		return true, fmt.Sprintf("MAE %s > %s (baseline %s × %.2f)", now.MAE, limit.StringFixed(2), base.MAE, 1+maxRel)
	}
	nb, bb := decimal.RequireFromString(now.Bias).Abs(), decimal.RequireFromString(base.Bias).Abs()
	if nb.Sub(bb).GreaterThan(decimal.RequireFromString("0.20")) {
		return true, fmt.Sprintf("|lệch TB| %s xấu hơn baseline %s quá 0,20", now.Bias, base.Bias)
	}
	return false, ""
}

func runRun(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	course := fs.String("course", "", "id lớp")
	limit := fs.Int("limit", 200, "số ca tối đa")
	baseline := fs.String("baseline", "", "baseline.json để so (thoát 1 nếu «Có lùi»)")
	write := fs.String("write-baseline", "", "ghi kết quả lần chạy này làm baseline")
	e2 := fs.Bool("e2", false, "chế độ E2: ca THREAD_* — độ tự tin của bài AI bị sửa / loại so với ngưỡng chuyển giảng viên")
	if fs.Parse(args) != nil {
		return 2
	}
	cid, err := uuid.Parse(*course)
	if err != nil {
		say(stderr, "cần --course <uuid>")
		return 2
	}
	ctx := context.Background()
	d, err := connect(ctx, getenv, !*e2)
	if err != nil {
		say(stderr, err)
		return 1
	}
	defer d.close(ctx)
	if *e2 {
		return runE2(ctx, d, cid, *limit, stdout, stderr)
	}
	rep, err := evaluate(ctx, d, cid, *limit)
	if err != nil {
		say(stderr, err)
		return 1
	}
	b, _ := json.MarshalIndent(rep, "", "  ")
	say(stdout, string(b))
	if *write != "" {
		if err := os.WriteFile(*write, append(b, '\n'), 0o644); err != nil { //nolint:gosec // baseline công khai (chỉ số)
			say(stderr, err)
			return 1
		}
	}
	if *baseline != "" {
		raw, err := os.ReadFile(*baseline)
		if err != nil {
			say(stderr, err)
			return 1
		}
		var base Report
		if err := json.Unmarshal(raw, &base); err != nil {
			say(stderr, "baseline hỏng:", err)
			return 1
		}
		if bad, why := regressed(rep, base, d.cfg.RegressionMaxRel); bad {
			say(stdout, "Có lùi:", why)
			return 1
		}
		say(stdout, "Không lùi so với baseline")
	}
	return 0
}

func evaluate(ctx context.Context, d *deps, course uuid.UUID, limit int) (Report, error) {
	q := store.New(d.pool)
	rows, err := q.RegressionRunCases(ctx, store.RegressionRunCasesParams{CourseID: course, PageLimit: int32(limit)}) //nolint:gosec // ≤ 5000
	if err != nil {
		return Report{}, fmt.Errorf("đọc ca: %w", err)
	}
	g := &grading.Grader{Pool: d.pool, LLM: d.rt.Gateway, Flags: d.rt.Guidance, Cfg: grading.DefaultEssayConfig(), Log: d.log}
	graded := map[uuid.UUID]*grading.Merged{}
	var ps []pair
	skipped := 0
	for _, r := range rows {
		var rubric []grading.RubricItem
		var exp map[string]string
		if json.Unmarshal(r.Rubric, &rubric) != nil || json.Unmarshal(r.Expected, &exp) != nil || r.CriterionID == nil || r.ExtractedText == nil || exp["rubric_hash"] != regress.RubricHash(r.Rubric) {
			skipped++ // rubric đã đổi / ca hỏng: không so được
			continue
		}
		m, ok := graded[r.SourceID]
		if !ok {
			lines, lerr := grading.GuidanceLinesAt(ctx, q, course, r.AssignmentID, r.GuidanceVersion, rubric)
			if lerr != nil {
				return Report{}, lerr
			}
			in := grading.PromptInput{Title: r.Title, Description: r.Description, Rubric: rubric, Text: grading.StripIdentityLines(*r.ExtractedText), Guidance: lines, GuidanceVersion: int(r.GuidanceVersion)}
			if r.ReferenceAnswer != nil {
				in.Reference = *r.ReferenceAnswer
			}
			res, gerr := g.GradeText(llm.WithIdentity(ctx, llm.Identity{CourseID: &course}), in)
			if gerr != nil {
				d.log.WarnContext(ctx, "regress: chấm lại lỗi", "error", gerr.Error()) // không kèm văn bản
				graded[r.SourceID] = nil
				skipped++
				continue
			}
			m = &res
			graded[r.SourceID] = m
		}
		if m == nil {
			skipped++
			continue
		}
		now, ok1 := m.Pass1.Scores[*r.CriterionID]
		oldS, e1 := decimal.NewFromString(exp["ai_score"])
		tch, e2 := decimal.NewFromString(exp["teacher_score"])
		if !ok1 || e1 != nil || e2 != nil {
			skipped++
			continue
		}
		ps = append(ps, pair{old: oldS, now: now, teacher: tch})
	}
	return compare(ps, skipped), nil
}

func runE2(ctx context.Context, d *deps, course uuid.UUID, limit int, stdout, stderr io.Writer) int {
	rows, err := store.New(d.pool).RegressionThreadCases(ctx, store.RegressionThreadCasesParams{CourseID: course, PageLimit: int32(limit)}) //nolint:gosec // ≤ 5000
	if err != nil {
		say(stderr, err)
		return 1
	}
	type bucket struct{ N, Confident int }
	by := map[string]*bucket{}
	for _, r := range rows {
		b := by[r.Kind]
		if b == nil {
			b = &bucket{}
			by[r.Kind] = b
		}
		b.N++
		if r.Confidence.Valid && r.Confidence.Decimal.GreaterThanOrEqual(r.EscalationThreshold) {
			b.Confident++ // bài AI bị sửa / loại nhưng vẫn đủ tự tin để KHÔNG chuyển giảng viên
		}
	}
	b, _ := json.MarshalIndent(by, "", "  ")
	say(stdout, string(b))
	return 0
}

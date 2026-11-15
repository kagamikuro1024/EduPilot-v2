// Package decision là API cấu hình + quan sát của decision model (SRS FEAT-decision-model 6; D60): bật / tắt cổng, ngưỡng, thời hạn, công tắc theo lớp,
// số liệu từ llm_audit. Không chạm đường nóng (đường nóng đọc ảnh chụp trong bộ nhớ của internal/llm). Chỉ ADMIN (kiểm ở handler).
package decision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/trace"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/llmconfig"
	"github.com/edupilot/backend-go/internal/store"
)

// Khoảng hợp lệ của ngưỡng từng cổng (AC4 US-DM-02) và của thời hạn.
var thresholdRange = map[llm.DecideGate][2]string{ //nolint:gochecknoglobals // bảng bất biến
	llm.GateThreads:   {"0.01", "0.20"},
	llm.GateInjection: {"0.30", "0.90"},
	llm.GateRouting:   {"0.70", "0.99"},
}

const (
	minTimeoutMS = 200
	maxTimeoutMS = 5000
)

// Trạng thái provider (mã; giao diện dịch sang chữ).
const (
	StatusFake          = "fake"
	StatusConnected     = "connected"
	StatusNoKey         = "no_key"
	StatusPaused        = "paused"
	StatusNotConfigured = "not_configured"
)

// Service là dịch vụ cấu hình decision.
type Service struct {
	Pool *pgxpool.Pool
	// Snapshot: ảnh chụp cấu hình của gateway tiến trình này; làm mới ngay sau khi ghi (các bản khác nhận qua Redis).
	Snapshot *llm.DecisionSnapshot
	Redis    *goredis.Client
	// Circuit trả trạng thái cầu dao của provider ("open" | "half_open" | "closed"); nil = coi như đóng.
	Circuit func(ctx context.Context, providerID string) string
	// AllowReal = LLM_DECISION_ALLOW_REAL; EnvKey = có GEMINI_API_KEY (khoá dùng khi bản ghi provider không có khoá).
	AllowReal, EnvKey bool
	Now               func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

type actor struct{ id *uuid.UUID }

func actorFrom(ctx context.Context) (actor, error) {
	p, ok := auth.FromContext(ctx)
	if !ok || p.Role != auth.RoleAdmin {
		return actor{}, llmconfig.ErrForbidden
	}
	id, err := uuid.Parse(p.Sub)
	if err != nil {
		return actor{}, nil // token thử: actor_id null trong audit_log
	}
	return actor{id: &id}, nil
}

func invalid(field, code, msg string) error {
	return &llmconfig.ErrInvalid{Field: field, Code: code, Message: msg}
}

func parseGate(g string) (llm.DecideGate, error) {
	if dg := llm.DecideGate(g); dg.Valid() {
		return dg, nil
	}
	return "", llmconfig.ErrNotFound
}

// ---- đọc ----

// Provider là provider decision đang bật.
type Provider struct {
	ID     uuid.UUID `json:"id"`
	Model  string    `json:"model"`
	Status string    `json:"status"`
	Fake   bool      `json:"fake"`
}

// Gate là một cổng toàn hệ thống.
type Gate struct {
	Gate           string          `json:"gate"`
	Enabled        bool            `json:"enabled"`
	Threshold      decimal.Decimal `json:"threshold"`
	TimeoutMS      int             `json:"timeout_ms"`
	Version        int             `json:"version"`
	CoursesEnabled int             `json:"courses_enabled"`
	CoursesTotal   int             `json:"courses_total"`
}

// Overview là phản hồi của GET /admin/llm/decision.
type Overview struct {
	Provider    *Provider `json:"provider"`
	Gates       []Gate    `json:"gates"`
	RealAllowed bool      `json:"real_allowed"`
}

// Overview trả provider + ba cổng.
func (s *Service) Overview(ctx context.Context) (Overview, error) {
	if _, err := actorFrom(ctx); err != nil {
		return Overview{}, err
	}
	q := store.New(s.Pool)
	out := Overview{RealAllowed: s.AllowReal}
	gs, err := q.DecisionGatesWithCounts(ctx)
	if err != nil {
		return out, fmt.Errorf("decision: đọc cổng: %w", err)
	}
	for _, g := range gs {
		out.Gates = append(out.Gates, toGate(g))
	}
	p, err := s.provider(ctx, q)
	if err != nil {
		return out, err
	}
	out.Provider = p
	return out, nil
}

func toGate(g store.DecisionGatesWithCountsRow) Gate {
	return Gate{Gate: g.Gate, Enabled: g.Enabled, Threshold: g.Threshold, TimeoutMS: int(g.TimeoutMs), Version: int(g.Version),
		CoursesEnabled: int(g.CoursesEnabled), CoursesTotal: int(g.CoursesTotal)}
}

// provider dựng trạng thái provider; nil = chưa cấu hình.
func (s *Service) provider(ctx context.Context, q *store.Queries) (*Provider, error) {
	row, err := q.DecisionProviderGet(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("decision: đọc provider: %w", err)
	}
	p := &Provider{ID: row.ID, Model: row.Model, Fake: row.Model == llm.FakeDecisionModel, Status: StatusConnected}
	switch {
	case p.Fake:
		p.Status = StatusFake
	case !row.HasKey && !s.EnvKey:
		p.Status = StatusNoKey
	}
	paused := s.Circuit != nil && s.Circuit(ctx, row.ID.String()) == "open"
	if !paused {
		n, err := q.DecisionBudgetBlockedSince(ctx, s.now().Add(-time.Hour))
		if err != nil {
			return nil, fmt.Errorf("decision: đọc ngân sách: %w", err)
		}
		paused = n > 0
	}
	if paused {
		p.Status = StatusPaused
	}
	return p, nil
}

// ---- ghi cổng toàn hệ thống ----

// GateIn là thân PUT cổng; trường nil = giữ nguyên.
type GateIn struct {
	Enabled   *bool
	Threshold *decimal.Decimal
	TimeoutMS *int
}

// PutGate sửa một cổng. version cũ → *llmconfig.ErrVersionConflict; ngoài khoảng → *llmconfig.ErrInvalid; cổng lạ → ErrNotFound.
func (s *Service) PutGate(ctx context.Context, gate string, in GateIn, version int) (Gate, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return Gate{}, err
	}
	g, err := parseGate(gate)
	if err != nil {
		return Gate{}, err
	}
	if in.Threshold != nil {
		r := thresholdRange[g]
		lo, hi := decimal.RequireFromString(r[0]), decimal.RequireFromString(r[1])
		if in.Threshold.LessThan(lo) || in.Threshold.GreaterThan(hi) || !in.Threshold.Equal(in.Threshold.Round(3)) {
			return Gate{}, invalid("threshold", "OUT_OF_RANGE", fmt.Sprintf("Ngưỡng từ %s đến %s.", r[0], r[1]))
		}
	}
	if in.TimeoutMS != nil && (*in.TimeoutMS < minTimeoutMS || *in.TimeoutMS > maxTimeoutMS) {
		return Gate{}, invalid("timeout_ms", "OUT_OF_RANGE", fmt.Sprintf("Thời hạn từ %d đến %d ms.", minTimeoutMS, maxTimeoutMS))
	}
	var out Gate
	err = s.inTx(ctx, func(q *store.Queries) error {
		before, err := q.DecisionGateGet(ctx, string(g))
		if err != nil {
			return fmt.Errorf("decision: đọc cổng: %w", err)
		}
		arg := store.DecisionGateUpdateParams{Gate: string(g), Version: int32(version), Actor: a.id, Enabled: in.Enabled} //nolint:gosec // phiên bản nhỏ
		if in.Threshold != nil {
			arg.Threshold = decimal.NullDecimal{Decimal: *in.Threshold, Valid: true}
		}
		if in.TimeoutMS != nil {
			t := int32(*in.TimeoutMS) //nolint:gosec // đã kiểm khoảng
			arg.TimeoutMs = &t
		}
		after, err := q.DecisionGateUpdate(ctx, arg)
		if errors.Is(err, pgx.ErrNoRows) {
			return &llmconfig.ErrVersionConflict{Current: int(before.Version)}
		}
		if err != nil {
			return fmt.Errorf("decision: sửa cổng: %w", err)
		}
		if err := audit(ctx, q, a, nil, "decision_gate", string(g), "UPDATE", gateAudit(before.Enabled, before.Threshold, int(before.TimeoutMs)), gateAudit(after.Enabled, after.Threshold, int(after.TimeoutMs))); err != nil {
			return err
		}
		out = Gate{Gate: after.Gate, Enabled: after.Enabled, Threshold: after.Threshold, TimeoutMS: int(after.TimeoutMs), Version: int(after.Version)}
		return nil
	})
	if err != nil {
		return Gate{}, err
	}
	s.changed(ctx)
	return out, nil
}

func gateAudit(enabled bool, th decimal.Decimal, timeout int) map[string]any {
	return map[string]any{"enabled": enabled, "threshold": th.String(), "timeout_ms": timeout}
}

// ---- theo lớp ----

// CourseRow là một dòng của danh sách lớp theo cổng.
type CourseRow struct {
	CourseID  uuid.UUID `json:"course_id"`
	Name      string    `json:"name"`
	ClassCode string    `json:"class_code"`
	Enabled   bool      `json:"enabled"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"-"`
}

// Courses trả một trang lớp (keyset created_at,id; limit đã được handler kiểm ≤ 100) kèm công tắc của cổng.
func (s *Service) Courses(ctx context.Context, gate, q string, curT *time.Time, curID *uuid.UUID, limit int) ([]CourseRow, error) {
	if _, err := actorFrom(ctx); err != nil {
		return nil, err
	}
	g, err := parseGate(gate)
	if err != nil {
		return nil, err
	}
	arg := store.DecisionCoursesPageParams{Gate: string(g), Q: escapeLike(strings.TrimSpace(q)), RowLimit: int32(limit)} //nolint:gosec // limit ≤ 101
	arg.CurT, arg.CurID = curT, curID
	rows, err := store.New(s.Pool).DecisionCoursesPage(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("decision: danh sách lớp: %w", err)
	}
	out := make([]CourseRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, CourseRow{CourseID: r.ID, Name: r.Name, ClassCode: r.ClassCode, Enabled: r.Enabled, Version: int(r.Version), CreatedAt: r.CreatedAt})
	}
	return out, nil
}

func escapeLike(q string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q)
}

// CourseGate là công tắc của một cổng ở một lớp.
type CourseGate struct {
	Gate      string `json:"gate"`
	Enabled   bool   `json:"enabled"`
	Effective bool   `json:"effective"`
	Version   int    `json:"version"`
}

// PutCourseGate bật / tắt cổng cho lớp. Hàng chưa có = phiên bản 0.
func (s *Service) PutCourseGate(ctx context.Context, course uuid.UUID, gate string, enabled bool, version int) (CourseGate, error) {
	a, err := actorFrom(ctx)
	if err != nil {
		return CourseGate{}, err
	}
	g, err := parseGate(gate)
	if err != nil {
		return CourseGate{}, err
	}
	var out CourseGate
	err = s.inTx(ctx, func(q *store.Queries) error {
		ok, err := q.CourseExists(ctx, course)
		if err != nil || !ok {
			if err != nil {
				return fmt.Errorf("decision: kiểm lớp: %w", err)
			}
			return llmconfig.ErrNotFound
		}
		cur, err := q.CourseGateGet(ctx, store.CourseGateGetParams{CourseID: course, Gate: string(g)})
		exists := err == nil
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("decision: đọc công tắc lớp: %w", err)
		}
		curVersion := 0
		if exists {
			curVersion = int(cur.Version)
		}
		if curVersion != version {
			return &llmconfig.ErrVersionConflict{Current: curVersion}
		}
		var after store.CourseGateUpdateRow
		if exists {
			after, err = q.CourseGateUpdate(ctx, store.CourseGateUpdateParams{CourseID: course, Gate: string(g), Enabled: enabled, Version: int32(version), Actor: a.id}) //nolint:gosec // phiên bản nhỏ
		} else {
			var ins store.CourseGateInsertRow
			ins, err = q.CourseGateInsert(ctx, store.CourseGateInsertParams{CourseID: course, Gate: string(g), Enabled: enabled, Actor: a.id})
			after = store.CourseGateUpdateRow(ins)
		}
		if errors.Is(err, pgx.ErrNoRows) { // đua với người khác giữa lúc đọc và ghi
			return &llmconfig.ErrVersionConflict{Current: curVersion + 1}
		}
		if err != nil {
			return fmt.Errorf("decision: sửa công tắc lớp: %w", err)
		}
		if err := audit(ctx, q, a, &course, "course_decision_gate", course.String()+":"+string(g), "UPDATE",
			map[string]any{"enabled": exists && cur.Enabled}, map[string]any{"enabled": after.Enabled}); err != nil {
			return err
		}
		out = CourseGate{Gate: after.Gate, Enabled: after.Enabled, Version: int(after.Version)}
		return nil
	})
	if err != nil {
		return CourseGate{}, err
	}
	s.changed(ctx)
	out.Effective = s.effective(ctx, course, g)
	return out, nil
}

// CourseGates trả ba cổng của lớp (`enabled`, `effective`, `version`).
func (s *Service) CourseGates(ctx context.Context, course uuid.UUID) ([]CourseGate, error) {
	if _, err := actorFrom(ctx); err != nil {
		return nil, err
	}
	q := store.New(s.Pool)
	ok, err := q.CourseExists(ctx, course)
	if err != nil {
		return nil, fmt.Errorf("decision: kiểm lớp: %w", err)
	}
	if !ok {
		return nil, llmconfig.ErrNotFound
	}
	rows, err := q.CourseGatesOf(ctx, course)
	if err != nil {
		return nil, fmt.Errorf("decision: đọc công tắc lớp: %w", err)
	}
	by := map[string]store.CourseGatesOfRow{}
	for _, r := range rows {
		by[r.Gate] = r
	}
	sys, err := q.DecisionGatesWithCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("decision: đọc cổng: %w", err)
	}
	prov, err := s.provider(ctx, q)
	if err != nil {
		return nil, err
	}
	ready := s.ready(prov)
	out := make([]CourseGate, 0, len(sys))
	for _, g := range sys {
		r := by[g.Gate]
		out = append(out, CourseGate{Gate: g.Gate, Enabled: r.Enabled, Effective: g.Enabled && r.Enabled && ready, Version: int(r.Version)})
	}
	return out, nil
}

// ready: provider có và đủ điều kiện (bản thật cần LLM_DECISION_ALLOW_REAL).
func (s *Service) ready(p *Provider) bool {
	return p != nil && (p.Fake || s.AllowReal)
}

func (s *Service) effective(ctx context.Context, course uuid.UUID, g llm.DecideGate) bool {
	cgs, err := s.CourseGates(ctx, course)
	if err != nil {
		return false
	}
	for _, c := range cgs {
		if c.Gate == string(g) {
			return c.Effective
		}
	}
	return false
}

// ---- quan sát ----

// ObsGate là một hàng của bảng «Quyết định nhanh».
type ObsGate struct {
	Gate           string  `json:"gate"`
	State          string  `json:"state"` // "off" | "on"
	CoursesEnabled int     `json:"courses_enabled"`
	CoursesTotal   int     `json:"courses_total"`
	Calls          int     `json:"calls"`
	FastPct        float64 `json:"fast_pct"`
	FallbackPct    float64 `json:"fallback_pct"`
	FailClosedPct  float64 `json:"fail_closed_pct"`
	P50MS          float64 `json:"p50_ms"`
	P95MS          float64 `json:"p95_ms"`
	ErrorPct       float64 `json:"error_pct"`
	CostVND        string  `json:"cost_vnd"`
}

// Observability tổng hợp từ llm_audit (task DECIDE) trong cửa sổ 24h | 7d. Không nội dung.
func (s *Service) Observability(ctx context.Context, window string) ([]ObsGate, error) {
	if _, err := actorFrom(ctx); err != nil {
		return nil, err
	}
	d := 24 * time.Hour
	switch window {
	case "", "24h":
	case "7d":
		d = 7 * 24 * time.Hour
	default:
		return nil, invalid("window", "INVALID", "Cửa sổ là 24h hoặc 7d.")
	}
	q := store.New(s.Pool)
	gs, err := q.DecisionGatesWithCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("decision: đọc cổng: %w", err)
	}
	rows, err := q.DecisionObservability(ctx, s.now().Add(-d))
	if err != nil {
		return nil, fmt.Errorf("decision: số liệu: %w", err)
	}
	by := map[string]store.DecisionObservabilityRow{}
	for _, r := range rows {
		by[r.Gate] = r
	}
	pct := func(n, total int32) float64 {
		if total == 0 {
			return 0
		}
		return float64(n) * 100 / float64(total)
	}
	out := make([]ObsGate, 0, len(gs))
	for _, g := range gs {
		o := ObsGate{Gate: g.Gate, State: "off", CoursesEnabled: int(g.CoursesEnabled), CoursesTotal: int(g.CoursesTotal), CostVND: "0"}
		if g.Enabled {
			o.State = "on"
		}
		if r, ok := by[g.Gate]; ok {
			o.Calls = int(r.Calls)
			o.FastPct, o.FallbackPct, o.FailClosedPct, o.ErrorPct = pct(r.Fast, r.Calls), pct(r.Fallback, r.Calls), pct(r.FailClosed, r.Calls), pct(r.Errors, r.Calls)
			o.P50MS, o.P95MS, o.CostVND = r.P50Ms, r.P95Ms, r.Cost.Round(0).String()
		}
		out = append(out, o)
	}
	return out, nil
}

// ---- hạ tầng ----

func (s *Service) inTx(ctx context.Context, fn func(q *store.Queries) error) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("decision: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err := fn(store.New(s.Pool).WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// changed làm mới ảnh chụp cục bộ ngay và báo mọi gateway khác (Redis); Redis lỗi → TTL 60 s của ảnh chụp vẫn nạp.
func (s *Service) changed(ctx context.Context) {
	c := context.WithoutCancel(ctx)
	if s.Snapshot != nil {
		_ = s.Snapshot.Load(c)
	}
	if s.Redis != nil {
		_ = llm.PublishDecisionCfg(c, s.Redis)
	}
}

func audit(ctx context.Context, q *store.Queries, a actor, course *uuid.UUID, entity, entityID, action string, before, after any) error {
	arg := store.InsertAuditLogParams{CourseID: course, ActorID: a.id, Entity: entity, EntityID: entityID, Action: action}
	arg.Before, _ = json.Marshal(before)
	arg.After, _ = json.Marshal(after)
	if sc := trace.SpanContextFromContext(ctx); sc.HasTraceID() {
		t := sc.TraceID().String()
		arg.TraceID = &t
	}
	if _, err := q.InsertAuditLog(ctx, arg); err != nil {
		return fmt.Errorf("decision: ghi audit_log: %w", err)
	}
	return nil
}

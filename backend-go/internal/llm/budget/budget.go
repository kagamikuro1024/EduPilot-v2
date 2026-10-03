// Package budget: ngân sách chi phí LLM (VND, decimal) theo hai phạm vi (hệ thống, lớp) × hai kỳ (ngày, tháng, múi giờ Asia/Ho_Chi_Minh).
// Bộ đếm Redis là số nguyên 1/10.000 đ (INCRBY) — cấm float (US-P1-01 AC12). Redis mất → ngừng cộng, đối soát từ llm_audit khi Redis về.
package budget

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"

	"github.com/edupilot/backend-go/internal/llm/cost"
	"github.com/edupilot/backend-go/internal/llmconfig"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

// State là trạng thái của phạm vi tệ nhất.
type State int

// Ba trạng thái (SRS 4.3): < 80 % ok, 80–<100 % warn, ≥ 100 % exhausted.
const (
	OK State = iota
	Warn
	Exhausted
)

// Ngưỡng cảnh báo (phần trăm).
const warnPct = 80

// TopicWarn là topic outbox khi vào `warn` lần đầu mỗi kỳ (SRS 6.5).
const TopicWarn = "llm.budget.warn"

// Limit là hạn mức một phạm vi; nil = không giới hạn.
type Limit struct{ Daily, Monthly *decimal.Decimal }

// LimitSource cấp hạn mức (ngân sách hệ thống / lớp) từ cấu hình.
type LimitSource interface {
	Budget(ctx context.Context, scope string, courseID *uuid.UUID) (llmconfig.Budget, error)
}

// Manager đếm chi phí và đánh giá trạng thái.
type Manager struct {
	rdb   *goredis.Client
	src   LimitSource
	pool  *pgxpool.Pool // outbox + đối soát; nil = bỏ qua cả hai
	clk   clock.Clock
	log   *slog.Logger
	loc   *time.Location
	mu    sync.Mutex
	cache map[string]cached
	ttl   time.Duration
	pfx   string // tiền tố khoá Redis (test cô lập); mặc định rỗng
}

type cached struct {
	l  Limit
	at time.Time
}

// New dựng Manager. ttl là lưới an toàn của cache hạn mức (vô hiệu chính qua Invalidate khi cấu hình đổi).
func New(rdb *goredis.Client, src LimitSource, pool *pgxpool.Pool, clk clock.Clock, log *slog.Logger) *Manager {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		loc = time.FixedZone("ICT", 7*3600) // Việt Nam không có giờ mùa hè
	}
	if clk == nil {
		clk = clock.Real{}
	}
	return &Manager{rdb: rdb, src: src, pool: pool, clk: clk, log: log, loc: loc, cache: map[string]cached{}, ttl: 30 * time.Second}
}

// WithPrefix đặt tiền tố cho mọi khoá Redis của Manager (dùng trong test để cô lập).
func (m *Manager) WithPrefix(p string) *Manager { m.pfx = p; return m }

// Invalidate bỏ cache hạn mức (gọi khi cấu hình nạp lại).
func (m *Manager) Invalidate() { m.mu.Lock(); m.cache = map[string]cached{}; m.mu.Unlock() }

func scopeKey(scope string, courseID *uuid.UUID) string {
	if courseID == nil {
		return scope + ":-"
	}
	return scope + ":" + courseID.String()
}

func (m *Manager) limit(ctx context.Context, scope string, courseID *uuid.UUID) Limit {
	k := scopeKey(scope, courseID)
	m.mu.Lock()
	if c, ok := m.cache[k]; ok && m.clk.Now().Sub(c.at) < m.ttl {
		m.mu.Unlock()
		return c.l
	}
	m.mu.Unlock()
	b, err := m.src.Budget(ctx, scope, courseID)
	if err != nil {
		m.log.WarnContext(ctx, "không đọc được hạn mức ngân sách LLM", "error", err.Error())
		return Limit{}
	}
	l := Limit{Daily: b.Daily, Monthly: b.Monthly}
	m.mu.Lock()
	m.cache[k] = cached{l, m.clk.Now()}
	m.mu.Unlock()
	return l
}

// Khoá Redis (SRS 5.5).
func (m *Manager) dayKey(scope, id string) string {
	d := m.clk.Now().In(m.loc).Format("20060102")
	if scope == "system" {
		return m.pfx + "ep:llm:budget:system:d:" + d
	}
	return m.pfx + "ep:llm:budget:course:" + id + ":d:" + d
}

func (m *Manager) monthKey(scope, id string) string {
	mo := m.clk.Now().In(m.loc).Format("200601")
	if scope == "system" {
		return m.pfx + "ep:llm:budget:system:m:" + mo
	}
	return m.pfx + "ep:llm:budget:course:" + id + ":m:" + mo
}

func (m *Manager) warnKey(scope, id, period string) string {
	p := m.clk.Now().In(m.loc).Format("20060102")
	if period == "month" {
		p = m.clk.Now().In(m.loc).Format("200601")
	}
	if id == "" {
		id = "-"
	}
	return m.pfx + "ep:llm:budgetwarn:" + scope + ":" + id + ":" + p
}

// pct trả phần trăm (đã nhân 100, số nguyên làm tròn xuống) của spent so với limit; ok=false khi không có hạn mức.
func pct(spentUnits int64, limit *decimal.Decimal) (int64, bool) {
	if limit == nil || limit.IsZero() {
		if limit != nil && limit.IsZero() && spentUnits >= 0 {
			return 100, true // hạn mức 0 đ = đã cạn
		}
		return 0, false
	}
	lu := cost.Units(*limit)
	if lu <= 0 {
		return 100, true
	}
	return spentUnits * 100 / lu, true
}

func stateOf(p int64) State {
	switch {
	case p >= 100:
		return Exhausted
	case p >= warnPct:
		return Warn
	}
	return OK
}

// State đánh giá trạng thái = tệ nhất của (hệ thống, lớp) × (ngày, tháng). Redis lỗi → OK (không bao giờ chặn chat vì lỗi hạ tầng).
func (m *Manager) State(ctx context.Context, courseID *uuid.UUID) State {
	worst := OK
	for _, sc := range m.scopes(courseID) {
		l := m.limit(ctx, sc.scope, sc.course)
		if l.Daily == nil && l.Monthly == nil {
			continue
		}
		vals, err := m.rdb.MGet(ctx, m.dayKey(sc.scope, sc.id), m.monthKey(sc.scope, sc.id)).Result()
		if err != nil {
			return OK
		}
		for i, lim := range []*decimal.Decimal{l.Daily, l.Monthly} {
			if p, ok := pct(toInt(vals[i]), lim); ok {
				worst = max(worst, stateOf(p))
			}
		}
	}
	return worst
}

type scopeRef struct {
	scope  string
	course *uuid.UUID
	id     string
}

func (m *Manager) scopes(courseID *uuid.UUID) []scopeRef {
	out := []scopeRef{{scope: "system"}}
	if courseID != nil {
		out = append(out, scopeRef{scope: "course", course: courseID, id: courseID.String()})
	}
	return out
}

func toInt(v any) int64 {
	s, ok := v.(string)
	if !ok {
		return 0
	}
	var n int64
	_, _ = fmt.Sscan(s, &n)
	return n
}

// Charge cộng chi phí vào bộ đếm hệ thống và lớp (ngày + tháng); vào `warn` lần đầu mỗi kỳ thì ghi MỘT dòng outbox.
func (m *Manager) Charge(ctx context.Context, courseID *uuid.UUID, c decimal.Decimal) {
	units := cost.Units(c)
	if units <= 0 {
		return
	}
	for _, sc := range m.scopes(courseID) {
		dk, mk := m.dayKey(sc.scope, sc.id), m.monthKey(sc.scope, sc.id)
		pipe := m.rdb.TxPipeline()
		d := pipe.IncrBy(ctx, dk, units)
		mo := pipe.IncrBy(ctx, mk, units)
		pipe.ExpireNX(ctx, dk, 40*time.Hour)
		pipe.ExpireNX(ctx, mk, 40*24*time.Hour)
		if _, err := pipe.Exec(ctx); err != nil {
			return // Redis mất: ngừng cộng; llm_audit vẫn đúng, đối soát khi Redis về
		}
		l := m.limit(ctx, sc.scope, sc.course)
		for _, p := range []struct {
			period string
			spent  int64
			lim    *decimal.Decimal
		}{{"day", d.Val(), l.Daily}, {"month", mo.Val(), l.Monthly}} {
			if pc, ok := pct(p.spent, p.lim); ok && pc >= warnPct {
				m.warnOnce(ctx, sc, p.period, pc)
			}
		}
	}
}

func (m *Manager) warnOnce(ctx context.Context, sc scopeRef, period string, pc int64) {
	ok, err := m.rdb.SetNX(ctx, m.warnKey(sc.scope, sc.id, period), "1", 40*24*time.Hour).Result()
	if err != nil || !ok {
		return
	}
	m.log.WarnContext(ctx, "ngân sách LLM chạm ngưỡng cảnh báo", "scope", sc.scope, "period", period, "pct", pc)
	if m.pool == nil {
		return
	}
	payload := map[string]any{"scope": sc.scope, "period": period, "pct": pc}
	if sc.course != nil {
		payload["course_id"] = sc.course.String()
	}
	if err := m.writeOutbox(ctx, payload); err != nil {
		m.log.ErrorContext(ctx, "không ghi được outbox llm.budget.warn", "error", err.Error())
		m.rdb.Del(ctx, m.warnKey(sc.scope, sc.id, period)) // cho lần sau thử lại
	}
}

func (m *Manager) writeOutbox(ctx context.Context, payload map[string]any) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if _, err := outbox.Write(ctx, tx, TopicWarn, payload); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Reconcile đặt lại bộ đếm từ llm_audit (ngày + tháng hiện tại): dùng khi Redis vừa trở lại sau khi mất.
func (m *Manager) Reconcile(ctx context.Context) error {
	if m.pool == nil {
		return nil
	}
	q := store.New(m.pool)
	now := m.clk.Now().In(m.loc)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, m.loc)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, m.loc)
	refs := []scopeRef{{scope: "system"}}
	rows, err := m.pool.Query(ctx, `select distinct course_id from llm_audit where course_id is not null and created_at >= $1`, monthStart)
	if err != nil {
		return fmt.Errorf("liệt kê lớp có chi phí: %w", err)
	}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		refs = append(refs, scopeRef{scope: "course", course: &id, id: id.String()})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, sc := range refs {
		for _, p := range []struct {
			key   string
			start time.Time
			ttl   time.Duration
		}{{m.dayKey(sc.scope, sc.id), dayStart, 40 * time.Hour}, {m.monthKey(sc.scope, sc.id), monthStart, 40 * 24 * time.Hour}} {
			total, err := q.LLMCostSum(ctx, store.LLMCostSumParams{FromTs: p.start, ToTs: now.Add(time.Minute), CourseID: sc.course})
			if err != nil {
				return fmt.Errorf("tổng chi phí: %w", err)
			}
			if err := m.rdb.Set(ctx, p.key, cost.Units(total), p.ttl).Err(); err != nil {
				return err
			}
		}
	}
	return nil
}

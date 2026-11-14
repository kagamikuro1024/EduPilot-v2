// Package observability: quan sát AI cho ADMIN và bản tổng hợp một lớp cho TEACHER (US-P10-01…03).
// Chỉ đọc llm_audit / escalation_tickets / chat_messages / grading_results (không cột nội dung); nội dung prompt (đã che) chỉ ra qua Open, có lý do + audit_log.
package observability

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/store"
)

// Lỗi của Service (handler đổi sang mã HTTP).
var (
	ErrReasonRequired = errors.New("observability: cần lý do 10–200 ký tự")
	ErrNotFound       = errors.New("observability: không tìm thấy")
	ErrExpired        = errors.New("observability: nội dung đã hết hạn lưu")
	ErrBadFilter      = errors.New("observability: bộ lọc không hợp lệ")
)

const (
	minReason = 10
	maxReason = 200
	vnZone    = "Asia/Ho_Chi_Minh"
)

// Service là nghiệp vụ quan sát AI.
type Service struct {
	Pool  *pgxpool.Pool
	Clock clock.Clock
	Log   *slog.Logger
	// JaegerBaseURL rỗng → không có liên kết trace.
	JaegerBaseURL string
}

func (s *Service) now() time.Time {
	if s.Clock != nil {
		return s.Clock.Now()
	}
	return time.Now().UTC()
}

// Window là cửa sổ tổng hợp: "24h" hoặc "7d".
func Window(w string) (time.Duration, bool) {
	switch w {
	case "", "24h":
		return 24 * time.Hour, true
	case "7d":
		return 7 * 24 * time.Hour, true
	}
	return 0, false
}

// Totals là một cửa sổ tổng hợp.
type Totals struct {
	Requests      int64   `json:"requests"`
	P50Ms         float64 `json:"p50_ms"`
	P95Ms         float64 `json:"p95_ms"`
	Errors        int64   `json:"errors"`
	ErrorRate     float64 `json:"error_rate"`
	Fallbacks     int64   `json:"fallbacks"`
	FallbackRate  float64 `json:"fallback_rate"`
	Cost          string  `json:"cost"`
	PIIMasked     int64   `json:"pii_masked"`
	WindowSeconds int64   `json:"window_seconds"`
}

// DayRow là một ngày của bảng 7 ngày.
type DayRow struct {
	Day         string  `json:"day"`
	Requests    int64   `json:"requests"`
	P95Ms       float64 `json:"p95_ms"`
	Errors      int64   `json:"errors"`
	Escalations int64   `json:"escalations"`
	PIIMasked   int64   `json:"pii_masked"`
	Cost        string  `json:"cost"`
}

// TaskCost là một dòng của bảng chi phí theo tác vụ.
type TaskCost struct {
	Task      string `json:"task"`
	Requests  int64  `json:"requests"`
	Cost      string `json:"cost"`
	TokensIn  int64  `json:"tokens_in"`
	TokensOut int64  `json:"tokens_out"`
}

// AdminSummary là phản hồi của `GET /admin/observability/summary`.
type AdminSummary struct {
	Window string     `json:"window"`
	Totals Totals     `json:"totals"`
	Days   []DayRow   `json:"days"`
	Tasks  []TaskCost `json:"tasks"`
}

func rate(n, d int64) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d)
}

func totals(r store.ObsWindowSummaryRow, window time.Duration) Totals {
	return Totals{Requests: r.Requests, P50Ms: r.P50Ms, P95Ms: r.P95Ms, Errors: r.Errors, ErrorRate: rate(r.Errors, r.Requests),
		Fallbacks: r.Fallbacks, FallbackRate: rate(r.Fallbacks, r.Requests), Cost: r.Cost.String(), PIIMasked: r.PiiMasked, WindowSeconds: int64(window.Seconds())}
}

func (s *Service) days(ctx context.Context, q *store.Queries, from, to time.Time, course *uuid.UUID) ([]DayRow, error) {
	ds, err := q.ObsDailySummary(ctx, store.ObsDailySummaryParams{FromTs: from, ToTs: to, CourseID: course})
	if err != nil {
		return nil, fmt.Errorf("observability: tổng hợp ngày: %w", err)
	}
	es, err := q.ObsEscalationsDaily(ctx, store.ObsEscalationsDailyParams{FromTs: from, ToTs: to, CourseID: course})
	if err != nil {
		return nil, fmt.Errorf("observability: chuyển giảng viên theo ngày: %w", err)
	}
	esc := map[string]int64{}
	for _, e := range es {
		esc[e.Day.Time.Format(time.DateOnly)] = e.N
	}
	out := make([]DayRow, 0, len(ds))
	for _, d := range ds {
		k := d.Day.Time.Format(time.DateOnly)
		out = append(out, DayRow{Day: k, Requests: d.Requests, P95Ms: d.P95Ms, Errors: d.Errors, Escalations: esc[k], PIIMasked: d.PiiMasked, Cost: d.Cost.String()})
	}
	return out, nil
}

// dayStart là 00:00 hôm nay (múi giờ Việt Nam) của `now`.
func dayStart(now time.Time) time.Time {
	loc, err := time.LoadLocation(vnZone)
	if err != nil {
		loc = time.FixedZone("ICT", 7*3600)
	}
	n := now.In(loc)
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, loc)
}

// Summary: dải 24 giờ (hoặc 7 ngày) + bảng 7 ngày + chi phí theo tác vụ, toàn hệ thống (ADMIN).
func (s *Service) Summary(ctx context.Context, window string) (AdminSummary, error) {
	dur, ok := Window(window)
	if !ok {
		return AdminSummary{}, ErrBadFilter
	}
	if window == "" {
		window = "24h"
	}
	now := s.now()
	q := store.New(s.Pool)
	win, err := q.ObsWindowSummary(ctx, store.ObsWindowSummaryParams{FromTs: now.Add(-dur), ToTs: now.Add(time.Second)})
	if err != nil {
		return AdminSummary{}, fmt.Errorf("observability: tổng hợp: %w", err)
	}
	from7 := dayStart(now).AddDate(0, 0, -6)
	days, err := s.days(ctx, q, from7, now.Add(time.Second), nil)
	if err != nil {
		return AdminSummary{}, err
	}
	ts, err := q.ObsCostByTask(ctx, store.ObsCostByTaskParams{FromTs: from7, ToTs: now.Add(time.Second)})
	if err != nil {
		return AdminSummary{}, fmt.Errorf("observability: chi phí theo tác vụ: %w", err)
	}
	tasks := make([]TaskCost, 0, len(ts))
	for _, t := range ts {
		tasks = append(tasks, TaskCost{Task: t.Task, Requests: t.Requests, Cost: t.Cost.String(), TokensIn: t.TokensIn, TokensOut: t.TokensOut})
	}
	return AdminSummary{Window: window, Totals: totals(store.ObsWindowSummaryRow(win), dur), Days: days, Tasks: tasks}, nil
}

// CourseSummary là bản tổng hợp MỘT lớp (TEACHER của lớp, hoặc ADMIN): không bảng yêu cầu, không nội dung, không chi phí theo nhà cung cấp.
type CourseSummary struct {
	CourseID  uuid.UUID `json:"course_id"`
	Window    string    `json:"window"`
	Requests  int64     `json:"requests"`
	ErrorRate float64   `json:"error_rate"`
	// Escalations: số phiếu chuyển giảng viên; PIIMasked: số lần che thông tin.
	Escalations   int64   `json:"escalations"`
	PIIMasked     int64   `json:"pii_masked"`
	Helpful       int64   `json:"helpful"`
	NotHelpful    int64   `json:"not_helpful"`
	ThreadCorrect int64   `json:"thread_correct"`
	ThreadReject  int64   `json:"thread_reject"`
	GradedChanged int64   `json:"graded_changed"`
	GradedTotal   int64   `json:"graded_total"`
	ChangeRate    float64 `json:"change_rate"`
	MeanAbsDev    string  `json:"mean_abs_deviation"`
	Days          []DayRow `json:"days"`
}

// CourseSummary tổng hợp 7 ngày của một lớp. Lớp không tồn tại → ErrNotFound.
func (s *Service) CourseSummary(ctx context.Context, course uuid.UUID) (CourseSummary, error) {
	now := s.now()
	from7 := dayStart(now).AddDate(0, 0, -6)
	to := now.Add(time.Second)
	q := store.New(s.Pool)
	win, err := q.ObsWindowSummary(ctx, store.ObsWindowSummaryParams{FromTs: from7, ToTs: to, CourseID: &course})
	if err != nil {
		return CourseSummary{}, fmt.Errorf("observability: tổng hợp lớp: %w", err)
	}
	ql, err := q.ObsCourseQuality(ctx, store.ObsCourseQualityParams{CourseID: course, FromTs: from7, ToTs: to})
	if err != nil {
		return CourseSummary{}, fmt.Errorf("observability: chất lượng lớp: %w", err)
	}
	days, err := s.days(ctx, q, from7, to, &course)
	if err != nil {
		return CourseSummary{}, err
	}
	return CourseSummary{CourseID: course, Window: "7d", Requests: win.Requests, ErrorRate: rate(win.Errors, win.Requests), Escalations: ql.Escalations, PIIMasked: win.PiiMasked,
		Helpful: ql.Helpful, NotHelpful: ql.NotHelpful, ThreadCorrect: ql.ThreadCorrect, ThreadReject: ql.ThreadReject, GradedChanged: ql.GradedChanged, GradedTotal: ql.GradedReviewed,
		ChangeRate: rate(ql.GradedChanged, ql.GradedReviewed), MeanAbsDev: ql.MeanAbsDeviation.String(), Days: days}, nil
}

// RequestsQuery là bộ lọc bảng yêu cầu.
type RequestsQuery struct {
	Filter   string // all | problem | privacy
	CourseID *uuid.UUID
	Task     string
	From, To *time.Time
	CursorAt *time.Time
	CursorID *uuid.UUID
	Limit    int // đã kẹp ≤ 100 ở handler; lấy thêm một dòng để biết còn trang sau
}

// RequestRow là một hàng bảng yêu cầu: KHÔNG có nội dung.
type RequestRow struct {
	ID            uuid.UUID  `json:"id"`
	At            time.Time  `json:"at"`
	Task          string     `json:"task"`
	Lane          string     `json:"lane"`
	Provider      *string    `json:"provider"`
	Model         *string    `json:"model"`
	LatencyMs     int32      `json:"latency_ms"`
	Status        string     `json:"status"`
	ErrorKind     *string    `json:"error_kind"`
	Degraded      bool       `json:"degraded"`
	FallbackIndex int32      `json:"fallback_index"`
	PIIMasked     int32      `json:"pii_masked_count"`
	CourseID      *uuid.UUID `json:"course_id"`
	CourseCode    *string    `json:"course_code"`
}

// Requests trả một trang (limit+1 dòng khi còn tiếp).
func (s *Service) Requests(ctx context.Context, rq RequestsQuery) ([]RequestRow, error) {
	switch rq.Filter {
	case "":
		rq.Filter = "all"
	case "all", "problem", "privacy":
	default:
		return nil, ErrBadFilter
	}
	arg := store.ObsRequestsPageParams{Filter: rq.Filter, CourseID: rq.CourseID, FromTs: rq.From, ToTs: rq.To, CursorAt: rq.CursorAt, CursorID: rq.CursorID, PageLimit: int32(rq.Limit + 1)} //nolint:gosec // ≤ 101
	if rq.Task != "" {
		arg.Task = &rq.Task
	}
	rows, err := store.New(s.Pool).ObsRequestsPage(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("observability: bảng yêu cầu: %w", err)
	}
	out := make([]RequestRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, RequestRow{ID: r.ID, At: r.CreatedAt, Task: r.Task, Lane: r.Lane, Provider: r.Provider, Model: r.Model, LatencyMs: r.LatencyMs, Status: r.Status,
			ErrorKind: r.ErrorKind, Degraded: r.Degraded, FallbackIndex: r.FallbackIndex, PIIMasked: r.PiiMaskedCount, CourseID: r.CourseID, CourseCode: r.CourseCode})
	}
	return out, nil
}

// Detail là nội dung một yêu cầu sau khi mở: prompt ĐÃ CHE.
type Detail struct {
	ID           uuid.UUID `json:"id"`
	At           time.Time `json:"at"`
	Task         string    `json:"task"`
	Provider     *string   `json:"provider"`
	Model        *string   `json:"model"`
	Status       string    `json:"status"`
	TraceID      string    `json:"trace_id"`
	TraceURL     *string   `json:"trace_url"`
	MaskedPrompt string    `json:"masked_prompt"`
	Tools        []string  `json:"tools"`
	Confidence   *string   `json:"confidence"`
}

// ValidReason cắt khoảng trắng và kiểm 10–200 ký tự (đếm rune).
func ValidReason(reason string) (string, bool) {
	r := strings.TrimSpace(reason)
	n := utf8.RuneCountInString(r)
	return r, n >= minReason && n <= maxReason
}

// Open mở chi tiết một yêu cầu: ghi audit_log TRƯỚC (cùng giao dịch), rồi đọc payload; mỗi lần mở một dòng.
// Không có yêu cầu → ErrNotFound (không audit); hết hạn lưu → ghi audit rồi ErrExpired; lỗi ghi audit → không trả nội dung.
func (s *Service) Open(ctx context.Context, actor, id uuid.UUID, reason, traceID string) (Detail, error) {
	reason, ok := ValidReason(reason)
	if !ok {
		return Detail{}, ErrReasonRequired
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Detail{}, fmt.Errorf("observability: mở giao dịch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)
	d, err := q.ObsRequestDetail(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, ErrNotFound
	}
	if err != nil {
		return Detail{}, fmt.Errorf("observability: đọc yêu cầu: %w", err)
	}
	after, _ := json.Marshal(map[string]string{"reason": reason}) //nolint:errchkjson // map[string]string luôn mã hoá được
	arg := store.InsertAuditLogParams{CourseID: d.CourseID, ActorID: &actor, Entity: "llm_request", EntityID: id.String(), Action: "VIEW_DETAIL", After: after}
	if traceID != "" {
		arg.TraceID = &traceID
	}
	if _, err := q.InsertAuditLog(ctx, arg); err != nil {
		return Detail{}, fmt.Errorf("observability: ghi audit_log: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Detail{}, fmt.Errorf("observability: commit: %w", err)
	}
	if !d.HasPayload {
		return Detail{}, ErrExpired
	}
	out := Detail{ID: d.ID, At: d.CreatedAt, Task: d.Task, Provider: d.Provider, Model: d.Model, Status: d.Status, TraceID: d.TraceID, MaskedPrompt: derefStr(d.MaskedPrompt), Tools: toolNames(d.Tools)}
	if s.JaegerBaseURL != "" {
		u := s.JaegerBaseURL + "/trace/" + d.TraceID
		out.TraceURL = &u
	}
	if d.Confidence != "" {
		out.Confidence = &d.Confidence
	}
	return out, nil
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func toolNames(raw []byte) []string {
	out := []string{}
	_ = json.Unmarshal(raw, &out)
	if out == nil {
		return []string{}
	}
	return out
}

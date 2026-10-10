package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"

	"github.com/shopspring/decimal"
)

// ErrToolNotRegistered: chạy một tool không có trong registry của kênh (agent Threads không có tool cá nhân).
var ErrToolNotRegistered = errors.New("agent: tool chưa đăng ký ở kênh này")

// Facts là dữ kiện tool trả (đưa vào lời nhắc trong khối rào; qua hook che ở internal/llm).
type Facts map[string]any

// Block là khối dữ liệu có cấu trúc cho giao diện (sự kiện SSE `block`).
type Block struct {
	Kind string `json:"kind"`
	Data any    `json:"data"`
}

// Source là một trích dẫn / nguồn của kết quả tool.
type Source struct {
	Title string `json:"title"`
	Page  int    `json:"page,omitempty"`
}

// Result là kết quả một tool. NoData: nguồn chưa nối / không có dữ liệu → trả câu mẫu, 0 lời gọi LLM, không bịa số.
type Result struct {
	Facts   Facts
	Block   *Block
	NoData  bool
	Sources []Source
}

// Tool là một công cụ Go. KHÔNG do LLM gọi (D47): Go định tuyến theo intent. Danh tính chỉ qua TrustedContext.
type Tool interface {
	Name() string
	// ArgsType là kiểu struct của tham số (struct{} nếu không có); dùng để kiểm "không tham số danh tính" bằng phản chiếu.
	ArgsType() reflect.Type
	Run(ctx context.Context, tc TrustedContext, args json.RawMessage) (Result, error)
}

// PersonalTool đọc dữ liệu cá nhân của người trong TrustedContext; CHỈ PrivateRegistry nhận. Phương thức không xuất ngăn gói ngoài tự khai tool cá nhân.
type PersonalTool interface {
	Tool
	personalTool()
}

// SharedTool dùng được ở cả hai kênh (chỉ search_library).
type SharedTool interface {
	Tool
	sharedTool()
}

// PrivateRegistry là registry của agent chat riêng.
type PrivateRegistry struct{ tools map[string]Tool }

// NewPrivateRegistry dựng registry rỗng.
func NewPrivateRegistry() *PrivateRegistry { return &PrivateRegistry{tools: map[string]Tool{}} }

// RegisterPersonal đăng ký tool cá nhân (đăng ký trùng tên → panic: lỗi lập trình).
func (r *PrivateRegistry) RegisterPersonal(t PersonalTool) { r.add(t) }

// RegisterShared đăng ký tool dùng chung hai kênh.
func (r *PrivateRegistry) RegisterShared(t SharedTool) { r.add(t) }

func (r *PrivateRegistry) add(t Tool) {
	if _, dup := r.tools[t.Name()]; dup {
		panic("agent: tool " + t.Name() + " đã đăng ký")
	}
	r.tools[t.Name()] = t
}

// Names trả tên các tool đã đăng ký, theo thứ tự chữ cái.
func (r *PrivateRegistry) Names() []string { return names(r.tools) }

// Tools trả các tool đã đăng ký (để test phản chiếu mọi kiểu tham số).
func (r *PrivateRegistry) Tools() []Tool { return tools(r.tools) }

// Run chạy một tool; không đăng ký → ErrToolNotRegistered.
func (r *PrivateRegistry) Run(ctx context.Context, name string, tc TrustedContext, args json.RawMessage) (Result, error) {
	return run(ctx, r.tools, name, tc, args)
}

// PublicRegistry là registry của agent Threads (kênh công khai): chỉ `search_library`. KHÔNG có RegisterPersonal (kiểm bằng phản chiếu) nên
// không thể đăng ký tool cá nhân vào đây.
type PublicRegistry struct{ tools map[string]Tool }

// NewPublicRegistry dựng registry công khai với đúng một tool.
func NewPublicRegistry(lib SharedTool) *PublicRegistry {
	return &PublicRegistry{tools: map[string]Tool{lib.Name(): lib}}
}

// Names trả tên tool đã đăng ký.
func (r *PublicRegistry) Names() []string { return names(r.tools) }

// Tools trả các tool đã đăng ký.
func (r *PublicRegistry) Tools() []Tool { return tools(r.tools) }

// Run chạy tool; tool cá nhân (hay bất kỳ tên lạ) → ErrToolNotRegistered.
func (r *PublicRegistry) Run(ctx context.Context, name string, tc TrustedContext, args json.RawMessage) (Result, error) {
	return run(ctx, r.tools, name, tc, args)
}

func names(m map[string]Tool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func tools(m map[string]Tool) []Tool {
	out := make([]Tool, 0, len(m))
	for _, k := range names(m) {
		out = append(out, m[k])
	}
	return out
}

func run(ctx context.Context, m map[string]Tool, name string, tc TrustedContext, args json.RawMessage) (Result, error) {
	t, ok := m[name]
	if !ok {
		return Result{}, ErrToolNotRegistered
	}
	return t.Run(ctx, tc, args)
}

// Nguồn dữ liệu: P5 / P6 / P8 chỉ cần cài đặt. (Facts, true) = có dữ liệu; (nil, false) = chưa có. Nguồn nhận TrustedContext, không nhận id từ ngoài.
type (
	// AttendanceSource: P5.
	AttendanceSource interface {
		Attendance(ctx context.Context, tc TrustedContext) (Facts, bool, error)
	}
	// ParticipationSource: P5.
	ParticipationSource interface {
		Participation(ctx context.Context, tc TrustedContext) (Facts, bool, error)
	}
	// GradeSource: P6. WhatIf nhận giả định đã phân tích bằng Go (không LLM) và tính bằng code thuần.
	GradeSource interface {
		Summary(ctx context.Context, tc TrustedContext) (Facts, bool, error)
		WhatIf(ctx context.Context, tc TrustedContext, assumed map[string]decimal.Decimal) (Facts, bool, error)
	}
	// GradeSchemeSource: P6 — công thức điểm đã được giảng viên xác nhận.
	GradeSchemeSource interface {
		Scheme(ctx context.Context, tc TrustedContext) (Facts, bool, error)
	}
	// ScheduleSource: US-P8-03 (calendar.Service).
	ScheduleSource interface {
		ExamSchedule(ctx context.Context, tc TrustedContext) (Facts, bool, error)
		Upcoming(ctx context.Context, tc TrustedContext, days int) (Facts, bool, error)
	}
	// LibrarySource: US-P8-02 (library.Service).
	LibrarySource interface {
		Search(ctx context.Context, tc TrustedContext, query string) (Facts, []Source, bool, error)
	}
)

type noArgs struct{}

func argsOf[T any](raw json.RawMessage) (T, error) {
	var v T
	if len(raw) == 0 {
		return v, nil
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return v, fmt.Errorf("agent: tham số tool: %w", err)
	}
	return v, nil
}

// nodata đổi (facts, có?, lỗi) của một nguồn thành Result; nguồn nil → NoData.
func result(kind string, f Facts, ok bool, err error) (Result, error) {
	if err != nil {
		return Result{}, err
	}
	if !ok {
		return Result{NoData: true}, nil
	}
	return Result{Facts: f, Block: &Block{Kind: kind, Data: f}}, nil
}

type personal struct{}

func (personal) personalTool() {}

type shared struct{}

func (shared) sharedTool() {}

// get_my_attendance (P5).
type attendanceTool struct {
	personal
	Src AttendanceSource
}

// NewAttendanceTool dựng get_my_attendance; src nil → NoData.
func NewAttendanceTool(src AttendanceSource) PersonalTool { return attendanceTool{Src: src} }
func (attendanceTool) Name() string                       { return "get_my_attendance" }
func (attendanceTool) ArgsType() reflect.Type             { return reflect.TypeFor[noArgs]() }
func (t attendanceTool) Run(ctx context.Context, tc TrustedContext, _ json.RawMessage) (Result, error) {
	if t.Src == nil {
		return Result{NoData: true}, nil
	}
	f, ok, err := t.Src.Attendance(ctx, tc)
	return result("attendance", f, ok, err)
}

// get_my_participation (P5).
type participationTool struct {
	personal
	Src ParticipationSource
}

// NewParticipationTool dựng get_my_participation; src nil → NoData.
func NewParticipationTool(src ParticipationSource) PersonalTool { return participationTool{Src: src} }
func (participationTool) Name() string                          { return "get_my_participation" }
func (participationTool) ArgsType() reflect.Type                { return reflect.TypeFor[noArgs]() }
func (t participationTool) Run(ctx context.Context, tc TrustedContext, _ json.RawMessage) (Result, error) {
	if t.Src == nil {
		return Result{NoData: true}, nil
	}
	f, ok, err := t.Src.Participation(ctx, tc)
	return result("participation", f, ok, err)
}

// get_my_grade_summary (P6).
type gradeTool struct {
	personal
	Src GradeSource
}

// NewGradeSummaryTool dựng get_my_grade_summary; src nil → NoData.
func NewGradeSummaryTool(src GradeSource) PersonalTool { return gradeTool{Src: src} }
func (gradeTool) Name() string                         { return "get_my_grade_summary" }
func (gradeTool) ArgsType() reflect.Type               { return reflect.TypeFor[noArgs]() }
func (t gradeTool) Run(ctx context.Context, tc TrustedContext, _ json.RawMessage) (Result, error) {
	if t.Src == nil {
		return Result{NoData: true}, nil
	}
	f, ok, err := t.Src.Summary(ctx, tc)
	return result("grade_summary", f, ok, err)
}

// WhatIfArgs là tham số của what_if_final_grade: giả định theo thành phần. Điểm là decimal (luật 5: cấm float64 cho điểm).
type WhatIfArgs struct {
	Assumed map[string]decimal.Decimal `json:"assumed"`
}

type whatIfTool struct {
	personal
	Src GradeSource
}

// NewWhatIfTool dựng what_if_final_grade; src nil → NoData.
func NewWhatIfTool(src GradeSource) PersonalTool { return whatIfTool{Src: src} }
func (whatIfTool) Name() string                  { return "what_if_final_grade" }
func (whatIfTool) ArgsType() reflect.Type        { return reflect.TypeFor[WhatIfArgs]() }
func (t whatIfTool) Run(ctx context.Context, tc TrustedContext, raw json.RawMessage) (Result, error) {
	a, err := argsOf[WhatIfArgs](raw)
	if err != nil {
		return Result{}, err
	}
	if t.Src == nil || len(a.Assumed) == 0 {
		return Result{NoData: true}, nil
	}
	f, ok, err := t.Src.WhatIf(ctx, tc, a.Assumed)
	return result("what_if", f, ok, err)
}

// get_exam_schedule (US-P8-03).
type examTool struct {
	personal
	Src ScheduleSource
}

// NewExamScheduleTool dựng get_exam_schedule; src nil → NoData.
func NewExamScheduleTool(src ScheduleSource) PersonalTool { return examTool{Src: src} }
func (examTool) Name() string                             { return "get_exam_schedule" }
func (examTool) ArgsType() reflect.Type                   { return reflect.TypeFor[noArgs]() }
func (t examTool) Run(ctx context.Context, tc TrustedContext, _ json.RawMessage) (Result, error) {
	if t.Src == nil {
		return Result{NoData: true}, nil
	}
	f, ok, err := t.Src.ExamSchedule(ctx, tc)
	return result("exam_schedule", f, ok, err)
}

// UpcomingArgs là tham số của get_upcoming_events: số ngày (1–30; ≤ 0 → 7, kẹp 30 — US-P8-03).
type UpcomingArgs struct {
	Days int `json:"days"`
}

type upcomingTool struct {
	personal
	Src ScheduleSource
}

// NewUpcomingEventsTool dựng get_upcoming_events; src nil → NoData.
func NewUpcomingEventsTool(src ScheduleSource) PersonalTool { return upcomingTool{Src: src} }
func (upcomingTool) Name() string                           { return "get_upcoming_events" }
func (upcomingTool) ArgsType() reflect.Type                 { return reflect.TypeFor[UpcomingArgs]() }
func (t upcomingTool) Run(ctx context.Context, tc TrustedContext, raw json.RawMessage) (Result, error) {
	a, err := argsOf[UpcomingArgs](raw)
	if err != nil {
		return Result{}, err
	}
	if a.Days <= 0 {
		a.Days = 7
	}
	a.Days = min(a.Days, 30)
	if t.Src == nil {
		return Result{NoData: true}, nil
	}
	f, ok, err := t.Src.Upcoming(ctx, tc, a.Days)
	return result("upcoming_events", f, ok, err)
}

// LibraryArgs là tham số của search_library.
type LibraryArgs struct {
	Query string `json:"query"`
}

type libraryTool struct {
	shared
	Src LibrarySource
}

// NewLibraryTool dựng search_library (hai kênh); src nil → NoData.
func NewLibraryTool(src LibrarySource) SharedTool { return libraryTool{Src: src} }
func (libraryTool) Name() string                  { return "search_library" }
func (libraryTool) ArgsType() reflect.Type        { return reflect.TypeFor[LibraryArgs]() }
func (t libraryTool) Run(ctx context.Context, tc TrustedContext, raw json.RawMessage) (Result, error) {
	a, err := argsOf[LibraryArgs](raw)
	if err != nil {
		return Result{}, err
	}
	if t.Src == nil || a.Query == "" {
		return Result{NoData: true}, nil
	}
	f, srcs, ok, err := t.Src.Search(ctx, tc, a.Query)
	r, err := result("library_results", f, ok, err)
	r.Sources = srcs
	return r, err
}

// DefaultPrivateRegistry đăng ký 6 tool cá nhân + search_library với nguồn tương ứng (nil = NoData cho tới khi P5 / P6 / P8 nối).
func DefaultPrivateRegistry(att AttendanceSource, part ParticipationSource, grade GradeSource, sched ScheduleSource, lib LibrarySource) *PrivateRegistry {
	r := NewPrivateRegistry()
	r.RegisterPersonal(NewAttendanceTool(att))
	r.RegisterPersonal(NewParticipationTool(part))
	r.RegisterPersonal(NewGradeSummaryTool(grade))
	r.RegisterPersonal(NewWhatIfTool(grade))
	r.RegisterPersonal(NewExamScheduleTool(sched))
	r.RegisterPersonal(NewUpcomingEventsTool(sched))
	r.RegisterShared(NewLibraryTool(lib))
	return r
}

// Package today: trang "Hôm nay" (FEAT-course-foundation US-P2-11, SRS 4.7). Mỗi nguồn việc là một Provider; bộ gộp xếp bằng
// LUẬT CỨNG (hàm thuần, xác định) — không có LLM nào ở đây (kiểm bằng grep import `internal/llm`).
package today

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Role là vai trong JWT; quyết định DẠNG phản hồi và những Kind được phép.
type Role string

// Các vai của người xem.
const (
	RoleStudent Role = "STUDENT"
	RoleTeacher Role = "TEACHER"
	RoleTA      Role = "TA"
	RoleAdmin   Role = "ADMIN"
)

// Kind là loại việc (SRS 4.7).
type Kind string

// Kind có ở P2.
const (
	KindVerifyEmail      Kind = "VERIFY_EMAIL"
	KindJoinCode         Kind = "JOIN_CODE"
	KindJoinPending      Kind = "JOIN_PENDING"
	KindJoinRequest      Kind = "JOIN_REQUEST"
	KindEmailMismatch    Kind = "EMAIL_MISMATCH"
	KindCourseSetup      Kind = "COURSE_SETUP"
	KindLLMProviderError Kind = "LLM_PROVIDER_ERROR"
	KindLLMBudgetOut     Kind = "LLM_BUDGET_EXHAUSTED"
	KindLLMBudgetWarn    Kind = "LLM_BUDGET_WARN"
	KindCourseNoTeacher  Kind = "COURSE_NO_TEACHER"
	KindInviteExpired    Kind = "INVITE_EXPIRED"
	KindQuestionReview   Kind = "QUESTION_REVIEW"  // US-PE-03
	KindExamInProgress   Kind = "EXAM_IN_PROGRESS" // US-PE-04
	KindExamOpen         Kind = "EXAM_OPEN"
	KindExamUpcoming     Kind = "EXAM_UPCOMING"
	KindExamSimilarity   Kind = "EXAM_SIMILARITY" // US-PE-07 (chỉ Giảng viên)
	KindExamResult       Kind = "EXAM_RESULT"     // US-PE-08: sinh viên, 7 ngày kể từ công bố
	KindExamAppealReply  Kind = "EXAM_APPEAL_REPLY"
	KindExamGradeError   Kind = "EXAM_GRADE_ERROR" // Giảng viên + TA
	KindExamAppeal       Kind = "EXAM_APPEAL"      // chỉ Giảng viên
	KindExamPublishHold  Kind = "EXAM_PUBLISH_HOLD"
)

// Bậc (số nhỏ = gấp hơn; có chỗ dự trữ cho phase sau) — SRS 4.7.
const (
	TierVerifyEmail      = 10
	TierJoinCode         = 20
	TierJoinPending      = 30
	TierEmailMismatch    = 40
	TierJoinRequest      = 45
	TierCourseSetup      = 80
	TierLLMProviderError = 10
	TierLLMBudgetOut     = 20
	TierLLMBudgetWarn    = 30
	TierCourseNoTeacher  = 40
	TierInviteExpired    = 50
	TierQuestionReview   = 90
	TierExamInProgress   = 5
	TierExamOpen         = 8
	TierExamUpcoming     = 42
	TierExamSimilarity   = 55
	TierExamGradeError   = 15
	TierExamAppeal       = 22
	TierExamPublishHold  = 32
	TierExamResult       = 44
	TierExamAppealReply  = 46
)

// allowed: Kind nào được phép trong phản hồi của vai nào. Bộ gộp LỌC theo bảng này nên một Provider lỡ trả nhầm
// (hoặc một phase sau đăng ký sai) cũng không làm lộ việc dành cho staff ra cho sinh viên.
func allowed(role Role, k Kind) bool {
	switch k {
	case KindVerifyEmail, KindJoinCode, KindJoinPending, KindExamInProgress, KindExamOpen, KindExamUpcoming, KindExamResult, KindExamAppealReply:
		return role == RoleStudent
	case KindJoinRequest, KindQuestionReview, KindExamGradeError:
		return role == RoleTeacher || role == RoleTA
	case KindEmailMismatch, KindCourseSetup, KindExamSimilarity, KindExamAppeal, KindExamPublishHold:
		return role == RoleTeacher
	case KindLLMProviderError, KindLLMBudgetOut, KindLLMBudgetWarn, KindCourseNoTeacher, KindInviteExpired:
		return role == RoleAdmin
	}
	return false
}

// CourseRef là một lớp ACTIVE của người xem; RoleInCourse là vai ghi danh ("TEACHER" | "TA" | "STUDENT").
type CourseRef struct {
	ID           uuid.UUID `json:"id"`
	ClassCode    string    `json:"class_code"`
	RoleInCourse string    `json:"-"`
}

// pendingRef là yêu cầu vào lớp đang chờ duyệt của CHÍNH người xem.
type pendingRef struct {
	Course CourseRef
	Since  time.Time
}

// Viewer do gateway dựng từ DB + JWT. Courses CHỈ chứa lớp ACTIVE của chính người này; Provider không nhận id nào khác.
type Viewer struct {
	UserID        uuid.UUID
	Role          Role
	EmailVerified bool
	MaskedEmail   string
	Courses       []CourseRef
	Now           time.Time
	pending       []pendingRef
}

// Scope là một lớp hoặc tất cả lớp của người xem.
type Scope struct{ CourseID uuid.UUID }

// All = "tất cả lớp của tôi".
func (s Scope) All() bool { return s.CourseID == uuid.Nil }

// Key là phần cuối của khoá cache: `all` hoặc id lớp.
func (s Scope) Key() string {
	if s.All() {
		return "all"
	}
	return s.CourseID.String()
}

// Scoped trả các lớp của Viewer nằm trong phạm vi (theo vai ghi danh khi role != "").
func (v Viewer) Scoped(s Scope, role string) []CourseRef {
	out := make([]CourseRef, 0, len(v.Courses))
	for _, c := range v.Courses {
		if (s.All() || c.ID == s.CourseID) && (role == "" || (role == "STAFF") == (c.RoleInCourse != "STUDENT")) {
			out = append(out, c)
		}
	}
	return out
}

// Item là một việc. Overdue / Tier / AgeMinutes dùng để xếp; json theo SRS 4.7 (khóa `href` rỗng khi không có đường dẫn).
type Item struct {
	ID              string     `json:"id"`
	Kind            Kind       `json:"kind"`
	Title           string     `json:"title"`
	Reason          string     `json:"reason"`
	Urgency         string     `json:"urgency"`
	Href            string     `json:"href"`
	Course          *CourseRef `json:"course"`
	AgeMinutes      int        `json:"age_minutes,omitempty"`
	EstimateMinutes int        `json:"estimate_minutes,omitempty"`
	Steps           []Step     `json:"steps,omitempty"`
	Tier            int        `json:"-"`
	Overdue         bool       `json:"-"`
}

// Step là một bước của `COURSE_SETUP`, tự tick theo dữ liệu.
type Step struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Done  bool   `json:"done"`
	Href  string `json:"href"`
}

// Provider là một nguồn việc. Items chỉ thấy Viewer và Scope; lỗi ⇒ bộ gộp bỏ Provider này và log warn.
type Provider interface {
	Name() string
	Items(ctx context.Context, v Viewer, s Scope) ([]Item, error)
}

// Aggregator gọi mọi Provider song song (hạn Timeout mỗi Provider, mặc định 150 ms), lọc theo vai, xếp, cắt.
type Aggregator struct {
	Log     *slog.Logger
	Timeout time.Duration

	mu        sync.RWMutex
	providers []Provider
}

// MaxItems là số việc tối đa trả về; Count vẫn là tổng.
const MaxItems = 50

// Register thêm một Provider (phase sau đăng ký Provider của mình).
func (a *Aggregator) Register(p Provider) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.providers = append(a.providers, p)
}

// Names trả tên các Provider đã đăng ký (theo thứ tự đăng ký).
func (a *Aggregator) Names() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]string, len(a.providers))
	for i, p := range a.providers {
		out[i] = p.Name()
	}
	return out
}

// Collect trả (đã xếp, tối đa MaxItems) và tổng số việc. Provider lỗi / quá hạn bị bỏ; ctx của yêu cầu hết hạn ⇒ trả lỗi (504).
func (a *Aggregator) Collect(ctx context.Context, v Viewer, s Scope) ([]Item, int, error) {
	a.mu.RLock()
	ps := append([]Provider(nil), a.providers...)
	a.mu.RUnlock()
	to := a.Timeout
	if to <= 0 {
		to = 150 * time.Millisecond
	}
	results := make([][]Item, len(ps))
	var wg sync.WaitGroup
	for i, p := range ps {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pctx, cancel := context.WithTimeout(ctx, to)
			defer cancel()
			items, err := safeItems(pctx, p, v, s)
			if err != nil {
				if ctx.Err() == nil && a.Log != nil {
					a.Log.WarnContext(ctx, "today: bỏ Provider lỗi", "provider", p.Name(), "error", err.Error())
				}
				return
			}
			results[i] = items
		}()
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	var all []Item
	for _, items := range results {
		for _, it := range items {
			if allowed(v.Role, it.Kind) {
				all = append(all, it)
			}
		}
	}
	Rank(all)
	total := len(all)
	if total > MaxItems {
		all = all[:MaxItems]
	}
	return all, total, nil
}

// safeItems chặn panic của một Provider (một Provider hỏng không được làm sập trang).
func safeItems(ctx context.Context, p Provider, v Viewer, s Scope) (items []Item, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return p.Items(ctx, v, s)
}

// Rank xếp tại chỗ theo khoá cứng (Overdue giảm dần, Tier tăng dần, tuổi giảm dần, mã lớp tăng dần, id tăng dần). Xác định: cùng đầu vào, cùng thứ tự.
func Rank(items []Item) {
	sort.SliceStable(items, func(i, j int) bool { return less(items[i], items[j]) })
}

func less(a, b Item) bool {
	if a.Overdue != b.Overdue {
		return a.Overdue
	}
	if a.Tier != b.Tier {
		return a.Tier < b.Tier
	}
	if a.AgeMinutes != b.AgeMinutes {
		return a.AgeMinutes > b.AgeMinutes
	}
	if ca, cb := classOf(a), classOf(b); ca != cb {
		return ca < cb
	}
	return a.ID < b.ID
}

func classOf(i Item) string {
	if i.Course == nil {
		return ""
	}
	return i.Course.ClassCode
}

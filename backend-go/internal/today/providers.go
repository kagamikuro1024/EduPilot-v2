package today

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/edupilot/backend-go/internal/store"
)

// signalTimeout: hạn CHUNG cho mọi tín hiệu cổng AI của Admin (đi qua Redis); nhỏ hơn hạn 150 ms của Provider.
const signalTimeout = 60 * time.Millisecond

// overdueAfter: yêu cầu vào lớp chờ lâu hơn mức này nổi lên trước mọi bậc (SRS 4.7).
const overdueAfter = 48 * time.Hour

// Age đổi khoảng thời gian thành "{n} phút" (< 60 phút), "{n} giờ" (< 48 giờ), "{n} ngày"; làm tròn xuống, không âm.
func Age(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%d phút", int(d/time.Minute))
	case d < 48*time.Hour:
		return fmt.Sprintf("%d giờ", int(d/time.Hour))
	}
	return fmt.Sprintf("%d ngày", int(d/(24*time.Hour)))
}

func urgency(it Item) string {
	switch {
	case it.Overdue:
		return "overdue"
	case it.Tier <= TierJoinRequest:
		return "high"
	}
	return "normal"
}

func mk(it Item) Item {
	it.Urgency = urgency(it)
	return it
}

func courseRef(c CourseRef) *CourseRef { return &CourseRef{ID: c.ID, ClassCode: c.ClassCode} }

func ids(cs []CourseRef) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.ID.String()
	}
	return out
}

// ----- Sinh viên -----

// StudentProvider: VERIFY_EMAIL, JOIN_CODE, JOIN_PENDING dùng ảnh chụp Viewer đã nạp (không truy vấn); bài thi (US-PE-04) dùng MỘT truy vấn cho mọi lớp trong phạm vi.
// Pool nil ⇒ không có việc bài thi.
type StudentProvider struct{ Pool *pgxpool.Pool }

// Name implements Provider.
func (StudentProvider) Name() string { return "student" }

// Items implements Provider.
func (p StudentProvider) Items(ctx context.Context, v Viewer, sc Scope) ([]Item, error) {
	if v.Role != RoleStudent {
		return nil, nil
	}
	var out []Item
	if !v.EmailVerified {
		out = append(out, mk(Item{
			ID: "VERIFY_EMAIL", Kind: KindVerifyEmail, Tier: TierVerifyEmail, EstimateMinutes: 1,
			Title: "Xác minh email của bạn", Href: "/verify-email",
			Reason: fmt.Sprintf("Chưa xác minh email thì chưa vào được lớp. Kiểm tra hộp thư %s.", v.MaskedEmail),
		}))
	}
	if len(v.Courses) == 0 && len(v.pending) == 0 {
		out = append(out, mk(Item{
			ID: "JOIN_CODE", Kind: KindJoinCode, Tier: TierJoinCode, EstimateMinutes: 1,
			Title: "Nhập mã tham gia lớp", Href: "/join",
			Reason: "Bạn chưa vào lớp nào. Nhập mã do giảng viên cung cấp để bắt đầu.",
		}))
	}
	exams, err := p.examItems(ctx, v, sc)
	if err != nil {
		return nil, err
	}
	out = append(out, exams...)
	if !sc.All() { // "một lớp" = lớp ACTIVE; yêu cầu chờ duyệt của lớp khác chỉ hiện ở "tất cả lớp của tôi"
		return out, nil
	}
	for _, p := range v.pending {
		age := v.Now.Sub(p.Since)
		out = append(out, mk(Item{
			ID: "JOIN_PENDING:" + p.Course.ID.String(), Kind: KindJoinPending, Tier: TierJoinPending, Course: courseRef(p.Course),
			Title:  "Chờ giảng viên duyệt",
			Reason: fmt.Sprintf("Yêu cầu vào lớp %s đã gửi %s trước.", p.Course.ClassCode, Age(age)), AgeMinutes: int(age / time.Minute),
		}))
	}
	return out, nil
}

// examHorizon: bài sắp mở trong khoảng này mới thành việc EXAM_UPCOMING.
const examHorizon = 48 * time.Hour

// examItems: EXAM_IN_PROGRESS (đang làm dở), EXAM_OPEN (đang mở, chưa làm), EXAM_UPCOMING (mở trong 48 giờ) — SRS FEAT-weekly-exam 4.10.
func (p StudentProvider) examItems(ctx context.Context, v Viewer, sc Scope) ([]Item, error) {
	courses := v.Scoped(sc, "STUDENT")
	if p.Pool == nil || len(courses) == 0 {
		return nil, nil
	}
	rows, err := store.New(p.Pool).TodayStudentExams(ctx, store.TodayStudentExamsParams{UserID: v.UserID, CourseIds: ids(courses), Now: v.Now, Horizon: v.Now.Add(examHorizon)})
	if err != nil {
		return nil, fmt.Errorf("today: bài thi: %w", err)
	}
	byCourse := map[uuid.UUID]CourseRef{}
	for _, c := range courses {
		byCourse[c.ID] = c
	}
	zone := ictZone()
	var out []Item
	for _, r := range rows {
		if r.OpensAt == nil || r.ClosesAt == nil || r.DurationMinutes == nil {
			continue
		}
		c := byCourse[r.CourseID]
		href := fmt.Sprintf("/exams/%s/take", r.ID)
		dur := int(*r.DurationMinutes)
		switch {
		case r.AttemptStatus != nil && r.AttemptDeadline != nil: // lượt đang làm (truy vấn đã lọc IN_PROGRESS còn hạn)
			left := max(1, int((r.AttemptDeadline.Sub(v.Now)+time.Minute-1)/time.Minute))
			out = append(out, mk(Item{ID: "EXAM_IN_PROGRESS:" + r.ID.String(), Kind: KindExamInProgress, Tier: TierExamInProgress, Course: courseRef(c), Href: href, EstimateMinutes: left,
				Title: fmt.Sprintf("Bài thi %s đang làm dở", r.Title), Reason: fmt.Sprintf("Còn %d phút. Làm tiếp.", left)}))
		case !r.OpensAt.After(v.Now):
			out = append(out, mk(Item{ID: "EXAM_OPEN:" + r.ID.String(), Kind: KindExamOpen, Tier: TierExamOpen, Course: courseRef(c), Href: href, EstimateMinutes: dur,
				Title:  fmt.Sprintf("Bài thi %s đang mở", r.Title),
				Reason: fmt.Sprintf("Mở đến %s. Bạn có %d phút để làm.", r.ClosesAt.In(zone).Format("15:04 02/01"), dur)}))
		default:
			weekday := [...]string{"Chủ nhật", "Thứ Hai", "Thứ Ba", "Thứ Tư", "Thứ Năm", "Thứ Sáu", "Thứ Bảy"}
			t := r.OpensAt.In(zone)
			out = append(out, mk(Item{ID: "EXAM_UPCOMING:" + r.ID.String(), Kind: KindExamUpcoming, Tier: TierExamUpcoming, Course: courseRef(c), Href: href, EstimateMinutes: dur,
				Title:  fmt.Sprintf("%s · %s, %s", r.Title, weekday[t.Weekday()], t.Format("15:04")),
				Reason: fmt.Sprintf("Làm trong %d phút. Chuẩn bị máy tính nếu có bài lập trình.", dur)}))
		}
	}
	return out, nil
}

// ----- Giảng viên / TA -----

// StaffProvider: JOIN_REQUEST, EMAIL_MISMATCH, COURSE_SETUP — mỗi nguồn dùng MỘT truy vấn tổng hợp cho mọi lớp trong phạm vi.
type StaffProvider struct{ Pool *pgxpool.Pool }

// Name implements Provider.
func (StaffProvider) Name() string { return "staff" }

// Items implements Provider.
func (p StaffProvider) Items(ctx context.Context, v Viewer, s Scope) ([]Item, error) {
	courses := v.Scoped(s, "STAFF")
	if (v.Role != RoleTeacher && v.Role != RoleTA) || len(courses) == 0 {
		return nil, nil // không tốn truy vấn cho vai khác
	}
	q := store.New(p.Pool)
	pend, err := q.TodayStaffPending(ctx, ids(courses))
	if err != nil {
		return nil, fmt.Errorf("today: yêu cầu chờ duyệt: %w", err)
	}
	byCourse := map[uuid.UUID]store.TodayStaffPendingRow{}
	for _, r := range pend {
		byCourse[r.CourseID] = r
	}
	var out []Item
	var teaching []CourseRef
	for _, c := range courses {
		if c.RoleInCourse == "TEACHER" {
			teaching = append(teaching, c)
		}
		r, ok := byCourse[c.ID]
		if !ok {
			continue
		}
		if r.Questions > 0 { // US-PE-03: câu hỏi PENDING chờ duyệt (Giảng viên và TA)
			age := v.Now.Sub(r.QuestionsOldest)
			out = append(out, mk(Item{
				ID: "QUESTION_REVIEW:" + c.ID.String(), Kind: KindQuestionReview, Tier: TierQuestionReview, Course: courseRef(c),
				Href:  fmt.Sprintf("/questions?review_status=PENDING&course=%s", c.ID),
				Title: fmt.Sprintf("%d câu hỏi chờ duyệt · lớp %s", r.Questions, c.ClassCode), Reason: fmt.Sprintf("Cũ nhất đã chờ %s.", Age(age)), AgeMinutes: int(age / time.Minute),
			}))
		}
		href := fmt.Sprintf("/class/members?course=%s&tab=pending", c.ID)
		if r.Pending > 0 {
			age := v.Now.Sub(r.Oldest)
			out = append(out, mk(Item{
				ID: "JOIN_REQUEST:" + c.ID.String(), Kind: KindJoinRequest, Tier: TierJoinRequest, Course: courseRef(c), Href: href,
				Title:  fmt.Sprintf("%d yêu cầu vào lớp %s đang chờ duyệt", r.Pending, c.ClassCode),
				Reason: fmt.Sprintf("Cũ nhất đã chờ %s.", Age(age)), AgeMinutes: int(age / time.Minute), Overdue: age > overdueAfter,
			}))
		}
		if r.Mismatch > 0 && c.RoleInCourse == "TEACHER" {
			age := v.Now.Sub(r.MismatchOldest)
			out = append(out, mk(Item{
				ID: "EMAIL_MISMATCH:" + c.ID.String(), Kind: KindEmailMismatch, Tier: TierEmailMismatch, Course: courseRef(c), Href: href,
				Title:  fmt.Sprintf("%d yêu cầu có email chưa khớp MSSV · lớp %s", r.Mismatch, c.ClassCode),
				Reason: "Cần bạn xác nhận: email đăng ký khác email trong danh sách lớp.", AgeMinutes: int(age / time.Minute),
			}))
		}
	}
	if len(teaching) > 0 {
		setup, err := q.TodaySetup(ctx, ids(teaching))
		if err != nil {
			return nil, fmt.Errorf("today: thiết lập lớp: %w", err)
		}
		byID := map[uuid.UUID]CourseRef{}
		for _, c := range teaching {
			byID[c.ID] = c
		}
		for _, r := range setup {
			c := byID[r.CourseID]
			steps := []Step{
				{Key: "share_code", Label: "Chia sẻ mã lớp", Done: r.ShareCode, Href: "/class/settings?course=" + c.ID.String()},
				{Key: "policy", Label: "Tải quy chế môn học", Done: r.Policy, Href: "/documents?course=" + c.ID.String()},
				{Key: "sessions", Label: "Tạo lịch buổi học", Done: r.Sessions, Href: "/calendar?course=" + c.ID.String()},
				{Key: "documents", Label: "Tải tài liệu", Done: r.Documents, Href: "/documents?course=" + c.ID.String()},
			}
			done := 0
			for _, st := range steps {
				if st.Done {
					done++
				}
			}
			if done == len(steps) || r.Dismissed != "" {
				continue
			}
			out = append(out, mk(Item{
				ID: "COURSE_SETUP:" + c.ID.String(), Kind: KindCourseSetup, Tier: TierCourseSetup, Course: courseRef(c), Steps: steps,
				Href:   "/class/settings?course=" + c.ID.String(),
				Title:  "Thiết lập lớp mới · " + c.ClassCode,
				Reason: fmt.Sprintf("%d/4 bước xong: chia sẻ mã lớp → tải quy chế môn học → tạo lịch buổi học → tải tài liệu.", done),
			}))
		}
	}
	return out, nil
}

// ----- Admin -----

// LLMSignals là phần dữ liệu về cổng AI mà gói này không tự đọc (không import internal/llm): gateway cấp adapter.
type LLMSignals interface {
	// BudgetPercent là % ngân sách AI hệ thống đã dùng hôm nay; ok=false khi không có hạn mức / không đọc được.
	BudgetPercent(ctx context.Context) (pct int, ok bool)
	// OpenCircuit báo mạch của nhà cung cấp (theo id) đang mở.
	OpenCircuit(ctx context.Context, providerID string) bool
}

// AdminProvider: lỗi nhà cung cấp AI, ngân sách, lớp không giảng viên, lời mời hết hạn.
type AdminProvider struct {
	Pool *pgxpool.Pool
	LLM  LLMSignals // nil = bỏ qua mạch và ngân sách
}

// Name implements Provider.
func (AdminProvider) Name() string { return "admin" }

// Items implements Provider.
func (p AdminProvider) Items(ctx context.Context, v Viewer, _ Scope) ([]Item, error) {
	if v.Role != RoleAdmin {
		return nil, nil // không tốn truy vấn cho vai khác
	}
	q := store.New(p.Pool)
	// Việc đọc từ DB trước: không phụ thuộc Redis. Tín hiệu cổng AI (mạch mở, % ngân sách) đi qua Redis nên đứng SAU và có hạn riêng —
	// Redis chậm / chết chỉ mất các tín hiệu đó, không làm Admin mất cả "lớp không giảng viên" và "lời mời hết hạn".
	provs, err := q.TodayAdminProviders(ctx)
	if err != nil {
		return nil, fmt.Errorf("today: nhà cung cấp AI: %w", err)
	}
	nt, err := q.TodayAdminCoursesNoTeacher(ctx)
	if err != nil {
		return nil, fmt.Errorf("today: lớp không giảng viên: %w", err)
	}
	n, err := q.TodayAdminExpiredInvites(ctx, v.Now)
	if err != nil {
		return nil, fmt.Errorf("today: lời mời hết hạn: %w", err)
	}

	sctx := ctx
	if p.LLM != nil {
		var cancel context.CancelFunc
		sctx, cancel = context.WithTimeout(ctx, signalTimeout)
		defer cancel()
	}
	var out []Item
	for _, pr := range provs {
		if pr.Ok && (p.LLM == nil || !p.LLM.OpenCircuit(sctx, pr.ID.String())) {
			continue
		}
		out = append(out, mk(Item{
			ID: "LLM_PROVIDER_ERROR:" + pr.ID.String(), Kind: KindLLMProviderError, Tier: TierLLMProviderError, Href: "/settings/llm",
			Title: "Nhà cung cấp AI đang lỗi", Reason: fmt.Sprintf("%s không phản hồi. Chat của sinh viên có thể dùng dự phòng.", pr.Name),
		}))
	}
	if p.LLM != nil {
		if pct, ok := p.LLM.BudgetPercent(sctx); ok && pct >= 80 {
			kind, tier := KindLLMBudgetWarn, TierLLMBudgetWarn
			if pct >= 100 {
				kind, tier = KindLLMBudgetOut, TierLLMBudgetOut
			}
			out = append(out, mk(Item{
				ID: string(kind), Kind: kind, Tier: tier, Href: "/settings/llm", Title: "Ngân sách AI",
				Reason: fmt.Sprintf("Chi phí AI hôm nay đã dùng %d %% ngân sách.", pct),
			}))
		}
	}
	for _, c := range nt {
		out = append(out, mk(Item{
			ID: "COURSE_NO_TEACHER:" + c.ID.String(), Kind: KindCourseNoTeacher, Tier: TierCourseNoTeacher, Course: &CourseRef{ID: c.ID, ClassCode: c.ClassCode},
			Href: "/admin/courses", Title: fmt.Sprintf("Lớp %s chưa có giảng viên", c.ClassCode),
			Reason: "Hãy gán giảng viên để lớp nhận thông báo và mở mã tham gia.",
		}))
	}
	if n > 0 {
		out = append(out, mk(Item{
			ID: "INVITE_EXPIRED", Kind: KindInviteExpired, Tier: TierInviteExpired, Href: "/admin/users?status=INVITED",
			Title: fmt.Sprintf("%d lời mời giảng viên đã hết hạn", n), Reason: "Gửi lại lời mời để họ vào được hệ thống.",
		}))
	}
	return out, nil
}

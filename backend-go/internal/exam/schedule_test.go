package exam_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/exam"
	"github.com/edupilot/backend-go/internal/platform/clock"
	"github.com/edupilot/backend-go/internal/platform/outbox"
)

func (r *rig) schedule(e exam.ExamDetail) (exam.ExamDetail, error) {
	return r.svc.ScheduleExam(r.t.Context(), r.teacher, r.course, e.ID)
}

// scheduled dựng một bài hợp lệ đã lên lịch (một câu MCQ).
func (r *rig) scheduled(title string) exam.ExamDetail {
	r.t.Helper()
	e := r.mustItems(r.newExam(goodIn(title)), r.approvedMCQ(title))
	d, err := r.schedule(e)
	require.NoError(r.t, err)
	return d
}

func (r *rig) message(topic string) outbox.Message {
	r.t.Helper()
	var m outbox.Message
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select id, topic, payload from outbox where topic=$1 order by created_at desc, id desc limit 1`, topic).Scan(&m.ID, &m.Topic, &m.Payload))
	return m
}

// ---- AC4 -------------------------------------------------------------------------------------------------------------------

// TestScheduleValidationsAll — AC4: lên lịch trả TOÀN BỘ lỗi một lần (không dừng ở lỗi đầu); không đạt thì không đổi gì.
func TestScheduleValidationsAll(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	has := func(err error, want ...string) {
		t.Helper()
		require.ElementsMatch(t, want, fieldCodes(t, err))
		st, code := apiStatus(t, err)
		require.Equal(t, 422, st)
		require.Equal(t, "VALIDATION_FAILED", code)
	}
	// 1) không mục + giờ mở ở quá khứ + thiếu thời lượng → ba lỗi cùng lúc
	past := time.Now().UTC().Add(-time.Hour)
	later := time.Now().UTC().Add(time.Hour)
	e := r.newExam(exam.ExamIn{Title: new("lỗi"), OpensAt: &past, ClosesAt: &later})
	_, err := r.schedule(e)
	has(err, "NO_ITEMS", "OPENS_IN_PAST", "VALUE_REQUIRED")
	// 2) mở sau 30 giây < EXAM_MIN_LEAD_SECONDS (60) → OPENS_IN_PAST; sau 90 giây thì đạt
	soon := time.Now().UTC().Add(30 * time.Second)
	e, err = r.svc.UpdateExam(t.Context(), r.teacher, r.course, e.ID, exam.ExamIn{OpensAt: &soon, ClosesAt: &later, DurationMinutes: new(30)}, e.Version)
	require.NoError(t, err)
	e = r.mustItems(e, r.approvedMCQ("a"))
	_, err = r.schedule(e)
	has(err, "OPENS_IN_PAST")
	ok := time.Now().UTC().Add(90 * time.Second)
	e, err = r.svc.UpdateExam(t.Context(), r.teacher, r.course, e.ID, exam.ExamIn{OpensAt: &ok}, e.Version)
	require.NoError(t, err)
	_, err = r.schedule(e)
	require.NoError(t, err)
	// 3) câu không còn APPROVED / đã lưu trữ → ITEM_NOT_APPROVED mỗi câu (giữ DRAFT)
	m1, m2, m3 := r.approvedMCQ("m1"), r.approvedMCQ("m2"), r.approvedMCQ("m3")
	e = r.mustItems(r.newExam(goodIn("duyệt lại")), m1, m2, m3)
	r.exec(`update question_bank set review_status='REJECTED' where id=$1`, m1)
	r.exec(`update question_bank set archived_at=now() where id=$2 and course_id=$1`, r.course, m2)
	_, err = r.schedule(e)
	has(err, "ITEM_NOT_APPROVED", "ITEM_NOT_APPROVED")
	cur, err := r.svc.GetExam(t.Context(), r.course, e.ID)
	require.NoError(t, err)
	require.Equal(t, "DRAFT", cur.Status)
	require.Equal(t, e.Version, cur.Version)
	// 4) câu code: thiếu test ẩn, tổng trọng số 0, lời giải mẫu chưa kiểm — mỗi điều kiện một mã
	c := r.approvedCode("code")
	e = r.mustItems(r.newExam(goodIn("code")), c)
	_, err = r.schedule(e)
	require.NoError(t, err)
	e = r.mustItems(r.newExam(goodIn("code 2")), c)
	r.exec(`update code_testcases set approved=false where problem_id=$1`, c)
	r.exec(`update code_problems set reference_verified_version=null where question_id=$1`, c)
	_, err = r.schedule(e)
	has(err, "CODE_TESTS_MISSING", "TOTAL_WEIGHT_ZERO", "REFERENCE_NOT_VERIFIED")
	r.exec(`update code_testcases set approved=true where problem_id=$1`, c)
	r.exec(`update code_testcases set weight=0 where problem_id=$1`, c)
	r.verify(c)
	_, err = r.schedule(e)
	has(err, "TOTAL_WEIGHT_ZERO")
	// 5) tổng thời gian chấm: 2 test × 3 × time_limit > JUDGE_MAX_TOTAL_SECONDS
	r.exec(`update code_testcases set weight=1 where problem_id=$1`, c)
	small := *r.svc
	small.Limits = exam.Limits{MaxTotalSeconds: 5}
	r.exec(`update code_problems set time_limit_ms=1000 where question_id=$1`, c) // 2 test × 3 s = 6 s > 5 s
	_, err = small.ScheduleExam(t.Context(), r.teacher, r.course, e.ID)
	has(err, "CODE_TIME_BUDGET_EXCEEDED")
	small.Limits = exam.Limits{MaxTotalSeconds: 6}
	_, err = small.ScheduleExam(t.Context(), r.teacher, r.course, e.ID)
	require.NoError(t, err, "đúng bằng ngân sách thì đạt")
	// 6) lớp lưu trữ → 409
	r.exec(`update courses set status='ARCHIVED', archived_at=now(), join_enabled=false where id=$1`, r.course)
	st, code := apiStatus(t, errOf(r.schedule(e)))
	require.Equal(t, 409, st)
	require.Equal(t, "COURSE_ARCHIVED", code)
}

// TestScheduleEffects — AC4: SCHEDULED + `audit_log` + outbox `exam.scheduled` cùng transaction; handler tạo MỘT thông báo cho mỗi sinh viên ACTIVE (không cho PENDING / REMOVED).
func TestScheduleEffects(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	svs := []uuid.UUID{r.student("ACTIVE"), r.student("ACTIVE"), r.student("ACTIVE")}
	pending, removed := r.student("PENDING"), r.student("REMOVED")
	e := r.mustItems(r.newExam(goodIn("Kiểm tra tuần 9")), r.approvedMCQ("a"))
	require.Zero(t, r.count(`select count(*) from outbox where topic='exam.scheduled'`))
	d, err := r.schedule(e)
	require.NoError(t, err)
	require.Equal(t, "SCHEDULED", d.Status)
	require.Equal(t, e.Version+1, d.Version)
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity_id=$1 and action='exam.schedule'`, e.ID.String()))
	var p struct {
		ExamID   uuid.UUID `json:"exam_id"`
		CourseID uuid.UUID `json:"course_id"`
	}
	m := r.message("exam.scheduled")
	require.NoError(t, json.Unmarshal(m.Payload, &p))
	require.Equal(t, e.ID, p.ExamID)
	require.Equal(t, r.course, p.CourseID)

	n := &exam.Notifier{Pool: r.pool}
	require.NoError(t, n.HandleScheduled(t.Context(), m))
	for _, sv := range svs {
		require.Equal(t, 1, r.count(`select count(*) from notifications where user_id=$1 and type='EXAM_SCHEDULED' and course_id=$2`, sv, r.course))
	}
	require.Zero(t, r.count(`select count(*) from notifications where user_id = any($1::uuid[]) and type='EXAM_SCHEDULED'`, []uuid.UUID{pending, removed, r.teacher, r.ta}))
	var title, link string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select title, link from notifications where user_id=$1`, svs[0]).Scan(&title, &link))
	opens := e.OpensAt.In(time.FixedZone("ICT", 7*3600))
	require.Equal(t, fmt.Sprintf("Bài thi Kiểm tra tuần 9 mở lúc %s, làm trong 45 phút", exam.ViTime(*e.OpensAt)), title)
	require.Contains(t, title, opens.Format("15:04 02/01"))
	require.Equal(t, "/exams/"+e.ID.String()+"/take", link)
	// xử lý lại cùng sự kiện: không thêm thông báo
	require.NoError(t, n.HandleScheduled(t.Context(), m))
	require.Equal(t, 3, r.count(`select count(*) from notifications where type='EXAM_SCHEDULED' and course_id=$1`, r.course))
}

// TestScheduleIdempotent — AC4: gọi lại khi đã SCHEDULED → trả bản hiện tại, không đổi version, không thêm audit / outbox.
func TestScheduleIdempotent(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	d := r.scheduled("lặp")
	again, err := r.schedule(d)
	require.NoError(t, err)
	require.Equal(t, d.Version, again.Version)
	require.Equal(t, "SCHEDULED", again.Status)
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='exam.scheduled'`))
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity_id=$1 and action='exam.schedule'`, d.ID.String()))
	// hai lần gọi đồng thời → vẫn một sự kiện
	e := r.mustItems(r.newExam(goodIn("song song")), r.approvedMCQ("p"))
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := r.schedule(e)
			require.NoError(t, err)
		}()
	}
	wg.Wait()
	require.Equal(t, 2, r.count(`select count(*) from outbox where topic='exam.scheduled'`))
}

// TestScheduleTeacherOnly — AC4 / SRS 2: lên lịch, bỏ lịch, gia hạn, xoá chỉ Giảng viên; TA → 403 `reason=role`; sinh viên và Admin cũng bị chặn.
func TestScheduleTeacherOnly(t *testing.T) {
	t.Parallel()
	for _, no := range []int{23, 24, 25, 26, 27} {
		var rt exam.Route
		for _, x := range exam.Routes() {
			if x.No == no {
				rt = x
			}
		}
		require.Equal(t, auth.TeacherRole, rt.Mode, "thao tác %d", no)
		for _, who := range []struct {
			jwt auth.Role
			m   auth.Membership
		}{{auth.RoleTA, auth.Membership{Found: true, Role: auth.RoleTA, Status: "ACTIVE"}}, {auth.RoleStudent, auth.Membership{Found: true, Role: auth.RoleStudent, Status: "ACTIVE"}},
			{auth.RoleAdmin, auth.Membership{Found: true, Role: auth.RoleTeacher, Status: "ACTIVE"}}} {
			code, reason := serve(t, rt, who.jwt, who.m)
			require.Equal(t, 403, code, "thao tác %d · %s", no, who.jwt)
			if who.jwt != auth.RoleAdmin {
				require.Equal(t, "role", reason, "thao tác %d · %s", no, who.jwt)
			}
		}
		code, _ := serve(t, rt, auth.RoleTeacher, auth.Membership{Found: true, Role: auth.RoleTeacher, Status: "ACTIVE"})
		require.Equal(t, 204, code)
	}
	for _, no := range []int{18, 20, 21, 22, 28} { // TA tạo / sửa nháp, xem trước, nhân bản được
		for _, x := range exam.Routes() {
			if x.No == no {
				require.Equal(t, auth.StaffRole, x.Mode, "thao tác %d", no)
			}
		}
	}
}

// ---- AC5 -------------------------------------------------------------------------------------------------------------------

// TestUnscheduleRules — AC5: bỏ lịch chỉ khi chưa tới giờ mở và chưa có lượt làm; ngược lại 409 EXAM_LOCKED với `reason` ∈ status | opened | has_attempts;
// bỏ lịch thu hồi thông báo cũ, báo "hoãn", và lần lên lịch sau báo lại.
func TestUnscheduleRules(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	sv := r.student("ACTIVE")
	lockedReason := func(err error) string {
		t.Helper()
		st, code := apiStatus(t, err)
		require.Equal(t, 409, st)
		require.Equal(t, "EXAM_LOCKED", code)
		return reasonOf(t, err)
	}
	d := r.scheduled("hoãn")
	n := &exam.Notifier{Pool: r.pool}
	require.NoError(t, n.HandleScheduled(t.Context(), r.message("exam.scheduled")))
	require.Equal(t, 1, r.count(`select count(*) from notifications where user_id=$1 and type='EXAM_SCHEDULED'`, sv))

	back, err := r.svc.UnscheduleExam(t.Context(), r.teacher, r.course, d.ID)
	require.NoError(t, err)
	require.Equal(t, "DRAFT", back.Status)
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity_id=$1 and action='exam.unschedule'`, d.ID.String()))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='exam.unscheduled'`))
	um := r.message("exam.unscheduled")
	require.NoError(t, n.HandleUnscheduled(t.Context(), um))
	require.NoError(t, n.HandleUnscheduled(t.Context(), um)) // idempotent
	require.Zero(t, r.count(`select count(*) from notifications where user_id=$1 and type='EXAM_SCHEDULED'`, sv), "thông báo lịch bị thu hồi")
	var title string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select title from notifications where user_id=$1 and type='EXAM_UNSCHEDULED'`, sv).Scan(&title))
	require.Equal(t, "Bài thi hoãn đã bị hoãn, chờ lịch mới", title)
	require.Equal(t, 1, r.count(`select count(*) from notifications where user_id=$1 and type='EXAM_UNSCHEDULED'`, sv))
	// DRAFT không bỏ lịch được
	require.Equal(t, "status", lockedReason(errOf(r.svc.UnscheduleExam(t.Context(), r.teacher, r.course, d.ID))))
	// lên lịch lại → báo lại
	_, err = r.schedule(back)
	require.NoError(t, err)
	require.NoError(t, n.HandleScheduled(t.Context(), r.message("exam.scheduled")))
	require.Equal(t, 1, r.count(`select count(*) from notifications where user_id=$1 and type='EXAM_SCHEDULED'`, sv))

	// đã có lượt làm → has_attempts
	r.exec(`insert into exam_attempts (course_id, exam_id, student_id, started_at, deadline_at) values ($1, $2, $3, now(), now() + interval '30 minutes')`, r.course, d.ID, sv)
	require.Equal(t, "has_attempts", lockedReason(errOf(r.svc.UnscheduleExam(t.Context(), r.teacher, r.course, d.ID))))
	// đã tới giờ mở (đồng hồ giả) → opened
	clk := clock.NewFake(time.Now().UTC())
	e2 := r.scheduled("sắp mở")
	clk.Advance(3 * time.Hour)
	require.Equal(t, "opened", lockedReason(errOf(r.at(clk).UnscheduleExam(t.Context(), r.teacher, r.course, e2.ID))))
	// OPEN / CLOSED / PUBLISHED → status
	for _, st := range []string{"OPEN", "CLOSED", "PUBLISHED"} {
		require.Equal(t, "status", lockedReason(errOf(r.svc.UnscheduleExam(t.Context(), r.teacher, r.course, r.useIn(r.approvedMCQ("u"+st), st)))), st)
	}
}

func reasonOf(t *testing.T, err error) string {
	t.Helper()
	b, jerr := json.Marshal(detailsOf(t, err))
	require.NoError(t, jerr)
	var d struct {
		Reason string `json:"reason"`
	}
	require.NoError(t, json.Unmarshal(b, &d))
	return d.Reason
}

// TestEditLockMatrix — AC5 / SRS 4.2.5: bảng trạng thái × trường. DRAFT sửa được mọi trường; từ SCHEDULED trở đi chỉ title / instructions / reveal_answers / appeal_days;
// các trường khác (kể cả closes_at, danh sách mục) → 409 EXAM_LOCKED; gửi lại đúng giá trị cũ thì không phải là thay đổi.
func TestEditLockMatrix(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	loose := []struct {
		name string
		in   exam.ExamIn
	}{
		{"title", exam.ExamIn{Title: new("tên mới")}},
		{"instructions", exam.ExamIn{Instructions: new("hướng dẫn mới")}},
		{"reveal_answers", exam.ExamIn{RevealAnswers: new(false)}},
		{"appeal_days", exam.ExamIn{AppealDays: new(3)}},
	}
	later := time.Now().UTC().Add(5 * time.Hour)
	strict := []struct {
		name string
		in   exam.ExamIn
	}{
		{"opens_at", exam.ExamIn{OpensAt: new(time.Now().UTC().Add(-3 * time.Hour))}},
		{"closes_at", exam.ExamIn{ClosesAt: &later}},
		{"duration_minutes", exam.ExamIn{DurationMinutes: new(20)}},
		{"shuffle_questions", exam.ExamIn{ShuffleQuestions: new(false)}},
		{"shuffle_options", exam.ExamIn{ShuffleOptions: new(false)}},
		{"max_score", exam.ExamIn{MaxScore: new(dec("20"))}},
		{"rounding_step", exam.ExamIn{RoundingStep: new(dec("0.5"))}},
		{"multi_scoring", exam.ExamIn{MultiScoring: new("ALL_OR_NOTHING")}},
	}
	for _, st := range []string{"DRAFT", "SCHEDULED", "OPEN", "CLOSED", "PUBLISHED"} {
		q := r.approvedMCQ("lock " + st)
		var id uuid.UUID
		if st == "DRAFT" {
			id = r.mustItems(r.newExam(goodIn("nháp")), q).ID
		} else {
			id = r.useIn(q, st)
		}
		upd := func(in exam.ExamIn) error {
			cur, err := r.svc.GetExam(t.Context(), r.course, id)
			require.NoError(t, err)
			_, err = r.svc.UpdateExam(t.Context(), r.teacher, r.course, id, in, cur.Version)
			return err
		}
		for _, f := range loose {
			require.NoError(t, upd(f.in), "%s · %s", st, f.name)
		}
		for _, f := range strict {
			err := upd(f.in)
			if st == "DRAFT" {
				require.NoError(t, err, "%s · %s", st, f.name)
				continue
			}
			sc, code := apiStatus(t, err)
			require.Equal(t, 409, sc, "%s · %s", st, f.name)
			require.Equal(t, "EXAM_LOCKED", code, "%s · %s", st, f.name)
			require.Contains(t, fmt.Sprint(detailsOf(t, err)), f.name)
		}
		// gửi lại đúng giá trị đang có: không phải thay đổi
		cur, err := r.svc.GetExam(t.Context(), r.course, id)
		require.NoError(t, err)
		require.NoError(t, upd(exam.ExamIn{OpensAt: cur.OpensAt, ClosesAt: cur.ClosesAt, DurationMinutes: cur.DurationMinutes, ShuffleQuestions: &cur.ShuffleQuestions, MaxScore: new(dec(cur.MaxScore))}), st)
	}
}

// ---- AC6 -------------------------------------------------------------------------------------------------------------------

// TestEffectiveStatus — AC6: effective_status theo đồng hồ (bảng).
func TestEffectiveStatus(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 12, 1, 8, 0, 0, 0, time.UTC)
	opens, closes := base, base.Add(time.Hour)
	at := func(d time.Duration) time.Time { return base.Add(d) }
	for _, c := range []struct {
		status string
		now    time.Time
		want   string
	}{
		{"DRAFT", at(2 * time.Hour), "DRAFT"},
		{"SCHEDULED", at(-time.Second), "SCHEDULED"},
		{"SCHEDULED", at(0), "OPEN"},
		{"SCHEDULED", at(59 * time.Minute), "OPEN"},
		{"SCHEDULED", at(time.Hour), "CLOSED"},
		{"SCHEDULED", at(48 * time.Hour), "CLOSED"},
		{"OPEN", at(-time.Hour), "OPEN"},
		{"OPEN", at(time.Hour - time.Nanosecond), "OPEN"},
		{"OPEN", at(time.Hour), "CLOSED"},
		{"CLOSED", at(-time.Hour), "CLOSED"},
		{"PUBLISHED", at(-time.Hour), "PUBLISHED"},
		{"PUBLISHED", at(time.Hour), "PUBLISHED"},
	} {
		require.Equal(t, c.want, exam.EffectiveStatus(c.status, &opens, &closes, c.now), "%s @ %s", c.status, c.now.Sub(base))
	}
	require.Equal(t, "SCHEDULED", exam.EffectiveStatus("SCHEDULED", nil, nil, base), "không có mốc giờ thì giữ nguyên")
}

func (r *rig) status(id uuid.UUID) string {
	r.t.Helper()
	var s string
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select status::text from exams where id=$1`, id).Scan(&s))
	return s
}

func (r *rig) ticker(clk clock.Clock) *exam.Ticker {
	return &exam.Ticker{Svc: r.at(clk), Log: nil}
}

// TestExamStateMachine — AC6: tick mở rồi đóng đúng giờ, mỗi chuyển một sự kiện outbox (exam.opened, exam.closed); đọc luôn dùng effective_status dù tick chưa chạy.
func TestExamStateMachine(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	d := r.scheduled("giờ giấc")
	clk := clock.NewFake(time.Now().UTC())
	tk := r.ticker(clk)
	require.NoError(t, tk.Tick(t.Context()))
	require.Equal(t, "SCHEDULED", r.status(d.ID), "chưa tới giờ mở")
	clk.Advance(2*time.Hour + time.Second)
	require.Equal(t, "OPEN", r.effective(clk, d.ID), "đọc đã thấy OPEN dù tick chưa chạy")
	require.Equal(t, "SCHEDULED", r.status(d.ID))
	require.NoError(t, tk.Tick(t.Context()))
	require.Equal(t, "OPEN", r.status(d.ID))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='exam.opened'`))
	clk.Advance(2 * time.Hour)
	require.Equal(t, "CLOSED", r.effective(clk, d.ID))
	require.NoError(t, tk.Tick(t.Context()))
	require.Equal(t, "CLOSED", r.status(d.ID))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='exam.closed'`))
	require.NoError(t, tk.Tick(t.Context()))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='exam.closed' and payload->>'exam_id'=$1`, d.ID.String()))
	// không có chuyển ngược: tick không đưa CLOSED về OPEN, bỏ lịch bị từ chối
	require.Equal(t, "CLOSED", r.status(d.ID))
}

func (r *rig) effective(clk clock.Clock, id uuid.UUID) string {
	r.t.Helper()
	d, err := r.at(clk).GetExam(r.t.Context(), r.course, id)
	require.NoError(r.t, err)
	return d.EffectiveStatus
}

// TestTickIdempotent — AC6: hai bộ lập lịch chạy cùng lúc (khoá leader hỏng) → vẫn MỘT chuyển, MỘT sự kiện.
func TestTickIdempotent(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	d := r.scheduled("lặp tick")
	clk := clock.NewFake(time.Now().UTC().Add(2*time.Hour + time.Minute))
	var wg sync.WaitGroup
	for range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			require.NoError(t, r.ticker(clk).Tick(t.Context()))
		}()
	}
	wg.Wait()
	require.Equal(t, "OPEN", r.status(d.ID))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='exam.opened' and payload->>'exam_id'=$1`, d.ID.String()))
}

// TestTickSkipsToClosed — AC6: SCHEDULED mà đã quá giờ đóng (máy chủ tắt dài) → thẳng CLOSED, không phát exam.opened.
func TestTickSkipsToClosed(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	d := r.scheduled("tắt lâu")
	clk := clock.NewFake(time.Now().UTC().Add(48 * time.Hour))
	require.NoError(t, r.ticker(clk).Tick(t.Context()))
	require.Equal(t, "CLOSED", r.status(d.ID))
	require.Zero(t, r.count(`select count(*) from outbox where topic='exam.opened' and payload->>'exam_id'=$1`, d.ID.String()))
	require.Equal(t, 1, r.count(`select count(*) from outbox where topic='exam.closed' and payload->>'exam_id'=$1`, d.ID.String()))
	// bài khác DRAFT / PUBLISHED không bị đụng
	dr := r.newExam(goodIn("nháp"))
	require.NoError(t, r.ticker(clk).Tick(t.Context()))
	require.Equal(t, "DRAFT", r.status(dr.ID))
}

// ---- AC7 -------------------------------------------------------------------------------------------------------------------

// TestExtendOnlyLater — AC7: chỉ lùi muộn hơn (mốc mới > cũ, ≤ cũ + 24 giờ); sai → 422 CLOSES_NOT_LATER / LIMIT_OUT_OF_RANGE; audit.
func TestExtendOnlyLater(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	d := r.scheduled("gia hạn")
	old := *d.ClosesAt
	for _, c := range []struct {
		name string
		to   time.Time
		want string
	}{
		{"bằng mốc cũ", old, "CLOSES_NOT_LATER"}, {"sớm hơn", old.Add(-time.Minute), "CLOSES_NOT_LATER"}, {"quá 24 giờ", old.Add(24*time.Hour + time.Second), "LIMIT_OUT_OF_RANGE"},
	} {
		_, err := r.svc.ExtendExam(t.Context(), r.teacher, r.course, d.ID, c.to)
		require.Equal(t, []string{c.want}, fieldCodes(t, err), c.name)
	}
	got, err := r.svc.ExtendExam(t.Context(), r.teacher, r.course, d.ID, old.Add(24*time.Hour))
	require.NoError(t, err, "đúng 24 giờ là biên hợp lệ")
	require.True(t, got.ClosesAt.Equal(old.Add(24*time.Hour)))
	require.Equal(t, "SCHEDULED", got.Status)
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity_id=$1 and action='exam.extend'`, d.ID.String()))
	// DRAFT không gia hạn được (đổi giờ đóng của nháp qua PUT)
	dr := r.newExam(goodIn("nháp"))
	_, err = r.svc.ExtendExam(t.Context(), r.teacher, r.course, dr.ID, dr.ClosesAt.Add(time.Hour))
	st, code := apiStatus(t, err)
	require.Equal(t, 409, st)
	require.Equal(t, "EXAM_LOCKED", code)
}

// TestExtendRecomputesRunningDeadlines — AC7: lượt IN_PROGRESS bị chặn bởi giờ đóng cũ được tính lại min(started_at + thời lượng, giờ đóng mới); lượt không bị chặn giữ nguyên.
func TestExtendRecomputesRunningDeadlines(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	svA, svB, svC := r.student("ACTIVE"), r.student("ACTIVE"), r.student("ACTIVE")
	d := r.scheduled("đang làm")
	opens, closes := *d.OpensAt, *d.ClosesAt
	// A bắt đầu 10 phút trước giờ đóng: deadline bị chặn ở giờ đóng cũ; B bắt đầu sớm: deadline = started + 45 phút (không bị chặn); C đã nộp.
	startA, startB := closes.Add(-10*time.Minute), opens.Add(5*time.Minute)
	r.exec(`update exams set status='OPEN' where id=$1`, d.ID)
	ins := func(who uuid.UUID, start, deadline time.Time, submitted bool) {
		if submitted {
			r.exec(`insert into exam_attempts (course_id, exam_id, student_id, status, started_at, deadline_at, submitted_at, submit_reason, auto_score, graded_at) values ($1,$2,$3,'GRADED',$4::timestamptz,$5,$4::timestamptz + interval '5 minutes','MANUAL',1,now())`, r.course, d.ID, who, start, deadline)
			return
		}
		r.exec(`insert into exam_attempts (course_id, exam_id, student_id, started_at, deadline_at) values ($1,$2,$3,$4,$5)`, r.course, d.ID, who, start, deadline)
	}
	ins(svA, startA, closes, false)
	ins(svB, startB, startB.Add(45*time.Minute), false)
	ins(svC, startA, closes, true)
	newClose := closes.Add(30 * time.Minute)
	got, err := r.svc.ExtendExam(t.Context(), r.teacher, r.course, d.ID, newClose)
	require.NoError(t, err)
	require.True(t, got.ClosesAt.Equal(newClose))
	deadline := func(who uuid.UUID) time.Time {
		var v time.Time
		require.NoError(t, r.pool.QueryRow(t.Context(), `select deadline_at from exam_attempts where exam_id=$1 and student_id=$2`, d.ID, who).Scan(&v))
		return v
	}
	require.True(t, deadline(svA).Equal(newClose), "A: min(bắt đầu + 45 phút = đóng cũ + 35, giờ đóng mới = đóng cũ + 30) = giờ đóng mới; có %s", deadline(svA))
	require.True(t, deadline(svB).Equal(startB.Add(45*time.Minute)), "B không đổi")
	require.True(t, deadline(svC).Equal(closes), "lượt đã nộp không đổi")
	// gia hạn lần nữa quá xa: deadline bị chặn bởi giờ đóng mới
	startD := newClose.Add(-5 * time.Minute)
	svD := r.student("ACTIVE")
	ins(svD, startD, newClose, false)
	_, err = r.svc.ExtendExam(t.Context(), r.teacher, r.course, d.ID, newClose.Add(time.Hour))
	require.NoError(t, err)
	require.True(t, deadline(svD).Equal(startD.Add(45*time.Minute)), "D: min(bắt đầu + 45, giờ đóng mới)")
}

// TestExtendAfterCloseRejected — AC7: bài CLOSED / PUBLISHED, hoặc đã quá giờ đóng (dù tick chưa chạy) → 409 EXAM_LOCKED.
func TestExtendAfterCloseRejected(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	for _, st := range []string{"CLOSED", "PUBLISHED"} {
		d, err := r.svc.GetExam(t.Context(), r.course, r.useIn(r.approvedMCQ("e"+st), st))
		require.NoError(t, err)
		sc, code := apiStatus(t, errOf(r.svc.ExtendExam(t.Context(), r.teacher, r.course, d.ID, d.ClosesAt.Add(time.Hour))))
		require.Equal(t, 409, sc, st)
		require.Equal(t, "EXAM_LOCKED", code, st)
	}
	d := r.scheduled("quá giờ")
	clk := clock.NewFake(d.ClosesAt.Add(time.Second))
	sc, code := apiStatus(t, errOf(r.at(clk).ExtendExam(t.Context(), r.teacher, r.course, d.ID, d.ClosesAt.Add(time.Hour))))
	require.Equal(t, 409, sc)
	require.Equal(t, "EXAM_LOCKED", code)
}

// ---- AC10 ------------------------------------------------------------------------------------------------------------------

// TestScheduleRaceWithQuestionReject — AC10: một câu bị REJECT ngay trước (giao dịch chưa commit) làm lên lịch thất bại ITEM_NOT_APPROVED: schedule kiểm lại TRONG transaction
// bằng khoá chia sẻ trên câu hỏi; không có cửa sổ "đọc APPROVED cũ rồi lên lịch".
func TestScheduleRaceWithQuestionReject(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	q := r.approvedMCQ("sắp bị loại")
	e := r.mustItems(r.newExam(goodIn("đua")), q)
	tx, err := r.pool.Begin(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback(context.WithoutCancel(t.Context())) })              // chạy trước pool.Close; Rollback sau Commit chỉ trả ErrTxClosed
	_, err = tx.Exec(t.Context(), `select 1 from question_bank where id=$1 for update`, q) // như Review đang chạy
	require.NoError(t, err)
	done := make(chan error, 1)
	go func() { _, err := r.schedule(e); done <- err }()
	select {
	case err := <-done:
		t.Fatalf("lên lịch phải chờ câu hỏi đang bị khoá, nhưng đã trả về: %v", err)
	case <-time.After(400 * time.Millisecond):
	}
	_, err = tx.Exec(t.Context(), `update question_bank set review_status='REJECTED' where id=$1`, q)
	require.NoError(t, err)
	require.NoError(t, tx.Commit(t.Context()))
	err = <-done
	require.Equal(t, []string{"ITEM_NOT_APPROVED"}, fieldCodes(t, err))
	require.Equal(t, "DRAFT", r.status(e.ID))
	// và chiều ngược lại: lên lịch xong thì Review phải chờ — không để câu bị loại GIỮA lúc kiểm và commit
	q2 := r.approvedMCQ("an toàn")
	e2 := r.mustItems(r.newExam(goodIn("đua 2")), q2)
	d, err := r.schedule(e2)
	require.NoError(t, err)
	require.Equal(t, "SCHEDULED", d.Status)
}

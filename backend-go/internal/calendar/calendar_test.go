package calendar_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/today"
)

func ids(items []map[string]any) []string {
	var out []string
	for _, i := range items {
		out = append(out, i["id"].(string))
	}
	return out
}

// TestCalendarUnionSources — AC2: ba nguồn gộp lúc đọc, đúng source/type, sắp theo (starts_at, id), chỉ lớp của người gọi.
func TestCalendarUnionSources(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	s := f.session(f.course, 1, 48*time.Hour, "Mật mã")
	e := f.exam(f.course, "Thi giữa kỳ", "SCHEDULED", 24*time.Hour)
	ev := f.event(f.course, "OTHER", "Nộp báo cáo", 72*time.Hour)
	other := f.newCourse("ACTIVE")
	f.session(other, 1, 10*time.Hour, "Lớp khác")
	items := f.items(f.sv, auth.RoleStudent, 0, 10*24*time.Hour)
	require.Equal(t, []string{"weekly_exam:" + e.String(), "class_session:" + s.String(), "calendar_event:" + ev.String()}, ids(items))
	require.Equal(t, "EXAM", items[0]["type"])
	require.Equal(t, "Buổi 1 · Mật mã", items[1]["title"])
	require.Equal(t, "CLASS_SESSION", items[1]["type"])
	require.Equal(t, "P.301", items[1]["location"])
	require.Equal(t, "OTHER", items[2]["type"])
}

// TestCalendarNoDuplication — AC2: tạo buổi học / bài thi không thêm dòng nào vào calendar_events.
func TestCalendarNoDuplication(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.session(f.course, 1, time.Hour, "A")
	f.exam(f.course, "Thi", "SCHEDULED", 2*time.Hour)
	require.Zero(t, count(f, `select count(*) from calendar_events`))
	require.Len(t, f.items(f.sv, auth.RoleStudent, 0, 24*time.Hour), 2)
}

// TestCalendarExcludesDraftExam — AC2: bài thi DRAFT không bao giờ có trong lịch.
func TestCalendarExcludesDraftExam(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.exam(f.course, "Nháp", "DRAFT", 0)
	require.Empty(t, f.items(f.sv, auth.RoleStudent, 0, 24*time.Hour))
}

// TestCalendarRangeLimit — AC2: from/to bắt buộc, khoảng > 62 ngày → 422 RANGE_TOO_LARGE; đúng 62 ngày vẫn được.
func TestCalendarRangeLimit(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	tok := f.tok(f.sv, auth.RoleStudent)
	st, _, b := f.do("GET", f.calURL(f.course, 0, 63*24*time.Hour), tok, "")
	require.Equal(t, 422, st)
	require.Contains(t, string(b), "RANGE_TOO_LARGE")
	st, _, _ = f.do("GET", f.calURL(f.course, 0, 62*24*time.Hour), tok, "")
	require.Equal(t, 200, st)
	st, _, _ = f.do("GET", "/api/v1/courses/"+f.course.String()+"/calendar", tok, "")
	require.Equal(t, 422, st)
	st, _, _ = f.do("GET", f.calURL(f.course, time.Hour, 0), tok, "") // to < from
	require.Equal(t, 422, st)
}

// TestCalendarCursor — AC2: phân trang con trỏ liền mạch, không trùng, không sót; limit > 100 → 422.
func TestCalendarCursor(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	for i := range 7 {
		f.event(f.course, "OTHER", fmt.Sprintf("E%d", i), time.Duration(i+1)*time.Hour)
	}
	f.event(f.course, "OTHER", "Cùng giờ", time.Hour) // trùng starts_at: tie-break theo id
	tok := f.tok(f.sv, auth.RoleStudent)
	var all []string
	path := f.calURL(f.course, 0, 24*time.Hour) + "&limit=3"
	for range 5 {
		st, _, b := f.do("GET", path, tok, "")
		require.Equal(t, 200, st, string(b))
		var p struct {
			Items      []map[string]any `json:"items"`
			NextCursor *string          `json:"next_cursor"`
		}
		require.NoError(t, json.Unmarshal(b, &p))
		all = append(all, ids(p.Items)...)
		if p.NextCursor == nil {
			break
		}
		path = f.calURL(f.course, 0, 24*time.Hour) + "&limit=3&cursor=" + *p.NextCursor
	}
	require.Len(t, all, 8)
	seen := map[string]bool{}
	for _, id := range all {
		require.False(t, seen[id], "trùng %s", id)
		seen[id] = true
	}
	st, _, _ := f.do("GET", f.calURL(f.course, 0, 24*time.Hour)+"&limit=101", tok, "")
	require.Equal(t, 422, st)
}

func (f *fx) createEvent(token, body string) (int, map[string]any) {
	st, _, b := f.do("POST", "/api/v1/courses/"+f.course.String()+"/calendar/events", token, body)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return st, m
}

func (f *fx) ev(title string, in time.Duration, extra string) string {
	return fmt.Sprintf(`{"type":"EXAM","title":%q,"starts_at":%q%s}`, title, f.clk.Now().Add(in).Format(time.RFC3339), extra)
}

// TestEventCRUD — AC3: TA tạo, TEACHER sửa và xoá; sinh viên không ghi được; sự kiện hiện đúng trong lịch với editable + version cho Staff.
func TestEventCRUD(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	ta, gv, sv := f.tok(f.ta, auth.RoleTA), f.tok(f.teacher, auth.RoleTeacher), f.tok(f.sv, auth.RoleStudent)
	st, m := f.createEvent(ta, f.ev("Thi giữa kỳ", 48*time.Hour, `,"location":"P.301","description":"Mang thẻ"`))
	require.Equal(t, 201, st)
	id := m["id"].(string)
	require.EqualValues(t, 1, m["version"])
	E := "/api/v1/courses/" + f.course.String() + "/calendar/events/" + id
	st, _, b := f.do("PUT", E, gv, f.ev("Thi giữa kỳ (đổi)", 50*time.Hour, `,"version":1`))
	require.Equal(t, 200, st, string(b))
	staff := f.items(f.ta, auth.RoleTA, 0, 7*24*time.Hour)
	require.Len(t, staff, 1)
	require.Equal(t, "Thi giữa kỳ (đổi)", staff[0]["title"])
	require.Equal(t, true, staff[0]["editable"])
	require.EqualValues(t, 2, staff[0]["version"])
	stu := f.items(f.sv, auth.RoleStudent, 0, 7*24*time.Hour)
	require.Equal(t, false, stu[0]["editable"])
	require.Nil(t, stu[0]["version"])
	st, _, _ = f.do("PUT", E, sv, f.ev("x", 50*time.Hour, `,"version":2`))
	require.Equal(t, 403, st)
	st, _, _ = f.do("DELETE", E, sv, "")
	require.Equal(t, 403, st)
	st, _, _ = f.do("DELETE", E, gv, "")
	require.Equal(t, 204, st)
	require.Empty(t, f.items(f.sv, auth.RoleStudent, 0, 7*24*time.Hour))
	st, _, _ = f.do("DELETE", E, gv, "")
	require.Equal(t, 404, st)
}

// TestEventValidation — AC3: tên 1–120, địa điểm ≤ 80, mô tả ≤ 1.000, starts_at ±2 năm, ends_at > starts_at.
func TestEventValidation(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	ta := f.tok(f.ta, auth.RoleTA)
	at := f.clk.Now().Add(time.Hour).Format(time.RFC3339)
	for name, tc := range map[string]struct {
		body string
		want int
		code string
	}{
		"tên rỗng":        {f.ev("  ", time.Hour, ""), 422, "VALIDATION_FAILED"},
		"tên 121":         {f.ev(strings.Repeat("a", 121), time.Hour, ""), 422, "VALIDATION_FAILED"},
		"tên 120":         {f.ev(strings.Repeat("a", 120), time.Hour, ""), 201, ""},
		"địa điểm 81":     {f.ev("x", time.Hour, `,"location":"`+strings.Repeat("a", 81)+`"`), 422, "VALIDATION_FAILED"},
		"mô tả 1001":      {f.ev("x", time.Hour, `,"description":"`+strings.Repeat("a", 1001)+`"`), 422, "VALIDATION_FAILED"},
		"loại lạ":         {`{"type":"CLASS","title":"x","starts_at":"` + at + `"}`, 422, "VALIDATION_FAILED"},
		"thiếu starts_at": {`{"type":"OTHER","title":"x"}`, 422, "VALIDATION_FAILED"},
		"ends = starts":   {f.ev("x", time.Hour, `,"ends_at":"`+at+`"`), 422, "EVENT_TIME_INVALID"},
		"ends < starts":   {f.ev("x", 2*time.Hour, `,"ends_at":"`+at+`"`), 422, "EVENT_TIME_INVALID"},
		"quá 2 năm sau":   {f.ev("x", 3*365*24*time.Hour, ""), 422, "EVENT_TIME_INVALID"},
		"quá 2 năm trước": {f.ev("x", -3*365*24*time.Hour, ""), 422, "EVENT_TIME_INVALID"},
		"trường lạ":       {f.ev("x", time.Hour, `,"color":"red"`), 422, "VALIDATION_FAILED"},
		"ends hợp lệ":     {f.ev("x", time.Hour, `,"ends_at":"`+f.clk.Now().Add(3*time.Hour).Format(time.RFC3339)+`"`), 201, ""},
	} {
		st, _, b := f.do("POST", "/api/v1/courses/"+f.course.String()+"/calendar/events", ta, tc.body)
		require.Equal(t, tc.want, st, "%s: %s", name, b)
		if tc.code != "" {
			require.Contains(t, string(b), tc.code, name)
		}
	}
}

// TestEventVersionConflict — AC3: sửa với version cũ → 409 VERSION_CONFLICT kèm bản hiện hành; If-Match cũng được.
func TestEventVersionConflict(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	ta := f.tok(f.ta, auth.RoleTA)
	_, m := f.createEvent(ta, f.ev("A", time.Hour, ""))
	E := "/api/v1/courses/" + f.course.String() + "/calendar/events/" + m["id"].(string)
	st, _, _ := f.do("PUT", E, ta, f.ev("B", time.Hour, `,"version":1`))
	require.Equal(t, 200, st)
	st, _, b := f.do("PUT", E, ta, f.ev("C", time.Hour, `,"version":1`))
	require.Equal(t, 409, st)
	require.Contains(t, string(b), "VERSION_CONFLICT")
	require.Contains(t, string(b), `"B"`) // bản hiện hành
	st, _, _ = f.do("PUT", E, ta, f.ev("D", time.Hour, ""), "If-Match", `W/"v2"`)
	require.Equal(t, 200, st)
	st, _, _ = f.do("PUT", E, ta, f.ev("E", time.Hour, ""))
	require.Equal(t, 422, st) // thiếu cả version lẫn If-Match
}

// TestEventAudited — AC3: tạo / sửa / xoá đều ghi audit_log; mỗi lần ghi một dòng outbox calendar.changed.
func TestEventAudited(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	ta := f.tok(f.ta, auth.RoleTA)
	_, m := f.createEvent(ta, f.ev("A", time.Hour, ""))
	E := "/api/v1/courses/" + f.course.String() + "/calendar/events/" + m["id"].(string)
	f.do("PUT", E, ta, f.ev("B", time.Hour, `,"version":1`))
	f.do("DELETE", E, ta, "")
	for _, a := range []string{"calendar.event.create", "calendar.event.update", "calendar.event.delete"} {
		require.Equal(t, 1, count(f, `select count(*) from audit_log where action = $1 and course_id = $2 and actor_id = $3`, a, f.course, f.ta), a)
	}
	require.Equal(t, 3, count(f, `select count(*) from outbox where topic = 'calendar.changed'`))
}

// TestEventArchivedCourse — AC3: lớp ARCHIVED → 409 COURSE_ARCHIVED; sự kiện của lớp khác → 404.
func TestEventArchivedCourse(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	arch := f.newCourse("ARCHIVED")
	f.enroll(arch, f.ta, "TA", "ACTIVE")
	ta := f.tok(f.ta, auth.RoleTA)
	st, _, b := f.do("POST", "/api/v1/courses/"+arch.String()+"/calendar/events", ta, f.ev("A", time.Hour, ""))
	require.Equal(t, 409, st)
	require.Contains(t, string(b), "COURSE_ARCHIVED")
	foreign := f.event(f.newCourse("ACTIVE"), "OTHER", "Lớp khác", time.Hour)
	st, _, _ = f.do("PUT", "/api/v1/courses/"+f.course.String()+"/calendar/events/"+foreign.String(), ta, f.ev("x", time.Hour, `,"version":1`))
	require.Equal(t, 404, st)
	st, _, _ = f.do("DELETE", "/api/v1/courses/"+f.course.String()+"/calendar/events/"+foreign.String(), ta, "")
	require.Equal(t, 404, st)
	require.Equal(t, 1, count(f, `select count(*) from calendar_events where id = $1`, foreign))
}

func (f *fx) etag(user uuid.UUID, role auth.Role) string {
	f.t.Helper()
	st, h, b := f.do("GET", f.calURL(f.course, 0, 7*24*time.Hour), f.tok(user, role), "")
	require.Equal(f.t, 200, st, string(b))
	return h.Get("ETag")
}

// TestCalendarETag304 — AC4: ETag = băm thân; If-None-Match trùng → 304 không thân.
func TestCalendarETag304(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.event(f.course, "OTHER", "A", time.Hour)
	tag := f.etag(f.sv, auth.RoleStudent)
	require.NotEmpty(t, tag)
	require.Equal(t, tag, f.etag(f.sv, auth.RoleStudent))
	st, _, b := f.do("GET", f.calURL(f.course, 0, 7*24*time.Hour), f.tok(f.sv, auth.RoleStudent), "", "If-None-Match", tag)
	require.Equal(t, http.StatusNotModified, st)
	require.Empty(t, b)
}

// TestCalendarETagChangesOnWrite — AC4: sửa sự kiện → ETag đổi, yêu cầu kế trả 200 với giờ mới.
func TestCalendarETagChangesOnWrite(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	_, m := f.createEvent(f.tok(f.ta, auth.RoleTA), f.ev("A", time.Hour, ""))
	tag := f.etag(f.sv, auth.RoleStudent)
	E := "/api/v1/courses/" + f.course.String() + "/calendar/events/" + m["id"].(string)
	st, _, _ := f.do("PUT", E, f.tok(f.ta, auth.RoleTA), f.ev("A", 2*time.Hour, `,"version":1`))
	require.Equal(t, 200, st)
	st, _, _ = f.do("GET", f.calURL(f.course, 0, 7*24*time.Hour), f.tok(f.sv, auth.RoleStudent), "", "If-None-Match", tag)
	require.Equal(t, 200, st)
	require.NotEqual(t, tag, f.etag(f.sv, auth.RoleStudent))
}

// TestCalendarETagChangesOnAttemptStart — AC4: bắt đầu làm bài → personal_state đổi → ETag của chính sinh viên đó đổi, của bạn học không đổi.
func TestCalendarETagChangesOnAttemptStart(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	e := f.exam(f.course, "Thi", "OPEN", -10*time.Minute)
	a1, a2 := f.etag(f.sv, auth.RoleStudent), f.etag(f.sv2, auth.RoleStudent)
	f.attempt(f.course, e, f.sv, "IN_PROGRESS")
	require.NotEqual(t, a1, f.etag(f.sv, auth.RoleStudent))
	require.Equal(t, a2, f.etag(f.sv2, auth.RoleStudent))
}

// TestCalendarETagChangesWithEffectiveStatus — AC4: qua giờ mở, trạng thái hiệu lực đổi SCHEDULED → OPEN dù không ai sửa dòng nào.
func TestCalendarETagChangesWithEffectiveStatus(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	f.exam(f.course, "Thi", "SCHEDULED", 10*time.Minute)
	before := f.etag(f.sv, auth.RoleStudent)
	require.Equal(t, "SCHEDULED", f.items(f.sv, auth.RoleStudent, 0, 7*24*time.Hour)[0]["status"])
	f.clk.Advance(15 * time.Minute)
	require.NotEqual(t, before, f.etag(f.sv, auth.RoleStudent))
	require.Equal(t, "OPEN", f.items(f.sv, auth.RoleStudent, -time.Hour, 7*24*time.Hour)[0]["status"])
}

// TestPersonalStateOwnOnly — AC5: personal_state chỉ của chính mình (NOT_STARTED / IN_PROGRESS / SUBMITTED); Staff không có; liên kết theo vai.
func TestPersonalStateOwnOnly(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	e := f.exam(f.course, "Thi", "OPEN", -10*time.Minute)
	state := func(u uuid.UUID, r auth.Role) any {
		return f.items(u, r, -time.Hour, 24*time.Hour)[0]["personal_state"]
	}
	require.Equal(t, "NOT_STARTED", state(f.sv, auth.RoleStudent))
	f.attempt(f.course, e, f.sv, "IN_PROGRESS")
	require.Equal(t, "IN_PROGRESS", state(f.sv, auth.RoleStudent))
	require.Equal(t, "NOT_STARTED", state(f.sv2, auth.RoleStudent), "không lộ trạng thái của bạn học")
	require.Nil(t, state(f.ta, auth.RoleTA))
	require.Nil(t, state(f.teacher, auth.RoleTeacher))
	_, err := f.pool.Exec(t.Context(), `update exam_attempts set status = 'GRADING', submitted_at = now(), submit_reason = 'MANUAL' where student_id = $1`, f.sv)
	require.NoError(t, err)
	require.Equal(t, "SUBMITTED", state(f.sv, auth.RoleStudent))
	require.Equal(t, "/exams/"+e.String()+"/take", f.items(f.sv, auth.RoleStudent, -time.Hour, 24*time.Hour)[0]["href"])
	require.Equal(t, "/exams/"+e.String(), f.items(f.ta, auth.RoleTA, -time.Hour, 24*time.Hour)[0]["href"])
}

// TestCalendarNoExamContentLeak — AC5: thân phản hồi của sinh viên không có điểm, câu hỏi, đáp án, mô tả.
func TestCalendarNoExamContentLeak(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	e := f.exam(f.course, "Thi", "OPEN", -10*time.Minute)
	_, err := f.pool.Exec(t.Context(), `update exams set instructions = 'ĐỀ-BÍ-MẬT' where id = $1`, e)
	require.NoError(t, err)
	f.attempt(f.course, e, f.sv2, "GRADING")
	st, _, b := f.do("GET", f.calURL(f.course, -time.Hour, 24*time.Hour), f.tok(f.sv, auth.RoleStudent), "")
	require.Equal(t, 200, st)
	for _, banned := range []string{"ĐỀ-BÍ-MẬT", "score", "question", "answer", "instructions", f.sv2.String()} {
		require.NotContains(t, strings.ToLower(string(b)), strings.ToLower(banned))
	}
}

// TestCalendarMatrix — AC15: STUDENT / TA / TEACHER đọc; ghi chỉ TA / TEACHER; ADMIN, người ngoài lớp, PENDING, REMOVED → 403; chưa đăng nhập → 401.
func TestCalendarMatrix(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	pending, removed, outsider := f.user("P", "STUDENT", "ACTIVE"), f.user("R", "STUDENT", "ACTIVE"), f.user("O", "STUDENT", "ACTIVE")
	f.enroll(f.course, pending, "STUDENT", "PENDING")
	f.enroll(f.course, removed, "STUDENT", "REMOVED")
	admin := f.user("AD", "ADMIN", "ACTIVE")
	type who struct {
		name string
		tok  string
		read int

		write int
	}
	cases := []who{
		{"student", f.tok(f.sv, auth.RoleStudent), 200, 403},
		{"ta", f.tok(f.ta, auth.RoleTA), 200, 201},
		{"teacher", f.tok(f.teacher, auth.RoleTeacher), 200, 201},
		{"admin", f.tok(admin, auth.RoleAdmin), 403, 403},
		{"outsider", f.tok(outsider, auth.RoleStudent), 403, 403},
		{"pending", f.tok(pending, auth.RoleStudent), 403, 403},
		{"removed", f.tok(removed, auth.RoleStudent), 403, 403},
		{"anon", "", 401, 401},
	}
	for _, c := range cases {
		st, _, b := f.do("GET", f.calURL(f.course, 0, 24*time.Hour), c.tok, "")
		require.Equal(t, c.read, st, "đọc %s: %s", c.name, b)
		st, _, b = f.do("POST", "/api/v1/courses/"+f.course.String()+"/calendar/events", c.tok, f.ev("x", time.Hour, ""))
		require.Equal(t, c.write, st, "ghi %s: %s", c.name, b)
	}
}

// TestChangeReflectedInCalendarAndTool — AC6: đổi giờ sự kiện → GET calendar kế tiếp và tool lịch thấy giờ mới ngay (không cache).
func TestChangeReflectedInCalendarAndTool(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	_, m := f.createEvent(f.tok(f.ta, auth.RoleTA), f.ev("Thi giữa kỳ", 48*time.Hour, ""))
	tc := tcOf(f)
	before, ok, err := f.svc.ExamSchedule(t.Context(), tc)
	require.NoError(t, err)
	require.True(t, ok)
	newStart := time.Date(2026, 12, 20, 7, 0, 0, 0, time.UTC)
	E := "/api/v1/courses/" + f.course.String() + "/calendar/events/" + m["id"].(string)
	st, _, b := f.do("PUT", E, f.tok(f.ta, auth.RoleTA), fmt.Sprintf(`{"type":"EXAM","title":"Thi giữa kỳ","starts_at":%q,"version":1}`, newStart.Format(time.RFC3339)))
	require.Equal(t, 200, st, string(b))
	f.clk.Set(newStart.Add(-48 * time.Hour))
	after, ok, err := f.svc.ExamSchedule(t.Context(), tc)
	require.NoError(t, err)
	require.True(t, ok)
	require.NotEqual(t, before, after)
	require.Contains(t, fmt.Sprint(after), "14:00")
	require.Contains(t, fmt.Sprint(after), "20/12")
}

// TestCalendarChangedInvalidatesToday — AC6: outbox calendar.changed → Invalidator xoá cache "Hôm nay" của thành viên lớp.
func TestCalendarChangedInvalidatesToday(t *testing.T) {
	t.Parallel()
	f := newFx(t)
	_, _ = f.createEvent(f.tok(f.ta, auth.RoleTA), f.ev("A", time.Hour, ""))
	var payload []byte
	require.NoError(t, f.pool.QueryRow(t.Context(), `select payload from outbox where topic = 'calendar.changed'`).Scan(&payload))
	keys := []string{today.CacheKey(f.sv, "all"), today.CacheKey(f.sv, f.course.String()), today.CacheKey(f.ta, "all")}
	for _, k := range keys {
		require.NoError(t, f.rdb.Set(t.Context(), k, "x", time.Minute).Err())
	}
	inv := today.Invalidator{Pool: f.pool, Redis: f.rdb}
	require.NoError(t, inv.Handle(t.Context(), outbox.Message{Topic: "calendar.changed", Payload: payload}))
	for _, k := range keys {
		n, err := f.rdb.Exists(t.Context(), k).Result()
		require.NoError(t, err)
		require.Zero(t, n, k)
	}
	require.Contains(t, today.Topics(), "calendar.changed")
}

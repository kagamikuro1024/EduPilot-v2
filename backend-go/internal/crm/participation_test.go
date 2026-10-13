package crm_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/agent"
	"github.com/edupilot/backend-go/internal/crm"
)

func (w world) speak(s session, student uuid.UUID, extra map[string]any) resp {
	b := map[string]any{"student_id": student.String(), "type": "SPEAK", "session_id": w.ses[0].String(), "client_event_id": uuid.NewString()}
	for k, v := range extra {
		b[k] = v
	}
	return w.r.call(http.MethodPost, "/courses/"+w.course.String()+"/participation", s, b)
}

func TestParticipationSpeakIdempotentClientID(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	ev := uuid.NewString()
	a := w.speak(w.T, w.students[0].ID, map[string]any{"client_event_id": ev})
	require.Equal(t, http.StatusCreated, a.code, string(a.body))
	require.Equal(t, "0.25", a.json()["points"], "điểm mặc định của phát biểu")
	b := w.speak(w.T, w.students[0].ID, map[string]any{"client_event_id": ev})
	require.Equal(t, http.StatusOK, b.code, string(b.body))
	require.Equal(t, a.json()["id"], b.json()["id"], "gửi lại cùng client_event_id ⇒ cùng sự kiện")
	require.Equal(t, 1, r.count(`select count(*) from participation_events where course_id = $1`, w.course))
	// bấm liên tiếp ba lần = ba sự kiện (mỗi lần một client_event_id)
	for range 3 {
		require.Equal(t, http.StatusCreated, w.speak(w.T, w.students[0].ID, nil).code)
	}
	require.Equal(t, 4, r.count(`select count(*) from participation_events where course_id = $1`, w.course))
	require.Equal(t, "1.00", r.scalar(`select sum(points)::text from participation_events where course_id = $1`, w.course)[:4])
	// lưới hiện số lần phát biểu của buổi
	for _, s := range items(w.grid(w.T, 0), "students") {
		if s["id"] == w.students[0].ID.String() {
			require.Equal(t, float64(4), s["speak_count"])
		}
	}
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity = 'participation' and action = 'create' and entity_id = $1`, a.json()["id"].(string)), "gửi lại không audit thêm")
}

func TestParticipationReplayOtherCourse409(t *testing.T) {
	r := newRig(t)
	w1, w2 := r.world(1), r.world(1)
	ev := uuid.NewString()
	require.Equal(t, http.StatusCreated, w1.speak(w1.T, w1.students[0].ID, map[string]any{"client_event_id": ev}).code)
	res := w2.speak(w2.T, w2.students[0].ID, map[string]any{"client_event_id": ev})
	require.Equal(t, http.StatusConflict, res.code, string(res.body))
	require.Equal(t, "IDEMPOTENCY_KEY_REUSED", res.errCode())
	require.NotContains(t, string(res.body), w1.students[0].ID.String(), "không lộ sự kiện của lớp khác")
	// cùng lớp nhưng người khác (TA dùng khoá của giảng viên) cũng không nhận lại sự kiện
	res = w1.speak(w1.A, w1.students[0].ID, map[string]any{"client_event_id": ev})
	require.Equal(t, http.StatusConflict, res.code, string(res.body))
	require.Equal(t, "IDEMPOTENCY_KEY_REUSED", res.errCode())
	require.Equal(t, 1, r.count(`select count(*) from participation_events`))
}

func TestParticipationUndo(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	id := w.speak(w.T, w.students[0].ID, nil).json()["id"].(string)
	path := "/courses/" + w.course.String() + "/participation/" + id
	require.Equal(t, http.StatusNoContent, r.call(http.MethodDelete, path, w.T, nil).code)
	require.Zero(t, r.count(`select count(*) from participation_events`))
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity = 'participation' and action = 'delete'`))
	require.Equal(t, http.StatusNotFound, r.call(http.MethodDelete, path, w.T, nil).code)
}

func TestParticipationDeleteOwnerOrTeacher(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	byTeacher := w.speak(w.T, w.students[0].ID, nil).json()["id"].(string)
	byTA := w.speak(w.A, w.students[0].ID, nil).json()["id"].(string)
	base := "/courses/" + w.course.String() + "/participation/"
	require.Equal(t, http.StatusForbidden, r.call(http.MethodDelete, base+byTeacher, w.A, nil).code, "TA không xoá sự kiện của người khác")
	require.Equal(t, http.StatusNoContent, r.call(http.MethodDelete, base+byTA, w.A, nil).code, "TA xoá sự kiện của chính mình")
	require.Equal(t, http.StatusNoContent, r.call(http.MethodDelete, base+byTeacher, w.T, nil).code)
	again := w.speak(w.A, w.students[0].ID, nil).json()["id"].(string)
	require.Equal(t, http.StatusNoContent, r.call(http.MethodDelete, base+again, w.T, nil).code, "TEACHER xoá được sự kiện của TA")
}

func TestParticipationStudentNotInCourse(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	for name, id := range map[string]uuid.UUID{"PENDING": w.pending.ID, "ngoài lớp": uuid.New()} {
		res := w.speak(w.T, id, nil)
		require.Equal(t, http.StatusUnprocessableEntity, res.code, name+": "+string(res.body))
		require.Equal(t, "STUDENT_NOT_IN_COURSE", res.errCode(), name)
	}
	other := r.world(1)
	res := w.speak(w.T, w.students[0].ID, map[string]any{"session_id": other.ses[0].String()})
	require.Equal(t, http.StatusUnprocessableEntity, res.code, "buổi của lớp khác")
	require.Zero(t, r.count(`select count(*) from participation_events`))
}

func TestParticipationRBAC(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	sv := r.mustLogin(w.students[0].Email)
	path := "/courses/" + w.course.String() + "/participation"
	for name, tc := range map[string]struct {
		s    session
		want int
	}{"giảng viên": {w.T, 201}, "trợ giảng": {w.A, 201}, "sinh viên": {sv, 403}, "ADMIN": {w.ADM, 403}, "ngoài lớp": {w.OUT, 403}, "không JWT": {session{}, 401}} {
		require.Equal(t, tc.want, w.speak(tc.s, w.students[0].ID, nil).code, "POST "+name)
		want := tc.want
		if want == 201 {
			want = 200
		}
		require.Equal(t, want, r.call(http.MethodGet, path, tc.s, nil).code, "GET "+name)
	}
}

// TestPointsDecimal — điểm cộng là decimal, bước 0,01, dấu theo loại, miền [-5, 5]; cấm float64.
func TestPointsDecimal(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	sid := w.students[0].ID
	ok := func(typ string, pts any, want string) {
		res := w.speak(w.T, sid, map[string]any{"type": typ, "points": pts})
		require.Equal(t, http.StatusCreated, res.code, fmt.Sprint(typ, pts, string(res.body)))
		require.Equal(t, want, res.json()["points"])
	}
	bad := func(typ string, pts any) {
		res := w.speak(w.T, sid, map[string]any{"type": typ, "points": pts})
		require.Equal(t, http.StatusUnprocessableEntity, res.code, fmt.Sprint(typ, pts, string(res.body)))
	}
	ok("SPEAK", "0.30", "0.30")
	ok("SPEAK", 0, "0.00")
	ok("BONUS", "5", "5.00")
	ok("BONUS", "0.01", "0.01")
	ok("PENALTY", "-5", "-5.00")
	ok("PENALTY", -0.5, "-0.50")
	bad("SPEAK", "-0.25")
	bad("BONUS", "0")
	bad("BONUS", "-1")
	bad("PENALTY", "1")
	bad("PENALTY", "0")
	bad("BONUS", "5.01")
	bad("PENALTY", "-5.01")
	bad("SPEAK", "0.255")
	bad("LAC", "1")
	res := w.speak(w.T, sid, map[string]any{"type": "BONUS"}) // BONUS bắt buộc có điểm
	require.Equal(t, http.StatusUnprocessableEntity, res.code)
	// 0,1 + 0,2 phải đúng 0,30 (float64 cho 0,30000000000000004)
	r.exec(`delete from participation_events`)
	ok("SPEAK", "0.10", "0.10")
	ok("SPEAK", "0.20", "0.20")
	require.Equal(t, "0.30", r.scalar(`select sum(points)::text from participation_events`))
}

func TestParticipationListCursor(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	for i := range 5 {
		require.Equal(t, http.StatusCreated, w.speak(w.T, w.students[i%2].ID, nil).code)
	}
	base := "/courses/" + w.course.String() + "/participation"
	p1 := r.call(http.MethodGet, base+"?limit=2", w.T, nil)
	require.Equal(t, http.StatusOK, p1.code, string(p1.body))
	require.Len(t, items(p1, "items"), 2)
	next := p1.json()["next_cursor"].(string)
	p2 := r.call(http.MethodGet, base+"?limit=2&cursor="+next, w.T, nil)
	require.Len(t, items(p2, "items"), 2)
	p3 := r.call(http.MethodGet, base+"?limit=2&cursor="+p2.json()["next_cursor"].(string), w.T, nil)
	require.Len(t, items(p3, "items"), 1)
	require.Nil(t, p3.json()["next_cursor"])
	mine := r.call(http.MethodGet, base+"?student_id="+w.students[0].ID.String(), w.T, nil)
	require.Len(t, items(mine, "items"), 3)
	require.Equal(t, http.StatusUnprocessableEntity, r.call(http.MethodGet, base+"?student_id=x", w.T, nil).code)
}

// ===== AC14: nguồn dữ liệu cho tool AI =====

func (w world) tc(u uuid.UUID) agent.TrustedContext {
	return agent.TrustedContext{UserID: u, CourseID: w.course, Role: "STUDENT"}
}

func TestToolMyAttendance(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	svc := &crm.Service{Pool: r.pool}
	_, ok, err := svc.Attendance(t.Context(), w.tc(w.students[0].ID))
	require.NoError(t, err)
	require.False(t, ok, "chưa có dòng nào ⇒ không có dữ liệu (tool trả câu mẫu, không gọi LLM)")
	// buổi 1: vắng; buổi 2: có mặt; sinh viên khác vắng cả hai (không được lẫn vào)
	require.Equal(t, http.StatusOK, w.put(w.T, 0, []rec{row(w.students[0].ID, "ABSENT", "note", "bệnh"), row(w.students[1].ID, "ABSENT")}, false).code)
	require.Equal(t, http.StatusOK, w.put(w.T, 1, []rec{row(w.students[0].ID, "PRESENT"), row(w.students[1].ID, "ABSENT")}, false).code)
	f, ok, err := svc.Attendance(t.Context(), w.tc(w.students[0].ID))
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, []any{2, 1, 0, 0, 1}, []any{f["recorded"], f["present"], f["late"], f["excused"], f["absent"]})
	recent := f["recent_absent"].([]map[string]any)
	require.Len(t, recent, 1)
	require.Equal(t, 1, recent[0]["session_no"])
	require.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, recent[0]["date"])
	require.NotContains(t, fmt.Sprint(f), "bệnh", "ghi chú điểm danh không đi vào kết quả tool")
	require.NotContains(t, fmt.Sprint(f), w.teacher.FullName, "không có tên người điểm")
}

func TestToolMyParticipation(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	svc := &crm.Service{Pool: r.pool}
	_, ok, err := svc.Participation(t.Context(), w.tc(w.students[0].ID))
	require.NoError(t, err)
	require.False(t, ok)
	for range 2 {
		require.Equal(t, http.StatusCreated, w.speak(w.T, w.students[0].ID, nil).code)
	}
	require.Equal(t, http.StatusCreated, w.speak(w.T, w.students[1].ID, nil).code)
	f, ok, err := svc.Participation(t.Context(), w.tc(w.students[0].ID))
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, []any{2, "0.50"}, []any{f["speak_count"], f["points_total"]}, "+0,50 cho hai lần phát biểu của CHÍNH mình")
	require.Len(t, f["recent"], 2)
	require.Equal(t, http.StatusCreated, w.speak(w.T, w.students[0].ID, nil).code)
	f, _, _ = svc.Participation(t.Context(), w.tc(w.students[0].ID))
	require.Equal(t, "0.75", f["points_total"])
}

// TestAgentSourcesReadOnlyOwnRows — tool chỉ đọc dòng của (lớp, người) trong trusted_context; lớp khác không lẫn vào.
func TestAgentSourcesReadOnlyOwnRows(t *testing.T) {
	r := newRig(t)
	w1, w2 := r.world(1), r.world(1)
	require.Equal(t, http.StatusOK, w2.put(w2.T, 0, []rec{row(w2.students[0].ID, "ABSENT")}, false).code)
	svc := &crm.Service{Pool: r.pool}
	_, ok, err := svc.Attendance(t.Context(), w1.tc(w2.students[0].ID)) // sinh viên lớp 2 hỏi trong ngữ cảnh lớp 1
	require.NoError(t, err)
	require.False(t, ok)
	_, ok, _ = svc.Attendance(t.Context(), w2.tc(w1.students[0].ID))
	require.False(t, ok)
	before := r.count(`select count(*) from audit_log`)
	_, _, _ = svc.Attendance(t.Context(), w2.tc(w2.students[0].ID))
	require.Equal(t, before, r.count(`select count(*) from audit_log`), "đọc của chính mình không ghi audit")
}

// TestChatAttendanceMatchesDB — AC14: qua đúng cửa tool của agent (`PrivateRegistry.Run`, danh tính chỉ từ trusted_context) khối `attendance` / `participation` khớp từng số với DB.
func TestChatAttendanceMatchesDB(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	svc := &crm.Service{Pool: r.pool}
	reg := agent.DefaultPrivateRegistry(svc, svc, nil, nil, nil)
	me, other := w.students[0].ID, w.students[1].ID
	require.Equal(t, http.StatusOK, w.put(w.T, 0, []rec{row(me, "ABSENT"), row(other, "ABSENT")}, false).code)
	require.Equal(t, http.StatusOK, w.put(w.T, 1, []rec{row(me, "ABSENT"), row(other, "PRESENT")}, false).code)
	require.Equal(t, http.StatusCreated, w.speak(w.T, me, nil).code)
	require.Equal(t, http.StatusCreated, w.speak(w.T, me, nil).code)
	att, err := reg.Run(t.Context(), "get_my_attendance", w.tc(me), nil)
	require.NoError(t, err)
	require.False(t, att.NoData)
	require.Equal(t, r.count(`select count(*) from attendance_records where student_id = $1 and status = 'ABSENT'`, me), att.Facts["absent"])
	require.Equal(t, r.count(`select count(*) from attendance_records where student_id = $1`, me), att.Facts["recorded"])
	part, err := reg.Run(t.Context(), "get_my_participation", w.tc(me), nil)
	require.NoError(t, err)
	require.Equal(t, r.count(`select count(*) from participation_events where student_id = $1 and type = 'SPEAK'`, me), part.Facts["speak_count"])
	require.Equal(t, r.scalar(`select sum(points)::numeric(6,2)::text from participation_events where student_id = $1`, me), part.Facts["points_total"])
}

// TestToolAttendanceNoteNotExposed — ghi chú điểm danh và tên người điểm không bao giờ vào kết quả tool (sinh viên không thấy ghi chú quan sát của giảng viên).
func TestToolAttendanceNoteNotExposed(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	svc := &crm.Service{Pool: r.pool}
	me := w.students[0].ID
	require.Equal(t, http.StatusOK, w.put(w.T, 0, []rec{row(me, "ABSENT", "note", "CANARY-GHI-CHU-RIENG")}, false).code)
	res, err := agent.DefaultPrivateRegistry(svc, svc, nil, nil, nil).Run(t.Context(), "get_my_attendance", w.tc(me), nil)
	require.NoError(t, err)
	out := fmt.Sprint(res.Facts, res.Block)
	require.NotContains(t, out, "CANARY-GHI-CHU-RIENG")
	require.NotContains(t, out, w.teacher.FullName)
}

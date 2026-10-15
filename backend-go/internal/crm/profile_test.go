package crm_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/llm"
	"github.com/edupilot/backend-go/internal/privacy"
)

func (w world) profileOf(s session, i int) resp {
	return w.r.call(http.MethodGet, "/courses/"+w.course.String()+"/students/"+w.students[i].ID.String()+"/profile", s, nil)
}

func (w world) sp(i int, tail string) string {
	return "/courses/" + w.course.String() + "/students/" + w.students[i].ID.String() + tail
}

func (r *rig) auditCount(entity, action string, id uuid.UUID) int {
	return r.count(`select count(*) from audit_log where entity = $1 and action = $2 and entity_id = $3`, entity, action, id.String())
}

// TestProfileOpenWritesOneAuditPerCall (+ WritesAuditEachOpen, NoStoreHeader): mở ba lần = ba request = ba dòng audit; no-store, không ETag.
func TestProfileOpenWritesOneAuditPerCall(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	s := w.students[0].ID
	for range 3 {
		res := w.profileOf(w.T, 0)
		require.Equal(t, http.StatusOK, res.code, string(res.body))
		require.Equal(t, "no-store", res.hdr.Get("Cache-Control"))
		require.Empty(t, res.hdr.Get("ETag"), "hồ sơ không ETag")
	}
	require.Equal(t, 3, r.auditCount("student_profile", "view", s))
	row := r.scalar(`select actor_id::text || '|' || course_id::text from audit_log where entity = 'student_profile' and entity_id = $1 limit 1`, s.String())
	require.Equal(t, w.teacher.ID.String()+"|"+w.course.String(), row)
	require.Equal(t, 1, func() int {
		w.profileOf(w.A, 0)
		return r.count(`select count(*) from audit_log where entity = 'student_profile' and actor_id = $1`, w.ta.ID)
	}(), "TA mở một lần = một dòng của TA")
}

// TestProfileBody: đủ khoá của hợp đồng; NONE không có câu; AT_RISK có câu + lý do (cùng nguồn với danh sách); không điểm, không nội dung chat.
func TestProfileBody(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	s := w.students[0].ID
	w.absentTwoWeeks(s)
	none := w.profileOf(w.T, 1).json()
	require.Equal(t, "NONE", none["risk"].(map[string]any)["level"])
	require.Nil(t, none["risk"].(map[string]any)["sentence"])
	require.NoError(t, r.svc().RunCourse(t.Context(), w.course, midweek()))
	p := w.profileOf(w.T, 0).json()
	risk := p["risk"].(map[string]any)
	require.Equal(t, "AT_RISK", risk["level"])
	require.Contains(t, risk["sentence"], "Cần chú ý: vắng 4/4 buổi trong 2 tuần gần nhất")
	require.NotEmpty(t, risk["reasons"])
	for _, k := range []string{"student", "attendance", "participation", "study", "activity"} {
		require.Contains(t, p, k)
	}
	require.Len(t, p["study"].(map[string]any)["weeks"], 8)
	body, _ := json.Marshal(p)
	for _, bad := range []string{"grade", "chat", "message"} {
		require.NotContains(t, string(body), bad)
	}
}

// TestProfileFailClosedWhenAuditFails: không ghi được audit → 503 và KHÔNG có dữ liệu trong thân.
func TestProfileFailClosedWhenAuditFails(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	r.exec(`create function deny_audit() returns trigger language plpgsql as $$ begin raise exception 'audit down'; end $$`)
	r.exec(`create trigger deny_profile_audit before insert on audit_log for each row when (new.entity = 'student_profile') execute function deny_audit()`)
	res := w.profileOf(w.T, 0)
	require.Equal(t, http.StatusServiceUnavailable, res.code)
	require.NotContains(t, string(res.body), w.students[0].ID.String(), "không trả dữ liệu khi chưa ghi được vết")
	require.Equal(t, http.StatusServiceUnavailable, r.call(http.MethodGet, w.sp(0, "/observations"), w.T, nil).code)
}

// TestProfileNotInCourse404 (+ TestNoAuditWhenProfileNotReturned): ngoài lớp / PENDING / REMOVED → 404 và KHÔNG có dòng audit.
func TestProfileNotInCourse404(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	gone := r.addUserRemoved(w)
	other := r.addUser(uniq("kl"), "STUDENT", "ACTIVE")
	for _, id := range []uuid.UUID{w.pending.ID, gone.ID, other.ID, uuid.New()} {
		res := r.call(http.MethodGet, "/courses/"+w.course.String()+"/students/"+id.String()+"/profile", w.T, nil)
		require.Equal(t, http.StatusNotFound, res.code, id.String())
		require.Equal(t, 0, r.auditCount("student_profile", "view", id))
	}
}

// TestProfileRBACMatrix (+ TestAllCourseRoutesGuarded ở contract): STUDENT (kể cả chính mình), ADMIN, nhân viên lớp khác → 403; TA / TEACHER ✓.
func TestProfileRBACMatrix(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	other := r.world(1) // lớp khác: giảng viên của lớp đó
	self := r.mustLogin(w.students[0].Email)
	routes := []struct{ method, tail string }{
		{http.MethodGet, "/profile"}, {http.MethodGet, "/observations"}, {http.MethodGet, "/attendance"}, {http.MethodGet, "/study-time"},
		{http.MethodPost, "/ai-summary"}, {http.MethodPost, "/outreach"}, {http.MethodPost, "/observations"},
	}
	for _, rt := range routes {
		for name, s := range map[string]session{"sinh viên (chính mình)": self, "admin": w.ADM, "giảng viên lớp khác": other.T, "người ngoài": w.OUT} {
			res := r.do(req{method: rt.method, path: w.sp(0, rt.tail), bearer: s.access, body: map[string]any{"body": "x"}, hdr: map[string]string{"Idempotency-Key": uuid.NewString()}})
			require.Equal(t, http.StatusForbidden, res.code, "%s %s %s: %s", name, rt.method, rt.tail, res.body)
		}
	}
	require.Equal(t, http.StatusOK, w.profileOf(w.A, 0).code)
	require.Equal(t, http.StatusOK, r.call(http.MethodGet, w.sp(0, "/observations"), w.A, nil).code)
}

// ---- ghi chú ----

func (w world) note(s session, i int, body string, extra map[string]any) resp {
	b := map[string]any{"body": body}
	for k, v := range extra {
		b[k] = v
	}
	return w.r.call(http.MethodPost, w.sp(i, "/observations"), s, b)
}

func TestObservationCRUD(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	res := w.note(w.T, 0, "Hay hỏi tốt", map[string]any{"tags": []string{"tích cực"}})
	require.Equal(t, http.StatusCreated, res.code, string(res.body))
	id := res.json()["id"].(string)
	require.Equal(t, "NOTE", res.json()["kind"])
	require.EqualValues(t, 1, res.json()["version"])
	w.note(w.A, 0, "Vắng hai buổi", nil)
	list := r.call(http.MethodGet, w.sp(0, "/observations"), w.A, nil)
	its := items(list, "items")
	require.Len(t, its, 2)
	require.Equal(t, "Vắng hai buổi", its[0]["body"], "mới nhất trước; TA đọc được ghi chú của giảng viên")
	up := r.call(http.MethodPut, w.sp(0, "/observations/"+id), w.T, map[string]any{"body": "Hay hỏi rất tốt", "tags": []string{}, "version": 1})
	require.Equal(t, http.StatusOK, up.code, string(up.body))
	require.EqualValues(t, 2, up.json()["version"])
	require.Equal(t, http.StatusNoContent, r.call(http.MethodDelete, w.sp(0, "/observations/"+id), w.T, nil).code)
	require.Equal(t, 0, r.count(`select count(*) from student_observations where id = $1`, id), "xoá thật")
	require.Equal(t, http.StatusNotFound, r.call(http.MethodDelete, w.sp(0, "/observations/"+id), w.T, nil).code)
}

func TestObservationValidation(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	tags := func(n, l int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = strings.Repeat("a", l)
		}
		return out
	}
	for name, c := range map[string]struct {
		body string
		x    map[string]any
	}{
		"rỗng": {"", nil}, "2001 ký tự": {strings.Repeat("a", 2001), nil}, "6 nhãn": {"x", map[string]any{"tags": tags(6, 3)}},
		"nhãn 31 ký tự": {"x", map[string]any{"tags": tags(1, 31)}}, "nhãn rỗng": {"x", map[string]any{"tags": []string{""}}}, "kind lạ": {"x", map[string]any{"kind": "SECRET"}},
		"NOTE đã sửa": {"x", map[string]any{"ai_edited": true}},
	} {
		res := w.note(w.T, 0, c.body, c.x)
		require.Equal(t, http.StatusUnprocessableEntity, res.code, "%s: %s", name, res.body)
	}
	require.Equal(t, http.StatusCreated, w.note(w.T, 0, strings.Repeat("a", 2000), map[string]any{"tags": tags(5, 30)}).code, "biên hợp lệ")
}

func TestObservationVersionConflict(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	id := w.note(w.T, 0, "Bản gốc", nil).json()["id"].(string)
	require.Equal(t, http.StatusOK, r.call(http.MethodPut, w.sp(0, "/observations/"+id), w.T, map[string]any{"body": "Sửa lần một", "version": 1}).code)
	res := r.call(http.MethodPut, w.sp(0, "/observations/"+id), w.T, map[string]any{"body": "Sửa lần hai", "version": 1})
	require.Equal(t, http.StatusConflict, res.code)
	cur := res.details()["current"].(map[string]any)
	require.Equal(t, "Sửa lần một", cur["body"], "kèm bản hiện tại để chọn")
}

// TestObservationAuthorOnlyEdit: TA không sửa / xoá ghi chú của giảng viên và ngược lại → 403; ghi chú nguyên vẹn.
func TestObservationAuthorOnlyEdit(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	byT := w.note(w.T, 0, "Của giảng viên", nil).json()["id"].(string)
	byA := w.note(w.A, 0, "Của TA", nil).json()["id"].(string)
	require.Equal(t, http.StatusForbidden, r.call(http.MethodPut, w.sp(0, "/observations/"+byT), w.A, map[string]any{"body": "TA sửa", "version": 1}).code)
	require.Equal(t, http.StatusForbidden, r.call(http.MethodDelete, w.sp(0, "/observations/"+byT), w.A, nil).code)
	require.Equal(t, http.StatusForbidden, r.call(http.MethodPut, w.sp(0, "/observations/"+byA), w.T, map[string]any{"body": "GV sửa", "version": 1}).code)
	require.Equal(t, http.StatusForbidden, r.call(http.MethodDelete, w.sp(0, "/observations/"+byA), w.T, nil).code)
	require.Equal(t, "Của giảng viên", r.scalar(`select body from student_observations where id = $1`, byT))
}

// TestObservationsReadAudited + TestAuditHasNoNoteContent: đọc danh sách ghi `view_observations`; mọi audit của ghi chú KHÔNG chứa nội dung.
func TestObservationsReadAudited(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	const canary = "CANARY-NOTE-9Q"
	id := w.note(w.T, 0, canary, nil).json()["id"].(string)
	r.call(http.MethodPut, w.sp(0, "/observations/"+id), w.T, map[string]any{"body": canary + " v2", "version": 1})
	r.call(http.MethodGet, w.sp(0, "/observations"), w.T, nil)
	r.call(http.MethodDelete, w.sp(0, "/observations/"+id), w.T, nil)
	require.Equal(t, 1, r.auditCount("student_profile", "view_observations", w.students[0].ID))
	require.Equal(t, 3, r.count(`select count(*) from audit_log where entity = 'student_observation'`), "create / update / delete")
	require.Equal(t, 0, r.count(`select count(*) from audit_log where course_id = $1 and (before::text like '%'||$2||'%' or after::text like '%'||$2||'%' or entity_id like '%'||$2||'%')`, w.course, canary), "audit không chứa nội dung ghi chú")
}

// ---- tab Chuyên cần ----

func TestStudentAttendanceTab(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	s := w.students[0].ID
	for i, st := range []string{"PRESENT", "ABSENT", "LATE"} {
		w.mark(w.pastSession(1, i+1), s, st)
	}
	r.exec(`update attendance_records set note = 'Ốm' where student_id = $1 and status = 'ABSENT'`, s)
	w.mark(w.pastSession(1, 9), w.students[1].ID, "ABSENT") // sinh viên khác: không lẫn
	res := r.call(http.MethodGet, w.sp(0, "/attendance?limit=2"), w.T, nil)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	its := items(res, "items")
	require.Len(t, its, 2)
	require.NotNil(t, res.json()["next_cursor"])
	next := r.call(http.MethodGet, w.sp(0, "/attendance?limit=2&cursor="+res.json()["next_cursor"].(string)), w.T, nil)
	all := append(its, items(next, "items")...)
	require.Len(t, all, 3)
	notes := 0
	for _, it := range all {
		if it["note"] == "Ốm" {
			notes++
		}
	}
	require.Equal(t, 1, notes, "nhân viên thấy ghi chú buổi")
}

// ---- nhận xét AI ----

type stubLLM struct {
	calls int
	reqs  []llm.Request
	reply string
	err   error
	block bool
}

func (s *stubLLM) Structured(ctx context.Context, r llm.Request, _ json.RawMessage) (json.RawMessage, error) {
	s.calls++
	s.reqs = append(s.reqs, r)
	if s.block {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if s.err != nil {
		return nil, s.err
	}
	return json.RawMessage(s.reply), nil
}

const okReply = `{"summary":"Sinh viên đi học khá đều, ít hoạt động trên hệ thống.","strengths":["Đi học đều"],"concerns":["Ít hỏi bài"],"suggestions":["Khuyến khích hỏi trên Threads","b","c","d"]}`

// dataFor cho sinh viên có số liệu: điểm danh + phát biểu.
func (w world) dataFor(i int) {
	w.mark(w.pastSession(1, 1), w.students[i].ID, "ABSENT")
	w.r.exec(`insert into participation_events (course_id, student_id, type, points, client_event_id, created_by) values ($1, $2, 'SPEAK', 0.25, gen_random_uuid(), $3)`, w.course, w.students[i].ID, w.teacher.ID)
}

// TestAISummaryPayloadHasNoIdentity + OneLLMCall: đầu vào chỉ là số liệu — không tên / MSSV / email / nội dung ghi chú — và đúng MỘT lời gọi INSIGHT làn NEAR_REALTIME.
func TestAISummaryPayloadHasNoIdentity(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	w.rename(0, "Nguyễn Hoàng Tuấn")
	r.exec(`update enrollments set student_code_snapshot = 'B21DCCN777' where course_id = $1 and user_id = $2`, w.course, w.students[0].ID)
	w.dataFor(0)
	w.note(w.T, 0, "CANARY-OBS-4F ghi chú riêng", nil)
	stub := &stubLLM{reply: okReply}
	svc := r.svc()
	svc.LLM = stub
	d, err := svc.AISummary(t.Context(), w.teacher.ID, w.course, w.students[0].ID)
	require.NoError(t, err)
	require.Equal(t, 1, stub.calls, "đúng MỘT lời gọi LLM")
	req := stub.reqs[0]
	require.Equal(t, llm.TaskInsight, req.Task)
	require.NotNil(t, req.Lane)
	require.Equal(t, llm.LaneNearRealtime, *req.Lane)
	var all strings.Builder
	for _, m := range req.Messages {
		all.WriteString(m.Content)
	}
	for _, bad := range []string{"Nguyễn Hoàng Tuấn", "Tuấn", "B21DCCN777", w.students[0].Email, "CANARY-OBS-4F", w.students[0].ID.String()} {
		require.NotContains(t, all.String(), bad)
	}
	require.Contains(t, all.String(), `"recorded":1`)
	require.Len(t, d.Suggestions, 3, "cắt về ≤ 3 mục")
	require.Equal(t, 0, r.count(`select count(*) from student_observations where kind = 'AI_SUMMARY'`), "không lưu cho tới khi giảng viên bấm Lưu")
}

// TestAISummaryRedactsInventedName: mô hình bịa tên một sinh viên trong roster → bị che trước khi trả.
func TestAISummaryRedactsInventedName(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	w.rename(1, "Trần Quốc Bảo")
	w.dataFor(0)
	svc := r.svc()
	svc.PII = piiFor(r)
	svc.LLM = &stubLLM{reply: `{"summary":"Nên học nhóm cùng Trần Quốc Bảo.","strengths":[],"concerns":[],"suggestions":[]}`}
	d, err := svc.AISummary(t.Context(), w.teacher.ID, w.course, w.students[0].ID)
	require.NoError(t, err)
	require.NotContains(t, d.Summary, "Trần Quốc Bảo")
}

func TestAISummaryNoDataNoLLM(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	stub := &stubLLM{reply: okReply}
	svc := r.svc()
	svc.LLM = stub
	_, err := svc.AISummary(t.Context(), w.teacher.ID, w.course, w.students[0].ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "NOT_ENOUGH_DATA")
	require.Zero(t, stub.calls)
}

func TestAISummaryLLMDown503(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	w.dataFor(0)
	svc := r.svc()
	svc.LLM = &stubLLM{err: &llm.ErrOverloaded{RetryAfter: 7 * time.Second}}
	_, err := svc.AISummary(t.Context(), w.teacher.ID, w.course, w.students[0].ID)
	require.Contains(t, err.Error(), "OVERLOADED")
	svc.LLM = &stubLLM{err: llm.ErrNotConfigured}
	_, err = svc.AISummary(t.Context(), w.teacher.ID, w.course, w.students[0].ID)
	require.Contains(t, err.Error(), "LLM_UNAVAILABLE")
}

func TestAISummaryTimeout504(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	w.dataFor(0)
	svc := r.svc()
	svc.AITimeout = 50 * time.Millisecond
	svc.LLM = &stubLLM{block: true}
	t0 := time.Now()
	_, err := svc.AISummary(t.Context(), w.teacher.ID, w.course, w.students[0].ID)
	require.Contains(t, err.Error(), "DEADLINE_EXCEEDED")
	require.Less(t, time.Since(t0), 2*time.Second)
}

func TestAISummaryRateLimited(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	w.dataFor(0)
	svc := r.svc()
	svc.Redis = r.rdb
	svc.AIPerHour = 2
	svc.LLM = &stubLLM{reply: okReply}
	for range 2 {
		_, err := svc.AISummary(t.Context(), w.teacher.ID, w.course, w.students[0].ID)
		require.NoError(t, err)
	}
	_, err := svc.AISummary(t.Context(), w.teacher.ID, w.course, w.students[0].ID)
	require.Contains(t, err.Error(), "RATE_LIMITED")
	_, err = svc.AISummary(t.Context(), w.ta.ID, w.course, w.students[0].ID)
	require.NoError(t, err, "hạn mức theo từng người")
}

// TestAISummaryEditedFlag: Lưu tạo ghi chú kind=AI_SUMMARY; ai_edited phản ánh bản lưu khác bản AI; NOTE không được đánh dấu.
func TestAISummaryEditedFlag(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	a := w.note(w.T, 0, "Nhận xét AI nguyên bản", map[string]any{"kind": "AI_SUMMARY", "ai_edited": false})
	b := w.note(w.T, 0, "Nhận xét AI đã sửa", map[string]any{"kind": "AI_SUMMARY", "ai_edited": true})
	require.Equal(t, http.StatusCreated, a.code, string(a.body))
	require.Equal(t, false, a.json()["ai_edited"])
	require.Equal(t, true, b.json()["ai_edited"])
	require.Equal(t, "AI_SUMMARY", b.json()["kind"])
}

func piiFor(r *rig) *privacy.Detector {
	return &privacy.Detector{Roster: &privacy.Roster{Src: privacy.StoreRoster{Pool: r.pool}, Redis: r.rdb, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}}
}

package course_test

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Lịch buổi học của US-P5-01 (FEAT-crm-360 SRS 4.2): xem trước, sửa, huỷ, buổi bù, số buổi, quyền.

// thứ Hai hằng tuần 09:00–11:30, 2026-10-12 → 2026-12-28, bỏ 2026-11-02 ⇒ 11 buổi + 1 ngày nghỉ (ví dụ của AC4).
func mondays(extra map[string]any) map[string]any {
	b := map[string]any{"weekdays": []int{1}, "start_time": "09:00", "end_time": "11:30", "from": "2026-10-12", "to": "2026-12-28", "exclude_dates": []string{"2026-11-02"}}
	for k, v := range extra {
		b[k] = v
	}
	return b
}

func (r *rig) previewGen(s session, courseID string, body map[string]any) resp { // xem trước: KHÔNG gửi Idempotency-Key
	return r.post(s, "/courses/"+courseID+"/sessions/generate?dry_run=true", body, nil)
}

func (r *rig) sessionIDs(s session, courseID string) []map[string]any {
	r.t.Helper()
	res := r.get(s, "/courses/"+courseID+"/sessions?limit=100")
	require.Equal(r.t, http.StatusOK, res.code, string(res.body))
	var out []map[string]any
	for _, it := range res.json()["items"].([]any) {
		out = append(out, it.(map[string]any))
	}
	return out
}

func TestGenerateDryRunNoWrites(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	outbox := r.count(`select count(*) from outbox`)
	audit := r.count(`select count(*) from audit_log where entity = 'class_sessions'`)
	res := r.previewGen(k.a.G, k.id, mondays(nil))
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	j := res.json()
	would := j["would_create"].([]any)
	require.Len(t, would, 11)
	first := would[0].(map[string]any)
	require.Equal(t, float64(1), first["session_no"])
	require.Equal(t, "2026-10-12T02:00:00Z", utc(t, first["starts_at"]), "09:00 ICT = 02:00 UTC")
	require.Equal(t, []any{map[string]any{"date": "2026-11-02", "reason": "EXCLUDED"}}, j["skipped"])
	require.Zero(t, r.count(`select count(*) from class_sessions where course_id = $1`, k.id))
	require.Equal(t, audit, r.count(`select count(*) from audit_log where entity = 'class_sessions'`), "xem trước không ghi audit_log")
	require.Equal(t, outbox, r.count(`select count(*) from outbox`), "xem trước không ghi outbox")
}

func TestGenerateDryRunNoIdempotencyKey(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	require.Equal(t, http.StatusOK, r.previewGen(k.a.G, k.id, mondays(nil)).code)
	// khoá của lần chạy thật không bị xem trước chiếm: cùng khoá dùng cho cả hai vẫn chạy thật được
	key := idemKey()
	pv := r.post(k.a.G, "/courses/"+k.id+"/sessions/generate?dry_run=true", mondays(nil), key)
	require.Equal(t, http.StatusOK, pv.code, string(pv.body))
	real := r.post(k.a.G, "/courses/"+k.id+"/sessions/generate", mondays(nil), key)
	require.Equal(t, http.StatusCreated, real.code, string(real.body))
	require.Equal(t, http.StatusUnprocessableEntity, r.post(k.a.G, "/courses/"+k.id+"/sessions/generate", mondays(nil), nil).code, "chạy thật vẫn đòi khoá")
	require.Equal(t, http.StatusUnprocessableEntity, r.post(k.a.G, "/courses/"+k.id+"/sessions/generate?dry_run=false", mondays(nil), nil).code)
}

func TestGeneratePreviewEqualsCreate(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	// đã có buổi 1 và 2 (ví dụ buổi cũ) để số xem trước phải nối tiếp, không bắt đầu từ 1
	require.Equal(t, http.StatusCreated, r.post(k.a.G, "/courses/"+k.id+"/sessions/generate", mondays(map[string]any{"from": "2026-09-07", "to": "2026-09-14", "exclude_dates": []string{}}), idemKey()).code)
	pv := r.previewGen(k.a.G, k.id, mondays(nil)).json()["would_create"].([]any)
	res := r.post(k.a.G, "/courses/"+k.id+"/sessions/generate", mondays(nil), idemKey())
	require.Equal(t, http.StatusCreated, res.code, string(res.body))
	require.Equal(t, float64(len(pv)), res.json()["created"])
	require.Equal(t, pv[0].(map[string]any)["session_no"], res.json()["first_session_no"])
	require.Equal(t, pv[len(pv)-1].(map[string]any)["session_no"], res.json()["last_session_no"])
	got := r.sessionIDs(k.a.G, k.id)[2:] // sau hai buổi đã có
	require.Len(t, got, len(pv))
	for i, it := range got {
		w := pv[i].(map[string]any)
		require.Equal(t, w["session_no"], it["session_no"], i)
		require.Equal(t, w["starts_at"], it["starts_at"], i)
		require.Equal(t, w["ends_at"], it["ends_at"], i)
	}
}

func TestGenerateValidation(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	for name, body := range map[string]map[string]any{
		"from > to":          mondays(map[string]any{"from": "2026-12-28", "to": "2026-10-12"}),
		"end ≤ start":        mondays(map[string]any{"end_time": "09:00"}),
		"weekdays rỗng":      mondays(map[string]any{"weekdays": []int{}}),
		"weekdays ngoài 1–7": mondays(map[string]any{"weekdays": []int{0}}),
		"room > 40":          mondays(map[string]any{"room": "P.0123456789012345678901234567890123456789"}),
		"ngày không có":      mondays(map[string]any{"from": "2026-02-30"}),
	} {
		res := r.previewGen(k.a.G, k.id, body)
		require.Equal(t, http.StatusUnprocessableEntity, res.code, name+": "+string(res.body))
		require.Equal(t, "VALIDATION_FAILED", res.errCode(), name)
	}
}

func TestGenerateTooMany(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	daily := map[string]any{"weekdays": []int{1, 2, 3, 4, 5, 6, 7}, "start_time": "08:00", "end_time": "09:00", "from": "2026-09-01", "to": "2026-11-13"} // 74 ngày
	for _, res := range []resp{r.previewGen(k.a.G, k.id, daily), r.post(k.a.G, "/courses/"+k.id+"/sessions/generate", daily, idemKey())} {
		require.Equal(t, http.StatusUnprocessableEntity, res.code, string(res.body))
		require.Contains(t, string(res.body), "TOO_MANY_SESSIONS")
		require.Contains(t, string(res.body), "74", "nêu số buổi sẽ tạo")
		require.Contains(t, string(res.body), "60", "nêu mức tối đa")
	}
	require.Zero(t, r.count(`select count(*) from class_sessions where course_id = $1`, k.id))
}

func TestGenerateOneTxAndAudit(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	changed := r.count(`select count(*) from outbox where topic = 'course.changed' and payload->>'course_id' = $1`, k.id)
	res := r.post(k.a.G, "/courses/"+k.id+"/sessions/generate", mondays(nil), idemKey())
	require.Equal(t, http.StatusCreated, res.code, string(res.body))
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity = 'class_sessions' and action = 'generate' and course_id = $1 and (after->>'created')::int = 11 and actor_id = $2`, k.id, k.a.gv.ID))
	require.Equal(t, changed+1, r.count(`select count(*) from outbox where topic = 'course.changed' and payload->>'course_id' = $1`, k.id), "đúng một sự kiện course.changed cho cả lô")
}

func TestGenerateIdempotentKey(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	key := idemKey()
	a := r.post(k.a.G, "/courses/"+k.id+"/sessions/generate", mondays(nil), key)
	b := r.post(k.a.G, "/courses/"+k.id+"/sessions/generate", mondays(nil), key)
	require.Equal(t, http.StatusCreated, a.code)
	require.JSONEq(t, string(a.body), string(b.body), "cùng khoá ⇒ cùng phản hồi")
	require.Equal(t, 11, r.count(`select count(*) from class_sessions where course_id = $1`, k.id))
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity = 'class_sessions' and action = 'generate' and course_id = $1`, k.id))
}

func TestGenerateRerunSkipsDuplicates(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	require.Equal(t, http.StatusCreated, r.post(k.a.G, "/courses/"+k.id+"/sessions/generate", mondays(nil), idemKey()).code)
	again := r.post(k.a.G, "/courses/"+k.id+"/sessions/generate", mondays(nil), idemKey())
	require.Equal(t, http.StatusCreated, again.code)
	require.Equal(t, float64(0), again.json()["created"])
	require.Equal(t, float64(11), again.json()["skipped"])
	require.Equal(t, 11, r.count(`select count(*) from class_sessions where course_id = $1`, k.id))
	pv := r.previewGen(k.a.G, k.id, mondays(nil)).json()
	require.Empty(t, pv["would_create"])
	dup, excl := 0, 0
	for _, s := range pv["skipped"].([]any) {
		switch s.(map[string]any)["reason"] {
		case "DUPLICATE":
			dup++
		case "EXCLUDED":
			excl++
		}
	}
	require.Equal(t, []int{11, 1}, []int{dup, excl})
}

func TestGenerateRollsBackOnError(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	// giả lập lỗi giữa chừng: buổi thứ 3 của lô làm DB từ chối
	_, err := r.pool.Exec(t.Context(), `create function zz_fail_session_3() returns trigger language plpgsql as $$ begin if new.session_no = 3 then raise exception 'giả lập'; end if; return new; end $$`)
	require.NoError(t, err)
	_, err = r.pool.Exec(t.Context(), `create trigger zz_fail before insert on class_sessions for each row execute function zz_fail_session_3()`)
	require.NoError(t, err)
	res := r.post(k.a.G, "/courses/"+k.id+"/sessions/generate", mondays(nil), idemKey())
	require.Equal(t, http.StatusInternalServerError, res.code, string(res.body))
	require.Zero(t, r.count(`select count(*) from class_sessions where course_id = $1`, k.id), "không buổi nào được tạo")
	require.Zero(t, r.count(`select count(*) from audit_log where entity = 'class_sessions' and course_id = $1`, k.id))
}

func TestGenerateConcurrentTwoKeys(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	var wg sync.WaitGroup
	out := make([]resp, 2)
	for i := range out {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out[i] = r.post(k.a.G, "/courses/"+k.id+"/sessions/generate", mondays(nil), idemKey())
		}()
	}
	wg.Wait()
	created, skipped := 0.0, 0.0
	for _, o := range out {
		require.Equal(t, http.StatusCreated, o.code, string(o.body))
		created += o.json()["created"].(float64)
		skipped += o.json()["skipped"].(float64)
	}
	require.Equal(t, []float64{11, 11}, []float64{created, skipped}, "buổi trùng bị bỏ ở người sau")
	require.Equal(t, "11|11", r.scalar(`select count(*)||'|'||count(distinct session_no) from class_sessions where course_id = $1`, k.id), "không trùng session_no")
}

// utc đổi một mốc thời gian JSON (offset nào cũng được) về UTC để so sánh theo thời điểm.
func utc(t *testing.T, v any) string {
	t.Helper()
	at, err := time.Parse(time.RFC3339, v.(string))
	require.NoError(t, err)
	return at.UTC().Format(time.RFC3339)
}

func (r *rig) putSession(s session, courseID, sid string, body map[string]any) resp {
	return r.put(s, "/courses/"+courseID+"/sessions/"+sid, body)
}

func sessionBody(starts, ends string, version any, extra map[string]any) map[string]any {
	b := map[string]any{"starts_at": starts, "ends_at": ends, "version": version}
	for k, v := range extra {
		b[k] = v
	}
	return b
}

func TestSessionUpdate(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	require.Equal(t, http.StatusCreated, r.post(k.a.G, "/courses/"+k.id+"/sessions/generate", mondays(nil), idemKey()).code)
	s := r.sessionIDs(k.a.G, k.id)[1]
	sid := s["id"].(string)
	res := r.putSession(k.a.T, k.id, sid, sessionBody("2026-10-20T14:00:00+07:00", "2026-10-20T16:30:00+07:00", s["version"], map[string]any{"room": "P.305", "topic": "Học bù"}))
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	got := res.json()["session"].(map[string]any)
	require.Equal(t, []any{float64(2), float64(2), "2026-10-20T07:00:00Z", "P.305", "Học bù"}, []any{got["session_no"], got["version"], utc(t, got["starts_at"]), got["room"], got["topic"]}, "session_no không đổi, version + 1")
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity = 'class_sessions' and action = 'update' and entity_id = $1 and actor_id = $2`, sid, k.a.ta.ID))
	// PUT thay toàn bộ: bỏ trống room / topic = xoá
	res = r.putSession(k.a.G, k.id, sid, sessionBody("2026-10-20T14:00:00+07:00", "2026-10-20T16:30:00+07:00", 2, nil))
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Nil(t, res.json()["session"].(map[string]any)["room"])
}

func TestSessionUpdateVersionConflict(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	require.Equal(t, http.StatusCreated, r.post(k.a.G, "/courses/"+k.id+"/sessions/generate", mondays(nil), idemKey()).code)
	sid := r.sessionIDs(k.a.G, k.id)[0]["id"].(string)
	body := sessionBody("2026-10-12T10:00:00+07:00", "2026-10-12T12:00:00+07:00", 1, nil)
	require.Equal(t, http.StatusOK, r.putSession(k.a.G, k.id, sid, body).code)
	res := r.putSession(k.a.T, k.id, sid, body) // người thứ hai còn giữ version 1
	require.Equal(t, http.StatusConflict, res.code, string(res.body))
	require.Equal(t, "VERSION_CONFLICT", res.errCode())
	cur := res.details()["current"].(map[string]any)
	require.Equal(t, []any{float64(2), "2026-10-12T03:00:00Z"}, []any{cur["version"], utc(t, cur["starts_at"])}, "kèm bản hiện tại")
}

func TestSessionUpdateDuplicate(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	require.Equal(t, http.StatusCreated, r.post(k.a.G, "/courses/"+k.id+"/sessions/generate", mondays(nil), idemKey()).code)
	items := r.sessionIDs(k.a.G, k.id)
	second := items[1]["starts_at"].(string)
	res := r.putSession(k.a.G, k.id, items[0]["id"].(string), sessionBody(second, "2026-10-19T04:30:00Z", 1, nil))
	require.Equal(t, http.StatusConflict, res.code, string(res.body))
	require.Equal(t, "SESSION_DUPLICATE", res.errCode())
	require.Equal(t, "1", r.scalar(`select version::text from class_sessions where id = $1`, items[0]["id"].(string)), "không ghi gì")
}

func TestSessionUpdateValidation(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	require.Equal(t, http.StatusCreated, r.post(k.a.G, "/courses/"+k.id+"/sessions/generate", mondays(nil), idemKey()).code)
	sid := r.sessionIDs(k.a.G, k.id)[0]["id"].(string)
	for name, body := range map[string]map[string]any{
		"end ≤ start": sessionBody("2026-10-12T10:00:00+07:00", "2026-10-12T10:00:00+07:00", 1, nil),
		"room > 40":   sessionBody("2026-10-12T10:00:00+07:00", "2026-10-12T12:00:00+07:00", 1, map[string]any{"room": fmt.Sprintf("%041d", 0)}),
		"topic > 200": sessionBody("2026-10-12T10:00:00+07:00", "2026-10-12T12:00:00+07:00", 1, map[string]any{"topic": fmt.Sprintf("%0201d", 0)}),
	} {
		res := r.putSession(k.a.G, k.id, sid, body)
		require.Equal(t, http.StatusUnprocessableEntity, res.code, name+": "+string(res.body))
	}
	require.Equal(t, "1", r.scalar(`select version::text from class_sessions where id = $1`, sid))
}

func TestSessionDeleteNoAttendance(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	require.Equal(t, http.StatusCreated, r.post(k.a.G, "/courses/"+k.id+"/sessions/generate", mondays(nil), idemKey()).code)
	sid := r.sessionIDs(k.a.G, k.id)[3]["id"].(string)
	require.Equal(t, http.StatusNoContent, r.del(k.a.T, "/courses/"+k.id+"/sessions/"+sid).code)
	require.Equal(t, 10, r.count(`select count(*) from class_sessions where course_id = $1`, k.id))
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity = 'class_sessions' and action = 'delete' and entity_id = $1`, sid))
	require.Equal(t, http.StatusNotFound, r.del(k.a.G, "/courses/"+k.id+"/sessions/"+sid).code, "xoá lại ⇒ không còn")
}

// markAttended chèn một dòng điểm danh cho buổi (đường ghi thật của lưới là US-P5-02).
func (r *rig) markAttended(k klass, sid string) {
	r.t.Helper()
	sv, _ := r.student("")
	r.enroll(k.course, sv.ID, "STUDENT", "ACTIVE")
	_, err := r.pool.Exec(r.t.Context(), `insert into attendance_records (course_id, session_id, student_id, status, marked_by) values ($1, $2, $3, 'PRESENT', $4)`, k.id, sid, sv.ID, k.a.gv.ID)
	require.NoError(r.t, err)
}

func TestSessionDeleteBlockedWithAttendance(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	require.Equal(t, http.StatusCreated, r.post(k.a.G, "/courses/"+k.id+"/sessions/generate", mondays(nil), idemKey()).code)
	s := r.sessionIDs(k.a.G, k.id)[0]
	sid := s["id"].(string)
	require.Equal(t, false, s["attendance"].(map[string]any)["has_records"])
	r.markAttended(k, sid)
	res := r.del(k.a.G, "/courses/"+k.id+"/sessions/"+sid)
	require.Equal(t, http.StatusConflict, res.code, string(res.body))
	require.Equal(t, "SESSION_HAS_ATTENDANCE", res.errCode())
	require.Equal(t, 11, r.count(`select count(*) from class_sessions where course_id = $1`, k.id))
	require.Equal(t, true, r.sessionIDs(k.a.G, k.id)[0]["attendance"].(map[string]any)["has_records"])
	// vẫn sửa giờ / chủ đề được
	require.Equal(t, http.StatusOK, r.putSession(k.a.G, k.id, sid, sessionBody("2026-10-12T10:00:00+07:00", "2026-10-12T12:00:00+07:00", 1, map[string]any{"topic": "Đổi giờ"})).code)
}

func TestMakeupSession(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	require.Equal(t, http.StatusCreated, r.post(k.a.G, "/courses/"+k.id+"/sessions/generate", mondays(nil), idemKey()).code)
	// buổi bù = tạo một buổi một ngày bằng generate (thứ của ngày đó, from = to), rồi đặt chủ đề bằng PUT
	makeup := map[string]any{"weekdays": []int{6}, "start_time": "13:00", "end_time": "15:00", "from": "2026-11-07", "to": "2026-11-07"}
	res := r.post(k.a.G, "/courses/"+k.id+"/sessions/generate", makeup, idemKey())
	require.Equal(t, http.StatusCreated, res.code, string(res.body))
	require.Equal(t, []any{float64(1), float64(12), float64(12)}, []any{res.json()["created"], res.json()["first_session_no"], res.json()["last_session_no"]})
	var sid string
	for _, it := range r.sessionIDs(k.a.G, k.id) {
		if it["session_no"] == float64(12) {
			sid = it["id"].(string)
		}
	}
	require.NotEmpty(t, sid)
	require.Equal(t, http.StatusOK, r.putSession(k.a.G, k.id, sid, sessionBody("2026-11-07T13:00:00+07:00", "2026-11-07T15:00:00+07:00", 1, map[string]any{"topic": "Học bù"})).code)
}

func TestSessionNoGapNotFilled(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	require.Equal(t, http.StatusCreated, r.post(k.a.G, "/courses/"+k.id+"/sessions/generate", mondays(nil), idemKey()).code)
	items := r.sessionIDs(k.a.G, k.id)
	require.Equal(t, http.StatusNoContent, r.del(k.a.G, "/courses/"+k.id+"/sessions/"+items[4]["id"].(string)).code) // buổi 5, ở giữa
	res := r.post(k.a.G, "/courses/"+k.id+"/sessions/generate", map[string]any{"weekdays": []int{6}, "start_time": "13:00", "end_time": "15:00", "from": "2027-01-02", "to": "2027-01-02"}, idemKey())
	require.Equal(t, float64(12), res.json()["first_session_no"], "khoảng trống ở giữa không được lấp")
	require.Zero(t, r.count(`select count(*) from class_sessions where course_id = $1 and session_no = 5`, k.id))
}

func TestSessionNoReusedAfterDeleteLast(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	require.Equal(t, http.StatusCreated, r.post(k.a.G, "/courses/"+k.id+"/sessions/generate", mondays(nil), idemKey()).code)
	items := r.sessionIDs(k.a.G, k.id)
	require.Equal(t, http.StatusNoContent, r.del(k.a.G, "/courses/"+k.id+"/sessions/"+items[10]["id"].(string)).code) // buổi 11, cuối cùng
	res := r.post(k.a.G, "/courses/"+k.id+"/sessions/generate", map[string]any{"weekdays": []int{6}, "start_time": "13:00", "end_time": "15:00", "from": "2027-01-02", "to": "2027-01-02"}, idemKey())
	require.Equal(t, float64(11), res.json()["first_session_no"], "xoá buổi cuối thì số đó được cấp lại (TLR-4)")
}

func TestSessionsRBACMatrix(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	require.Equal(t, http.StatusCreated, r.post(k.a.G, "/courses/"+k.id+"/sessions/generate", mondays(nil), idemKey()).code)
	s := r.sessionIDs(k.a.G, k.id)[0]
	sid := s["id"].(string)
	_, member := r.student("")
	body := sessionBody("2026-10-12T10:00:00+07:00", "2026-10-12T12:00:00+07:00", s["version"], nil)
	for name, tc := range map[string]struct {
		s    session
		want int
	}{
		"giảng viên": {k.a.G, 200}, "trợ giảng": {k.a.T, 200}, "ADMIN": {k.a.A, 403}, "sinh viên ngoài lớp": {k.a.S, 403},
		"sinh viên khác": {member, 403}, "giảng viên lớp khác": {k.a.G2, 403}, "không JWT": {session{}, 401},
	} {
		body["version"] = mustAtoi(t, r.scalar(`select version::text from class_sessions where id = $1`, sid))
		require.Equal(t, tc.want, r.putSession(tc.s, k.id, sid, body).code, "PUT "+name)
		pv := r.previewGen(tc.s, k.id, mondays(nil))
		require.Equal(t, tc.want, pv.code, "xem trước "+name+": "+string(pv.body))
		del := r.del(tc.s, "/courses/"+k.id+"/sessions/"+uuid.NewString()) // buổi không có: người có quyền nhận 404, người khác 401 / 403
		if tc.want == 200 {
			require.Equal(t, http.StatusNotFound, del.code, "DELETE "+name)
		} else {
			require.Equal(t, tc.want, del.code, "DELETE "+name)
		}
	}
	// STUDENT trong lớp đọc được lịch nhưng không thấy trạng thái điểm danh
	in, svIn := r.student("")
	r.enroll(k.course, in.ID, "STUDENT", "ACTIVE")
	for _, it := range r.sessionIDs(svIn, k.id) {
		require.NotContains(t, it, "attendance")
	}
	require.Contains(t, r.sessionIDs(k.a.G, k.id)[0], "attendance")
}

func mustAtoi(t *testing.T, s string) int {
	t.Helper()
	var n int
	_, err := fmt.Sscan(s, &n)
	require.NoError(t, err)
	return n
}

func TestSessionsArchived(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	require.Equal(t, http.StatusCreated, r.post(k.a.G, "/courses/"+k.id+"/sessions/generate", mondays(nil), idemKey()).code)
	sid := r.sessionIDs(k.a.G, k.id)[0]["id"].(string)
	require.Equal(t, http.StatusOK, r.post(k.a.A, "/admin/courses/"+k.id+"/archive", nil, nil).code)
	for name, res := range map[string]resp{
		"generate": r.post(k.a.G, "/courses/"+k.id+"/sessions/generate", mondays(nil), idemKey()),
		"PUT":      r.putSession(k.a.G, k.id, sid, sessionBody("2026-10-12T10:00:00+07:00", "2026-10-12T12:00:00+07:00", 1, nil)),
		"DELETE":   r.del(k.a.G, "/courses/"+k.id+"/sessions/"+sid),
	} {
		require.Equal(t, http.StatusConflict, res.code, name+": "+string(res.body))
		require.Equal(t, "COURSE_ARCHIVED", res.errCode(), name)
	}
	require.Equal(t, 11, r.count(`select count(*) from class_sessions where course_id = $1`, k.id))
}

func TestSessionCrossCourse404(t *testing.T) {
	r := newRig(t)
	k1, k2 := r.klass(nil), r.klass(nil)
	require.Equal(t, http.StatusCreated, r.post(k2.a.G, "/courses/"+k2.id+"/sessions/generate", mondays(nil), idemKey()).code)
	sid := r.sessionIDs(k2.a.G, k2.id)[0]["id"].(string)
	// giảng viên lớp 1 gọi buổi của lớp 2 qua URL lớp mình ⇒ 404, không đụng dữ liệu
	require.Equal(t, http.StatusNotFound, r.putSession(k1.a.G, k1.id, sid, sessionBody("2026-10-12T10:00:00+07:00", "2026-10-12T12:00:00+07:00", 1, nil)).code)
	require.Equal(t, http.StatusNotFound, r.del(k1.a.G, "/courses/"+k1.id+"/sessions/"+sid).code)
	require.Equal(t, http.StatusNotFound, r.del(k1.a.G, "/courses/"+k1.id+"/sessions/khong-phai-uuid").code)
	require.Equal(t, 11, r.count(`select count(*) from class_sessions where course_id = $1`, k2.id))
}

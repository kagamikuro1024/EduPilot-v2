package crm_test

import (
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// ===== AC1: đọc lưới =====

func TestGridListsActiveStudentsOnly(t *testing.T) {
	r := newRig(t)
	w := r.world(3)
	removed := r.addUserRemoved(w)
	_ = removed
	res := w.grid(w.T, 0)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Equal(t, "no-store", res.hdr.Get("Cache-Control"), "điểm danh không cache")
	st := items(res, "students")
	require.Len(t, st, 3, "PENDING và REMOVED không có trong lưới")
	var names []string
	for _, s := range st {
		names = append(names, s["full_name"].(string))
		require.Nil(t, s["record"], "chưa có dòng ⇒ record null")
		require.Equal(t, float64(0), s["speak_count"])
	}
	require.True(t, sort.StringsAreSorted(names), "sắp theo tên: %v", names)
	ses := res.json()["session"].(map[string]any)
	require.Equal(t, w.ses[0].String(), ses["id"])
	require.Nil(t, ses["attendance_done_at"])
}

func TestGridSortsByNameWithoutDiacritics(t *testing.T) {
	r := newRig(t)
	w := r.world(0)
	for _, n := range []string{"Đặng An", "Bùi Bình", "Ân Dũng", "Cao Chi"} {
		u := r.addUserNamed(w, n)
		_ = u
	}
	var got []string
	for _, s := range items(w.grid(w.T, 0), "students") {
		got = append(got, s["full_name"].(string))
	}
	require.Equal(t, []string{"Ân Dũng", "Bùi Bình", "Cao Chi", "Đặng An"}, got)
}

func TestGridStaffOnly(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	sv := r.mustLogin(w.students[0].Email)
	for name, tc := range map[string]struct {
		s    session
		want int
	}{"giảng viên": {w.T, 200}, "trợ giảng": {w.A, 200}, "sinh viên trong lớp": {sv, 403}, "ADMIN": {w.ADM, 403}, "ngoài lớp": {w.OUT, 403}, "không JWT": {session{}, 401}} {
		require.Equal(t, tc.want, w.grid(tc.s, 0).code, name)
		require.Equal(t, tc.want, w.put(tc.s, 0, []rec{}, false).code, "PUT "+name)
	}
}

func TestGridCrossCourse404(t *testing.T) {
	r := newRig(t)
	w1, w2 := r.world(1), r.world(1)
	res := r.call(http.MethodGet, "/courses/"+w1.course.String()+"/sessions/"+w2.ses[0].String()+"/attendance", w1.T, nil)
	require.Equal(t, http.StatusNotFound, res.code)
	require.Equal(t, http.StatusNotFound, r.call(http.MethodPut, "/courses/"+w1.course.String()+"/sessions/"+w2.ses[0].String()+"/attendance", w1.T, map[string]any{"records": []rec{}}).code)
	require.Equal(t, http.StatusNotFound, r.call(http.MethodGet, "/courses/"+w1.course.String()+"/sessions/khong-phai-uuid/attendance", w1.T, nil).code)
}

// ===== AC2: ghi cả lưới =====

func TestPutGridOneTxWithAudit(t *testing.T) {
	r := newRig(t)
	w := r.world(30)
	before := r.count(`select count(*) from outbox where topic = 'crm.attendance_saved'`)
	recs := w.all("PRESENT")
	for i := range 3 {
		recs[i]["status"] = "ABSENT"
	}
	recs[3]["status"] = "LATE"
	res := w.put(w.T, 0, recs, false)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	out := items(res, "records")
	require.Len(t, out, 30)
	for _, o := range out {
		require.Equal(t, float64(1), o["version"])
	}
	require.Nil(t, res.json()["done_at"])
	require.Equal(t, 30, r.count(`select count(*) from attendance_records where session_id = $1`, w.ses[0]))
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity = 'attendance' and entity_id = $1 and action = 'save' and actor_id = $2`, w.ses[0].String(), w.teacher.ID), "đúng MỘT dòng audit cho cả lưới")
	require.Equal(t, before+1, r.count(`select count(*) from outbox where topic = 'crm.attendance_saved'`))
	require.Equal(t, "3", r.scalar(`select after->'counts'->>'ABSENT' from audit_log where entity = 'attendance' and entity_id = $1`, w.ses[0].String()), "đếm theo trạng thái")
	require.Equal(t, 30, r.count(`select jsonb_array_length(after->'changes') from audit_log where entity = 'attendance' and entity_id = $1`, w.ses[0].String()))
	require.Equal(t, 30, r.count(`select count(*) from attendance_records where session_id = $1 and marked_by = $2`, w.ses[0], w.teacher.ID), "marked_by là người gọi")
}

func TestPutGridNoopRowsKeepVersion(t *testing.T) {
	r := newRig(t)
	w := r.world(4)
	require.Equal(t, http.StatusOK, w.put(w.T, 0, w.all("PRESENT"), false).code)
	// đổi MỘT hàng (có version), ba hàng còn lại giống hệt ⇒ chỉ hàng đổi tăng version
	recs := w.all("PRESENT")
	recs[1]["status"], recs[1]["version"] = "ABSENT", 1
	res := w.put(w.T, 0, recs, false)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	vers := map[string]float64{}
	for _, o := range items(res, "records") {
		vers[o["student_id"].(string)] = o["version"].(float64)
	}
	require.Equal(t, float64(2), vers[w.students[1].ID.String()])
	for _, i := range []int{0, 2, 3} {
		require.Equal(t, float64(1), vers[w.students[i].ID.String()], "hàng giống hệt giữ version")
	}
	require.Equal(t, 2, r.count(`select count(*) from audit_log where entity = 'attendance' and entity_id = $1`, w.ses[0].String()))
}

func TestPutGridIdempotent(t *testing.T) {
	r := newRig(t)
	w := r.world(5)
	recs := w.all("PRESENT")
	recs[0]["status"] = "ABSENT"
	require.Equal(t, http.StatusOK, w.put(w.T, 0, recs, false).code)
	audits := r.count(`select count(*) from audit_log where entity = 'attendance'`)
	events := r.count(`select count(*) from outbox where topic = 'crm.attendance_saved'`)
	again := w.put(w.T, 0, recs, false)
	require.Equal(t, http.StatusOK, again.code, string(again.body))
	require.Equal(t, audits, r.count(`select count(*) from audit_log where entity = 'attendance'`), "gửi lại cùng nội dung: không audit mới")
	require.Equal(t, events, r.count(`select count(*) from outbox where topic = 'crm.attendance_saved'`), "không outbox mới")
	require.Equal(t, "1", r.scalar(`select max(version)::text from attendance_records where session_id = $1`, w.ses[0]))
}

func TestPutGridAuditNoNames(t *testing.T) {
	r := newRig(t)
	w := r.world(3)
	note := "xin phép đi khám bệnh viện"
	recs := w.all("ABSENT")
	recs[0]["note"] = note
	require.Equal(t, http.StatusOK, w.put(w.T, 0, recs, true).code)
	rows, err := r.pool.Query(t.Context(), `select coalesce(before::text, '') || coalesce(after::text, '') from audit_log where entity = 'attendance'`)
	require.NoError(t, err)
	defer rows.Close()
	n := 0
	for rows.Next() {
		var s string
		require.NoError(t, rows.Scan(&s))
		n++
		require.NotContains(t, s, "Sv 0", "audit không có tên sinh viên")
		require.NotContains(t, s, note, "audit không có ghi chú")
		require.NotContains(t, s, w.students[0].Email)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, 1, n)
}

func TestPutGrid30Under500ms(t *testing.T) {
	r := newRig(t)
	w := r.world(30)
	var d []time.Duration
	for i := range 2 {
		for round := range 5 { // 10 lần ghi lưới 30 hàng: mỗi buổi luân phiên đổi trạng thái
			st := []string{"ABSENT", "LATE", "PRESENT", "EXCUSED", "ABSENT"}[round]
			recs := w.all(st)
			if round > 0 || i > 0 {
				cur := w.grid(w.T, i)
				require.Equal(t, http.StatusOK, cur.code)
				for k, s := range items(cur, "students") {
					if s["record"] != nil {
						recs[k]["version"] = s["record"].(map[string]any)["version"]
					}
				}
			}
			t0 := time.Now()
			res := w.put(w.T, i, recs, false)
			d = append(d, time.Since(t0))
			require.Equal(t, http.StatusOK, res.code, string(res.body))
		}
	}
	sort.Slice(d, func(a, b int) bool { return d[a] < d[b] })
	require.LessOrEqual(t, d[len(d)-1], 500*time.Millisecond, "30 hàng ≤ 500 ms: %v", d)
}

// ===== AC3: xung đột =====

func TestPutGridConflictAllOrNothing(t *testing.T) {
	r := newRig(t)
	w := r.world(4)
	require.Equal(t, http.StatusOK, w.put(w.T, 0, w.all("PRESENT"), false).code)
	// T1 (giảng viên) đổi sinh viên 0 → ABSENT (version 1 → 2)
	require.Equal(t, http.StatusOK, w.put(w.T, 0, []rec{row(w.students[0].ID, "ABSENT", "version", 1)}, false).code)
	// T2 (TA) còn giữ version 1: đổi sinh viên 0 → LATE (xung đột) và sinh viên 1 → ABSENT (hợp lệ)
	res := w.put(w.A, 0, []rec{row(w.students[0].ID, "LATE", "version", 1), row(w.students[1].ID, "ABSENT", "version", 1)}, false)
	require.Equal(t, http.StatusConflict, res.code, string(res.body))
	require.Equal(t, "VERSION_CONFLICT", res.errCode())
	cs := res.details()["conflicts"].([]any)
	require.Len(t, cs, 1)
	c := cs[0].(map[string]any)
	require.Equal(t, w.students[0].ID.String(), c["student_id"])
	require.Equal(t, "LATE", c["mine"].(map[string]any)["status"])
	cur := c["current"].(map[string]any)
	require.Equal(t, []any{"ABSENT", float64(2)}, []any{cur["status"], cur["version"]})
	require.Equal(t, w.teacher.FullName, cur["marked_by"].(map[string]any)["full_name"])
	require.NotEmpty(t, cur["marked_at"])
	require.Equal(t, "PRESENT", r.scalar(`select status::text from attendance_records where session_id = $1 and student_id = $2`, w.ses[0], w.students[1].ID), "tất cả hoặc không: hàng hợp lệ cũng không được ghi")
	// chọn "lấy của thầy": gửi lại với version mới thì qua
	require.Equal(t, http.StatusOK, w.put(w.A, 0, []rec{row(w.students[0].ID, "LATE", "version", 2), row(w.students[1].ID, "ABSENT", "version", 1)}, false).code)
}

func TestPutGridUnchangedRowNoConflict(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	require.Equal(t, http.StatusOK, w.put(w.T, 0, w.all("PRESENT"), false).code)
	require.Equal(t, http.StatusOK, w.put(w.T, 0, []rec{row(w.students[0].ID, "ABSENT", "version", 1)}, false).code)
	// TA gửi lại hàng cũ với trạng thái ĐÚNG như DB (không đổi) và version cũ ⇒ không xung đột
	res := w.put(w.A, 0, []rec{row(w.students[0].ID, "ABSENT", "version", 1)}, false)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	// hai người cùng mở lưới trống: người thứ hai thiếu version cho hàng đã có (khác nội dung) ⇒ 409 đúng, không ghi đè lặng lẽ
	res = w.put(w.A, 0, []rec{row(w.students[1].ID, "LATE")}, false)
	require.Equal(t, http.StatusConflict, res.code)
}

// ===== AC4: dữ liệu sai =====

func TestPutGridValidation(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	long := strings.Repeat("a", 201)
	many := make([]rec, 1001)
	for i := range many {
		many[i] = row(uuid.New(), "PRESENT")
	}
	cases := map[string]struct {
		recs []rec
		code string
	}{
		"sinh viên PENDING": {[]rec{row(w.pending.ID, "PRESENT")}, "STUDENT_NOT_IN_COURSE"},
		"sinh viên lạ":      {[]rec{row(uuid.New(), "PRESENT")}, "STUDENT_NOT_IN_COURSE"},
		"trùng student_id":  {[]rec{row(w.students[0].ID, "PRESENT"), row(w.students[0].ID, "ABSENT")}, "VALIDATION_FAILED"},
		"status lạ":         {[]rec{row(w.students[0].ID, "NGHI")}, "VALIDATION_FAILED"},
		"note > 200":        {[]rec{row(w.students[0].ID, "ABSENT", "note", long)}, "VALIDATION_FAILED"},
		"> 1.000 hàng":      {many, "VALIDATION_FAILED"},
	}
	for name, tc := range cases {
		res := w.put(w.T, 0, tc.recs, false)
		require.Equal(t, http.StatusUnprocessableEntity, res.code, name+": "+string(res.body))
		require.Equal(t, tc.code, res.errCode(), name)
	}
	bad := w.put(w.T, 0, []rec{row(w.pending.ID, "PRESENT"), row(uuid.New(), "PRESENT")}, false)
	require.Len(t, bad.details()["student_ids"], 2, "kèm danh sách")
	require.Zero(t, r.count(`select count(*) from attendance_records where course_id = $1`, w.course))
}

func TestPutGridArchived(t *testing.T) {
	r := newRig(t)
	w := r.world(2)
	r.exec(`update courses set status = 'ARCHIVED', archived_at = now(), join_enabled = false where id = $1`, w.course)
	res := w.put(w.T, 0, w.all("PRESENT"), false)
	require.Equal(t, http.StatusConflict, res.code, string(res.body))
	require.Equal(t, "COURSE_ARCHIVED", res.errCode())
	require.Equal(t, http.StatusOK, w.grid(w.T, 0).code, "vẫn đọc được")
}

// TestPutGridDBDownAtomic — AC4: DB không dùng được ⇒ 503; lỗi giữa chừng ⇒ không ghi nửa chừng.
func TestPutGridDBDownAtomic(t *testing.T) {
	r := newRig(t)
	w := r.world(3)
	r.exec(`create function zz_boom() returns trigger language plpgsql as $$ begin if new.note = 'boom' then raise exception 'giả lập'; end if; return new; end $$`)
	r.exec(`create trigger zz_boom before insert on attendance_records for each row execute function zz_boom()`)
	recs := w.all("ABSENT")
	recs[2]["note"] = "boom"
	res := w.put(w.T, 0, recs, false)
	require.Equal(t, http.StatusInternalServerError, res.code, string(res.body))
	require.Zero(t, r.count(`select count(*) from attendance_records where course_id = $1`, w.course), "không hàng nào được ghi")
	require.Zero(t, r.count(`select count(*) from audit_log where entity = 'attendance'`), "không audit nào")
}

// ===== AC5: hoàn tất và sửa sau =====

func TestCompleteSetsDoneAt(t *testing.T) {
	r := newRig(t)
	w := r.world(5)
	res := w.put(w.T, 0, []rec{row(w.students[0].ID, "ABSENT")}, true)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.NotNil(t, res.json()["done_at"])
	require.Equal(t, 5, r.count(`select count(*) from attendance_records where session_id = $1`, w.ses[0]), "sinh viên ACTIVE còn thiếu được điền Có mặt")
	require.Equal(t, 4, r.count(`select count(*) from attendance_records where session_id = $1 and status = 'PRESENT' and version = 1`, w.ses[0]))
	require.Equal(t, w.teacher.ID.String(), r.scalar(`select attendance_done_by::text from class_sessions where id = $1`, w.ses[0]))
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity = 'attendance' and action = 'complete'`))
	st := items(w.grid(w.T, 0), "students")
	for _, s := range st {
		require.NotNil(t, s["record"], "mọi sinh viên ACTIVE đều có dòng sau khi hoàn tất")
	}
	require.NotNil(t, w.grid(w.T, 0).json()["session"].(map[string]any)["attendance_done_at"])
	// sinh viên vào lớp sau không có dòng cho buổi cũ (không bịa)
	late := r.addUserNamed(w, "Vào Muộn")
	_ = late
	for _, s := range items(w.grid(w.T, 0), "students") {
		if s["full_name"] == "Vào Muộn" {
			require.Nil(t, s["record"])
		}
	}
}

func TestAutosaveDoesNotComplete(t *testing.T) {
	r := newRig(t)
	w := r.world(3)
	require.Equal(t, http.StatusOK, w.put(w.T, 0, []rec{row(w.students[0].ID, "ABSENT")}, false).code)
	require.Equal(t, "", r.scalar(`select attendance_done_at::text from class_sessions where id = $1`, w.ses[0]))
	require.Equal(t, 1, r.count(`select count(*) from attendance_records where session_id = $1`, w.ses[0]), "tự lưu chỉ ghi hàng đã chạm")
}

func TestEditAfterCompleteAudited(t *testing.T) {
	r := newRig(t)
	w := r.world(3)
	require.Equal(t, http.StatusOK, w.put(w.T, 0, nil, true).code)
	res := w.put(w.A, 0, []rec{row(w.students[1].ID, "ABSENT", "version", 1)}, false)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity = 'attendance' and action = 'edit_after_complete' and actor_id = $1`, w.ta.ID))
	require.Equal(t, "PRESENT", r.scalar(`select (after->'changes'->0->>'from') from audit_log where action = 'edit_after_complete'`))
	require.Equal(t, "ABSENT", r.scalar(`select (after->'changes'->0->>'to') from audit_log where action = 'edit_after_complete'`))
	// gửi lại nội dung vừa hoàn tất lần đầu (mạng chập chờn) không tạo "complete" thứ hai
	require.Equal(t, http.StatusOK, w.put(w.T, 0, nil, true).code)
	require.Equal(t, 1, r.count(`select count(*) from audit_log where action = 'complete'`))
	require.Equal(t, 1, r.count(`select count(*) from audit_log where action = 'edit_after_complete'`))
}

// TestConflictDetailsHaveCurrent — AC3: mỗi xung đột mang `mine` và `current` đủ để dựng "{tên} đã đổi: …" và gửi lại đúng version.
func TestConflictDetailsHaveCurrent(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	require.Equal(t, http.StatusOK, w.put(w.T, 0, w.all("PRESENT"), false).code)
	require.Equal(t, http.StatusOK, w.put(w.T, 0, []rec{row(w.students[0].ID, "EXCUSED", "version", 1)}, false).code)
	res := w.put(w.A, 0, []rec{row(w.students[0].ID, "ABSENT", "version", 1)}, false)
	require.Equal(t, http.StatusConflict, res.code)
	c := res.details()["conflicts"].([]any)[0].(map[string]any)
	cur := c["current"].(map[string]any)
	require.Equal(t, []any{"EXCUSED", float64(2), w.teacher.FullName}, []any{cur["status"], cur["version"], cur["marked_by"].(map[string]any)["full_name"]})
	require.Equal(t, "ABSENT", c["mine"].(map[string]any)["status"])
}

// TestCRMRoutesRBAC — AC15: năm route điểm danh / tham gia × sáu danh tính.
func TestCRMRoutesRBAC(t *testing.T) {
	r := newRig(t)
	w := r.world(1)
	sv := r.mustLogin(w.students[0].Email)
	pid := w.speak(w.T, w.students[0].ID, nil).json()["id"].(string)
	base := "/courses/" + w.course.String()
	for who, tc := range map[string]struct {
		s    session
		want int
	}{"giảng viên": {w.T, 200}, "trợ giảng": {w.A, 200}, "sinh viên": {sv, 403}, "ADMIN": {w.ADM, 403}, "ngoài lớp": {w.OUT, 403}, "không JWT": {session{}, 401}} {
		require.Equal(t, tc.want, w.grid(tc.s, 0).code, "GET attendance "+who)
		require.Equal(t, tc.want, w.put(tc.s, 0, w.all("PRESENT"), false).code, "PUT attendance "+who)
		require.Equal(t, tc.want, r.call(http.MethodGet, base+"/participation", tc.s, nil).code, "GET participation "+who)
		if tc.want != 200 {
			require.Equal(t, tc.want, w.speak(tc.s, w.students[0].ID, nil).code, "POST participation "+who)
			require.Equal(t, tc.want, r.call(http.MethodDelete, base+"/participation/"+pid, tc.s, nil).code, "DELETE participation "+who)
		}
	}
}

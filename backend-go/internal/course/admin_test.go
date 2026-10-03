package course_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/course"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
)

func (r *rig) scalar(sql string, args ...any) string {
	r.t.Helper()
	var s *string
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), sql, args...).Scan(&s))
	if s == nil {
		return ""
	}
	return *s
}

func (r *rig) count(sql string, args ...any) int {
	r.t.Helper()
	var n int
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), sql, args...).Scan(&n))
	return n
}

func idemKey() map[string]string {
	return map[string]string{"Idempotency-Key": "k-" + uuid.NewString()}
}

type actors struct {
	adm, gv, gv2, ta, ta2, sv store.User
	A, G, G2, T, T2, S        session
}

func (r *rig) actors() actors {
	r.t.Helper()
	a := actors{
		adm: r.addUser(uniq("adm"), store.UserRoleADMIN, store.UserStatusACTIVE),
		gv:  r.addUser(uniq("gv"), store.UserRoleTEACHER, store.UserStatusACTIVE),
		gv2: r.addUser(uniq("gv2"), store.UserRoleTEACHER, store.UserStatusACTIVE),
		ta:  r.addUser(uniq("ta"), store.UserRoleTA, store.UserStatusACTIVE),
		ta2: r.addUser(uniq("ta2"), store.UserRoleTA, store.UserStatusACTIVE),
		sv:  r.addUser(uniq("sv"), store.UserRoleSTUDENT, store.UserStatusACTIVE),
	}
	a.A, a.G, a.G2 = r.mustLogin(a.adm.Email), r.mustLogin(a.gv.Email), r.mustLogin(a.gv2.Email)
	a.T, a.T2, a.S = r.mustLogin(a.ta.Email), r.mustLogin(a.ta2.Email), r.mustLogin(a.sv.Email)
	return a
}

func classCode() string { return "C" + strings.ToUpper(uuid.NewString()[:6]) }

func (r *rig) post(s session, path string, body any, hdr map[string]string) resp {
	return r.do(req{method: http.MethodPost, path: path, bearer: s.access, body: body, hdr: hdr})
}

func (r *rig) put(s session, path string, body any) resp {
	return r.do(req{method: http.MethodPut, path: path, bearer: s.access, body: body})
}

func (r *rig) get(s session, path string) resp {
	return r.do(req{method: http.MethodGet, path: path, bearer: s.access})
}

func (r *rig) open(s session, extra map[string]any) resp {
	body := map[string]any{"subject_code": "INT1006", "class_code": classCode(), "name": "An ninh mạng", "semester": "2026-2027-HK1"}
	for k, v := range extra {
		body[k] = v
	}
	return r.post(s, "/admin/courses", body, idemKey())
}

func (r *rig) mustOpen(s session, extra map[string]any) string {
	r.t.Helper()
	res := r.open(s, extra)
	require.Equal(r.t, http.StatusCreated, res.code, string(res.body))
	return res.json()["id"].(string)
}

func (r *rig) assign(s session, id string, body map[string]any) resp {
	return r.post(s, "/admin/courses/"+id+"/assign", body, nil)
}

func (r *rig) staffOf(id string) (teacher string, tas []string) {
	r.t.Helper()
	rows, err := r.pool.Query(r.t.Context(), `select role_in_course::text, user_id::text from enrollments where course_id = $1 and status = 'ACTIVE' and role_in_course in ('TEACHER','TA') order by user_id`, id)
	require.NoError(r.t, err)
	defer rows.Close()
	for rows.Next() {
		var role, uid string
		require.NoError(r.t, rows.Scan(&role, &uid))
		if role == "TEACHER" {
			teacher = uid
		} else {
			tas = append(tas, uid)
		}
	}
	require.NoError(r.t, rows.Err())
	return
}

// ===== AC1 =====

func TestCreateCourse(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	res := r.open(a.A, map[string]any{"class_code": "999001", "capacity": 40})
	require.Equal(t, http.StatusCreated, res.code, string(res.body))
	m := res.json()
	require.Len(t, m, 8)
	require.Equal(t, "ACTIVE", m["status"])
	require.EqualValues(t, 1, m["version"])
	require.Nil(t, m["teacher"])
	require.NotContains(t, string(res.body), "join_code", "mã tham gia không bao giờ nằm trong phản hồi của admin/courses")
	var code string
	var enabled, approval bool
	require.NoError(t, r.pool.QueryRow(t.Context(), `select join_code, join_enabled, join_require_approval from courses where id = $1`, m["id"]).Scan(&code, &enabled, &approval))
	require.Regexp(t, `^[ABCDEFGHJKMNPQRSTUVWXYZ23456789]{7}$`, code)
	require.True(t, enabled)
	require.False(t, approval)

	// kèm giảng viên và trợ giảng: gán trong cùng giao dịch
	res = r.open(a.A, map[string]any{"teacher_id": a.gv.ID, "ta_ids": []string{a.ta.ID.String()}})
	require.Equal(t, http.StatusCreated, res.code, string(res.body))
	require.Equal(t, a.gv.ID.String(), res.json()["teacher"].(map[string]any)["id"])
	teacher, tas := r.staffOf(res.json()["id"].(string))
	require.Equal(t, a.gv.ID.String(), teacher)
	require.Equal(t, []string{a.ta.ID.String()}, tas)
	require.Equal(t, 2, r.count(`select count(*) from outbox where topic = 'course.assigned' and payload->>'course_id' = $1`, res.json()["id"]))
}

func TestCreateCourseIdempotent(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	key := idemKey()
	body := map[string]any{"subject_code": "INT1006", "class_code": "999002", "name": "An ninh mạng", "semester": "2026-2027-HK1"}
	first := r.post(a.A, "/admin/courses", body, key)
	second := r.post(a.A, "/admin/courses", body, key)
	require.Equal(t, http.StatusCreated, first.code, string(first.body))
	require.Equal(t, http.StatusCreated, second.code)
	require.JSONEq(t, string(first.body), string(second.body))
	require.Equal(t, "true", second.hdr.Get("Idempotent-Replayed"))
	require.Equal(t, 1, r.count(`select count(*) from courses where class_code = '999002'`))
	require.Equal(t, http.StatusUnprocessableEntity, r.post(a.A, "/admin/courses", body, nil).code, "thiếu Idempotency-Key")
}

func TestCreateCourseDuplicateClassCode(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	r.mustOpen(a.A, map[string]any{"class_code": "DUP-001"})
	res := r.open(a.A, map[string]any{"class_code": "DUP-001"})
	require.Equal(t, http.StatusConflict, res.code)
	require.Equal(t, "CONFLICT", res.errCode())
	require.Equal(t, "class_code", res.details()["field"])
	require.Equal(t, 1, r.count(`select count(*) from courses where class_code = 'DUP-001'`))
}

func TestCreateCourseValidation(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	long := strings.Repeat("a", 121)
	for name, tc := range map[string]struct {
		extra map[string]any
		field string
	}{
		"class_code ngắn":       {map[string]any{"class_code": "ab"}, "class_code"},
		"class_code ký tự lạ":   {map[string]any{"class_code": "ab cd"}, "class_code"},
		"class_code dài":        {map[string]any{"class_code": strings.Repeat("a", 21)}, "class_code"},
		"semester sai dạng":     {map[string]any{"semester": "2026-HK1"}, "semester"},
		"semester HK4":          {map[string]any{"semester": "2026-2027-HK4"}, "semester"},
		"name rỗng":             {map[string]any{"name": "  "}, "name"},
		"name 121":              {map[string]any{"name": long}, "name"},
		"capacity 0":            {map[string]any{"capacity": 0}, "capacity"},
		"capacity 1001":         {map[string]any{"capacity": 1001}, "capacity"},
		"subject_code lạ":       {map[string]any{"subject_code": "a b"}, "subject_code"},
		"join_code sai bảng":    {map[string]any{"join_code": "ABCDEF0"}, "join_code"},
		"teacher_id không uuid": {map[string]any{"teacher_id": "x"}, "teacher_id"},
	} {
		res := r.open(a.A, tc.extra)
		require.Equal(t, http.StatusUnprocessableEntity, res.code, name+" "+string(res.body))
		require.Contains(t, string(res.body), `"`+tc.field+`"`, name)
	}
	require.Equal(t, http.StatusUnprocessableEntity, r.open(a.A, map[string]any{"role": "x"}).code, "khoá lạ")
	require.Equal(t, 0, r.count(`select count(*) from courses where created_by = $1`, a.adm.ID), "không lớp nào được tạo")
	// subject_code được chuẩn hoá chữ hoa
	id := r.mustOpen(a.A, map[string]any{"subject_code": "int1006"})
	require.Equal(t, "INT1006", r.scalar(`select subject_code from courses where id = $1`, id))
}

// ===== AC3 =====

func TestUpdateCourse(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	id := r.mustOpen(a.A, nil)
	res := r.put(a.A, "/admin/courses/"+id, map[string]any{"name": "Tên mới", "semester": "2026-2027-HK2", "capacity": 50, "version": 1})
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Equal(t, "Tên mới", res.json()["name"])
	require.EqualValues(t, 50, res.json()["capacity"])
	require.EqualValues(t, 2, res.json()["version"])
	require.Equal(t, `W/"v2"`, res.hdr.Get("ETag"))
	// capacity null = bỏ giới hạn; khoá vắng = giữ nguyên
	res = r.put(a.A, "/admin/courses/"+id, map[string]any{"capacity": nil, "version": 2})
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Nil(t, res.json()["capacity"])
	res = r.put(a.A, "/admin/courses/"+id, map[string]any{"name": "Chỉ đổi tên", "version": 3})
	require.Equal(t, "2026-2027-HK2", res.json()["semester"])
	// trùng class_code của lớp khác
	other := r.mustOpen(a.A, map[string]any{"class_code": "OTHER-1"})
	_ = other
	dup := r.put(a.A, "/admin/courses/"+id, map[string]any{"class_code": "OTHER-1", "version": 4})
	require.Equal(t, http.StatusConflict, dup.code)
	require.Equal(t, "class_code", dup.details()["field"])
	require.Equal(t, http.StatusUnprocessableEntity, r.put(a.A, "/admin/courses/"+id, map[string]any{"class_code": "x", "version": 4}).code)
	require.Equal(t, http.StatusNotFound, r.put(a.A, "/admin/courses/"+uuid.NewString(), map[string]any{"name": "x", "version": 1}).code)
	require.Equal(t, http.StatusNotFound, r.put(a.A, "/admin/courses/khong-phai-uuid", map[string]any{"name": "x", "version": 1}).code)
	require.Equal(t, 3, r.count(`select count(*) from audit_log where entity = 'course' and entity_id = $1 and action = 'course_updated'`, id))
}

func TestUpdateCapacityBelowActive(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	id := r.mustOpen(a.A, nil)
	cid := uuid.MustParse(id)
	for i := 0; i < 3; i++ {
		u := r.addUser(uniq("s"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
		r.enroll(cid, u.ID, "STUDENT", "ACTIVE")
	}
	res := r.put(a.A, "/admin/courses/"+id, map[string]any{"capacity": 2, "version": 1})
	require.Equal(t, http.StatusUnprocessableEntity, res.code, string(res.body))
	require.Contains(t, string(res.body), "CAPACITY_BELOW_ACTIVE")
	require.Equal(t, http.StatusOK, r.put(a.A, "/admin/courses/"+id, map[string]any{"capacity": 3, "version": 1}).code)
}

func TestUpdateVersionConflict(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	id := r.mustOpen(a.A, nil)
	require.Equal(t, http.StatusOK, r.put(a.A, "/admin/courses/"+id, map[string]any{"name": "Lần một", "version": 1}).code)
	res := r.put(a.A, "/admin/courses/"+id, map[string]any{"name": "Lần hai", "version": 1})
	require.Equal(t, http.StatusConflict, res.code)
	require.Equal(t, "VERSION_CONFLICT", res.errCode())
	require.EqualValues(t, 2, res.details()["current_version"])
	require.Equal(t, "Lần một", res.details()["current"].(map[string]any)["name"])
	require.Equal(t, http.StatusUnprocessableEntity, r.put(a.A, "/admin/courses/"+id, map[string]any{"name": "x"}).code, "thiếu version")
}

func TestArchiveCourse(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	id := r.mustOpen(a.A, map[string]any{"teacher_id": a.gv.ID})
	cid := uuid.MustParse(id)
	r.enroll(cid, a.sv.ID, "STUDENT", "ACTIVE")
	res := r.post(a.A, "/admin/courses/"+id+"/archive", nil, nil)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Equal(t, "ARCHIVED", res.json()["status"])
	require.EqualValues(t, 2, res.json()["version"])
	var enabled bool
	var archivedAt *time.Time
	require.NoError(t, r.pool.QueryRow(t.Context(), `select join_enabled, archived_at from courses where id = $1`, id).Scan(&enabled, &archivedAt))
	require.False(t, enabled, "mã tham gia ngừng ngay")
	require.NotNil(t, archivedAt)
	require.Equal(t, 2, r.count(`select count(*) from enrollments where course_id = $1 and status = 'ACTIVE'`, id), "thành viên giữ nguyên")
	// vẫn đọc được lớp (chỉ đọc)
	require.Equal(t, http.StatusOK, r.get(a.S, "/courses/"+id).code)
	require.Equal(t, http.StatusOK, r.get(a.G, "/courses/"+id).code)
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity = 'course' and entity_id = $1 and action = 'course_archived'`, id))
}

func TestArchiveIdempotent(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	id := r.mustOpen(a.A, nil)
	first := r.post(a.A, "/admin/courses/"+id+"/archive", nil, nil)
	second := r.post(a.A, "/admin/courses/"+id+"/archive", nil, nil)
	require.Equal(t, http.StatusOK, second.code, string(second.body))
	require.JSONEq(t, string(first.body), string(second.body), "lần hai không đổi gì")
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity = 'course' and entity_id = $1 and action = 'course_archived'`, id))
	require.Equal(t, http.StatusNotFound, r.post(a.A, "/admin/courses/"+uuid.NewString()+"/archive", nil, nil).code)
}

func TestArchivedRejectsWrites(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	id := r.mustOpen(a.A, map[string]any{"teacher_id": a.gv.ID})
	require.Equal(t, http.StatusOK, r.post(a.A, "/admin/courses/"+id+"/archive", nil, nil).code)
	for name, res := range map[string]resp{
		"sửa":                    r.put(a.A, "/admin/courses/"+id, map[string]any{"name": "x", "version": 2}),
		"gán":                    r.assign(a.A, id, map[string]any{"teacher_id": a.gv2.ID}),
		"trợ giảng (Admin)":      r.put(a.A, "/courses/"+id+"/assistants", map[string]any{"ta_ids": []string{a.ta.ID.String()}}),
		"trợ giảng (giảng viên)": r.put(a.G, "/courses/"+id+"/assistants", map[string]any{"ta_ids": []string{a.ta.ID.String()}}),
	} {
		require.Equal(t, http.StatusConflict, res.code, name+" "+string(res.body))
		require.Equal(t, "COURSE_ARCHIVED", res.errCode(), name)
		require.Equal(t, "Lớp này đã được lưu trữ.", res.json()["message"], name)
	}
	teacher, tas := r.staffOf(id)
	require.Equal(t, a.gv.ID.String(), teacher)
	require.Empty(t, tas)
}

// ===== AC4 =====

func TestAssignTeacher(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	id := r.mustOpen(a.A, nil)
	res := r.assign(a.A, id, map[string]any{"teacher_id": a.gv.ID, "ta_ids": []string{a.ta.ID.String()}})
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	m := res.json()
	require.Equal(t, a.gv.ID.String(), m["teacher"].(map[string]any)["id"])
	require.Len(t, m["assistants"], 1)
	ch := m["changed"].(map[string]any)
	require.Equal(t, true, ch["teacher"])
	require.Equal(t, []any{a.ta.ID.String()}, ch["added_ta"])
	require.Equal(t, []any{}, ch["removed_ta"])
	require.Equal(t, "ADMIN", r.scalar(`select joined_via::text from enrollments where course_id = $1 and user_id = $2`, id, a.gv.ID))
	// người INVITED vẫn gán được
	inv := r.addUser(uniq("inv"), store.UserRoleTEACHER, store.UserStatusINVITED)
	id2 := r.mustOpen(a.A, nil)
	require.Equal(t, http.StatusOK, r.assign(a.A, id2, map[string]any{"teacher_id": inv.ID}).code)
}

func TestAssignRejectsWrongRole(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	id := r.mustOpen(a.A, nil)
	disabled := r.addUser(uniq("dis"), store.UserRoleTEACHER, store.UserStatusDISABLED)
	for name, body := range map[string]map[string]any{
		"teacher_id là TA":      {"teacher_id": a.ta.ID},
		"teacher_id là SV":      {"teacher_id": a.sv.ID},
		"teacher_id là ADMIN":   {"teacher_id": a.adm.ID},
		"teacher_id bị khoá":    {"teacher_id": disabled.ID},
		"teacher_id không có":   {"teacher_id": uuid.NewString()},
		"ta_ids có giảng viên":  {"ta_ids": []string{a.gv.ID.String()}},
		"ta_ids có SV":          {"ta_ids": []string{a.ta.ID.String(), a.sv.ID.String()}},
		"ta_ids không có người": {"ta_ids": []string{uuid.NewString()}},
	} {
		res := r.assign(a.A, id, body)
		require.Equal(t, http.StatusUnprocessableEntity, res.code, name+" "+string(res.body))
	}
	res := r.assign(a.A, id, map[string]any{"teacher_id": a.ta.ID})
	require.Contains(t, string(res.body), "Người này không phải giảng viên.")
	teacher, tas := r.staffOf(id)
	require.Empty(t, teacher)
	require.Empty(t, tas, "lỗi ở người thứ hai thì người thứ nhất cũng không được gán")
}

func TestAssignReplacesTAs(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	id := r.mustOpen(a.A, map[string]any{"ta_ids": []string{a.ta.ID.String()}})
	_, tas := r.staffOf(id)
	require.Equal(t, []string{a.ta.ID.String()}, tas)
	// vắng khoá và null = giữ nguyên
	require.Equal(t, http.StatusOK, r.assign(a.A, id, map[string]any{"teacher_id": a.gv.ID}).code)
	_, tas = r.staffOf(id)
	require.Equal(t, []string{a.ta.ID.String()}, tas)
	require.Equal(t, http.StatusOK, r.assign(a.A, id, map[string]any{"ta_ids": nil}).code)
	_, tas = r.staffOf(id)
	require.Equal(t, []string{a.ta.ID.String()}, tas, "null = giữ nguyên")
	// thay toàn bộ
	res := r.assign(a.A, id, map[string]any{"ta_ids": []string{a.ta2.ID.String()}})
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Equal(t, []any{a.ta2.ID.String()}, res.json()["changed"].(map[string]any)["added_ta"])
	require.Equal(t, []any{a.ta.ID.String()}, res.json()["changed"].(map[string]any)["removed_ta"])
	_, tas = r.staffOf(id)
	require.Equal(t, []string{a.ta2.ID.String()}, tas)
	// [] = gỡ hết
	require.Equal(t, http.StatusOK, r.assign(a.A, id, map[string]any{"ta_ids": []string{}}).code)
	_, tas = r.staffOf(id)
	require.Empty(t, tas)
	// gán lại người từng bị gỡ dùng lại dòng cũ
	require.Equal(t, http.StatusOK, r.assign(a.A, id, map[string]any{"ta_ids": []string{a.ta.ID.String()}}).code)
	require.Equal(t, 1, r.count(`select count(*) from enrollments where course_id = $1 and user_id = $2`, id, a.ta.ID))
}

func TestAssignOldTeacherLosesAccessNow(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	id := r.mustOpen(a.A, map[string]any{"teacher_id": a.gv.ID})
	require.Equal(t, http.StatusOK, r.get(a.G, "/courses/"+id).code)
	require.Equal(t, http.StatusForbidden, r.get(a.G2, "/courses/"+id).code)
	require.Equal(t, http.StatusOK, r.assign(a.A, id, map[string]any{"teacher_id": a.gv2.ID}).code)
	old := r.get(a.G, "/courses/"+id)
	require.Equal(t, http.StatusForbidden, old.code, "giảng viên cũ mất quyền ở yêu cầu kế tiếp")
	require.Equal(t, "course", old.details()["reason"])
	require.Equal(t, http.StatusOK, r.get(a.G2, "/courses/"+id).code)
	require.Equal(t, "REMOVED", r.scalar(`select status::text from enrollments where course_id = $1 and user_id = $2`, id, a.gv.ID))
}

func TestAssignOneActiveTeacher(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	id := r.mustOpen(a.A, nil)
	for _, gv := range []store.User{a.gv, a.gv2, a.gv, a.gv2, a.gv} {
		require.Equal(t, http.StatusOK, r.assign(a.A, id, map[string]any{"teacher_id": gv.ID}).code)
		require.Equal(t, 1, r.count(`select count(*) from enrollments where course_id = $1 and role_in_course = 'TEACHER' and status = 'ACTIVE'`, id))
	}
	require.Equal(t, 2, r.count(`select count(*) from enrollments where course_id = $1 and role_in_course = 'TEACHER'`, id), "dòng cũ được dùng lại, không thêm dòng mới")
}

func TestAssignAtomicWithOutbox(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	id := r.mustOpen(a.A, nil)
	before := r.count(`select count(*) from outbox where topic = 'course.assigned'`)
	// hỏng ở người thứ ba ⇒ không dòng ghi danh, không dòng outbox
	bad := r.assign(a.A, id, map[string]any{"teacher_id": a.gv.ID, "ta_ids": []string{a.ta.ID.String(), a.sv.ID.String()}})
	require.Equal(t, http.StatusUnprocessableEntity, bad.code)
	require.Equal(t, before, r.count(`select count(*) from outbox where topic = 'course.assigned'`))
	teacher, tas := r.staffOf(id)
	require.Empty(t, teacher)
	require.Empty(t, tas)
	// thành công: mỗi người MỚI một dòng outbox, payload đủ trường
	require.Equal(t, http.StatusOK, r.assign(a.A, id, map[string]any{"teacher_id": a.gv.ID, "ta_ids": []string{a.ta.ID.String()}}).code)
	require.Equal(t, before+2, r.count(`select count(*) from outbox where topic = 'course.assigned'`))
	var payload string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select payload::text from outbox where topic = 'course.assigned' and payload->>'user_id' = $1`, a.gv.ID.String()).Scan(&payload))
	var p map[string]string
	require.NoError(t, json.Unmarshal([]byte(payload), &p))
	require.Equal(t, map[string]string{"course_id": id, "user_id": a.gv.ID.String(), "role": "TEACHER", "assigned_by": a.adm.ID.String()}, p)
}

func TestAssignNoRenotify(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	id := r.mustOpen(a.A, nil)
	body := map[string]any{"teacher_id": a.gv.ID, "ta_ids": []string{a.ta.ID.String()}}
	require.Equal(t, http.StatusOK, r.assign(a.A, id, body).code)
	n := r.count(`select count(*) from outbox where topic = 'course.assigned' and payload->>'course_id' = $1`, id)
	require.Equal(t, 2, n)
	res := r.assign(a.A, id, body)
	require.Equal(t, http.StatusOK, res.code)
	require.Equal(t, false, res.json()["changed"].(map[string]any)["teacher"])
	require.Equal(t, n, r.count(`select count(*) from outbox where topic = 'course.assigned' and payload->>'course_id' = $1`, id), "không gán lại người đã có ⇒ không thông báo lại")
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity = 'course' and entity_id = $1 and action = 'course_assigned'`, id), "không đổi gì ⇒ không ghi audit")
}

func TestAssignAudit(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	id := r.mustOpen(a.A, map[string]any{"teacher_id": a.gv.ID})
	require.Equal(t, http.StatusOK, r.assign(a.A, id, map[string]any{"teacher_id": a.gv2.ID}).code)
	var before, after, actor string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select before::text, after::text, actor_id::text from audit_log where entity = 'course' and entity_id = $1 and action = 'course_assigned' order by created_at desc, id desc limit 1`, id).Scan(&before, &after, &actor))
	require.Contains(t, before, a.gv.ID.String())
	require.Contains(t, after, a.gv2.ID.String())
	require.NotContains(t, after, a.gv.ID.String())
	require.Equal(t, a.adm.ID.String(), actor)
}

// ===== AC6 =====

func TestTeacherManagesOwnTAs(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	id := r.mustOpen(a.A, map[string]any{"teacher_id": a.gv.ID})
	res := r.put(a.G, "/courses/"+id+"/assistants", map[string]any{"ta_ids": []string{a.ta.ID.String(), a.ta2.ID.String()}})
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Len(t, res.json()["assistants"], 2)
	_, tas := r.staffOf(id)
	require.Len(t, tas, 2)
	res = r.put(a.G, "/courses/"+id+"/assistants", map[string]any{"ta_ids": []string{a.ta2.ID.String()}})
	require.Equal(t, []any{a.ta.ID.String()}, res.json()["changed"].(map[string]any)["removed_ta"])
	require.Equal(t, http.StatusOK, r.put(a.A, "/courses/"+id+"/assistants", map[string]any{"ta_ids": []string{}}).code, "Admin cũng được")
	// teacher_id không được gửi qua đường này; thiếu ta_ids ⇒ 422; người sai vai ⇒ 422
	require.Equal(t, http.StatusUnprocessableEntity, r.put(a.G, "/courses/"+id+"/assistants", map[string]any{"teacher_id": a.gv2.ID, "ta_ids": []string{}}).code)
	require.Equal(t, http.StatusUnprocessableEntity, r.put(a.G, "/courses/"+id+"/assistants", map[string]any{}).code)
	require.Equal(t, http.StatusUnprocessableEntity, r.put(a.G, "/courses/"+id+"/assistants", map[string]any{"ta_ids": []string{a.sv.ID.String()}}).code)
}

func TestOtherTeacherCannotManageTAs(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	id := r.mustOpen(a.A, map[string]any{"teacher_id": a.gv.ID, "ta_ids": []string{a.ta.ID.String()}})
	for name, s := range map[string]session{"giảng viên lớp khác": a.G2, "TA của lớp": a.T, "sinh viên": a.S} {
		res := r.put(s, "/courses/"+id+"/assistants", map[string]any{"ta_ids": []string{}})
		require.Equal(t, http.StatusForbidden, res.code, name)
		res = r.get(s, "/courses/"+id+"/assistant-candidates")
		require.Equal(t, http.StatusForbidden, res.code, name)
	}
	_, tas := r.staffOf(id)
	require.Equal(t, []string{a.ta.ID.String()}, tas)
}

func TestAssistantCandidates(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	id := r.mustOpen(a.A, map[string]any{"teacher_id": a.gv.ID})
	_, err := r.pool.Exec(t.Context(), `update users set full_name = 'Nguyễn Thị Hằng' where id = $1`, a.ta.ID)
	require.NoError(t, err)
	inv := r.addUser(uniq("hang.moi"), store.UserRoleTA, store.UserStatusINVITED)
	dis := r.addUser(uniq("hang.khoa"), store.UserRoleTA, store.UserStatusDISABLED)
	_, err = r.pool.Exec(t.Context(), `update users set full_name = 'Hằng Khoá' where id = $1`, dis.ID)
	require.NoError(t, err)

	res := r.get(a.G, "/courses/"+id+"/assistant-candidates?q=hang")
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	items := res.json()["items"].([]any)
	var ids []string
	for _, it := range items {
		m := it.(map[string]any)
		require.Len(t, m, 3, "chỉ id, full_name, email")
		ids = append(ids, m["id"].(string))
	}
	require.Contains(t, ids, a.ta.ID.String(), "tên có dấu tìm được bằng chữ không dấu")
	require.NotContains(t, ids, dis.ID.String(), "DISABLED không có mặt")
	require.NotContains(t, ids, a.gv.ID.String(), "chỉ vai TA")
	// INVITED có mặt khi tìm theo tiền tố email
	res = r.get(a.G, "/courses/"+id+"/assistant-candidates?q=hang.moi")
	require.Contains(t, string(res.body), inv.ID.String())
	// ≤ 20
	for i := 0; i < 25; i++ {
		r.addUser(uniq("zz.ta"), store.UserRoleTA, store.UserStatusACTIVE)
	}
	res = r.get(a.A, "/courses/"+id+"/assistant-candidates?q=zz.ta")
	require.Len(t, res.json()["items"], 20)
	require.Equal(t, http.StatusUnprocessableEntity, r.get(a.G, "/courses/"+id+"/assistant-candidates?q="+strings.Repeat("a", 101)).code)
}

func TestChangeTeacherMidTerm(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	id := r.mustOpen(a.A, map[string]any{"teacher_id": a.gv.ID})
	require.Equal(t, http.StatusOK, r.assign(a.A, id, map[string]any{"teacher_id": a.gv2.ID}).code)
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity = 'course' and entity_id = $1 and action = 'course_created'`, id))
	require.Equal(t, 2, r.count(`select count(*) from audit_log where entity = 'course' and entity_id = $1 and action = 'course_assigned'`, id), "ghi cả lần gán đầu và lần đổi giảng viên")
	var after string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select before::text || after::text from audit_log where entity_id = $1 and action = 'course_assigned' order by created_at desc limit 1`, id).Scan(&after))
	require.Contains(t, after, a.gv.ID.String())
	require.Contains(t, after, a.gv2.ID.String())
	require.Equal(t, 2, r.count(`select count(*) from outbox where topic = 'course.assigned' and payload->>'course_id' = $1`, id))
}

// ===== AC7 =====

func (r *rig) notify(user uuid.UUID, course *uuid.UUID, title string, at time.Time) uuid.UUID {
	r.t.Helper()
	var id uuid.UUID
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into notifications (user_id, course_id, type, title, body, link, created_at) values ($1, $2, 'COURSE_ASSIGNED', $3, 'thân', '/class/members', $4) returning id`,
		user, course, title, at).Scan(&id))
	return id
}

func TestNotificationsList(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	base := time.Now().UTC().Add(-time.Hour)
	for i := 0; i < 7; i++ {
		r.notify(a.gv.ID, nil, fmt.Sprintf("tb %d", i), base.Add(time.Duration(i)*time.Minute))
	}
	res := r.get(a.G, "/notifications?limit=3")
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	m := res.json()
	items := m["items"].([]any)
	require.Len(t, items, 3)
	require.Equal(t, "tb 6", items[0].(map[string]any)["title"], "mới nhất trước")
	for _, k := range []string{"id", "type", "title", "body", "link", "course_id", "read_at", "created_at"} {
		require.Contains(t, items[0], k)
	}
	require.EqualValues(t, 7, m["unread_count"])
	var seen []string
	next := m["next_cursor"]
	for _, it := range items {
		seen = append(seen, it.(map[string]any)["title"].(string))
	}
	for next != nil {
		page := r.get(a.G, "/notifications?limit=3&cursor="+next.(string)).json()
		for _, it := range page["items"].([]any) {
			seen = append(seen, it.(map[string]any)["title"].(string))
		}
		next = page["next_cursor"]
	}
	require.Equal(t, []string{"tb 6", "tb 5", "tb 4", "tb 3", "tb 2", "tb 1", "tb 0"}, seen)
	require.Equal(t, http.StatusUnprocessableEntity, r.get(a.G, "/notifications?limit=101").code)
	require.Equal(t, http.StatusUnprocessableEntity, r.get(a.G, "/notifications?unread_only=maybe").code)
	require.Equal(t, http.StatusUnauthorized, r.do(req{method: http.MethodGet, path: "/notifications"}).code)
}

func TestNotificationsOnlyMine(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	r.notify(a.gv.ID, nil, "của giảng viên", time.Now().UTC())
	mine := r.notify(a.sv.ID, nil, "của sinh viên", time.Now().UTC())
	res := r.get(a.S, "/notifications?user_id="+a.gv.ID.String())
	require.Equal(t, http.StatusOK, res.code)
	require.Len(t, res.json()["items"], 1)
	require.Equal(t, mine.String(), res.json()["items"].([]any)[0].(map[string]any)["id"])
	require.NotContains(t, string(res.body), "của giảng viên")
}

func TestNotificationReadIdempotent(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	id := r.notify(a.gv.ID, nil, "x", time.Now().UTC())
	first := r.post(a.G, "/notifications/"+id.String()+"/read", nil, nil)
	require.Equal(t, http.StatusNoContent, first.code, string(first.body))
	readAt := r.scalar(`select read_at::text from notifications where id = $1`, id)
	require.NotEmpty(t, readAt)
	require.Equal(t, http.StatusNoContent, r.post(a.G, "/notifications/"+id.String()+"/read", nil, nil).code)
	require.Equal(t, readAt, r.scalar(`select read_at::text from notifications where id = $1`, id), "đọc lần hai không đổi read_at")
	unread := r.get(a.G, "/notifications?unread_only=true")
	require.Empty(t, unread.json()["items"])
	require.EqualValues(t, 0, unread.json()["unread_count"])
}

func TestNotificationOtherUser404(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	id := r.notify(a.gv.ID, nil, "riêng tư", time.Now().UTC())
	res := r.post(a.S, "/notifications/"+id.String()+"/read", nil, nil)
	require.Equal(t, http.StatusNotFound, res.code)
	require.Empty(t, r.scalar(`select read_at::text from notifications where id = $1`, id), "không đổi dữ liệu của người khác")
	require.Equal(t, http.StatusNotFound, r.post(a.S, "/notifications/"+uuid.NewString()+"/read", nil, nil).code)
	require.Equal(t, http.StatusNotFound, r.post(a.S, "/notifications/khong-phai-uuid/read", nil, nil).code)
}

func TestNotificationsUnreadCount(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	var ids []uuid.UUID
	for i := 0; i < 5; i++ {
		ids = append(ids, r.notify(a.gv.ID, nil, fmt.Sprint(i), time.Now().UTC().Add(time.Duration(i)*time.Second)))
	}
	r.notify(a.sv.ID, nil, "người khác", time.Now().UTC())
	require.EqualValues(t, 5, r.get(a.G, "/notifications?limit=1").json()["unread_count"], "đếm toàn bộ, không phải theo trang")
	require.Equal(t, http.StatusNoContent, r.post(a.G, "/notifications/"+ids[0].String()+"/read", nil, nil).code)
	res := r.get(a.G, "/notifications?limit=2").json()
	require.EqualValues(t, 4, res["unread_count"])
	only := r.get(a.G, "/notifications?unread_only=true&limit=100").json()
	require.Len(t, only["items"], 4)
}

// ===== AC8 =====

func TestAdminCoursesRBACMatrix(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	id := r.mustOpen(a.A, map[string]any{"teacher_id": a.gv.ID, "ta_ids": []string{a.ta.ID.String()}})
	other := r.mustOpen(a.A, nil)
	issuer := auth.NewIssuer(rigSecret, 0, r.clk)
	token := func(u store.User, jwtRole auth.Role) string {
		tok, err := issuer.Issue(u.ID.String(), jwtRole, u.Email)
		require.NoError(t, err)
		return tok
	}
	type op struct {
		name, method, path string
		body               any
		hdr                map[string]string
	}
	ops := []op{
		{"GET", http.MethodGet, "/admin/courses", nil, nil},
		{"POST", http.MethodPost, "/admin/courses", map[string]any{"subject_code": "INT1006", "class_code": "RB-" + uuid.NewString()[:6], "name": "x", "semester": "2026-2027-HK1"}, idemKey()},
		{"PUT", http.MethodPut, "/admin/courses/" + other, map[string]any{"name": "x", "version": 1}, nil},
		{"assign", http.MethodPost, "/admin/courses/" + other + "/assign", map[string]any{"teacher_id": a.gv.ID}, nil},
		{"archive", http.MethodPost, "/admin/courses/" + id + "/archive", nil, nil},
	}
	cases := 0
	run := func(label string, o op, tok string, want func(code int) bool) {
		t.Helper()
		cases++
		if o.method == http.MethodPost && o.path == "/admin/courses" {
			o.body = map[string]any{"subject_code": "INT1006", "class_code": "RB-" + uuid.NewString()[:6], "name": "x", "semester": "2026-2027-HK1"}
			o.hdr = idemKey()
		}
		res := r.do(req{method: o.method, path: o.path, bearer: tok, body: o.body, hdr: o.hdr})
		require.Truef(t, want(res.code), "%s %s: status=%d %s", label, o.name, res.code, res.body)
		if res.code == http.StatusForbidden {
			require.Equal(t, "role", res.details()["reason"], "%s %s", label, o.name)
		}
	}
	forbidden := func(c int) bool { return c == http.StatusForbidden }
	for _, o := range ops {
		run("sinh viên", o, token(a.sv, auth.RoleStudent), forbidden)
		run("trợ giảng", o, token(a.ta, auth.RoleTA), forbidden)
		run("giảng viên", o, token(a.gv, auth.RoleTeacher), forbidden)
		run("trợ giảng của chính lớp", o, token(a.ta, auth.RoleTA), forbidden)
		run("giảng viên của chính lớp", o, token(a.gv, auth.RoleTeacher), forbidden)
		run("vai JWT thắng DB (DB là ADMIN, JWT là STUDENT)", o, token(a.adm, auth.RoleStudent), forbidden)
		run("không JWT", o, "", func(c int) bool { return c == http.StatusUnauthorized })
		run("Admin", o, token(a.adm, auth.RoleAdmin), func(c int) bool { return c >= 200 && c < 300 })
		// sau lượt Admin (PUT đã tăng version) chỉ cần: không bị chặn quyền
		run("vai JWT thắng DB (DB là SV, JWT là ADMIN)", o, token(a.sv, auth.RoleAdmin), func(c int) bool { return c != http.StatusForbidden && c != http.StatusUnauthorized && c < 500 })
	}
	require.GreaterOrEqual(t, cases, 40)
}

// ===== AC9 =====

func TestAdminCoursesList(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	id1 := r.mustOpen(a.A, map[string]any{"class_code": "LST-001", "name": "An ninh mạng", "semester": "2026-2027-HK1", "capacity": 30, "teacher_id": a.gv.ID, "ta_ids": []string{a.ta.ID.String(), a.ta2.ID.String()}})
	id2 := r.mustOpen(a.A, map[string]any{"class_code": "LST-002", "name": "Cơ sở dữ liệu", "semester": "2026-2027-HK2"})
	c1 := uuid.MustParse(id1)
	for i := 0; i < 3; i++ {
		r.enroll(c1, r.addUser(uniq("s"), store.UserRoleSTUDENT, store.UserStatusACTIVE).ID, "STUDENT", "ACTIVE")
	}
	r.enroll(c1, r.addUser(uniq("p"), store.UserRoleSTUDENT, store.UserStatusACTIVE).ID, "STUDENT", "PENDING")
	require.Equal(t, http.StatusOK, r.post(a.A, "/admin/courses/"+id2+"/archive", nil, nil).code)

	res := r.get(a.A, "/admin/courses?limit=100")
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	var first map[string]any
	for _, it := range res.json()["items"].([]any) {
		if it.(map[string]any)["id"] == id1 {
			first = it.(map[string]any)
		}
	}
	require.NotNil(t, first)
	require.Len(t, first, 12)
	require.Equal(t, "LST-001", first["class_code"])
	require.Equal(t, a.gv.ID.String(), first["teacher"].(map[string]any)["id"])
	require.EqualValues(t, 2, first["assistants_count"])
	require.EqualValues(t, 3, first["students_active"])
	require.EqualValues(t, 1, first["students_pending"])
	require.EqualValues(t, 30, first["capacity"])

	list := func(q string) []string {
		var codes []string
		for _, it := range r.get(a.A, "/admin/courses?"+q).json()["items"].([]any) {
			codes = append(codes, it.(map[string]any)["class_code"].(string))
		}
		return codes
	}
	require.Equal(t, []string{"LST-002"}, list("status=ARCHIVED&q=LST"))
	require.Equal(t, []string{"LST-001"}, list("status=ACTIVE&q=LST"))
	require.Equal(t, []string{"LST-002"}, list("semester=2026-2027-HK2&q=LST"))
	require.Equal(t, []string{"LST-002"}, list("q=co+so+du+lieu"), "tên tìm không dấu")
	require.Equal(t, []string{"LST-001"}, list("q=lst-001"))
	require.Equal(t, http.StatusUnprocessableEntity, r.get(a.A, "/admin/courses?status=NOPE").code)
}

func TestAdminCoursesNoJoinCodeNoStudents(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	id := r.mustOpen(a.A, map[string]any{"teacher_id": a.gv.ID})
	r.enroll(uuid.MustParse(id), a.sv.ID, "STUDENT", "ACTIVE")
	var code string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select join_code from courses where id = $1`, id).Scan(&code))
	for _, res := range []resp{
		r.get(a.A, "/admin/courses?limit=100"),
		r.put(a.A, "/admin/courses/"+id, map[string]any{"name": "x", "version": 1}),
		r.post(a.A, "/admin/courses/"+id+"/archive", nil, nil),
		r.assign(a.A, id, map[string]any{"teacher_id": a.gv.ID}),
	} {
		body := string(res.body)
		require.NotContains(t, body, "join_code")
		require.NotContains(t, body, code)
		require.NotContains(t, body, a.sv.Email)
		require.NotContains(t, body, a.sv.ID.String(), "Admin không thấy danh sách sinh viên")
	}
}

func TestAdminCoursesNoNPlusOne(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	for i := 0; i < 25; i++ {
		r.mustOpen(a.A, map[string]any{"teacher_id": a.gv.ID})
	}
	before := r.qc.n.Load()
	res := r.get(a.A, "/admin/courses?limit=100")
	require.Equal(t, http.StatusOK, res.code)
	require.GreaterOrEqual(t, len(res.json()["items"].([]any)), 25)
	require.EqualValues(t, 1, r.qc.n.Load()-before, "đúng một câu SQL tổng hợp cho cả trang")
}

func TestAdminCoursesCursor(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	want := map[string]bool{}
	for i := 0; i < 7; i++ {
		id := r.mustOpen(a.A, map[string]any{"class_code": fmt.Sprintf("CUR-%03d", i)})
		want[id] = true
	}
	seen := map[string]bool{}
	cursor := ""
	for pages := 0; pages < 10; pages++ {
		q := "/admin/courses?limit=3"
		if cursor != "" {
			q += "&cursor=" + cursor
		}
		res := r.get(a.A, q)
		require.Equal(t, http.StatusOK, res.code)
		for _, it := range res.json()["items"].([]any) {
			id := it.(map[string]any)["id"].(string)
			require.False(t, seen[id], "trùng giữa các trang")
			seen[id] = true
		}
		next, _ := res.json()["next_cursor"].(string)
		if next == "" {
			break
		}
		cursor = next
	}
	for id := range want {
		require.True(t, seen[id])
	}
	require.Equal(t, http.StatusUnprocessableEntity, r.get(a.A, "/admin/courses?limit=0").code)
	require.Equal(t, http.StatusUnprocessableEntity, r.get(a.A, "/admin/courses?cursor=hong").code)
}

// ===== AC5 (trình xử lý; luồng worker thật ở internal/integration) =====

func (r *rig) notifier() *course.Notifier {
	return &course.Notifier{Pool: r.pool, AppPublicURL: "https://localhost"}
}

type queued struct {
	id      uuid.UUID
	payload []byte
}

func (r *rig) assignedMessages(courseID string) []queued {
	r.t.Helper()
	rows, err := r.pool.Query(r.t.Context(), `select id, payload from outbox where topic = 'course.assigned' and payload->>'course_id' = $1 order by created_at, id`, courseID)
	require.NoError(r.t, err)
	defer rows.Close()
	var out []queued
	for rows.Next() {
		var q queued
		require.NoError(r.t, rows.Scan(&q.id, &q.payload))
		out = append(out, q)
	}
	require.NoError(r.t, rows.Err())
	return out
}

func (r *rig) deliverAssigned(courseID string) {
	r.t.Helper()
	for _, m := range r.assignedMessages(courseID) {
		require.NoError(r.t, r.notifier().HandleAssigned(context.Background(), outbox.Message{ID: m.id, Topic: course.TopicAssigned, Payload: m.payload}))
	}
}

func TestAssignNotificationTexts(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	_, err := r.pool.Exec(t.Context(), `update users set full_name = 'Trần Văn Giảng' where id = $1`, a.gv.ID)
	require.NoError(t, err)
	id := r.mustOpen(a.A, map[string]any{"name": "An ninh mạng", "class_code": "NT-761987", "teacher_id": a.gv.ID, "ta_ids": []string{a.ta.ID.String()}})
	r.deliverAssigned(id)
	var code string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select join_code from courses where id = $1`, id).Scan(&code))

	tn := r.get(a.G, "/notifications").json()["items"].([]any)
	require.Len(t, tn, 1)
	m := tn[0].(map[string]any)
	require.Equal(t, "Bạn được phân công lớp An ninh mạng – NT-761987", m["title"])
	require.Equal(t, fmt.Sprintf("Mã tham gia: %s. Chia sẻ mã hoặc đường dẫn https://localhost/join/%s cho sinh viên.", code, code), m["body"])
	require.Equal(t, "/class/settings?course="+id, m["link"])
	require.Equal(t, id, m["course_id"])
	require.Equal(t, "COURSE_ASSIGNED", m["type"])

	tan := r.get(a.T, "/notifications").json()["items"].([]any)
	require.Len(t, tan, 1)
	m = tan[0].(map[string]any)
	require.Equal(t, "Bạn được phân công làm trợ giảng lớp An ninh mạng – NT-761987", m["title"])
	require.Equal(t, "Giảng viên phụ trách: Trần Văn Giảng.", m["body"])
	require.Equal(t, "/class/members?course="+id, m["link"])
	require.NotContains(t, m["body"], code, "TA không nhận mã tham gia")
}

func TestAssignNotifyOnce(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	id := r.mustOpen(a.A, map[string]any{"teacher_id": a.gv.ID})
	r.deliverAssigned(id)
	r.deliverAssigned(id) // giao lại cùng tin
	require.Equal(t, 1, r.count(`select count(*) from notifications where user_id = $1 and type = 'COURSE_ASSIGNED'`, a.gv.ID))
	var key string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select dedupe_key from notifications where user_id = $1`, a.gv.ID).Scan(&key))
	require.Equal(t, "course.assigned:"+r.assignedMessages(id)[0].id.String(), key)
}

func TestAssignRemovedBeforeWorkerGetsNothing(t *testing.T) {
	r := newRig(t)
	a := r.actors()
	id := r.mustOpen(a.A, map[string]any{"teacher_id": a.gv.ID})
	require.Equal(t, http.StatusOK, r.assign(a.A, id, map[string]any{"teacher_id": a.gv2.ID}).code)
	r.deliverAssigned(id)
	require.Equal(t, 0, r.count(`select count(*) from notifications where user_id = $1`, a.gv.ID), "giảng viên đã bị thay không nhận mã tham gia")
	require.Equal(t, 1, r.count(`select count(*) from notifications where user_id = $1`, a.gv2.ID))
}

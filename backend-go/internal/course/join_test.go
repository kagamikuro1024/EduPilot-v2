package course_test

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/store"
)

func (r *rig) del(s session, path string) resp {
	return r.do(req{method: http.MethodDelete, path: path, bearer: s.access})
}

// student tạo sinh viên ACTIVE đã xác minh email (student_code tuỳ chọn) và đăng nhập.
func (r *rig) student(code string) (store.User, session) {
	r.t.Helper()
	u := r.addUser(uniq("sv"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	_, err := r.pool.Exec(r.t.Context(), `update users set email_verified_at = now() where id = $1`, u.ID)
	require.NoError(r.t, err)
	if code != "" {
		_, err = r.pool.Exec(r.t.Context(), `update users set student_code = $2 where id = $1`, u.ID, code)
		require.NoError(r.t, err)
	}
	return u, r.mustLogin(u.Email)
}

func (r *rig) joinCodeOf(courseID string) string {
	return strings.TrimSpace(r.scalar(`select join_code from courses where id = $1`, courseID))
}

func (r *rig) preview(s session, code string) resp {
	return r.post(s, "/courses/join/preview", map[string]any{"code": code}, nil)
}

func (r *rig) join(s session, code string) resp {
	return r.post(s, "/courses/join", map[string]any{"code": code}, nil)
}

type klass struct {
	a      actors
	id     string
	code   string
	course uuid.UUID
}

// klass dựng một lớp có giảng viên và (tuỳ chọn) TA qua API Admin thật.
func (r *rig) klass(extra map[string]any) klass {
	r.t.Helper()
	a := r.actors()
	body := map[string]any{"teacher_id": a.gv.ID, "ta_ids": []string{a.ta.ID.String()}}
	for k, v := range extra {
		body[k] = v
	}
	id := r.mustOpen(a.A, body)
	return klass{a: a, id: id, code: r.joinCodeOf(id), course: uuid.MustParse(id)}
}

func (r *rig) setJoin(k klass, body map[string]any) resp {
	if _, ok := body["version"]; !ok {
		body["version"] = r.count(`select version from courses where id = $1`, k.id)
	}
	return r.put(k.a.G, "/courses/"+k.id+"/join-settings", body)
}

func (r *rig) enrollment(courseID string, user uuid.UUID) (status, warning, snapshot, via string) {
	r.t.Helper()
	row := r.pool.QueryRow(r.t.Context(), `select status::text, coalesce(warning,''), coalesce(student_code_snapshot,''), joined_via::text from enrollments where course_id = $1 and user_id = $2`, courseID, user)
	require.NoError(r.t, row.Scan(&status, &warning, &snapshot, &via))
	return
}

// ===== AC1 =====

func TestJoinPreviewShape(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	_, sv := r.student("")
	res := r.preview(sv, strings.ToLower(k.code))
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	m := res.json()
	require.Len(t, m, 5)
	require.Equal(t, "OPEN", m["state"])
	require.Equal(t, "An ninh mạng", m["name"])
	require.Equal(t, "2026-2027-HK1", m["semester"])
	require.Len(t, m["teachers"], 1)
	body := string(res.body)
	for _, bad := range []string{k.code, k.id, "join_code", "students", "\"id\""} {
		require.NotContains(t, body, bad)
	}
	require.Equal(t, 0, r.count(`select count(*) from enrollments where course_id = $1 and role_in_course = 'STUDENT'`, k.id), "xem trước không ghi danh")
	// các trạng thái còn lại
	require.Equal(t, http.StatusOK, r.setJoin(k, map[string]any{"require_approval": true}).code)
	require.Equal(t, "REQUIRES_APPROVAL", r.preview(sv, k.code).json()["state"])
	require.Equal(t, http.StatusOK, r.join(sv, k.code).code)
	require.Equal(t, "PENDING", r.preview(sv, k.code).json()["state"])
	_, sv2 := r.student("")
	require.Equal(t, http.StatusOK, r.setJoin(k, map[string]any{"require_approval": false, "capacity": 1}).code)
	u3, sv3 := r.student("")
	r.enroll(k.course, u3.ID, "STUDENT", "ACTIVE")
	require.Equal(t, "FULL", r.preview(sv2, k.code).json()["state"])
	require.Equal(t, "ALREADY_MEMBER", r.preview(sv3, k.code).json()["state"])
}

func TestJoinOpen(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	u, sv := r.student("")
	res := r.join(sv, k.code)
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	require.Equal(t, map[string]any{"course_id": k.id, "status": "ACTIVE", "already_member": false}, res.json())
	st, warn, _, via := r.enrollment(k.id, u.ID)
	require.Equal(t, "ACTIVE", st)
	require.Empty(t, warn)
	require.Equal(t, "CODE", via)
	// vào được lớp ngay
	require.Equal(t, http.StatusOK, r.get(sv, "/courses/"+k.id).code)
	mine := r.get(sv, "/me/courses").json()["items"].([]any)
	require.Len(t, mine, 1)
}

func TestJoinCodeCaseInsensitive(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	_, sv := r.student("")
	spaced := " " + strings.ToLower(k.code[:3]) + " " + strings.ToLower(k.code[3:]) + "\t"
	require.Equal(t, http.StatusOK, r.join(sv, spaced).code)
}

func TestJoinSnapshotStudentCode(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	u, sv := r.student("B20DCCN001")
	require.Equal(t, http.StatusOK, r.join(sv, k.code).code)
	_, _, snap, _ := r.enrollment(k.id, u.ID)
	require.Equal(t, "B20DCCN001", snap)
	// đổi MSSV sau đó KHÔNG đổi ảnh chụp
	res := r.put(sv, "/me/profile", map[string]any{"student_code": "B20DCCN999", "version": 1})
	require.Equal(t, http.StatusOK, res.code, string(res.body))
	_, _, snap, _ = r.enrollment(k.id, u.ID)
	require.Equal(t, "B20DCCN001", snap)
	// sinh viên không khai MSSV ⇒ snapshot rỗng
	u2, sv2 := r.student("")
	require.Equal(t, http.StatusOK, r.join(sv2, k.code).code)
	_, _, snap, _ = r.enrollment(k.id, u2.ID)
	require.Empty(t, snap)
}

func TestJoinAudit(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	u, sv := r.student("B20DCCN001")
	require.Equal(t, http.StatusOK, r.join(sv, k.code).code)
	require.Equal(t, http.StatusOK, r.join(sv, k.code).code)
	require.Equal(t, 1, r.count(`select count(*) from audit_log where entity = 'enrollment' and action = 'member_joined' and actor_id = $1 and course_id = $2`, u.ID, k.id), "vào hai lần vẫn một dòng audit")
	var after string
	require.NoError(t, r.pool.QueryRow(t.Context(), `select coalesce(before::text,'') || coalesce(after::text,'') from audit_log where entity = 'enrollment' and action = 'member_joined' and actor_id = $1`, u.ID).Scan(&after))
	for _, bad := range []string{u.Email, "B20DCCN001", k.code} {
		require.NotContains(t, after, bad)
	}
}

// ===== AC2 =====

func TestJoinTwiceOneEnrollment(t *testing.T) {
	r := newRig(t)
	k := r.klass(map[string]any{"capacity": 1})
	u, sv := r.student("")
	first := r.join(sv, k.code)
	second := r.join(sv, k.code)
	require.Equal(t, http.StatusOK, first.code)
	require.Equal(t, http.StatusOK, second.code, string(second.body))
	require.Equal(t, false, first.json()["already_member"])
	require.Equal(t, true, second.json()["already_member"])
	require.Equal(t, 1, r.count(`select count(*) from enrollments where course_id = $1 and user_id = $2`, k.id, u.ID))
	// lớp đầy nhưng người đã trong lớp vẫn nhận already_member, không COURSE_FULL
	require.Equal(t, true, r.join(sv, k.code).json()["already_member"])
	// có Idempotency-Key: phát lại đúng phản hồi
	_, sv2 := r.student("")
	key := idemKey()
	a := r.post(sv2, "/courses/join", map[string]any{"code": k.code}, key)
	b := r.post(sv2, "/courses/join", map[string]any{"code": k.code}, key)
	require.Equal(t, a.code, b.code)
	require.JSONEq(t, string(a.body), string(b.body))
	require.Equal(t, "true", b.hdr.Get("Idempotent-Replayed"))
}

func TestJoinConcurrentOneEnrollment(t *testing.T) {
	r := newRig(t)
	k := r.klass(map[string]any{"capacity": 5})
	u, sv := r.student("")
	var wg sync.WaitGroup
	codes := make([]resp, 50)
	for i := range codes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes[i] = r.join(sv, k.code)
		}()
	}
	wg.Wait()
	fresh := 0
	for _, c := range codes {
		require.Equal(t, http.StatusOK, c.code, string(c.body))
		if c.json()["already_member"] == false {
			fresh++
		}
	}
	require.Equal(t, 1, fresh, "đúng một yêu cầu tạo bản ghi")
	require.Equal(t, 1, r.count(`select count(*) from enrollments where course_id = $1 and user_id = $2`, k.id, u.ID))
	require.Equal(t, 1, r.count(`select count(*) from enrollments where course_id = $1 and role_in_course = 'STUDENT' and status = 'ACTIVE'`, k.id), "sĩ số không đếm hai lần")
}

// ===== AC3 =====

// sixCauses dựng sáu nguyên nhân thất bại và trả mã dùng để thử cho từng nguyên nhân.
func (r *rig) sixCauses() (map[string]string, session) {
	r.t.Helper()
	u, sv := r.student("")
	_ = u
	domain := r.klass(nil)
	require.Equal(r.t, http.StatusOK, r.setJoin(domain, map[string]any{"allowed_email_domain": "khac.edu.vn"}).code)
	disabled := r.klass(nil)
	require.Equal(r.t, http.StatusOK, r.setJoin(disabled, map[string]any{"enabled": false}).code)
	expired := r.klass(nil)
	_, err := r.pool.Exec(r.t.Context(), `update courses set join_expires_at = now() - interval '1 hour' where id = $1`, expired.id)
	require.NoError(r.t, err)
	old := r.klass(nil)
	oldCode := old.code
	require.Equal(r.t, http.StatusOK, r.post(old.a.G, "/courses/"+old.id+"/join-code/regenerate", nil, nil).code)
	archived := r.klass(nil)
	require.Equal(r.t, http.StatusOK, r.post(archived.a.A, "/admin/courses/"+archived.id+"/archive", nil, nil).code)
	return map[string]string{"sai mã": "ZZZZZZZ", "mã cũ": oldCode, "mã tắt": disabled.code, "mã hết hạn": expired.code, "lớp lưu trữ": archived.code, "sai tên miền": domain.code}, sv
}

func TestJoinUniformFailure(t *testing.T) {
	r := newRig(t)
	causes, _ := r.sixCauses()
	seen := map[string]string{}
	for name, code := range causes {
		// mỗi lần một người dùng + IP mới ⇒ không dính giới hạn đoán mã
		for _, op := range []string{"preview", "join"} {
			_, sv := r.student("")
			var res resp
			if op == "preview" {
				res = r.preview(sv, code)
			} else {
				res = r.join(sv, code)
			}
			require.Equal(t, http.StatusNotFound, res.code, name+" "+op+" "+string(res.body))
			m := res.json()
			delete(m, "trace_id")
			b := fmt.Sprint(m)
			require.Equal(t, "JOIN_CODE_INVALID", m["code"], name)
			require.Equal(t, "Mã không hợp lệ hoặc đã hết hạn. Kiểm tra lại với giảng viên.", m["message"], name)
			if prev, ok := seen["x"]; ok {
				require.Equal(t, prev, b, name+" "+op)
			}
			seen["x"] = b
		}
	}
	// mã sai định dạng cũng cùng khuôn
	_, sv := r.student("")
	res := r.join(sv, "../etc")
	require.Equal(t, http.StatusNotFound, res.code)
	require.Equal(t, "JOIN_CODE_INVALID", res.errCode())
}

func TestJoinFullIsDistinct(t *testing.T) {
	r := newRig(t)
	k := r.klass(map[string]any{"capacity": 1})
	u1, _ := r.student("")
	r.enroll(k.course, u1.ID, "STUDENT", "ACTIVE")
	_, sv := r.student("")
	res := r.join(sv, k.code)
	require.Equal(t, http.StatusConflict, res.code)
	require.Equal(t, "COURSE_FULL", res.errCode())
	require.Equal(t, "Lớp đã đủ sĩ số. Hãy báo giảng viên.", res.json()["message"])
	// duyệt vẫn chờ: lớp cần duyệt mà đầy ⇒ vẫn COURSE_FULL (không để chờ vô ích)
	require.Equal(t, http.StatusOK, r.setJoin(k, map[string]any{"require_approval": true}).code)
	require.Equal(t, http.StatusConflict, r.join(sv, k.code).code)
}

// ===== AC11 =====

func TestJoinUnverified403(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	u := r.addUser(uniq("unv"), store.UserRoleSTUDENT, store.UserStatusACTIVE) // chưa xác minh
	s := r.mustLogin(u.Email)
	for _, res := range []resp{r.preview(s, k.code), r.join(s, k.code)} {
		require.Equal(t, http.StatusForbidden, res.code, string(res.body))
		require.Equal(t, "EMAIL_NOT_VERIFIED", res.errCode())
	}
	require.Equal(t, 0, r.count(`select count(*) from enrollments where user_id = $1`, u.ID))
}

func TestJoinNonStudent403(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	for name, s := range map[string]session{"ADMIN": k.a.A, "TEACHER": k.a.G2, "TA": k.a.T2} {
		for _, res := range []resp{r.preview(s, k.code), r.join(s, k.code)} {
			require.Equal(t, http.StatusForbidden, res.code, name)
			require.Equal(t, "role", res.details()["reason"], name)
		}
	}
	require.Equal(t, 0, r.count(`select count(*) from enrollments where course_id = $1 and role_in_course = 'STUDENT'`, k.id))
}

func TestJoinNoJWT401(t *testing.T) {
	r := newRig(t)
	for _, p := range []string{"/courses/join/preview", "/courses/join"} {
		res := r.do(req{method: http.MethodPost, path: p, body: map[string]any{"code": "AN7K2MQ"}})
		require.Equal(t, http.StatusUnauthorized, res.code, p)
	}
}

package crm_test

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

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

func (r *rig) exec(sql string, args ...any) {
	r.t.Helper()
	_, err := r.pool.Exec(r.t.Context(), sql, args...)
	require.NoError(r.t, err, sql)
}

func (r *rig) call(method, path string, s session, body any) resp {
	return r.do(req{method: method, path: path, bearer: s.access, body: body})
}

// world: một lớp ACTIVE có giảng viên, TA, n sinh viên ACTIVE (tên "Sv 01"…), hai buổi; kèm Admin, một sinh viên ngoài lớp, một sinh viên PENDING.
type world struct {
	r              *rig
	course         uuid.UUID
	teacher, ta    store.User
	T, A, ADM, OUT session // giảng viên, TA, admin, sinh viên ngoài lớp
	students       []store.User
	pending        store.User
	ses            []uuid.UUID
}

func (r *rig) world(n int) world {
	r.t.Helper()
	w := world{r: r}
	adm := r.addUser(uniq("adm"), store.UserRoleADMIN, store.UserStatusACTIVE)
	w.teacher = r.addUser(uniq("gv"), store.UserRoleTEACHER, store.UserStatusACTIVE)
	w.ta = r.addUser(uniq("ta"), store.UserRoleTA, store.UserStatusACTIVE)
	out := r.addUser(uniq("out"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	w.T, w.A, w.ADM, w.OUT = r.mustLogin(w.teacher.Email), r.mustLogin(w.ta.Email), r.mustLogin(adm.Email), r.mustLogin(out.Email)
	w.course = r.course(adm.ID, "C"+strings.ToUpper(uuid.NewString()[:6]))
	r.enroll(w.course, w.teacher.ID, "TEACHER", "ACTIVE")
	r.enroll(w.course, w.ta.ID, "TA", "ACTIVE")
	for i := 1; i <= n; i++ {
		u := r.addUser(uniq("sv"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
		r.exec(`update users set full_name = $1 where id = $2`, fmt.Sprintf("Sv %02d", i), u.ID)
		r.enroll(w.course, u.ID, "STUDENT", "ACTIVE")
		w.students = append(w.students, u)
	}
	w.pending = r.addUser(uniq("pend"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	r.enroll(w.course, w.pending.ID, "STUDENT", "PENDING")
	base := time.Now().Add(-2 * time.Hour)
	for i := 1; i <= 2; i++ {
		var id uuid.UUID
		require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `insert into class_sessions (course_id, session_no, starts_at, ends_at) values ($1, $2, $3, $4) returning id`,
			w.course, i, base.Add(time.Duration(i)*24*time.Hour), base.Add(time.Duration(i)*24*time.Hour+2*time.Hour)).Scan(&id))
		w.ses = append(w.ses, id)
	}
	return w
}

func (w world) path(i int, tail string) string {
	return "/courses/" + w.course.String() + "/sessions/" + w.ses[i].String() + tail
}

func (w world) grid(s session, i int) resp {
	return w.r.call(http.MethodGet, w.path(i, "/attendance"), s, nil)
}

type rec = map[string]any

func (w world) put(s session, i int, records []rec, complete bool) resp {
	return w.r.call(http.MethodPut, w.path(i, "/attendance"), s, map[string]any{"records": records, "complete": complete})
}

func row(id uuid.UUID, status string, extra ...any) rec {
	m := rec{"student_id": id.String(), "status": status}
	for i := 0; i+1 < len(extra); i += 2 {
		m[extra[i].(string)] = extra[i+1]
	}
	return m
}

func (w world) all(status string) []rec {
	out := make([]rec, len(w.students))
	for i, u := range w.students {
		out[i] = row(u.ID, status)
	}
	return out
}

func items(x resp, key string) []map[string]any {
	var out []map[string]any
	for _, it := range x.json()[key].([]any) {
		out = append(out, it.(map[string]any))
	}
	return out
}

// addUserNamed ghi danh thêm một sinh viên ACTIVE tên `name` vào lớp.
func (r *rig) addUserNamed(w world, name string) store.User {
	r.t.Helper()
	u := r.addUser(uniq("sv"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	r.exec(`update users set full_name = $1 where id = $2`, name, u.ID)
	r.enroll(w.course, u.ID, "STUDENT", "ACTIVE")
	return u
}

// addUserRemoved ghi danh một sinh viên REMOVED (đã bị mời ra).
func (r *rig) addUserRemoved(w world) store.User {
	r.t.Helper()
	u := r.addUser(uniq("gone"), store.UserRoleSTUDENT, store.UserStatusACTIVE)
	r.enroll(w.course, u.ID, "STUDENT", "REMOVED")
	return u
}

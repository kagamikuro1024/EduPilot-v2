package today_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/edupilot/backend-go/internal/grade"
	"github.com/edupilot/backend-go/internal/platform/outbox"
	"github.com/edupilot/backend-go/internal/store"
	"github.com/edupilot/backend-go/internal/today"
)

func gradeItems(res resp) []map[string]any {
	var out []map[string]any
	if rec, ok := res.json()["recommended"].(map[string]any); ok && rec["kind"] == "GRADE_SCHEME_UNCONFIRMED" {
		out = append(out, rec)
	}
	for _, a := range actionsOf(res) {
		if m := a.(map[string]any); m["kind"] == "GRADE_SCHEME_UNCONFIRMED" {
			out = append(out, m)
		}
	}
	return out
}

func (r *rig) policyDoc(c cls) {
	r.t.Helper()
	r.exec(`insert into documents (course_id, title, type, status) values ($1, 'Quy chế', 'COURSE_POLICY', 'READY')`, c.id)
}

// draft: bản nháp qua được bộ kiểm khi ready, nếu không thiếu mọi luật (mục chưa rõ).
func (r *rig) draft(c cls, owner store.User, ready bool) string {
	r.t.Helper()
	defer func() { require.NoError(r.t, (&grade.Service{Pool: r.pool}).RecomputeUnclear(r.t.Context(), c.id)) }() // số mục chưa rõ nằm ở cột (#16)
	sc := r.scalar(`insert into grade_schemes (course_id, scheme_no, created_by) values ($1, 1, $2) returning id`, c.id, owner.ID)
	if !ready {
		r.exec(`insert into grade_components (course_id, scheme_id, key, name, grp, weight) values ($1, $2, 'weekly', 'Hằng tuần', 'QT', 20)`, c.id, sc)
		return sc
	}
	r.exec(`insert into grade_components (course_id, scheme_id, key, name, grp, weight, position) values ($1, $2, 'weekly', 'Hằng tuần', 'QT', 40, 0), ($1, $2, 'final', 'Cuối kỳ', 'CK', 60, 1)`, c.id, sc)
	r.exec(`insert into grade_rules (course_id, scheme_id, kind, applicable) values ($1, $2, 'BONUS', false), ($1, $2, 'PENALTY', false)`, c.id, sc)
	r.exec(`insert into grade_rules (course_id, scheme_id, kind, params) values ($1, $2, 'ROUNDING', '{"step":"0.5","mode":"HALF_UP"}'),
		($1, $2, 'LETTER_SCALE', '{"bands":[{"min":"5","letter":"A"},{"min":"0","letter":"F"}]}')`, c.id, sc)
	return sc
}

// AC5: tiêu đề, lý do theo trạng thái, href, bậc 70; TEACHER và TA thấy, STUDENT và ADMIN không.
func TestProviderGradeSchemeUnconfirmed(t *testing.T) {
	r := newRig(t)
	c := r.klass("763001")
	gv, gvSess := r.teaching(c)
	ta, taSess := r.person(store.UserRoleTA, true)
	r.enrollAt(c, ta, "TA", "ACTIVE", "", 24*time.Hour)
	_, svSess := r.person(store.UserRoleSTUDENT, true)
	_, adSess := r.person(store.UserRoleADMIN, true)

	want := func(reason string) {
		t.Helper()
		for name, s := range map[string]session{"giảng viên": gvSess, "trợ giảng": taSess} {
			items := gradeItems(r.tget(s, "/me/today"))
			require.Len(t, items, 1, name)
			require.Equal(t, "Công thức điểm chưa xác nhận · 763001", items[0]["title"], name)
			require.Equal(t, reason, items[0]["reason"], name)
			require.Equal(t, fmt.Sprintf("/gradebook/scheme?course=%s", c.id), items[0]["href"], name)
		}
	}
	want("Lớp chưa có quy chế môn học.")
	r.policyDoc(c)
	want("Đã có quy chế, chưa trích công thức.")
	sc := r.draft(c, gv, false)
	items := gradeItems(r.tget(gvSess, "/me/today"))
	require.Regexp(t, `^Bản nháp còn \d+ mục chưa rõ\.$`, items[0]["reason"])
	r.exec(`delete from grade_components where scheme_id = $1`, sc)
	r.exec(`delete from grade_schemes where id = $1`, sc)
	r.draft(c, gv, true)
	want("Bản nháp sẵn sàng, chờ xác nhận.")

	for name, s := range map[string]session{"sinh viên": svSess, "admin": adSess} {
		res := r.tget(s, "/me/today")
		require.Empty(t, gradeItems(res), name)
		require.NotContains(t, string(res.body), "GRADE_SCHEME_UNCONFIRMED", name)
	}
	require.Equal(t, today.TierGradeSchemeUnconfirmed, 70)
}

// Xác nhận công thức → mục biến mất: topic `grade.scheme_confirmed` xoá cache của Staff, lần đọc sau không còn mục.
func TestProviderClearedOnConfirm(t *testing.T) {
	r := newRig(t)
	c := r.klass("763002")
	gv, gvSess := r.teaching(c)
	r.policyDoc(c)
	sc := r.draft(c, gv, true)
	require.Len(t, gradeItems(r.tget(gvSess, "/me/today")), 1)

	r.exec(`update grade_schemes set status = 'CONFIRMED', confirmed_by = $2, confirmed_at = now() where id = $1`, sc, gv.ID)
	require.Len(t, gradeItems(r.tgetC(gvSess, "/me/today")), 1, "cache còn giữ bản cũ tới khi có sự kiện")

	payload, _ := json.Marshal(map[string]string{"course_id": c.sid()})
	inv := today.Invalidator{Pool: r.pool, Redis: r.rdb}
	require.NoError(t, inv.Handle(t.Context(), outbox.Message{Topic: grade.TopicSchemeConfirmed, Payload: payload}))
	require.Empty(t, gradeItems(r.tgetC(gvSess, "/me/today")))
}

// Lớp đã lưu trữ không còn mục.
func TestProviderGradeSkipsArchived(t *testing.T) {
	r := newRig(t)
	c := r.klass("763003")
	_, gvSess := r.teaching(c)
	require.Len(t, gradeItems(r.tget(gvSess, "/me/today")), 1)
	r.exec(`update courses set status = 'ARCHIVED', archived_at = now(), join_enabled = false where id = $1`, c.id)
	require.Empty(t, gradeItems(r.tget(gvSess, "/me/today")))
}

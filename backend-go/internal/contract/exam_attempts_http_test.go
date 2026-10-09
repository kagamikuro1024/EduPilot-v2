package contract

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
)

// TestStudentGuardStates — US-PE-05 AC14: sinh viên PENDING / REMOVED / ngoài lớp → 403 `reason=course` ở MỌI route làm bài; Giảng viên / TA / Admin → 403 `reason=role` hoặc 403;
// lớp ARCHIVED → bắt đầu 409 COURSE_ARCHIVED. TestRemovedMidExam403: bị mời ra giữa giờ → mọi ghi tiếp theo 403 ở yêu cầu kế (guard không cache), lượt đã có không bị xoá.
func TestStudentGuardStates(t *testing.T) { runGuardStates(t, false) }

// TestRemovedMidExam403 — xem trên.
func TestRemovedMidExam403(t *testing.T) { runGuardStates(t, true) }

func runGuardStates(t *testing.T, midExam bool) {
	r := &runner{t: t, rig: getRig(t), cache: map[string]string{}}
	r.prod, r.test = mustLoad(t)
	ctx := context.Background()
	db := r.rig.deps.DB
	mk := func(role string) uuid.UUID {
		var id uuid.UUID
		if err := db.QueryRow(ctx, `insert into users (email, full_name, role, status, password_hash, email_verified_at) values ($1, 'Người Thử', $2::user_role, 'ACTIVE', 'x', now()) returning id`,
			"ct-gs-"+uuid.NewString()[:8]+"@example.test", role).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	adminID, gvID, svID, pendID, remID, outID := mk("ADMIN"), mk("TEACHER"), mk("STUDENT"), mk("STUDENT"), mk("STUDENT"), mk("STUDENT")
	var course uuid.UUID
	if err := db.QueryRow(ctx, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp', '2026-2027-HK1', $2, $3) returning id`,
		"GS-"+uuid.NewString()[:6], "GS"+strings.ToUpper(strings.NewReplacer("0", "X", "1", "Y", "O", "Z", "I", "W", "L", "V").Replace(uuid.NewString()[:5])), adminID).Scan(&course); err != nil {
		t.Fatal(err)
	}
	enrol := func(u uuid.UUID, role, status string) {
		if _, err := db.Exec(ctx, `insert into enrollments (course_id, user_id, role_in_course, status, joined_via, removed_at) values ($1, $2, $3::enrollment_role, $4::enrollment_status, 'ADMIN', case when $4 = 'REMOVED' then now() end)`, course, u, role, status); err != nil {
			t.Fatal(err)
		}
	}
	enrol(gvID, "TEACHER", "ACTIVE")
	enrol(svID, "STUDENT", "ACTIVE")
	enrol(pendID, "STUDENT", "PENDING")
	enrol(remID, "STUDENT", "REMOVED")
	tok := map[string]string{
		"GV": r.rig.token(t, gvID.String(), auth.RoleTeacher), "SV": r.rig.token(t, svID.String(), auth.RoleStudent), "PENDING": r.rig.token(t, pendID.String(), auth.RoleStudent),
		"REMOVED": r.rig.token(t, remID.String(), auth.RoleStudent), "OUT": r.rig.token(t, outID.String(), auth.RoleStudent), "ADMIN": r.rig.token(t, adminID.String(), auth.RoleAdmin),
	}
	// bài đang mở có một câu
	qb := "/api/v1/courses/" + course.String() + "/questions"
	_, b := r.must(call{method: "POST", path: qb, token: tok["GV"], body: `{"type":"TRUE_FALSE","title":"tf","topic":"t","stem":"1<2","value":true}`}, 201)
	var q struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
	}
	_ = json.Unmarshal(b, &q)
	r.must(call{method: "PUT", path: qb + "/" + q.ID + "/review", token: tok["GV"], body: `{"decision":"APPROVE","version":` + itoa(q.Version) + `}`}, 200)
	now := time.Now().UTC()
	eb := "/api/v1/courses/" + course.String() + "/exams"
	_, b = r.must(call{method: "POST", path: eb, token: tok["GV"], headers: map[string]string{"Idempotency-Key": "gs-" + uuid.NewString()}, body: `{"title":"Giữa kỳ","opens_at":"` + now.Add(2*time.Hour).Format(time.RFC3339) + `","closes_at":"` + now.Add(4*time.Hour).Format(time.RFC3339) + `","duration_minutes":45}`}, 201)
	var e struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
	}
	_ = json.Unmarshal(b, &e)
	r.must(call{method: "PUT", path: eb + "/" + e.ID + "/items", token: tok["GV"], body: `{"items":[{"question_id":"` + q.ID + `","points":"1"}],"version":` + itoa(e.Version) + `}`}, 200)
	r.must(call{method: "POST", path: eb + "/" + e.ID + "/schedule", token: tok["GV"]}, 200)
	if _, err := db.Exec(ctx, `update exams set status='OPEN', opens_at=now() - interval '1 minute', closes_at=now() + interval '3 hours' where id=$1`, e.ID); err != nil {
		t.Fatal(err)
	}
	one := eb + "/" + e.ID
	tab := uuid.NewString()
	hdr := func(idem bool) map[string]string {
		h := map[string]string{"X-Exam-Tab": tab}
		if idem {
			h["Idempotency-Key"] = "gs-" + uuid.NewString()
		}
		return h
	}
	aid := uuid.NewString()
	routes := []struct {
		method, path, body string
		idem               bool
	}{
		{"POST", one + "/attempts", "", true}, {"GET", one + "/attempts/mine", "", false}, {"PUT", one + "/attempts/" + aid + "/answers", `{"items":[]}`, false},
		{"POST", one + "/attempts/" + aid + "/takeover", "", false}, {"POST", one + "/attempts/" + aid + "/submit", "", true}, {"GET", one + "/attempts/" + aid + "/result", "", false},
	}
	forbidden := func(who string, reason string) {
		t.Helper()
		for _, o := range routes {
			st, _, body := r.do(call{method: o.method, path: o.path, token: tok[who], headers: hdr(o.idem), body: o.body})
			if st != http.StatusForbidden {
				t.Errorf("%s %s với %s → %d (cần 403): %s", o.method, o.path, who, st, body)
				continue
			}
			var d struct {
				Details struct {
					Reason string `json:"reason"`
				} `json:"details"`
			}
			_ = json.Unmarshal([]byte(body), &d)
			if reason != "" && d.Details.Reason != reason {
				t.Errorf("%s %s với %s → reason %q (cần %q)", o.method, o.path, who, d.Details.Reason, reason)
			}
		}
	}
	forbidden("PENDING", "course")
	forbidden("REMOVED", "course")
	forbidden("OUT", "course")
	forbidden("GV", "role")
	forbidden("ADMIN", "")

	if !midExam {
		// lớp lưu trữ → bắt đầu 409
		if _, err := db.Exec(ctx, `update courses set status='ARCHIVED', archived_at=now(), join_enabled=false where id=$1`, course); err != nil {
			t.Fatal(err)
		}
		r.must(call{method: "POST", path: one + "/attempts", token: tok["SV"], headers: hdr(true)}, 409)
		return
	}
	// giữa giờ: sinh viên đang làm, rồi bị mời ra
	_, b = r.must(call{method: "POST", path: one + "/attempts", token: tok["SV"], headers: hdr(true)}, 201)
	var st struct {
		Attempt struct {
			ID string `json:"id"`
		} `json:"attempt"`
		Items []struct {
			ItemID string `json:"item_id"`
		} `json:"items"`
	}
	_ = json.Unmarshal(b, &st)
	att := one + "/attempts/" + st.Attempt.ID
	save := `{"items":[{"item_id":"` + st.Items[0].ItemID + `","answer":{"value":true}}]}`
	r.must(call{method: "PUT", path: att + "/answers", token: tok["SV"], headers: hdr(false), body: save}, 200)
	if _, err := db.Exec(ctx, `update enrollments set status='REMOVED', removed_at=now() where course_id=$1 and user_id=$2`, course, svID); err != nil {
		t.Fatal(err)
	}
	r.must(call{method: "PUT", path: att + "/answers", token: tok["SV"], headers: hdr(false), body: save}, 403)
	r.must(call{method: "POST", path: att + "/submit", token: tok["SV"], headers: hdr(true)}, 403)
	r.must(call{method: "POST", path: att + "/takeover", token: tok["SV"], headers: hdr(false)}, 403)
	var n int
	if err := db.QueryRow(ctx, `select count(*) from exam_attempts a join exam_answers x on x.attempt_id = a.id where a.id=$1`, st.Attempt.ID).Scan(&n); err != nil || n != 1 {
		t.Errorf("lượt và câu trả lời đã lưu phải còn nguyên sau khi bị mời ra: n=%d err=%v", n, err)
	}
}

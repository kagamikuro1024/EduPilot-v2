package contract

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
)

// TestQuestionListPermissionMatrix — US-PE-03 AC11: MỌI thao tác `…/questions…` (16 thao tác) chỉ Giảng viên / TA của lớp; Sinh viên của lớp, ADMIN và người ngoài lớp → 403;
// không token → 401. Sinh viên không có route nào để đọc câu hỏi (kể cả đáp án).
func TestQuestionListPermissionMatrix(t *testing.T) {
	r := &runner{t: t, rig: getRig(t), cache: map[string]string{}}
	r.prod, r.test = mustLoad(t)
	ctx := context.Background()
	db := r.rig.deps.DB
	mk := func(role string) uuid.UUID {
		var id uuid.UUID
		if err := db.QueryRow(ctx, `insert into users (email, full_name, role, status, password_hash, email_verified_at) values ($1, 'Người Thử', $2::user_role, 'ACTIVE', 'x', now()) returning id`,
			"ct-mx-"+uuid.NewString()[:8]+"@example.test", role).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	adminID, gvID, taID, svID, outID := mk("ADMIN"), mk("TEACHER"), mk("TA"), mk("STUDENT"), mk("STUDENT")
	var course uuid.UUID
	if err := db.QueryRow(ctx, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp', '2026-2027-HK1', $2, $3) returning id`,
		"MX-"+uuid.NewString()[:6], "MX"+strings.ToUpper(strings.NewReplacer("0", "X", "1", "Y", "O", "Z", "I", "W", "L", "V").Replace(uuid.NewString()[:5])), adminID).Scan(&course); err != nil {
		t.Fatal(err)
	}
	for id, role := range map[uuid.UUID]string{gvID: "TEACHER", taID: "TA", svID: "STUDENT"} {
		if _, err := db.Exec(ctx, `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, $3::enrollment_role, 'ACTIVE', 'ADMIN')`, course, id, role); err != nil {
			t.Fatal(err)
		}
	}
	tok := map[string]string{
		"TEACHER": r.rig.token(t, gvID.String(), auth.RoleTeacher), "TA": r.rig.token(t, taID.String(), auth.RoleTA),
		"STUDENT": r.rig.token(t, svID.String(), auth.RoleStudent), "ADMIN": r.rig.token(t, adminID.String(), auth.RoleAdmin), "OUTSIDER": r.rig.token(t, outID.String(), auth.RoleStudent),
	}
	base := "/api/v1/courses/" + course.String() + "/questions"
	_, b := r.must(call{method: "POST", path: base, token: tok["TEACHER"], body: `{"type":"CODE","title":"c","topic":"t","stem":"s"}`}, 201)
	var q struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(b, &q)
	one := base + "/" + q.ID
	idem := func() map[string]string { return map[string]string{"Idempotency-Key": "ct-" + uuid.NewString()} }
	ops := []struct {
		method, path, body string
		idem               bool
	}{
		{"GET", base, "", false}, {"POST", base, `{"type":"TRUE_FALSE","title":"t","topic":"t","stem":"s","value":true}`, false},
		{"POST", base + "/suggest", `{"kind":"MCQ","topic":"x","count":1}`, true}, {"GET", one, "", false}, {"PUT", one, `{"type":"CODE","title":"c","topic":"t","stem":"s","version":1}`, false},
		{"POST", one + "/archive", "", false}, {"POST", one + "/duplicate", "", false}, {"PUT", one + "/review", `{"decision":"REQUEST","version":1}`, false},
		{"PUT", one + "/code", `{"languages":["cpp17"],"version":1}`, false}, {"GET", one + "/testcases", "", false},
		{"POST", one + "/testcases", `{"name":"a","input":"1","expected":"1"}`, false}, {"POST", one + "/testcases/import", "", false},
		{"POST", one + "/testcases/approve", `{"ids":["` + uuid.NewString() + `"]}`, false}, {"PUT", one + "/testcases/" + uuid.NewString(), `{"weight":1}`, false},
		{"DELETE", one + "/testcases/" + uuid.NewString(), "", false}, {"POST", one + "/reference/verify", "", true},
	}
	if len(ops) != 16 {
		t.Fatalf("cần đủ 16 thao tác, có %d", len(ops))
	}
	for _, o := range ops {
		h := map[string]string{}
		if o.idem {
			h = idem()
		}
		for _, role := range []string{"STUDENT", "ADMIN", "OUTSIDER"} {
			if st, _, body := r.do(call{method: o.method, path: o.path, token: tok[role], headers: h, body: o.body}); st != http.StatusForbidden {
				t.Errorf("%s %s với %s → %d (cần 403): %s", o.method, o.path, role, st, body)
			}
		}
		if st, _, _ := r.do(call{method: o.method, path: o.path, headers: h, body: o.body}); st != http.StatusUnauthorized {
			t.Errorf("%s %s không token → %d (cần 401)", o.method, o.path, st)
		}
		for _, role := range []string{"TEACHER", "TA"} {
			if st, _, body := r.do(call{method: o.method, path: o.path, token: tok[role], headers: h, body: o.body}); st == http.StatusForbidden || st == http.StatusUnauthorized || st >= 500 {
				t.Errorf("%s %s với %s → %d (Staff phải qua guard): %s", o.method, o.path, role, st, body)
			}
		}
	}
}

func mustLoad(t *testing.T) (*Spec, *Spec) {
	t.Helper()
	prod, test := loadBoth(t)
	return prod, test
}

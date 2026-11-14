package contract

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

// guidanceScenarios: «Lưu ý chấm» (SRS FEAT-grading-guidance 6) với mọi status đã khai báo. Dùng một bài tập ESSAY riêng của lớp `x.cid`.
func (r *runner) guidanceScenarios(x examRig) {
	ctx := context.Background()
	db := r.rig.deps.DB
	course := "/api/v1/courses/" + x.cid
	none := "00000000-0000-7000-8000-000000000000"

	// bài tập ESSAY dùng riêng
	due := time.Now().UTC().Add(72 * time.Hour).Format(time.RFC3339)
	asgBody := `{"title":"BT lưu ý","submission_code":"BTG1","due_at":"` + due + `","max_score":"10","channels":["IN_APP"],"rubric":[{"name":"Khái niệm","max_score":"4"},{"name":"Ví dụ","max_score":"6"}],"reference_answer":"` + strings.Repeat("Đáp án chuẩn. ", 8) + `"}`
	_, b := r.must(call{method: "POST", path: course + "/assignments", token: x.gv, headers: x.idem(), body: asgBody}, 201)
	var asg struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(b, &asg)
	gd := course + "/assignments/" + asg.ID + "/guidance"

	// cờ: GET (200 / 401 / 403 / 404)
	fl := course + "/guidance-flags"
	r.must(call{method: "GET", path: fl, token: x.gv}, 200)
	r.must(call{method: "GET", path: fl, token: x.admin}, 200)
	r.must(call{method: "GET", path: fl}, 401)
	for _, who := range []string{x.ta, x.sv} {
		r.must(call{method: "GET", path: fl, token: who}, 403)
	}
	r.must(call{method: "GET", path: "/api/v1/courses/khong-phai-uuid/guidance-flags", token: x.gv}, 404)

	// cờ: PUT (200 / 401 / 403 / 404 / 409 / 422)
	put := fl + "/GRADING_GUIDANCE"
	r.must(call{method: "PUT", path: put, token: x.gv, body: `{"enabled":false,"version":1}`}, 200) // tắt → version 2
	r.must(call{method: "PUT", path: put, token: x.gv, body: `{"enabled":true,"version":1}`}, 409)
	r.must(call{method: "PUT", path: put, body: `{"enabled":true,"version":2}`}, 401)
	for _, who := range []string{x.ta, x.sv} {
		r.must(call{method: "PUT", path: put, token: who, body: `{"enabled":true,"version":2}`}, 403)
	}
	r.must(call{method: "PUT", path: fl + "/REGRESSION_CAPTURE", token: x.gv, body: `{"enabled":true,"version":1}`}, 403) // chỉ ADMIN
	r.must(call{method: "PUT", path: fl + "/REGRESSION_CAPTURE", token: x.admin, body: `{"enabled":true,"version":1}`}, 200)
	r.must(call{method: "PUT", path: fl + "/LA", token: x.gv, body: `{"enabled":true,"version":1}`}, 422)
	r.must(call{method: "PUT", path: put, token: x.gv, body: `{"version":2}`}, 422)
	r.must(call{method: "PUT", path: "/api/v1/courses/khong-phai-uuid/guidance-flags/GRADING_GUIDANCE", token: x.gv, body: `{"enabled":true,"version":1}`}, 404)

	// tổng quan (200 / 401 / 403 / 404 / 422)
	r.must(call{method: "GET", path: gd, token: x.gv}, 200)
	r.must(call{method: "GET", path: gd, token: x.ta}, 200)
	r.must(call{method: "GET", path: gd}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "GET", path: gd, token: who}, 403)
	}
	r.must(call{method: "GET", path: course + "/assignments/" + none + "/guidance", token: x.gv}, 404)
	r.must(call{method: "GET", path: gd + "?limit=0", token: x.gv}, 422)

	// rút lưu ý: 401 / 403 / 404 / 409 (tắt, thiếu dữ liệu) / 202 / 429
	rf := gd + "/reflect"
	r.must(call{method: "POST", path: rf, headers: x.idem()}, 401)
	for _, who := range []string{x.ta, x.sv, x.admin} {
		r.must(call{method: "POST", path: rf, token: who, headers: x.idem()}, 403)
	}
	r.must(call{method: "POST", path: rf, token: x.gv, headers: x.idem()}, 409)                    // GUIDANCE_DISABLED (cờ đang tắt)
	r.must(call{method: "PUT", path: put, token: x.gv, body: `{"enabled":true,"version":2}`}, 200) // bật → version 3
	r.must(call{method: "POST", path: rf, token: x.gv, headers: x.idem()}, 409)                    // NOT_ENOUGH_DATA
	// cờ tắt trả 409 trước khi tra bài tập, nên 404 chỉ kiểm được sau khi bật
	r.must(call{method: "POST", path: course + "/assignments/" + none + "/guidance/reflect", token: x.gv, headers: x.idem()}, 404)
	tick := 0
	reviewed := func(n int) {
		for range n {
			tick++ // mốc duyệt luôn tăng: dữ liệu mới = khoá idempotent mới
			var stu, sub uuid.UUID
			if err := db.QueryRow(ctx, `insert into users (email, full_name, role) values ($1, 'SV G', 'STUDENT') returning id`, "ct-g-"+uuid.NewString()[:8]+"@example.test").Scan(&stu); err != nil {
				r.t.Fatal(err)
			}
			if _, err := db.Exec(ctx, `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN')`, x.cid, stu); err != nil {
				r.t.Fatal(err)
			}
			if err := db.QueryRow(ctx, `insert into submissions (course_id, assignment_id, student_id, source, version, blob_key, file_name, file_size, file_mime, sha256, match_status, match_method)
				values ($1, $2, $3, 'IN_APP', 1, 'k/'||gen_random_uuid()::text, 'a.txt', 10, 'text/plain', 'x', 'MATCHED', 'IN_APP') returning id`, x.cid, asg.ID, stu).Scan(&sub); err != nil {
				r.t.Fatal(err)
			}
			if _, err := db.Exec(ctx, `with rid as (select rubric->0->>'id' as id from assignments where id = $2)
				insert into grading_results (course_id, submission_id, run_state, status, criteria, ai_scores, ai_total_before_late, total, reviewed_by, reviewed_at, first_reviewed_by, first_reviewed_at)
				select $1, $3, 'DONE', 'REVIEWED', jsonb_build_array(jsonb_build_object('id', rid.id, 'score', '1.00', 'comment', '', 'edited', true, 'evidence', '[]'::jsonb)),
				       jsonb_build_object(rid.id, '2.00'), 2, 1, g.u, now() + ($4 || ' seconds')::interval, g.u, now() from rid, (select created_by as u from assignments where id = $2) g`, x.cid, asg.ID, sub, tick); err != nil {
				r.t.Fatal(err)
			}
		}
	}
	reviewed(6)
	_, b = r.must(call{method: "POST", path: rf, token: x.gv, headers: x.idem()}, 202)
	var j1 struct {
		JobID string `json:"job_id"`
	}
	_ = json.Unmarshal(b, &j1)
	_, b = r.must(call{method: "POST", path: rf, token: x.gv, headers: x.idem()}, 202)
	var j2 struct {
		JobID string `json:"job_id"`
	}
	_ = json.Unmarshal(b, &j2)
	if j1.JobID == "" || j1.JobID != j2.JobID {
		r.t.Fatalf("hai lần rút cùng dữ liệu phải cùng job: %q ≠ %q", j1.JobID, j2.JobID)
	}
	for range 2 { // mỗi lần có bài duyệt mới là dữ liệu mới → việc mới; lần thứ 4 trong giờ bị trần
		reviewed(1)
		r.must(call{method: "POST", path: rf, token: x.gv, headers: x.idem()}, 202)
	}
	reviewed(1)
	r.must(call{method: "POST", path: rf, token: x.gv, headers: x.idem()}, 429)
	r.must(call{method: "GET", path: gd, token: x.gv}, 200)
}

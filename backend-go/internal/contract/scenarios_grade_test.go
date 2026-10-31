package contract

import (
	"context"
	"encoding/json"
	"strings"
)

// gradeScenarios: công thức điểm — thao tác POST …/grade-scheme/extract (US-P6-03, SRS FEAT-gradebook 6.1 #1) với mọi status đã khai báo.
// Tài liệu được chèn thẳng vào DB (đã READY) vì việc trích dùng đoạn đã lập chỉ mục; việc nền không chạy trong bộ hợp đồng (không worker).
func (r *runner) gradeScenarios(x examRig) {
	db := r.rig.deps.DB
	ctx := context.Background()
	base := "/api/v1/courses/" + x.cid
	ex := base + "/grade-scheme/extract"
	doc := func(typ, status string, chars int) string {
		var id string
		if err := db.QueryRow(ctx, `insert into documents (course_id, title, type, status) values ($1, 'Quy chế', $2::document_type, $3::document_status) returning id`, x.cid, typ, status).Scan(&id); err != nil {
			r.t.Fatalf("chèn tài liệu: %v", err)
		}
		if chars > 0 {
			if _, err := db.Exec(ctx, `insert into content_chunks (document_id, course_ids, ord, page_no, text) values ($1, array[$2::uuid], 0, 1, $3)`, id, x.cid, strings.Repeat("a", chars)); err != nil {
				r.t.Fatalf("chèn đoạn: %v", err)
			}
		}
		return id
	}
	body := func(id string, replace bool) string {
		raw, _ := json.Marshal(map[string]any{"document_id": id, "replace_draft": replace})
		return string(raw)
	}
	policy := doc("COURSE_POLICY", "READY", 100)

	// 202: việc nền được xếp hàng; 401 / 403 / 404 / 422 theo guard và kiểm đầu vào.
	_, raw := r.must(call{method: "POST", path: ex, token: x.gv, headers: x.idem(), body: body(policy, false)}, 202)
	var acc struct {
		JobID string `json:"job_id"`
	}
	_ = json.Unmarshal(raw, &acc)
	if acc.JobID == "" {
		r.t.Fatalf("202 không có job_id: %s", raw)
	}
	r.must(call{method: "POST", path: ex, headers: x.idem(), body: body(policy, false)}, 401)
	for _, who := range []string{x.ta, x.sv, x.admin} {
		r.must(call{method: "POST", path: ex, token: who, headers: x.idem(), body: body(policy, false)}, 403)
	}
	r.must(call{method: "POST", path: "/api/v1/courses/khong-phai-uuid/grade-scheme/extract", token: x.gv, headers: x.idem(), body: body(policy, false)}, 404)
	r.must(call{method: "POST", path: ex, token: x.gv, headers: x.idem(), body: body("00000000-0000-7000-8000-000000000000", false)}, 404)
	r.must(call{method: "POST", path: ex, token: x.gv, headers: x.idem(), body: `{}`}, 422)
	r.must(call{method: "POST", path: ex, token: x.gv, headers: x.idem(), body: body(doc("LECTURE", "READY", 100), false)}, 422)         // DOCUMENT_NOT_POLICY
	r.must(call{method: "POST", path: ex, token: x.gv, headers: x.idem(), body: body(doc("COURSE_POLICY", "READY", 60001), false)}, 422) // POLICY_TOO_LONG
	// 409: chưa READY; bản nháp đã sửa tay.
	r.must(call{method: "POST", path: ex, token: x.gv, headers: x.idem(), body: body(doc("COURSE_POLICY", "PROCESSING", 0), false)}, 409)
	var teacher string
	if err := db.QueryRow(ctx, `select user_id from enrollments where course_id = $1 and role_in_course = 'TEACHER'`, x.cid).Scan(&teacher); err != nil {
		r.t.Fatalf("giảng viên của lớp: %v", err)
	}
	if _, err := db.Exec(ctx, `insert into grade_schemes (course_id, scheme_no, created_by, version) values ($1, 1, $2, 2)`, x.cid, teacher); err != nil {
		r.t.Fatalf("chèn bản nháp: %v", err)
	}
	r.must(call{method: "POST", path: ex, token: x.gv, headers: x.idem(), body: body(policy, false)}, 409)
	r.must(call{method: "POST", path: ex, token: x.gv, headers: x.idem(), body: body(policy, true)}, 202) // replace_draft

	// GET …/grade-scheme/status (US-P6-05): Staff đọc; STUDENT / ADMIN 403; chưa đăng nhập 401; id lạ 404.
	st := base + "/grade-scheme/status"
	r.must(call{method: "GET", path: st, token: x.gv}, 200)
	r.must(call{method: "GET", path: st, token: x.ta}, 200)
	r.must(call{method: "GET", path: st}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "GET", path: st, token: who}, 403)
	}
	r.must(call{method: "GET", path: "/api/v1/courses/khong-phai-uuid/grade-scheme/status", token: x.gv}, 404)

	// Chưa có công thức đã xác nhận: xem trước và giải trình → 409 SCHEME_NOT_CONFIRMED (US-P6-06).
	var student string
	if err := db.QueryRow(ctx, `select user_id from enrollments where course_id = $1 and role_in_course = 'STUDENT' and status = 'ACTIVE' order by user_id limit 1`, x.cid).Scan(&student); err != nil {
		r.t.Fatalf("sinh viên của lớp: %v", err)
	}
	r.must(call{method: "GET", path: base + "/gradebook/preview", token: x.gv}, 409)
	r.must(call{method: "GET", path: base + "/gradebook/students/" + student + "/explain", token: x.gv}, 409)

	// US-P6-04: GET / PUT …/grade-scheme, POST …/impact, POST …/confirm.
	gs := base + "/grade-scheme"
	r.must(call{method: "GET", path: gs, token: x.gv}, 200)
	r.must(call{method: "GET", path: gs + "?diff=confirmed", token: x.ta}, 200)
	r.must(call{method: "GET", path: gs}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "GET", path: gs, token: who}, 403)
	}
	r.must(call{method: "GET", path: "/api/v1/courses/khong-phai-uuid/grade-scheme", token: x.gv}, 404)

	rules := `[{"kind":"BONUS","applicable":false},{"kind":"PENALTY","applicable":false},{"kind":"ROUNDING","params":{"step":"0.5","mode":"HALF_UP"}},{"kind":"LETTER_SCALE","params":{"bands":[{"min":"5","letter":"A"},{"min":"0","letter":"F"}]}}]`
	valid := `"components":[{"key":"weekly","name":"Hằng tuần","grp":"QT","weight":"40"},{"key":"final","name":"Cuối kỳ","grp":"CK","weight":"60"}],"rules":` + rules
	put := func(version int, comps string) call {
		return call{method: "PUT", path: gs, token: x.gv, body: `{"version":` + itoa(version) + `,` + comps + `}`}
	}
	r.must(put(2, valid), 200) // bản nháp chèn tay có version 2 → 3
	r.must(put(2, valid), 409) // version cũ
	r.must(put(3, `"components":[{"key":"A","name":"x","grp":"QT","weight":"abc"}],"rules":[]`), 422)
	r.must(call{method: "PUT", path: gs, body: `{"version":3,` + valid + `}`}, 401)
	for _, who := range []string{x.ta, x.sv, x.admin} {
		r.must(call{method: "PUT", path: gs, token: who, body: `{"version":3,` + valid + `}`}, 403)
	}
	r.must(call{method: "PUT", path: "/api/v1/courses/khong-phai-uuid/grade-scheme", token: x.gv, body: `{"version":3,` + valid + `}`}, 404)

	im := gs + "/impact"
	for _, who := range []string{x.gv, x.ta} {
		r.must(call{method: "POST", path: im, token: who, body: `{` + valid + `}`}, 200)
	}
	r.must(call{method: "POST", path: im, token: x.gv, body: `{"components":[{"key":"A","name":"x","grp":"QT","weight":"abc"}],"rules":[]}`}, 422)
	r.must(call{method: "POST", path: im, body: `{` + valid + `}`}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "POST", path: im, token: who, body: `{` + valid + `}`}, 403)
	}
	r.must(call{method: "POST", path: "/api/v1/courses/khong-phai-uuid/grade-scheme/impact", token: x.gv, body: `{` + valid + `}`}, 404)

	cf := gs + "/confirm"
	confirm := func(token string, version int) call {
		return call{method: "POST", path: cf, token: token, headers: x.idem(), body: `{"version":` + itoa(version) + `}`}
	}
	r.must(confirm("", 3), 401)
	for _, who := range []string{x.ta, x.sv, x.admin} {
		r.must(confirm(who, 3), 403)
	}
	r.must(call{method: "POST", path: "/api/v1/courses/khong-phai-uuid/grade-scheme/confirm", token: x.gv, headers: x.idem(), body: `{"version":3}`}, 404)
	r.must(call{method: "POST", path: cf, token: x.gv, headers: x.idem(), body: `{}`}, 422)
	r.must(confirm(x.gv, 2), 409) // version cũ
	r.must(put(3, `"components":[{"key":"weekly","name":"Hằng tuần","grp":"QT","weight":"20"}],"rules":[]`), 200)
	r.must(confirm(x.gv, 4), 422) // còn mục chưa rõ
	r.must(put(4, valid), 200)
	r.must(confirm(x.gv, 5), 200)
	r.must(confirm(x.gv, 6), 409) // không còn bản nháp

	// US-P6-06: bảng điểm, xem trước, sửa ô, công bố cột, giải trình, lịch sử (công thức đã CONFIRMED ở trên).
	var weekly, finalItem, exam string
	for k, dst := range map[string]*string{"weekly": &weekly, "final": &finalItem} {
		if err := db.QueryRow(ctx, `select i.id from grade_items i join grade_schemes s on s.id = i.scheme_id where i.course_id = $1 and s.status = 'CONFIRMED' and i.key = $2`, x.cid, k).Scan(dst); err != nil {
			r.t.Fatalf("cột điểm %s: %v", k, err)
		}
	}
	gb := base + "/gradebook"
	r.must(call{method: "GET", path: gb + "?limit=10", token: x.gv}, 200)
	r.must(call{method: "GET", path: gb, token: x.ta}, 200)
	r.must(call{method: "GET", path: gb}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "GET", path: gb, token: who}, 403)
	}
	r.must(call{method: "GET", path: "/api/v1/courses/khong-phai-uuid/gradebook", token: x.gv}, 404)
	r.must(call{method: "GET", path: gb + "?limit=0", token: x.gv}, 422)
	r.must(call{method: "GET", path: gb + "?cursor=%25", token: x.gv}, 422)

	pv := gb + "/preview"
	r.must(call{method: "GET", path: pv, token: x.gv}, 200)
	r.must(call{method: "GET", path: pv, token: x.ta}, 200)
	r.must(call{method: "GET", path: pv}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "GET", path: pv, token: who}, 403)
	}
	r.must(call{method: "GET", path: "/api/v1/courses/khong-phai-uuid/gradebook/preview", token: x.gv}, 404)
	r.must(call{method: "GET", path: pv + "?limit=0", token: x.gv}, 422)

	ge := base + "/grade-entries"
	entry := func(token, item, score string, version string) call {
		return call{method: "PUT", path: ge, token: token, body: `{"item_id":"` + item + `","student_id":"` + student + `","score":` + score + `,"version":` + version + `}`}
	}
	r.must(entry(x.gv, weekly, `"8,5"`, "null"), 200)
	r.must(entry(x.gv, weekly, `"9"`, "null"), 409) // ô đã có mà version null
	r.must(entry(x.gv, weekly, `"9"`, "1"), 200)
	r.must(entry(x.gv, weekly, `"11"`, "2"), 422)    // ngoài thang
	r.must(entry(x.gv, weekly, `"8,333"`, "2"), 422) // quá 2 chữ số thập phân
	r.must(entry(x.gv, weekly, `"abc"`, "2"), 422)
	r.must(entry(x.gv, "00000000-0000-7000-8000-00000000dead", `"5"`, "null"), 404)
	r.must(call{method: "PUT", path: ge, token: x.gv, body: `{}`}, 422)
	r.must(entry(x.ta, weekly, `"5"`, "2"), 403)
	r.must(entry(x.sv, weekly, `"5"`, "2"), 403)
	r.must(entry(x.admin, weekly, `"5"`, "2"), 403)
	r.must(entry("", weekly, `"5"`, "2"), 401)
	r.must(call{method: "PUT", path: "/api/v1/courses/khong-phai-uuid/grade-entries", token: x.gv, body: `{"item_id":"` + weekly + `","student_id":"` + student + `","score":"5","version":2}`}, 404)
	r.must(entry(x.gv, weekly, `null`, "3"), 409) // version cũ: ô đã ở version 2
	r.must(entry(x.gv, finalItem, `"7"`, "null"), 200)
	r.must(entry(x.gv, finalItem, `null`, "1"), 200) // xoá ô

	var teacherID string
	if err := db.QueryRow(ctx, `select user_id from enrollments where course_id = $1 and role_in_course = 'TEACHER'`, x.cid).Scan(&teacherID); err != nil {
		r.t.Fatalf("giảng viên của lớp: %v", err)
	}
	if err := db.QueryRow(ctx, `insert into exams (course_id, title, created_by, status, max_score, opens_at, closes_at, duration_minutes) values ($1, 'Thi giữa kỳ', $2, 'SCHEDULED', 10, now() + interval '1 day', now() + interval '2 days', 45) returning id`, x.cid, teacherID).Scan(&exam); err != nil {
		r.t.Fatalf("chèn bài thi: %v", err)
	}
	if _, err := db.Exec(ctx, `update grade_items set source = 'EXAM', exam_id = $2 where id = $1`, finalItem, exam); err != nil {
		r.t.Fatalf("đổi nguồn cột: %v", err)
	}
	r.must(entry(x.gv, finalItem, `"7"`, "null"), 409) // ITEM_NOT_EDITABLE

	pub := base + "/grade-items/" + weekly + "/publish"
	r.must(call{method: "POST", path: pub, token: x.gv, body: `{"published":true}`}, 200)
	r.must(call{method: "POST", path: pub, token: x.gv, body: `{"published":true}`}, 200) // idempotent
	r.must(call{method: "POST", path: pub, token: x.gv, body: `{}`}, 422)
	r.must(call{method: "POST", path: pub, token: x.ta, body: `{"published":true}`}, 403)
	r.must(call{method: "POST", path: pub, token: x.sv, body: `{"published":true}`}, 403)
	r.must(call{method: "POST", path: pub, body: `{"published":true}`}, 401)
	r.must(call{method: "POST", path: base + "/grade-items/" + finalItem + "/publish", token: x.gv, body: `{"published":true}`}, 409) // ITEM_NOT_PUBLISHABLE
	r.must(call{method: "POST", path: base + "/grade-items/khong-phai-uuid/publish", token: x.gv, body: `{"published":true}`}, 404)

	ex1 := gb + "/students/" + student + "/explain"
	r.must(call{method: "GET", path: ex1, token: x.gv}, 200)
	r.must(call{method: "GET", path: ex1, token: x.ta}, 200)
	r.must(call{method: "GET", path: ex1}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "GET", path: ex1, token: who}, 403)
	}
	r.must(call{method: "GET", path: gb + "/students/00000000-0000-7000-8000-00000000dead/explain", token: x.gv}, 404)
	r.must(call{method: "GET", path: gb + "/students/khong-phai-uuid/explain", token: x.gv}, 404)

	hi := gb + "/students/" + student + "/history"
	r.must(call{method: "GET", path: hi, token: x.gv}, 200)
	r.must(call{method: "GET", path: hi, token: x.ta}, 200)
	r.must(call{method: "GET", path: hi}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "GET", path: hi, token: who}, 403)
	}
	r.must(call{method: "GET", path: hi + "?cursor=%25", token: x.gv}, 422)
	r.must(call{method: "GET", path: gb + "/students/khong-phai-uuid/history", token: x.gv}, 404)

	// sổ đã chốt: sửa ô, công bố cột → 409 GRADEBOOK_LOCKED
	if _, err := db.Exec(ctx, `update gradebooks set status = 'LOCKED', snapshot_version = 1 where course_id = $1`, x.cid); err != nil {
		r.t.Fatalf("khoá sổ: %v", err)
	}
	r.must(entry(x.gv, weekly, `"5"`, "2"), 409)
	r.must(call{method: "POST", path: pub, token: x.gv, body: `{"published":false}`}, 409)
	if _, err := db.Exec(ctx, `update gradebooks set status = 'OPEN', snapshot_version = 0 where course_id = $1`, x.cid); err != nil {
		r.t.Fatalf("mở sổ: %v", err)
	}
}

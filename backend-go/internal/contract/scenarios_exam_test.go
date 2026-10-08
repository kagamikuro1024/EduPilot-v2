package contract

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"math/rand/v2"
	"mime/multipart"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
)

// zipBody dựng thân multipart (trường `file`) chứa một zip các tệp name → nội dung.
func zipBody(files map[string]string, pad int) (body, ctype string) {
	var zb bytes.Buffer
	zw := zip.NewWriter(&zb)
	for n, c := range files {
		w, _ := zw.Create(n)
		_, _ = w.Write([]byte(c))
	}
	if pad > 0 { // tệp không nén được nên zip phình thật
		w, _ := zw.Create("pad.out")
		b := make([]byte, pad)
		_, _ = rand.NewChaCha8([32]byte{7}).Read(b)
		_, _ = w.Write(b)
	}
	_ = zw.Close()
	var mb bytes.Buffer
	mw := multipart.NewWriter(&mb)
	fw, _ := mw.CreateFormFile("file", "tests.zip")
	_, _ = fw.Write(zb.Bytes())
	_ = mw.Close()
	return mb.String(), mw.FormDataContentType()
}

// examScenarios: thao tác 1–16 của SRS FEAT-weekly-exam 6.2 (ngân hàng câu hỏi, US-PE-03) với mọi status đã khai báo.
func (r *runner) examScenarios() {
	r.freshIP()
	db := r.rig.deps.DB
	mkUser := func(role string) uuid.UUID {
		var id uuid.UUID
		if err := db.QueryRow(context.Background(), `insert into users (email, full_name, role, status, password_hash, email_verified_at) values ($1, 'Người Thử', $2::user_role, 'ACTIVE', 'x', now()) returning id`,
			"ct-ex-"+uuid.NewString()[:8]+"@example.test", role).Scan(&id); err != nil {
			r.t.Fatalf("tạo người dùng: %v", err)
		}
		return id
	}
	adminID, gvID, taID, svID := mkUser("ADMIN"), mkUser("TEACHER"), mkUser("TA"), mkUser("STUDENT")
	admin, gv := r.rig.token(r.t, adminID.String(), auth.RoleAdmin), r.rig.token(r.t, gvID.String(), auth.RoleTeacher)
	ta, sv := r.rig.token(r.t, taID.String(), auth.RoleTA), r.rig.token(r.t, svID.String(), auth.RoleStudent)
	idem := func() map[string]string { return map[string]string{"Idempotency-Key": "ct-" + uuid.NewString()} }
	newCourse := func() string {
		body := `{"subject_code":"INT1006","class_code":"` + "EX-" + uuid.NewString()[:8] + `","name":"An ninh mạng","semester":"2026-2027-HK1","teacher_id":"` + gvID.String() + `","ta_ids":["` + taID.String() + `"]}`
		_, b := r.must(call{method: "POST", path: "/api/v1/admin/courses", token: admin, headers: idem(), body: body}, 201)
		var c struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(b, &c)
		if _, err := db.Exec(context.Background(), `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN')`, c.ID, svID); err != nil {
			r.t.Fatal(err)
		}
		return c.ID
	}
	cid, arch := newCourse(), newCourse()
	base := func(c string) string { return "/api/v1/courses/" + c + "/questions" }
	type created struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
	}
	mk := func(course, body string) created {
		_, b := r.must(call{method: "POST", path: base(course), token: gv, body: body}, 201)
		var c created
		_ = json.Unmarshal(b, &c)
		return c
	}
	const mcq = `{"type":"MCQ_SINGLE","title":"2+2","topic":"Số học","difficulty":"EASY","stem":"2+2=?","options":[{"body":"3"},{"body":"4"},{"body":"Tất cả các đáp án trên"}],"correct":[1]}`
	const codeQ = `{"type":"CODE","title":"a+b","topic":"Cơ bản","stem":"Đọc a b, in a+b."}`

	// 1, 2: danh sách và tạo.
	q1 := mk(cid, mcq)
	_, b := r.must(call{method: "POST", path: base(cid), token: gv, headers: idem(), body: `{"type":"TRUE_FALSE","title":"Đúng sai","topic":"Số học","stem":"1<2","value":true}`}, 201)
	_ = b
	r.must(call{method: "POST", path: base(cid), headers: idem(), body: mcq}, 401)
	r.must(call{method: "POST", path: base(cid), token: sv, body: mcq}, 403)
	r.must(call{method: "POST", path: base(cid), token: admin, body: mcq}, 403)
	r.must(call{method: "POST", path: base("khong-phai-uuid"), token: gv, body: mcq}, 404)
	r.must(call{method: "POST", path: base(cid), token: gv, body: `{"type":"MCQ_SINGLE","title":"x","topic":"y","stem":"z","options":[{"body":"a"},{"body":"b"}],"correct":[0,1]}`}, 422)
	r.must(call{method: "POST", path: base(cid), token: gv, body: `{"type":"SHORT","title":"x","topic":"y","stem":"z"}`}, 422)
	qa := mk(arch, mcq)
	ca := mk(arch, codeQ)
	r.must(call{method: "POST", path: base(arch) + "/" + ca.ID + "/testcases", token: gv, body: `{"name":"t1","input":"1 2","expected":"3","is_sample":true}`}, 201)
	var tcA struct {
		ID string `json:"id"`
	}
	_, tb := r.must(call{method: "POST", path: base(arch) + "/" + ca.ID + "/testcases", token: gv, body: `{"name":"t2","input":"2 2","expected":"4"}`}, 201)
	_ = json.Unmarshal(tb, &tcA)
	r.must(call{method: "POST", path: "/api/v1/admin/courses/" + arch + "/archive", token: admin}, 200)

	r.must(call{method: "POST", path: base(arch), token: gv, body: mcq}, 409) // lớp đã lưu trữ

	lst := base(cid)
	r.must(call{method: "GET", path: lst, token: gv}, 200)
	r.must(call{method: "GET", path: lst + "?review_status=DRAFT&type=MCQ_SINGLE&q=2%2B2&limit=1", token: ta}, 200)
	r.must(call{method: "GET", path: lst}, 401)
	r.must(call{method: "GET", path: lst, token: sv}, 403)
	r.must(call{method: "GET", path: lst, token: admin}, 403)
	r.must(call{method: "GET", path: base("khong-phai-uuid"), token: gv}, 404)
	r.must(call{method: "GET", path: lst + "?limit=0", token: gv}, 422)

	// 3, 4: chi tiết và sửa (khoá lạc quan).
	one := lst + "/" + q1.ID
	r.must(call{method: "GET", path: one, token: gv}, 200)
	r.must(call{method: "GET", path: one}, 401)
	r.must(call{method: "GET", path: one, token: sv}, 403)
	r.must(call{method: "GET", path: lst + "/" + uuid.NewString(), token: gv}, 404)
	upd := `{"type":"MCQ_SINGLE","title":"2+2 (sửa)","topic":"Số học","difficulty":"EASY","stem":"2+2=?","options":[{"body":"3"},{"body":"4"},{"body":"5"}],"correct":[1],"version":1}`
	r.must(call{method: "PUT", path: one, token: gv, body: upd}, 200)
	r.must(call{method: "PUT", path: one, token: gv, body: upd}, 409)
	r.must(call{method: "PUT", path: one, body: upd}, 401)
	r.must(call{method: "PUT", path: one, token: sv, body: upd}, 403)
	r.must(call{method: "PUT", path: lst + "/" + uuid.NewString(), token: gv, body: upd}, 404)
	r.must(call{method: "PUT", path: one, token: gv, body: `{"type":"MCQ_SINGLE","title":"x","topic":"y","stem":"z","options":[{"body":"a"},{"body":"b"}],"correct":[],"version":2}`}, 422)

	// 5, 6: lưu trữ và nhân bản.
	spare := mk(cid, mcq)
	r.must(call{method: "POST", path: lst + "/" + spare.ID + "/duplicate", token: ta}, 201)
	r.must(call{method: "POST", path: lst + "/" + spare.ID + "/duplicate"}, 401)
	r.must(call{method: "POST", path: lst + "/" + spare.ID + "/duplicate", token: sv}, 403)
	r.must(call{method: "POST", path: lst + "/" + uuid.NewString() + "/duplicate", token: gv}, 404)
	r.must(call{method: "POST", path: base(arch) + "/" + qa.ID + "/duplicate", token: gv}, 409)
	r.must(call{method: "POST", path: lst + "/" + spare.ID + "/archive", token: gv}, 200)
	r.must(call{method: "POST", path: lst + "/" + spare.ID + "/archive"}, 401)
	r.must(call{method: "POST", path: lst + "/" + spare.ID + "/archive", token: sv}, 403)
	r.must(call{method: "POST", path: lst + "/" + uuid.NewString() + "/archive", token: gv}, 404)
	r.must(call{method: "POST", path: base(arch) + "/" + qa.ID + "/archive", token: gv}, 409)

	// 7: duyệt.
	rv := func(q string) string { return lst + "/" + q + "/review" }
	_, gb := r.must(call{method: "GET", path: one, token: gv}, 200)
	var cur created
	_ = json.Unmarshal(gb, &cur)
	r.must(call{method: "PUT", path: rv(q1.ID), token: gv, body: `{"decision":"REQUEST","version":` + itoa(cur.Version) + `}`}, 200)
	r.must(call{method: "PUT", path: rv(q1.ID), token: gv, body: `{"decision":"REQUEST","version":` + itoa(cur.Version+1) + `}`}, 409) // PENDING không gửi duyệt lại
	r.must(call{method: "PUT", path: rv(q1.ID), body: `{"decision":"APPROVE","version":1}`}, 401)
	r.must(call{method: "PUT", path: rv(q1.ID), token: sv, body: `{"decision":"APPROVE","version":1}`}, 403)
	r.must(call{method: "PUT", path: rv(uuid.NewString()), token: gv, body: `{"decision":"APPROVE","version":1}`}, 404)
	r.must(call{method: "PUT", path: rv(q1.ID), token: ta, body: `{"decision":"BAN","version":` + itoa(cur.Version+1) + `}`}, 422)
	r.must(call{method: "PUT", path: rv(q1.ID), token: ta, body: `{"decision":"APPROVE","version":` + itoa(cur.Version+1) + `}`}, 200)

	// 8: cấu hình bài code.
	cq := mk(cid, codeQ)
	code := lst + "/" + cq.ID + "/code"
	cfg := `{"languages":["cpp17"],"time_limit_ms":2000,"memory_limit_mb":128,"checker":"EXACT","starter_code":{"cpp17":"int main(){}"},"reference":{"language":"cpp17","source":"int main(){}"},"version":` + itoa(cq.Version) + `}`
	r.must(call{method: "PUT", path: code, token: gv, body: cfg}, 200)
	r.must(call{method: "PUT", path: code, token: gv, body: cfg}, 409)
	r.must(call{method: "PUT", path: code, body: cfg}, 401)
	r.must(call{method: "PUT", path: code, token: sv, body: cfg}, 403)
	r.must(call{method: "PUT", path: lst + "/" + q1.ID + "/code", token: gv, body: cfg}, 404) // câu không phải CODE
	r.must(call{method: "PUT", path: base(arch) + "/" + ca.ID + "/code", token: gv, body: cfg}, 409)
	_, cb := r.must(call{method: "GET", path: lst + "/" + cq.ID, token: gv}, 200)
	var cc created
	_ = json.Unmarshal(cb, &cc)
	r.must(call{method: "PUT", path: code, token: gv, body: `{"languages":[],"version":` + itoa(cc.Version) + `}`}, 422)

	// 9–12, 14: test.
	tcs := lst + "/" + cq.ID + "/testcases"
	r.must(call{method: "GET", path: tcs, token: gv}, 200)
	r.must(call{method: "GET", path: tcs}, 401)
	r.must(call{method: "GET", path: tcs, token: sv}, 403)
	r.must(call{method: "GET", path: lst + "/" + q1.ID + "/testcases", token: gv}, 404)
	r.must(call{method: "GET", path: tcs + "?limit=0", token: gv}, 422)
	_, tb1 := r.must(call{method: "POST", path: tcs, token: gv, body: `{"name":"sample1","input":"1 2","expected":"3","is_sample":true}`}, 201)
	var t1 struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(tb1, &t1)
	r.must(call{method: "POST", path: tcs, token: ta, body: `{"name":"hidden1","input":"5 5","expected":"10","weight":2}`}, 201)
	r.must(call{method: "POST", path: tcs, body: `{"name":"x","input":"1","expected":"1"}`}, 401)
	r.must(call{method: "POST", path: tcs, token: sv, body: `{"name":"x","input":"1","expected":"1"}`}, 403)
	r.must(call{method: "POST", path: lst + "/" + q1.ID + "/testcases", token: gv, body: `{"name":"x","input":"1","expected":"1"}`}, 404)
	r.must(call{method: "POST", path: base(arch) + "/" + ca.ID + "/testcases", token: gv, body: `{"name":"x","input":"1","expected":"1"}`}, 409)
	r.must(call{method: "POST", path: tcs, token: gv, body: `{"name":"tên có dấu cách","input":"1","expected":"1"}`}, 422)
	r.must(call{method: "GET", path: tcs + "?limit=1", token: gv}, 200)
	tc := tcs + "/" + t1.ID
	r.must(call{method: "PUT", path: tc, token: gv, body: `{"weight":3,"position":2}`}, 200)
	r.must(call{method: "PUT", path: tc, body: `{"weight":3}`}, 401)
	r.must(call{method: "PUT", path: tc, token: sv, body: `{"weight":3}`}, 403)
	r.must(call{method: "PUT", path: tcs + "/" + uuid.NewString(), token: gv, body: `{"weight":3}`}, 404)
	r.must(call{method: "PUT", path: base(arch) + "/" + ca.ID + "/testcases/" + tcA.ID, token: gv, body: `{"weight":3}`}, 409)
	r.must(call{method: "PUT", path: tc, token: gv, body: `{"weight":5000}`}, 422)
	r.must(call{method: "DELETE", path: tc}, 401)
	r.must(call{method: "DELETE", path: tc, token: sv}, 403)
	r.must(call{method: "DELETE", path: tcs + "/" + uuid.NewString(), token: gv}, 404)
	r.must(call{method: "DELETE", path: base(arch) + "/" + ca.ID + "/testcases/" + tcA.ID, token: gv}, 409)
	r.must(call{method: "DELETE", path: tc, token: gv}, 204)
	ap := tcs + "/approve"
	r.must(call{method: "POST", path: ap, token: gv, body: `{"ids":["` + uuid.NewString() + `"]}`}, 200)
	r.must(call{method: "POST", path: ap, body: `{"ids":["` + uuid.NewString() + `"]}`}, 401)
	r.must(call{method: "POST", path: ap, token: sv, body: `{"ids":["` + uuid.NewString() + `"]}`}, 403)
	r.must(call{method: "POST", path: lst + "/" + q1.ID + "/testcases/approve", token: gv, body: `{"ids":["` + uuid.NewString() + `"]}`}, 404)
	r.must(call{method: "POST", path: base(arch) + "/" + ca.ID + "/testcases/approve", token: gv, body: `{"ids":["` + uuid.NewString() + `"]}`}, 409)
	r.must(call{method: "POST", path: ap, token: gv, body: `{"ids":[]}`}, 422)

	// 13: nhập zip.
	imp := tcs + "/import"
	good, ct := zipBody(map[string]string{"sample1.in": "1 2\n", "sample1.out": "3\n", "t2.in": "2 2\n", "t2.out": "4\n", "t10.in": "3 3\n", "t10.out": "6\n"}, 0)
	r.must(call{method: "POST", path: imp + "?dry_run=true", token: gv, body: good, ctype: ct}, 200)
	r.must(call{method: "POST", path: imp, token: gv, body: good, ctype: ct}, 201)
	r.must(call{method: "POST", path: imp + "?mode=replace", token: ta, body: good, ctype: ct}, 201)
	r.must(call{method: "POST", path: imp, body: good, ctype: ct}, 401)
	r.must(call{method: "POST", path: imp, token: sv, body: good, ctype: ct}, 403)
	r.must(call{method: "POST", path: lst + "/" + q1.ID + "/testcases/import", token: gv, body: good, ctype: ct}, 404)
	r.must(call{method: "POST", path: base(arch) + "/" + ca.ID + "/testcases/import", token: gv, body: good, ctype: ct}, 409)
	big, bct := zipBody(map[string]string{"t.in": "1", "t.out": "1"}, 90<<10)
	r.must(call{method: "POST", path: imp, token: gv, body: big, ctype: bct}, 413)
	bad, bdct := zipBody(map[string]string{"t.in": "1"}, 0) // thiếu nửa cặp
	r.must(call{method: "POST", path: imp, token: gv, body: bad, ctype: bdct}, 422)
	r.must(call{method: "POST", path: imp + "?mode=bad", token: gv, body: good, ctype: ct}, 422)

	// 15, 16: việc nền (202 + job id).
	vf := lst + "/" + cq.ID + "/reference/verify"
	r.must(call{method: "POST", path: vf, token: gv, headers: idem()}, 202)
	r.must(call{method: "POST", path: vf, token: gv}, 422) // thiếu Idempotency-Key
	r.must(call{method: "POST", path: vf, headers: idem()}, 401)
	r.must(call{method: "POST", path: vf, token: sv, headers: idem()}, 403)
	r.must(call{method: "POST", path: lst + "/" + q1.ID + "/reference/verify", token: gv, headers: idem()}, 404)
	r.must(call{method: "POST", path: base(arch) + "/" + ca.ID + "/reference/verify", token: gv, headers: idem()}, 409)
	sg := lst + "/suggest"
	r.must(call{method: "POST", path: sg, token: gv, headers: idem(), body: `{"kind":"MCQ","topic":"Mật mã","difficulty":"MEDIUM","count":3}`}, 202)
	r.must(call{method: "POST", path: sg, headers: idem(), body: `{"kind":"MCQ","topic":"x","count":1}`}, 401)
	r.must(call{method: "POST", path: sg, token: sv, headers: idem(), body: `{"kind":"MCQ","topic":"x","count":1}`}, 403)
	r.must(call{method: "POST", path: base("khong-phai-uuid") + "/suggest", token: gv, headers: idem(), body: `{"kind":"MCQ","topic":"x","count":1}`}, 404)
	r.must(call{method: "POST", path: base(arch) + "/suggest", token: gv, headers: idem(), body: `{"kind":"MCQ","topic":"x","count":1}`}, 409)
	r.must(call{method: "POST", path: sg, token: gv, headers: idem(), body: `{"kind":"MCQ","topic":"x","count":99}`}, 422)
}

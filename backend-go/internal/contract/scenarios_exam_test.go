package contract

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"math/rand/v2"
	"mime/multipart"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

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

	r.examExamScenarios(examRig{cid: cid, arch: arch, q1: q1.ID, gv: gv, ta: ta, sv: sv, admin: admin, idem: idem})
	r.examAttemptScenarios(examRig{cid: cid, arch: arch, q1: q1.ID, gv: gv, ta: ta, sv: sv, admin: admin, idem: idem})
	r.examCodeScenarios(examRig{cid: cid, arch: arch, q1: q1.ID, gv: gv, ta: ta, sv: sv, admin: admin, idem: idem}, cq.ID)
	r.examResultScenarios(examRig{cid: cid, arch: arch, q1: q1.ID, gv: gv, ta: ta, sv: sv, admin: admin, idem: idem})
	r.documentScenarios(examRig{cid: cid, arch: arch, q1: q1.ID, gv: gv, ta: ta, sv: sv, admin: admin, idem: idem})
}

type examRig struct {
	cid, arch, q1, gv, ta, sv, admin string
	idem                             func() map[string]string
}

// examExamScenarios: thao tác 17–25, 27, 28 (bài thi, US-PE-04) với mọi status đã khai báo. q1 là một câu MCQ đã duyệt của lớp `cid`; `arch` là lớp đã lưu trữ.
func (r *runner) examExamScenarios(x examRig) {
	ex := "/api/v1/courses/" + x.cid + "/exams"
	archEx := "/api/v1/courses/" + x.arch + "/exams"
	now := time.Now().UTC()
	opens, closes := now.Add(2*time.Hour).Format(time.RFC3339), now.Add(4*time.Hour).Format(time.RFC3339)
	create := `{"title":"Kiểm tra tuần 9","opens_at":"` + opens + `","closes_at":"` + closes + `","duration_minutes":45}`
	type exm struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
	}
	parse := func(b []byte) exm {
		var e exm
		_ = json.Unmarshal(b, &e)
		return e
	}

	// 18: tạo.
	_, b := r.must(call{method: "POST", path: ex, token: x.gv, headers: x.idem(), body: create}, 201)
	e := parse(b)
	r.must(call{method: "POST", path: ex, headers: x.idem(), body: create}, 401)
	r.must(call{method: "POST", path: ex, token: x.sv, headers: x.idem(), body: create}, 403)
	r.must(call{method: "POST", path: ex, token: x.admin, headers: x.idem(), body: create}, 403)
	r.must(call{method: "POST", path: ex, token: x.gv, body: create}, 422) // thiếu Idempotency-Key
	r.must(call{method: "POST", path: ex, token: x.gv, headers: x.idem(), body: `{"title":"","duration_minutes":1}`}, 422)
	r.must(call{method: "POST", path: ex, token: x.gv, headers: x.idem(), body: `{"title":"kind do client","kind":"CODE"}`}, 422) // `kind` suy ra từ mục, không nhận từ client
	r.must(call{method: "POST", path: "/api/v1/courses/khong-phai-uuid/exams", token: x.gv, headers: x.idem(), body: create}, 404)
	r.must(call{method: "POST", path: archEx, token: x.gv, headers: x.idem(), body: create}, 409)
	_, b = r.must(call{method: "POST", path: ex, token: x.ta, headers: x.idem(), body: `{"title":"Bản nháp của TA"}`}, 201) // TA tạo / sửa nháp được
	spare := parse(b)

	// 17: danh sách.
	r.must(call{method: "GET", path: ex, token: x.gv}, 200)
	r.must(call{method: "GET", path: ex + "?status=DRAFT&limit=1", token: x.ta}, 200)
	r.must(call{method: "GET", path: ex, token: x.sv}, 200) // sinh viên: bản giới hạn, không DRAFT
	r.must(call{method: "GET", path: ex}, 401)
	r.must(call{method: "GET", path: ex, token: x.admin}, 403)
	r.must(call{method: "GET", path: "/api/v1/courses/khong-phai-uuid/exams", token: x.gv}, 404)
	r.must(call{method: "GET", path: ex + "?status=BAN", token: x.gv}, 422)

	// 19: chi tiết.
	one := ex + "/" + e.ID
	r.must(call{method: "GET", path: one, token: x.gv}, 200)
	r.must(call{method: "GET", path: one}, 401)
	r.must(call{method: "GET", path: one, token: x.admin}, 403)
	r.must(call{method: "GET", path: one, token: x.sv}, 404) // DRAFT: sinh viên không thấy
	r.must(call{method: "GET", path: ex + "/" + uuid.NewString(), token: x.gv}, 404)

	// 20: sửa (khoá lạc quan).
	upd := `{"title":"Kiểm tra tuần 9 (sửa)","version":` + itoa(e.Version) + `}`
	_, b = r.must(call{method: "PUT", path: one, token: x.gv, body: upd}, 200)
	e = parse(b)
	r.must(call{method: "PUT", path: one, token: x.gv, body: upd}, 409)
	r.must(call{method: "PUT", path: one, body: upd}, 401)
	r.must(call{method: "PUT", path: one, token: x.sv, body: upd}, 403)
	r.must(call{method: "PUT", path: ex + "/" + uuid.NewString(), token: x.gv, body: upd}, 404)
	r.must(call{method: "PUT", path: archEx + "/" + uuid.NewString(), token: x.gv, body: upd}, 409)
	r.must(call{method: "PUT", path: one, token: x.gv, body: `{"duration_minutes":1,"version":` + itoa(e.Version) + `}`}, 422)

	// 21: mục của bài.
	it := one + "/items"
	r.must(call{method: "PUT", path: it, token: x.gv, body: `{"items":[],"version":` + itoa(e.Version) + `}`}, 422)
	_, b = r.must(call{method: "PUT", path: it, token: x.gv, body: `{"items":[{"question_id":"` + x.q1 + `","points":"2.50"}],"version":` + itoa(e.Version) + `}`}, 200)
	stale := e.Version
	e = parse(b)
	r.must(call{method: "PUT", path: it, token: x.gv, body: `{"items":[{"question_id":"` + x.q1 + `","points":"2.50"}],"version":` + itoa(stale) + `}`}, 409)
	r.must(call{method: "PUT", path: it, body: `{"items":[],"version":1}`}, 401)
	r.must(call{method: "PUT", path: it, token: x.sv, body: `{"items":[],"version":1}`}, 403)
	r.must(call{method: "PUT", path: ex + "/" + uuid.NewString() + "/items", token: x.gv, body: `{"items":[],"version":1}`}, 404)
	r.must(call{method: "PUT", path: archEx + "/" + uuid.NewString() + "/items", token: x.gv, body: `{"items":[],"version":1}`}, 409)

	// 22: xem trước.
	r.must(call{method: "GET", path: one + "/preview", token: x.gv}, 200)
	r.must(call{method: "GET", path: one + "/preview", token: x.ta}, 200)
	r.must(call{method: "GET", path: one + "/preview"}, 401)
	r.must(call{method: "GET", path: one + "/preview", token: x.sv}, 403)
	r.must(call{method: "GET", path: ex + "/" + uuid.NewString() + "/preview", token: x.gv}, 404)

	// 28: nhân bản (bản sao không có giờ, không có mục → lên lịch lỗi 422 đủ danh sách).
	_, b = r.must(call{method: "POST", path: one + "/clone", token: x.ta}, 201)
	clone := parse(b)
	r.must(call{method: "POST", path: one + "/clone"}, 401)
	r.must(call{method: "POST", path: one + "/clone", token: x.sv}, 403)
	r.must(call{method: "POST", path: ex + "/" + uuid.NewString() + "/clone", token: x.gv}, 404)
	r.must(call{method: "POST", path: archEx + "/" + uuid.NewString() + "/clone", token: x.gv}, 409)

	// 23: lên lịch (chỉ Giảng viên).
	sch := one + "/schedule"
	r.must(call{method: "POST", path: ex + "/" + spare.ID + "/schedule", token: x.gv}, 422) // nháp của TA: không giờ, không mục
	r.must(call{method: "POST", path: sch, token: x.ta}, 403)
	r.must(call{method: "POST", path: sch, token: x.sv}, 403)
	r.must(call{method: "POST", path: sch}, 401)
	r.must(call{method: "POST", path: ex + "/" + uuid.NewString() + "/schedule", token: x.gv}, 404)
	r.must(call{method: "POST", path: archEx + "/" + uuid.NewString() + "/schedule", token: x.gv}, 409)
	r.must(call{method: "POST", path: sch, token: x.gv}, 200)
	r.must(call{method: "POST", path: sch, token: x.gv}, 200) // gọi lại: idempotent
	r.must(call{method: "GET", path: one, token: x.sv}, 200)  // đã lên lịch: sinh viên thấy bản giới hạn
	r.must(call{method: "GET", path: ex, token: x.sv}, 200)
	r.must(call{method: "DELETE", path: one, token: x.gv}, 409) // không phải DRAFT
	r.must(call{method: "PUT", path: it, token: x.gv, body: `{"items":[{"question_id":"` + x.q1 + `","points":"1"}],"version":` + itoa(e.Version+1) + `}`}, 409)

	// 25: gia hạn.
	ext := one + "/extend"
	r.must(call{method: "POST", path: ext, token: x.gv, body: `{"closes_at":"` + now.Add(5*time.Hour).Format(time.RFC3339) + `"}`}, 200)
	r.must(call{method: "POST", path: ext, token: x.gv, body: `{"closes_at":"` + now.Add(time.Hour).Format(time.RFC3339) + `"}`}, 422)
	r.must(call{method: "POST", path: ext, body: `{"closes_at":"` + closes + `"}`}, 401)
	r.must(call{method: "POST", path: ext, token: x.ta, body: `{"closes_at":"` + closes + `"}`}, 403)
	r.must(call{method: "POST", path: ex + "/" + uuid.NewString() + "/extend", token: x.gv, body: `{"closes_at":"` + closes + `"}`}, 404)
	r.must(call{method: "POST", path: ex + "/" + clone.ID + "/extend", token: x.gv, body: `{"closes_at":"` + closes + `"}`}, 409) // nháp
	r.must(call{method: "POST", path: archEx + "/" + uuid.NewString() + "/extend", token: x.gv, body: `{"closes_at":"` + closes + `"}`}, 409)

	// 24: bỏ lịch.
	un := one + "/unschedule"
	r.must(call{method: "POST", path: un, token: x.gv}, 200)
	r.must(call{method: "POST", path: un, token: x.gv}, 409) // đã về DRAFT
	r.must(call{method: "POST", path: un}, 401)
	r.must(call{method: "POST", path: un, token: x.ta}, 403)
	r.must(call{method: "POST", path: ex + "/" + uuid.NewString() + "/unschedule", token: x.gv}, 404)

	// 27: xoá (chỉ nháp, chỉ Giảng viên).
	r.must(call{method: "DELETE", path: ex + "/" + clone.ID}, 401)
	r.must(call{method: "DELETE", path: ex + "/" + clone.ID, token: x.ta}, 403)
	r.must(call{method: "DELETE", path: ex + "/" + uuid.NewString(), token: x.gv}, 404)
	r.must(call{method: "DELETE", path: archEx + "/" + uuid.NewString(), token: x.gv}, 409)
	r.must(call{method: "DELETE", path: ex + "/" + clone.ID, token: x.gv}, 204)
}

// examAttemptScenarios: thao tác 29, 30, 31, 39, 40, 41 (lượt làm của sinh viên, US-PE-05) với mọi status đã khai báo.
func (r *runner) examAttemptScenarios(x examRig) {
	db := r.rig.deps.DB
	ex := "/api/v1/courses/" + x.cid + "/exams"
	archEx := "/api/v1/courses/" + x.arch + "/exams"
	now := time.Now().UTC()
	create := `{"title":"Bài làm","opens_at":"` + now.Add(2*time.Hour).Format(time.RFC3339) + `","closes_at":"` + now.Add(4*time.Hour).Format(time.RFC3339) + `","duration_minutes":45}`
	_, b := r.must(call{method: "POST", path: ex, token: x.gv, headers: x.idem(), body: create}, 201)
	var e struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
	}
	_ = json.Unmarshal(b, &e)
	r.must(call{method: "PUT", path: ex + "/" + e.ID + "/items", token: x.gv, body: `{"items":[{"question_id":"` + x.q1 + `","points":"2"}],"version":` + itoa(e.Version) + `}`}, 200)
	r.must(call{method: "POST", path: ex + "/" + e.ID + "/schedule", token: x.gv}, 200)
	if _, err := db.Exec(context.Background(), `update exams set status='OPEN', opens_at=now() - interval '1 minute', closes_at=now() + interval '3 hours' where id=$1`, e.ID); err != nil {
		r.t.Fatal(err)
	}
	one := ex + "/" + e.ID
	tabA, tabB := uuid.NewString(), uuid.NewString()
	hdr := func(tab string, idem bool) map[string]string {
		h := map[string]string{"X-Exam-Tab": tab}
		if idem {
			h["Idempotency-Key"] = "ct-" + uuid.NewString()
		}
		return h
	}

	// 29: bắt đầu.
	r.must(call{method: "POST", path: one + "/attempts", token: x.sv, headers: map[string]string{"X-Exam-Tab": tabA}}, 422) // thiếu Idempotency-Key
	r.must(call{method: "POST", path: one + "/attempts", headers: hdr(tabA, true)}, 401)
	r.must(call{method: "POST", path: one + "/attempts", token: x.gv, headers: hdr(tabA, true)}, 403)
	r.must(call{method: "POST", path: one + "/attempts", token: x.ta, headers: hdr(tabA, true)}, 403)
	r.must(call{method: "POST", path: one + "/attempts", token: x.admin, headers: hdr(tabA, true)}, 403)
	r.must(call{method: "POST", path: "/api/v1/courses/khong-phai-uuid/exams/" + e.ID + "/attempts", token: x.sv, headers: hdr(tabA, true)}, 404)
	r.must(call{method: "POST", path: ex + "/" + uuid.NewString() + "/attempts", token: x.sv, headers: hdr(tabA, true)}, 404)
	r.must(call{method: "POST", path: archEx + "/" + uuid.NewString() + "/attempts", token: x.sv, headers: hdr(tabA, true)}, 409)
	_, b = r.must(call{method: "POST", path: one + "/attempts", token: x.sv, headers: hdr(tabA, true)}, 201)
	var st struct {
		Attempt struct {
			ID string `json:"id"`
		} `json:"attempt"`
		Items []struct {
			ItemID  string `json:"item_id"`
			Options []struct {
				ID string `json:"id"`
			} `json:"options"`
		} `json:"items"`
	}
	_ = json.Unmarshal(b, &st)
	r.must(call{method: "POST", path: one + "/attempts", token: x.sv, headers: hdr(tabA, true)}, 200) // làm tiếp cùng lượt
	att := one + "/attempts/" + st.Attempt.ID

	// 30: lượt của tôi.
	r.must(call{method: "GET", path: one + "/attempts/mine", token: x.sv}, 200)
	r.must(call{method: "GET", path: one + "/attempts/mine"}, 401)
	r.must(call{method: "GET", path: one + "/attempts/mine", token: x.gv}, 403)
	r.must(call{method: "GET", path: ex + "/" + uuid.NewString() + "/attempts/mine", token: x.sv}, 404)

	// 31: lưu câu trả lời (một nơi được ghi).
	ans := `{"items":[{"item_id":"` + st.Items[0].ItemID + `","answer":{"option_ids":["` + st.Items[0].Options[0].ID + `"]}}]}`
	r.must(call{method: "PUT", path: att + "/answers", token: x.sv, headers: hdr(tabA, false), body: ans}, 200)
	r.must(call{method: "PUT", path: att + "/answers", token: x.sv, body: ans}, 422) // thiếu X-Exam-Tab
	r.must(call{method: "PUT", path: att + "/answers", headers: hdr(tabA, false), body: ans}, 401)
	r.must(call{method: "PUT", path: att + "/answers", token: x.gv, headers: hdr(tabA, false), body: ans}, 403)
	r.must(call{method: "PUT", path: one + "/attempts/" + uuid.NewString() + "/answers", token: x.sv, headers: hdr(tabA, false), body: ans}, 404)
	r.must(call{method: "PUT", path: att + "/answers", token: x.sv, headers: hdr(tabB, false), body: ans}, 409) // tab khác khi tab A vừa ghi
	r.must(call{method: "PUT", path: att + "/answers", token: x.sv, headers: hdr(tabA, false), body: `{"items":[{"item_id":"` + st.Items[0].ItemID + `","answer":{"option_ids":["` + uuid.NewString() + `"]}}]}`}, 422)

	// 39: takeover.
	r.must(call{method: "POST", path: att + "/takeover", token: x.sv, headers: hdr(tabB, false), body: `{"reload":false}`}, 200)
	r.must(call{method: "POST", path: att + "/takeover", headers: hdr(tabB, false)}, 401)
	r.must(call{method: "POST", path: att + "/takeover", token: x.gv, headers: hdr(tabB, false)}, 403)
	r.must(call{method: "POST", path: one + "/attempts/" + uuid.NewString() + "/takeover", token: x.sv, headers: hdr(tabB, false)}, 404)
	r.must(call{method: "POST", path: att + "/takeover", token: x.sv}, 422) // thiếu X-Exam-Tab

	// 41: kết quả khi chưa công bố → 409 (thân không chứa dữ liệu).
	r.must(call{method: "GET", path: att + "/result", token: x.sv}, 409)
	r.must(call{method: "GET", path: att + "/result"}, 401)
	r.must(call{method: "GET", path: att + "/result", token: x.gv}, 403)
	r.must(call{method: "GET", path: one + "/attempts/" + uuid.NewString() + "/result", token: x.sv}, 404)

	// 40: nộp bài.
	r.must(call{method: "POST", path: att + "/submit", token: x.sv, headers: map[string]string{"X-Exam-Tab": tabB}}, 422) // thiếu Idempotency-Key
	r.must(call{method: "POST", path: att + "/submit", headers: hdr(tabB, true)}, 401)
	r.must(call{method: "POST", path: att + "/submit", token: x.gv, headers: hdr(tabB, true)}, 403)
	r.must(call{method: "POST", path: one + "/attempts/" + uuid.NewString() + "/submit", token: x.sv, headers: hdr(tabB, true)}, 404)
	r.must(call{method: "POST", path: att + "/submit", token: x.sv, headers: hdr(tabB, true)}, 200)
	r.must(call{method: "POST", path: att + "/submit", token: x.sv, headers: hdr(tabB, true)}, 409) // khoá khác: đã nộp
	r.must(call{method: "POST", path: att + "/takeover", token: x.sv, headers: hdr(tabB, false)}, 409)
	r.must(call{method: "PUT", path: att + "/answers", token: x.sv, headers: hdr(tabB, false), body: ans}, 409)
	r.must(call{method: "POST", path: one + "/attempts", token: x.sv, headers: hdr(tabB, true)}, 409) // đã nộp
	r.must(call{method: "GET", path: one + "/attempts/mine", token: x.sv}, 200)                       // tóm tắt, không đề

	// 41: sau công bố.
	if _, err := db.Exec(context.Background(), `update exams set status='PUBLISHED', published_at=now() where id=$1`, e.ID); err != nil {
		r.t.Fatal(err)
	}
	r.must(call{method: "GET", path: att + "/result", token: x.sv}, 200)
}

// examCodeScenarios: thao tác 32–37 (bài code trong lượt làm, US-PE-06) với mọi status đã khai báo (429 của `draft` được miễn — xem exempt.go).
func (r *runner) examCodeScenarios(x examRig, codeQ string) {
	ctx := context.Background()
	db := r.rig.deps.DB
	ex := "/api/v1/courses/" + x.cid + "/exams"
	for _, q := range []string{
		`update question_bank set review_status='APPROVED', reviewed_by=created_by, reviewed_at=now() where id=$1`,
		`update code_testcases set approved=true where problem_id=$1`,
		`update code_problems set reference_verified_version=tests_version, reference_verified_at=now() where question_id=$1`,
	} {
		if _, err := db.Exec(ctx, q, codeQ); err != nil {
			r.t.Fatal(err)
		}
	}
	now := time.Now().UTC()
	_, b := r.must(call{method: "POST", path: ex, token: x.gv, headers: x.idem(), body: `{"title":"Bài code","opens_at":"` + now.Add(2*time.Hour).Format(time.RFC3339) + `","closes_at":"` + now.Add(4*time.Hour).Format(time.RFC3339) + `","duration_minutes":45}`}, 201)
	var e struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
	}
	_ = json.Unmarshal(b, &e)
	r.must(call{method: "PUT", path: ex + "/" + e.ID + "/items", token: x.gv, body: `{"items":[{"question_id":"` + codeQ + `","points":"3"}],"version":` + itoa(e.Version) + `}`}, 200)
	r.must(call{method: "POST", path: ex + "/" + e.ID + "/schedule", token: x.gv}, 200)
	if _, err := db.Exec(ctx, `update exams set status='OPEN', opens_at=now() - interval '1 minute', closes_at=now() + interval '3 hours' where id=$1`, e.ID); err != nil {
		r.t.Fatal(err)
	}
	one := ex + "/" + e.ID
	tabA, tabB := uuid.NewString(), uuid.NewString()
	hdr := func(tab string, idem bool) map[string]string {
		h := map[string]string{"X-Exam-Tab": tab}
		if idem {
			h["Idempotency-Key"] = "ct-" + uuid.NewString()
		}
		return h
	}
	_, b = r.must(call{method: "POST", path: one + "/attempts", token: x.sv, headers: hdr(tabA, true)}, 201)
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
	it := st.Items[0].ItemID
	codeP := att + "/code/" + it
	src := `{"language":"cpp17","source":"int main(){return 0;}"}`

	// 32: bản nháp.
	r.must(call{method: "PUT", path: codeP + "/draft", token: x.sv, headers: hdr(tabA, false), body: `{"language":"cpp17","source":"int main(){}","base_rev":0}`}, 200)
	r.must(call{method: "PUT", path: codeP + "/draft", token: x.sv, headers: hdr(tabA, false), body: `{"language":"cpp17","source":"x","base_rev":0}`}, 409) // rev đã là 1: DRAFT_CONFLICT
	r.must(call{method: "PUT", path: codeP + "/draft", token: x.sv, headers: hdr(tabA, false), body: `{"language":"c11","source":"x","base_rev":0}`}, 422)   // c11 không được phép
	r.must(call{method: "PUT", path: codeP + "/draft", headers: hdr(tabA, false), body: src}, 401)
	r.must(call{method: "PUT", path: codeP + "/draft", token: x.gv, headers: hdr(tabA, false), body: `{"language":"cpp17","source":"x","base_rev":1}`}, 403)
	r.must(call{method: "PUT", path: att + "/code/" + uuid.NewString() + "/draft", token: x.sv, headers: hdr(tabA, false), body: `{"language":"cpp17","source":"x","base_rev":1}`}, 404)
	r.must(call{method: "PUT", path: codeP + "/draft", token: x.sv, headers: hdr(tabB, false), body: `{"language":"cpp17","source":"x","base_rev":1}`}, 409) // tab khác đang là nơi ghi

	// 33: chạy thử. Chưa có `ep:judge:up` (không có worker) → 503; đặt khoá → 202; quá 10 lần / 10 phút → 429.
	r.must(call{method: "POST", path: codeP + "/run", token: x.sv, headers: hdr(tabA, true), body: src}, 503)
	if err := r.rig.deps.Redis.Set(ctx, "ep:judge:up", "1", time.Minute).Err(); err != nil {
		r.t.Fatal(err)
	}
	defer r.rig.deps.Redis.Del(ctx, "ep:judge:up")
	_, b = r.must(call{method: "POST", path: codeP + "/run", token: x.sv, headers: hdr(tabA, true), body: src}, 202)
	var run struct {
		RunID string `json:"run_id"`
	}
	_ = json.Unmarshal(b, &run)
	r.must(call{method: "POST", path: codeP + "/run", token: x.sv, headers: map[string]string{"X-Exam-Tab": tabA}, body: src}, 422) // thiếu Idempotency-Key
	r.must(call{method: "POST", path: codeP + "/run", headers: hdr(tabA, true), body: src}, 401)
	r.must(call{method: "POST", path: codeP + "/run", token: x.gv, headers: hdr(tabA, true), body: src}, 403)
	r.must(call{method: "POST", path: att + "/code/" + uuid.NewString() + "/run", token: x.sv, headers: hdr(tabA, true), body: src}, 404)
	r.must(call{method: "POST", path: codeP + "/run", token: x.sv, headers: hdr(tabA, true), body: `{"language":"c11","source":"x"}`}, 422)
	r.must(call{method: "POST", path: codeP + "/run", token: x.sv, headers: hdr(tabB, true), body: src}, 409)
	var uid string
	if err := db.QueryRow(ctx, `select student_id::text from exam_attempts where id=$1`, st.Attempt.ID).Scan(&uid); err != nil {
		r.t.Fatal(err)
	}
	runKey := "ep:exam:run:" + uid
	for i := range 10 {
		if err := r.rig.deps.Redis.ZAdd(ctx, runKey, redis.Z{Score: float64(time.Now().UnixMilli()), Member: "fill-" + itoa(i)}).Err(); err != nil {
			r.t.Fatal(err)
		}
	}
	r.must(call{method: "POST", path: codeP + "/run", token: x.sv, headers: hdr(tabA, true), body: src}, 429)
	r.rig.deps.Redis.Del(ctx, runKey)

	// 34: kết quả chạy thử.
	rp := att + "/runs/" + run.RunID
	r.must(call{method: "GET", path: rp, token: x.sv}, 200)
	r.must(call{method: "GET", path: rp}, 401)
	r.must(call{method: "GET", path: rp, token: x.gv}, 403)
	r.must(call{method: "GET", path: att + "/runs/" + uuid.NewString(), token: x.sv}, 404)

	// 35: nộp lời giải (lần hai trong 15 s → 429).
	_, b = r.must(call{method: "POST", path: codeP + "/submit", token: x.sv, headers: hdr(tabA, true), body: src}, 202)
	var sub struct {
		ID string `json:"submission_id"`
	}
	_ = json.Unmarshal(b, &sub)
	r.must(call{method: "POST", path: codeP + "/submit", token: x.sv, headers: hdr(tabA, true), body: src}, 429)
	r.must(call{method: "POST", path: codeP + "/submit", token: x.sv, headers: map[string]string{"X-Exam-Tab": tabA}, body: src}, 422) // thiếu Idempotency-Key
	r.must(call{method: "POST", path: codeP + "/submit", headers: hdr(tabA, true), body: src}, 401)
	r.must(call{method: "POST", path: codeP + "/submit", token: x.gv, headers: hdr(tabA, true), body: src}, 403)
	r.must(call{method: "POST", path: att + "/code/" + uuid.NewString() + "/submit", token: x.sv, headers: hdr(tabA, true), body: src}, 404)
	r.must(call{method: "POST", path: codeP + "/submit", token: x.sv, headers: hdr(tabB, true), body: src}, 409)
	r.must(call{method: "POST", path: codeP + "/submit", token: x.sv, headers: hdr(tabA, true), body: `{"language":"cpp17","source":""}`}, 422)

	// 36, 37: lịch sử và một bản nộp.
	r.must(call{method: "GET", path: codeP + "/submissions", token: x.sv}, 200)
	r.must(call{method: "GET", path: codeP + "/submissions"}, 401)
	r.must(call{method: "GET", path: codeP + "/submissions", token: x.gv}, 403)
	r.must(call{method: "GET", path: one + "/attempts/" + uuid.NewString() + "/code/" + it + "/submissions", token: x.sv}, 404)
	r.must(call{method: "GET", path: codeP + "/submissions?limit=0", token: x.sv}, 422)
	sp := att + "/submissions/" + sub.ID
	r.must(call{method: "GET", path: sp, token: x.sv}, 200)
	r.must(call{method: "GET", path: sp}, 401)
	r.must(call{method: "GET", path: sp, token: x.gv}, 403)
	r.must(call{method: "GET", path: att + "/submissions/" + uuid.NewString(), token: x.sv}, 404)

	// ---- US-PE-07: sự kiện liêm chính (38, 51), khoá chat (43), so độ giống (52–54 + chi tiết) ----
	aid := st.Attempt.ID
	evBody := `{"events":[{"type":"PASTE","client_at":"` + time.Now().UTC().Format(time.RFC3339) + `","meta":{"chars":812,"clipboard":"SECRET"}},{"type":"TAB_HIDDEN","meta":{"duration_ms":4000}}]}`
	r.must(call{method: "POST", path: att + "/events", token: x.sv, body: evBody}, 204)
	r.must(call{method: "POST", path: att + "/events", body: evBody}, 401)
	r.must(call{method: "POST", path: att + "/events", token: x.gv, body: evBody}, 403)
	r.must(call{method: "POST", path: one + "/attempts/" + uuid.NewString() + "/events", token: x.sv, body: evBody}, 404)
	r.must(call{method: "POST", path: att + "/events", token: x.sv, body: `{}`}, 422)
	evs := one + "/events"
	r.must(call{method: "GET", path: evs + "?attempt=" + aid, token: x.gv}, 200)
	r.must(call{method: "GET", path: evs + "?attempt=" + aid}, 401)
	for _, who := range []string{x.ta, x.sv, x.admin} {
		r.must(call{method: "GET", path: evs + "?attempt=" + aid, token: who}, 403) // TA, sinh viên (kể cả của chính mình), Admin
	}
	r.must(call{method: "GET", path: evs + "?attempt=" + uuid.NewString(), token: x.gv}, 404)
	r.must(call{method: "GET", path: evs, token: x.gv}, 422)
	r.must(call{method: "GET", path: "/api/v1/me/exam-lock", token: x.sv}, 200)
	r.must(call{method: "GET", path: "/api/v1/me/exam-lock"}, 401)

	sim := one + "/similarity"
	_, mb := r.must(call{method: "POST", path: ex, token: x.gv, headers: x.idem(), body: `{"title":"Bài không có câu code"}`}, 201)
	var mcq struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(mb, &mcq)
	r.must(call{method: "POST", path: ex + "/" + mcq.ID + "/similarity/run", token: x.gv}, 422) // không có câu lập trình
	r.must(call{method: "POST", path: sim + "/run", token: x.gv}, 409)                          // bài chưa đóng
	if _, err := db.Exec(ctx, `update exams set status='CLOSED', opens_at=now() - interval '4 hours', closes_at=now() - interval '1 minute' where id=$1`, e.ID); err != nil {
		r.t.Fatal(err)
	}
	r.must(call{method: "POST", path: sim + "/run", token: x.gv}, 202)
	r.must(call{method: "POST", path: sim + "/run"}, 401)
	r.must(call{method: "POST", path: sim + "/run", token: x.ta}, 403)
	r.must(call{method: "POST", path: ex + "/" + uuid.NewString() + "/similarity/run", token: x.gv}, 404)
	// hai bên của cặp phải là hai lượt khác nhau (`attempt_a < attempt_b`): dựng lượt thứ hai cho một người dùng khác và một bản nộp của lượt đó
	var att2, sub2, other string
	if err := db.QueryRow(ctx, `select id::text from users where role = 'TEACHER' limit 1`).Scan(&other); err != nil {
		r.t.Fatal(err)
	}
	if err := db.QueryRow(ctx, `insert into exam_attempts (course_id, exam_id, student_id, started_at, deadline_at, status, submitted_at, submit_reason) values ($1, $2, $3, now() - interval '1 hour', now() - interval '10 minutes', 'GRADING', now() - interval '20 minutes', 'MANUAL') returning id::text`, x.cid, e.ID, other).Scan(&att2); err != nil {
		r.t.Fatal(err)
	}
	if err := db.QueryRow(ctx, `insert into code_submissions (course_id, exam_id, attempt_id, item_id, problem_id, student_id, kind, language, source, source_sha256)
		select course_id, exam_id, $2::uuid, item_id, problem_id, $3::uuid, kind, language, source || ' /* bản 2 */', source_sha256 from code_submissions where id=$1 returning id::text`, sub.ID, att2, other).Scan(&sub2); err != nil {
		r.t.Fatal(err)
	}
	lo, hi := aid, att2
	sa, sb := sub.ID, sub2
	if lo > hi {
		lo, hi, sa, sb = hi, lo, sb, sa
	}
	var pairID string
	if err := db.QueryRow(ctx, `insert into similarity_reports (course_id, exam_id, problem_id, run_id, submission_a, submission_b, attempt_a, attempt_b, score, shared_fingerprints, flagged)
		values ($1, $2, $3, $4, $5, $6, $7, $8, 0.873, 40, true) returning id::text`, x.cid, e.ID, codeQ, uuid.NewString(), sa, sb, lo, hi).Scan(&pairID); err != nil {
		r.t.Fatal(err)
	}
	r.must(call{method: "GET", path: sim, token: x.gv}, 200)
	r.must(call{method: "GET", path: sim + "?flagged=true&limit=1", token: x.gv}, 200)
	r.must(call{method: "GET", path: sim}, 401)
	r.must(call{method: "GET", path: sim, token: x.ta}, 403)
	r.must(call{method: "GET", path: ex + "/" + uuid.NewString() + "/similarity", token: x.gv}, 404)
	r.must(call{method: "GET", path: sim + "?limit=0", token: x.gv}, 422)
	sp1 := sim + "/" + pairID
	r.must(call{method: "GET", path: sp1, token: x.gv}, 200)
	r.must(call{method: "GET", path: sp1}, 401)
	r.must(call{method: "GET", path: sp1, token: x.sv}, 403)
	r.must(call{method: "GET", path: sim + "/" + uuid.NewString(), token: x.gv}, 404)
	r.must(call{method: "PUT", path: sp1 + "/review", token: x.gv, body: `{"state":"CLEARED","note":"Cùng cách làm tự nhiên"}`}, 200)
	r.must(call{method: "PUT", path: sp1 + "/review", body: `{"state":"CLEARED"}`}, 401)
	r.must(call{method: "PUT", path: sp1 + "/review", token: x.ta, body: `{"state":"CLEARED"}`}, 403)
	r.must(call{method: "PUT", path: sim + "/" + uuid.NewString() + "/review", token: x.gv, body: `{"state":"CLEARED"}`}, 404)
	r.must(call{method: "PUT", path: sp1 + "/review", token: x.gv, body: `{"state":"CHEAT"}`}, 422)
}

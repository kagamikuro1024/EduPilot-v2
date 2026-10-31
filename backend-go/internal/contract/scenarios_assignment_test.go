package contract

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// assignmentScenarios: bài tập (US-P7-02, SRS FEAT-assignments 6 #1–#8) với mọi status đã khai báo. Rubric gợi ý dùng bản ghi demo của provider fake (JSON chủ đích, Σ = 100).
func (r *runner) assignmentScenarios(x examRig) {
	ctx := context.Background()
	db := r.rig.deps.DB
	base := "/api/v1/courses/" + x.cid + "/assignments"
	due := time.Now().UTC().Add(72 * time.Hour).Format(time.RFC3339)
	rubric := `[{"name":"Khái niệm","max_score":"4","guidance":"Nêu đúng định nghĩa"},{"name":"Ví dụ","max_score":"6"}]`
	body := func(code string) string {
		return `{"title":"BT03","submission_code":"` + code + `","due_at":"` + due + `","max_score":"10","channels":["IN_APP"],"rubric":` + rubric + `,"reference_answer":"` + strings.Repeat("Đáp án chuẩn. ", 8) + `"}`
	}
	type asg struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
	}
	parse := func(b []byte) asg {
		var a asg
		_ = json.Unmarshal(b, &a)
		return a
	}

	// 2: tạo.
	_, b := r.must(call{method: "POST", path: base, token: x.gv, headers: x.idem(), body: body("BT03")}, 201)
	a := parse(b)
	r.must(call{method: "POST", path: base, headers: x.idem(), body: body("BT04")}, 401)
	for _, who := range []string{x.ta, x.sv, x.admin} {
		r.must(call{method: "POST", path: base, token: who, headers: x.idem(), body: body("BT04")}, 403)
	}
	r.must(call{method: "POST", path: "/api/v1/courses/khong-phai-uuid/assignments", token: x.gv, headers: x.idem(), body: body("BT04")}, 404)
	r.must(call{method: "POST", path: "/api/v1/courses/" + x.arch + "/assignments", token: x.gv, headers: x.idem(), body: body("BT04")}, 409) // COURSE_ARCHIVED
	r.must(call{method: "POST", path: base, token: x.gv, headers: x.idem(), body: body("BT03")}, 422)                                         // mã nộp thư trùng
	r.must(call{method: "POST", path: base, token: x.gv, headers: x.idem(), body: strings.Replace(body("BT05"), `"10"`, `"10.1"`, 1)}, 422)   // SCORE_STEP

	// 1: danh sách.
	r.must(call{method: "GET", path: base, token: x.gv}, 200)
	r.must(call{method: "GET", path: base, token: x.ta}, 200)
	r.must(call{method: "GET", path: base, token: x.sv}, 200) // chưa công bố → rỗng
	r.must(call{method: "GET", path: base}, 401)
	r.must(call{method: "GET", path: base, token: x.admin}, 403)
	r.must(call{method: "GET", path: "/api/v1/courses/khong-phai-uuid/assignments", token: x.gv}, 404)
	r.must(call{method: "GET", path: base + "?limit=0", token: x.gv}, 422)

	// 3: chi tiết; bài nháp 404 với sinh viên.
	A := base + "/" + a.ID
	r.must(call{method: "GET", path: A, token: x.gv}, 200)
	r.must(call{method: "GET", path: A, token: x.ta}, 200)
	r.must(call{method: "GET", path: A}, 401)
	r.must(call{method: "GET", path: A, token: x.admin}, 403)
	r.must(call{method: "GET", path: A, token: x.sv}, 404)
	r.must(call{method: "GET", path: base + "/00000000-0000-7000-8000-000000000000", token: x.gv}, 404)

	// 8: gợi ý rubric — bản ghi demo cho 200; thiếu đáp án 422; mọi nhà cung cấp lỗi 503.
	sg := A + "/rubric/suggest"
	demo := filepath.Join(r.t.TempDir(), "demo.json")
	if err := os.WriteFile(demo, []byte(`{"recordings":[{"name":"suggest","match":["ĐÁP ÁN CHUẨN (dữ liệu)"],"response":{"criteria":[{"name":"Khái niệm","guidance":"Nêu đúng","weight_pct":40},{"name":"Ví dụ","guidance":"Có ví dụ","weight_pct":60}]}}]}`), 0o600); err != nil {
		r.t.Fatal(err)
	}
	ctl := r.rig.deps.LLM.Registry.Fake()
	if err := ctl.LoadDemo(demo); err != nil {
		r.t.Fatal(err)
	}
	defer func() { _ = ctl.LoadDemo("") }()
	r.must(call{method: "POST", path: sg, token: x.gv, headers: x.idem()}, 200)
	r.must(call{method: "POST", path: sg, token: x.ta, headers: x.idem()}, 200)
	r.must(call{method: "POST", path: sg, headers: x.idem()}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "POST", path: sg, token: who, headers: x.idem()}, 403)
	}
	r.must(call{method: "POST", path: base + "/00000000-0000-7000-8000-000000000000/rubric/suggest", token: x.gv, headers: x.idem()}, 404)
	_, b = r.must(call{method: "POST", path: base, token: x.gv, headers: x.idem(), body: `{"title":"Không đáp án","submission_code":"NOREF","due_at":"` + due + `"}`}, 201)
	r.must(call{method: "POST", path: base + "/" + parse(b).ID + "/rubric/suggest", token: x.gv, headers: x.idem()}, 422) // REFERENCE_ANSWER_REQUIRED
	r.must(call{method: "POST", path: "/api/v1/_test/llm/fake", token: x.admin, body: `{"error_rate":1,"error_kind":"AUTH"}`}, 200)
	r.must(call{method: "POST", path: sg, token: x.gv, headers: x.idem()}, 503) // LLM_UNAVAILABLE
	r.must(call{method: "POST", path: "/api/v1/_test/llm/fake", token: x.admin, body: `{"error_rate":0,"error_kind":"SERVER"}`}, 200)

	// 7: lưu rubric.
	R := A + "/rubric"
	rb := func(v int, crit string) string { return `{"version":` + itoa(v) + `,"criteria":` + crit + `}` }
	r.must(call{method: "PUT", path: R, token: x.gv, body: rb(a.Version, `[{"name":"Một","max_score":"10"}]`)}, 200)
	r.must(call{method: "PUT", path: R, token: x.gv, body: rb(a.Version, rubric)}, 409) // version cũ
	r.must(call{method: "PUT", path: R, token: x.gv, body: rb(a.Version+1, `[{"name":"Một","max_score":"9"}]`)}, 422)
	r.must(call{method: "PUT", path: R, body: rb(1, rubric)}, 401)
	for _, who := range []string{x.ta, x.sv, x.admin} {
		r.must(call{method: "PUT", path: R, token: who, body: rb(1, rubric)}, 403)
	}
	r.must(call{method: "PUT", path: base + "/00000000-0000-7000-8000-000000000000/rubric", token: x.gv, body: rb(1, rubric)}, 404)
	a.Version++

	// 4: sửa.
	put := func(v int, title string) string {
		return `{"version":` + itoa(v) + `,"title":"` + title + `","due_at":"` + due + `","max_score":"10","channels":["IN_APP"]}`
	}
	_, b = r.must(call{method: "PUT", path: A, token: x.gv, body: put(a.Version, "BT03 sửa")}, 200)
	r.must(call{method: "PUT", path: A, token: x.gv, body: put(a.Version, "Cũ")}, 409)
	r.must(call{method: "PUT", path: A, token: x.gv, body: put(parse(b).Version, "")}, 422)
	r.must(call{method: "PUT", path: A, body: put(1, "x")}, 401)
	for _, who := range []string{x.ta, x.sv, x.admin} {
		r.must(call{method: "PUT", path: A, token: who, body: put(1, "x")}, 403)
	}
	r.must(call{method: "PUT", path: base + "/00000000-0000-7000-8000-000000000000", token: x.gv, body: put(1, "x")}, 404)

	// 6: công bố — thiếu rubric 422; hai lần 409.
	P := A + "/publish"
	_, b = r.must(call{method: "POST", path: base, token: x.gv, headers: x.idem(), body: `{"title":"Chưa rubric","submission_code":"NORUB","due_at":"` + due + `"}`}, 201)
	r.must(call{method: "POST", path: base + "/" + parse(b).ID + "/publish", token: x.gv, headers: x.idem()}, 422) // RUBRIC_REQUIRED
	r.must(call{method: "POST", path: P, token: x.gv, headers: x.idem()}, 200)
	r.must(call{method: "POST", path: P, token: x.gv, headers: x.idem()}, 409) // ALREADY_PUBLISHED
	r.must(call{method: "POST", path: P, headers: x.idem()}, 401)
	for _, who := range []string{x.ta, x.sv, x.admin} {
		r.must(call{method: "POST", path: P, token: who, headers: x.idem()}, 403)
	}
	r.must(call{method: "POST", path: base + "/00000000-0000-7000-8000-000000000000/publish", token: x.gv, headers: x.idem()}, 404)
	r.must(call{method: "GET", path: A, token: x.sv}, 200) // đã công bố: bản rút gọn

	// 9–11: sinh viên — mine, presign, submit.
	r.studentSubmitScenarios(x, base, A, a.ID)

	// 5: xoá — bài nộp → 409; bài sạch 204.
	if _, err := db.Exec(ctx, `insert into submissions (course_id, assignment_id, student_id, source, version, blob_key, file_name, file_size, file_mime, sha256, match_status, match_method, submitted_at)
		select $1, $2, e.user_id, 'IN_APP', 99, 'k', 'a.pdf', 1, 'application/pdf', 'x', 'MATCHED', 'IN_APP', now() from enrollments e where e.course_id = $1 and e.role_in_course = 'STUDENT' limit 1`, x.cid, a.ID); err != nil {
		r.t.Fatalf("chèn bài nộp: %v", err)
	}
	r.must(call{method: "DELETE", path: A, token: x.gv}, 409) // HAS_SUBMISSIONS
	_, b = r.must(call{method: "POST", path: base, token: x.gv, headers: x.idem(), body: `{"title":"Xoá được","submission_code":"DEL1","due_at":"` + due + `"}`}, 201)
	D := base + "/" + parse(b).ID
	r.must(call{method: "DELETE", path: D}, 401)
	for _, who := range []string{x.ta, x.sv, x.admin} {
		r.must(call{method: "DELETE", path: D, token: who}, 403)
	}
	r.must(call{method: "DELETE", path: D, token: x.gv}, 204)
	r.must(call{method: "DELETE", path: D, token: x.gv}, 404)
}

// studentSubmitScenarios: thao tác 9 (mine), 10 (presign), 11 (submit) của SRS FEAT-assignments 6 với mọi status đã khai báo. `A` là bài đã công bố, kênh IN_APP, hạn còn 72 giờ.
func (r *runner) studentSubmitScenarios(x examRig, base, A, aid string) {
	ctx := context.Background()
	db := r.rig.deps.DB
	due := time.Now().UTC().Add(72 * time.Hour).Format(time.RFC3339)
	mk := func(extra, code string, publish bool) string { // bài mới (đã có rubric) — trả id
		_, b := r.must(call{method: "POST", path: base, token: x.gv, headers: x.idem(), body: `{"title":"` + code + `","submission_code":"` + code + `","due_at":"` + due + `","rubric":[{"name":"Một","max_score":"10"}]` + extra + `}`}, 201)
		var o struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(b, &o)
		if publish {
			r.must(call{method: "POST", path: base + "/" + o.ID + "/publish", token: x.gv, headers: x.idem()}, 200)
		}
		return o.ID
	}
	// 9: mine.
	M := A + "/mine"
	_, raw := r.must(call{method: "GET", path: M, token: x.sv}, 200)
	if strings.Contains(string(raw), "guidance") || strings.Contains(string(raw), "reference_answer") || !strings.Contains(string(raw), `"state":"NOT_SUBMITTED"`) {
		r.t.Fatalf("mine lộ dữ liệu nhân viên hoặc sai trạng thái: %s", raw)
	}
	r.must(call{method: "GET", path: M}, 401)
	for _, who := range []string{x.ta, x.gv, x.admin} {
		r.must(call{method: "GET", path: M, token: who}, 403)
	}
	r.must(call{method: "GET", path: base + "/00000000-0000-7000-8000-000000000000/mine", token: x.sv}, 404)
	draft := mk("", "NHAP2", false)
	r.must(call{method: "GET", path: base + "/" + draft + "/mine", token: x.sv}, 404) // ASSIGNMENT_NOT_FOUND: không lộ bài nháp

	// 10: presign.
	data := []byte("Bài làm của tôi.\n")
	sum := sha256.Sum256(data)
	hexSum := fmt.Sprintf("%x", sum)
	body := func(name string, size int) string {
		return fmt.Sprintf(`{"filename":%q,"size_bytes":%d,"sha256":%q}`, name, size, hexSum)
	}
	type pres struct {
		UploadID string            `json:"upload_id"`
		URL      string            `json:"url"`
		Headers  map[string]string `json:"headers"`
	}
	presign := func(path string) pres {
		_, b := r.must(call{method: "POST", path: path + "/submit/presign", token: x.sv, body: body("bai.txt", len(data))}, 200)
		var p pres
		_ = json.Unmarshal(b, &p)
		return p
	}
	put := func(p pres, payload []byte) {
		req, _ := http.NewRequest(http.MethodPut, p.URL, bytes.NewReader(payload))
		for k, v := range p.Headers {
			if k != "Content-Length" {
				req.Header.Set(k, v)
			}
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			r.t.Fatalf("PUT bài nộp: %v", err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			r.t.Fatalf("PUT bài nộp → %d", resp.StatusCode)
		}
	}
	if want := base64.StdEncoding.EncodeToString(sum[:]); true {
		p := presign(A)
		if p.Headers["x-amz-checksum-sha256"] != want || p.Headers["Content-Type"] != "text/plain; charset=utf-8" {
			r.t.Fatalf("presign: header ký sai: %v", p.Headers)
		}
		// 11: submit — chưa PUT thì 422 UPLOAD_INCOMPLETE; PUT rồi 201; vé đã dùng 404.
		r.must(call{method: "POST", path: A + "/submit", token: x.sv, headers: x.idem(), body: `{"upload_id":"` + p.UploadID + `"}`}, 422)
		put(p, data)
		k := x.idem()
		_, b1 := r.must(call{method: "POST", path: A + "/submit", token: x.sv, headers: k, body: `{"upload_id":"` + p.UploadID + `"}`}, 201)
		_, b2 := r.must(call{method: "POST", path: A + "/submit", token: x.sv, headers: k, body: `{"upload_id":"` + p.UploadID + `"}`}, 201)
		if string(b1) != string(b2) {
			r.t.Fatalf("nộp lại cùng khoá phải phát lại: %s ≠ %s", b1, b2)
		}
		r.must(call{method: "POST", path: A + "/submit", token: x.sv, headers: x.idem(), body: `{"upload_id":"` + p.UploadID + `"}`}, 404)
		r.must(call{method: "POST", path: A + "/submit", token: x.sv, headers: x.idem(), body: `{}`}, 422)
	}
	_, raw = r.must(call{method: "GET", path: M, token: x.sv}, 200)
	if !strings.Contains(string(raw), `"state":"SUBMITTED"`) || !strings.Contains(string(raw), `"file_name":"bai.txt"`) {
		r.t.Fatalf("mine sau khi nộp: %s", raw)
	}
	r.must(call{method: "POST", path: A + "/submit/presign", body: body("bai.txt", len(data))}, 401)
	r.must(call{method: "POST", path: A + "/submit", headers: x.idem(), body: `{"upload_id":"` + uuid.NewString() + `"}`}, 401)
	for _, who := range []string{x.ta, x.gv, x.admin} {
		r.must(call{method: "POST", path: A + "/submit/presign", token: who, body: body("bai.txt", len(data))}, 403)
		r.must(call{method: "POST", path: A + "/submit", token: who, headers: x.idem(), body: `{"upload_id":"` + uuid.NewString() + `"}`}, 403)
	}
	nf := base + "/00000000-0000-7000-8000-000000000000"
	r.must(call{method: "POST", path: nf + "/submit/presign", token: x.sv, body: body("bai.txt", len(data))}, 404)
	r.must(call{method: "POST", path: base + "/" + draft + "/submit/presign", token: x.sv, body: body("bai.txt", len(data))}, 404)
	r.must(call{method: "POST", path: nf + "/submit", token: x.sv, headers: x.idem(), body: `{"upload_id":"` + uuid.NewString() + `"}`}, 404)
	r.must(call{method: "POST", path: A + "/submit/presign", token: x.sv, body: body("bai.exe", len(data))}, 422) // FILE_TYPE_NOT_ALLOWED
	r.must(call{method: "POST", path: A + "/submit/presign", token: x.sv, body: body("bai.txt", 0)}, 422)         // FILE_EMPTY
	r.must(call{method: "POST", path: A + "/submit/presign", token: x.sv, body: body("bai.txt", 20<<20+1)}, 422)  // FILE_TOO_LARGE
	uploadOnly := mk(`,"channels":["UPLOAD"]`, "UPONLY", true)
	r.must(call{method: "POST", path: base + "/" + uploadOnly + "/submit/presign", token: x.sv, body: body("bai.txt", len(data))}, 409) // CHANNEL_NOT_ALLOWED
	closed := mk(`,"allow_late":false`, "CLOSED1", true)
	if _, err := db.Exec(ctx, `update assignments set due_at = now() - interval '1 hour' where id = $1`, closed); err != nil {
		r.t.Fatalf("lùi hạn: %v", err)
	}
	r.must(call{method: "POST", path: base + "/" + closed + "/submit/presign", token: x.sv, body: body("bai.txt", len(data))}, 409) // SUBMISSION_CLOSED
	// khoá nộp lại: kết quả của bản mới nhất đã REVIEWED → 409 SUBMISSION_LOCKED (câu chung)
	if _, err := db.Exec(ctx, `insert into grading_results (course_id, submission_id, run_state, status, reviewed_at)
		select s.course_id, s.id, 'DONE', 'REVIEWED', now() from submissions s where s.assignment_id = $1 and s.version = 1`, aid); err != nil {
		r.t.Fatalf("chèn kết quả: %v", err)
	}
	p := presign(A)
	put(p, data)
	r.must(call{method: "POST", path: A + "/submit", token: x.sv, headers: x.idem(), body: `{"upload_id":"` + p.UploadID + `"}`}, 409)
	// hạn 10 lượt presign / phút / người → 429
	got429 := false
	for range 30 {
		if st, _, _ := r.do(call{method: "POST", path: A + "/submit/presign", token: x.sv, body: body("bai.txt", len(data))}); st == http.StatusTooManyRequests {
			got429 = true
			break
		}
	}
	if !got429 {
		r.t.Fatal("presign bài nộp: không thấy 429 sau 30 lượt")
	}
}

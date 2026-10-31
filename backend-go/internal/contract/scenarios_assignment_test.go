package contract

import (
	"archive/zip"
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
	r.uploadScenarios(x, base)
	r.appealScenarios(x, base)
	r.versionsScenarios(x, base)
	r.integrationScenarios(x)

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

// uploadScenarios: thao tác 13 (upload), 14 (danh sách chưa khớp), 16 (gán tay), 17 (tải tệp), 35 / 36 (bỏ qua / hoàn tác) của SRS FEAT-assignments 6 với mọi status đã khai báo.
func (r *runner) uploadScenarios(x examRig, base string) {
	ctx := context.Background()
	db := r.rig.deps.DB
	due := time.Now().UTC().Add(72 * time.Hour).Format(time.RFC3339)
	mk := func(code string, publish bool, channels string) string {
		_, b := r.must(call{method: "POST", path: base, token: x.gv, headers: x.idem(), body: `{"title":"` + code + `","submission_code":"` + code + `","due_at":"` + due + `","channels":` + channels + `,"rubric":[{"name":"Một","max_score":"10"}]}`}, 201)
		var o struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(b, &o)
		if publish {
			r.must(call{method: "POST", path: base + "/" + o.ID + "/publish", token: x.gv, headers: x.idem()}, 200)
		}
		return o.ID
	}
	asg := mk("UPL1", true, `["IN_APP","UPLOAD"]`)
	A := base + "/" + asg
	type pres struct {
		UploadID string            `json:"upload_id"`
		URL      string            `json:"url"`
		Headers  map[string]string `json:"headers"`
	}
	upload := func(token, name string, data []byte, doPut bool) pres {
		h := sha256.Sum256(data)
		body := fmt.Sprintf(`{"purpose":"submission_upload","filename":%q,"size_bytes":%d,"sha256":%q}`, name, len(data), fmt.Sprintf("%x", h))
		_, b := r.must(call{method: "POST", path: strings.TrimSuffix(base, "/assignments") + "/uploads/presign", token: token, body: body}, 200)
		var p pres
		_ = json.Unmarshal(b, &p)
		if doPut {
			req, _ := http.NewRequest(http.MethodPut, p.URL, bytes.NewReader(data))
			for k, v := range p.Headers {
				if k != "Content-Length" {
					req.Header.Set(k, v)
				}
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				r.t.Fatalf("PUT: %v", err)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				r.t.Fatalf("PUT → %d", resp.StatusCode)
			}
		}
		return p
	}
	up := func(token, path, id string) call {
		return call{method: "POST", path: path + "/submissions/upload", token: token, headers: x.idem(), body: `{"upload_id":"` + id + `"}`}
	}
	// 13: tệp lẻ không MSSV → 200, vào Chưa khớp.
	single := upload(x.ta, "khong-mssv.txt", []byte("Bài làm không ghi MSSV.\n"), true)
	_, raw := r.must(up(x.ta, A, single.UploadID), 200)
	if !strings.Contains(string(raw), `"unmatched":1`) {
		r.t.Fatalf("tệp lẻ không MSSV phải vào Chưa khớp: %s", raw)
	}
	r.must(up(x.ta, A, single.UploadID), 404) // vé đã dùng
	// ZIP 6 mục → 202 + việc nền.
	var zb bytes.Buffer
	zw := zip.NewWriter(&zb)
	for i := range 6 {
		f, _ := zw.Create(fmt.Sprintf("bai-%d.txt", i))
		_, _ = fmt.Fprintf(f, "Bài làm số %d.\n", i)
	}
	_ = zw.Close()
	z := upload(x.gv, "lop.zip", zb.Bytes(), true)
	r.must(up(x.gv, A, z.UploadID), 202)
	// lỗi: chưa PUT → 422; ngoài vai → 403; bài lạ → 404; bài nháp / kênh tắt → 409.
	noPut := upload(x.ta, "chua-tai.txt", []byte("x\n"), false)
	r.must(up(x.ta, A, noPut.UploadID), 422)
	r.must(call{method: "POST", path: A + "/submissions/upload", headers: x.idem(), body: `{"upload_id":"` + noPut.UploadID + `"}`}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(up(who, A, noPut.UploadID), 403)
	}
	r.must(up(x.ta, base+"/00000000-0000-7000-8000-000000000000", noPut.UploadID), 404)
	r.must(up(x.ta, base+"/"+mk("UPLDRAFT", false, `["UPLOAD"]`), noPut.UploadID), 409) // ASSIGNMENT_NOT_PUBLISHED
	r.must(up(x.ta, base+"/"+mk("UPLOFF", true, `["IN_APP"]`), noPut.UploadID), 409)    // CHANNEL_NOT_ALLOWED

	// 14: tab Chưa khớp.
	sub := strings.TrimSuffix(base, "/assignments") + "/submissions"
	list := func() (id string, version int64) {
		_, b := r.must(call{method: "GET", path: sub + "?match_status=UNMATCHED", token: x.gv}, 200)
		var pg struct {
			Items []struct {
				ID      string `json:"id"`
				Version int64  `json:"version"`
			} `json:"items"`
		}
		_ = json.Unmarshal(b, &pg)
		if len(pg.Items) == 0 {
			r.t.Fatalf("Chưa khớp rỗng: %s", b)
		}
		return pg.Items[0].ID, pg.Items[0].Version
	}
	r.must(call{method: "GET", path: sub + "?match_status=UNMATCHED", token: x.ta}, 200)
	r.must(call{method: "GET", path: sub + "?match_status=UNMATCHED"}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "GET", path: sub + "?match_status=UNMATCHED", token: who}, 403)
	}
	r.must(call{method: "GET", path: "/api/v1/courses/khong-phai-uuid/submissions?match_status=UNMATCHED", token: x.gv}, 404)
	r.must(call{method: "GET", path: sub + "?match_status=NOPE", token: x.gv}, 422)
	r.must(call{method: "GET", path: sub, token: x.gv}, 200) // mặc định: hàng chờ chấm (MATCHED + DRAFT)
	id1, v1 := list()

	// 17: tải tệp (URL ký 5 phút).
	r.must(call{method: "GET", path: sub + "/" + id1 + "/file", token: x.gv}, 200)
	r.must(call{method: "GET", path: sub + "/" + id1 + "/file", token: x.ta}, 200)
	r.must(call{method: "GET", path: sub + "/" + id1 + "/file"}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "GET", path: sub + "/" + id1 + "/file", token: who}, 403)
	}
	r.must(call{method: "GET", path: sub + "/00000000-0000-7000-8000-000000000000/file", token: x.gv}, 404)

	// 16: gán tay — sai version 409; sinh viên ngoài lớp 422; thành công 200.
	var student string
	if err := db.QueryRow(ctx, `select user_id from enrollments where course_id = $1 and role_in_course = 'STUDENT' and status = 'ACTIVE' order by user_id limit 1`, x.cid).Scan(&student); err != nil {
		r.t.Fatalf("sinh viên của lớp: %v", err)
	}
	mt := func(version int64, who string) string {
		return fmt.Sprintf(`{"assignment_id":%q,"student_id":%q,"version":%d}`, asg, who, version)
	}
	r.must(call{method: "PUT", path: sub + "/" + id1 + "/match", token: x.gv, body: mt(v1+1, student)}, 409)
	r.must(call{method: "PUT", path: sub + "/" + id1 + "/match", token: x.gv, body: mt(v1, uuid.NewString())}, 422)
	r.must(call{method: "PUT", path: sub + "/" + id1 + "/match", body: mt(v1, student)}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "PUT", path: sub + "/" + id1 + "/match", token: who, body: mt(v1, student)}, 403)
	}
	r.must(call{method: "PUT", path: sub + "/00000000-0000-7000-8000-000000000000/match", token: x.gv, body: mt(v1, student)}, 404)
	r.must(call{method: "PUT", path: sub + "/" + id1 + "/match", token: x.ta, body: mt(v1, student)}, 200)
	r.must(call{method: "PUT", path: sub + "/" + id1 + "/match", token: x.ta, body: mt(v1, student)}, 409) // đã gán

	// 35 / 36: bỏ qua và hoàn tác.
	other := upload(x.ta, "bai-khac.txt", []byte("Một bài khác không MSSV.\n"), true)
	r.must(up(x.ta, A, other.UploadID), 200)
	id2, v2 := list()
	ds := func(version int64, reason string) string {
		return fmt.Sprintf(`{"reason":%q,"version":%d}`, reason, version)
	}
	r.must(call{method: "POST", path: sub + "/" + id2 + "/dismiss", token: x.gv, body: ds(v2, "")}, 422)
	r.must(call{method: "POST", path: sub + "/" + id2 + "/dismiss", token: x.gv, body: ds(v2+1, "Nộp nhầm lớp")}, 409)
	r.must(call{method: "POST", path: sub + "/" + id2 + "/dismiss", body: ds(v2, "x")}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "POST", path: sub + "/" + id2 + "/dismiss", token: who, body: ds(v2, "x")}, 403)
	}
	r.must(call{method: "POST", path: sub + "/00000000-0000-7000-8000-000000000000/dismiss", token: x.gv, body: ds(v2, "x")}, 404)
	r.must(call{method: "POST", path: sub + "/" + id2 + "/dismiss", token: x.ta, body: ds(v2, "Nộp nhầm lớp")}, 204)
	r.must(call{method: "POST", path: sub + "/" + id2 + "/dismiss", token: x.ta, body: ds(v2, "Nộp nhầm lớp")}, 409)
	r.must(call{method: "DELETE", path: sub + "/" + id2 + "/dismiss"}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "DELETE", path: sub + "/" + id2 + "/dismiss", token: who}, 403)
	}
	r.must(call{method: "DELETE", path: sub + "/00000000-0000-7000-8000-000000000000/dismiss", token: x.gv}, 404)
	r.must(call{method: "DELETE", path: sub + "/" + id2 + "/dismiss", token: x.gv}, 204)
	r.must(call{method: "DELETE", path: sub + "/" + id2 + "/dismiss", token: x.gv}, 409) // chưa bỏ qua
	// US-P7-07: hàng đợi chấm (grade-all, grade, retry) với mọi status đã khai báo.
	ga := A + "/grade-all"
	r.must(call{method: "POST", path: ga, headers: x.idem()}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "POST", path: ga, token: who, headers: x.idem()}, 403)
	}
	r.must(call{method: "POST", path: base + "/00000000-0000-7000-8000-000000000000/grade-all", token: x.gv, headers: x.idem()}, 404)
	r.must(call{method: "POST", path: ga, token: x.ta, headers: x.idem()}, 202) // id1 đã MATCHED, chưa có kết quả
	var student2 string
	if err := db.QueryRow(ctx, `select user_id from enrollments where course_id = $1 and role_in_course = 'STUDENT' and status = 'ACTIVE' and user_id <> $2 order by user_id limit 1`, x.cid, student).Scan(&student2); err != nil {
		r.t.Fatalf("sinh viên thứ hai: %v", err)
	}
	third := upload(x.ta, "bai-ba.txt", []byte("Bài thứ ba không MSSV.\n"), true)
	r.must(up(x.ta, A, third.UploadID), 200)
	id3, v3 := list()
	r.must(call{method: "PUT", path: sub + "/" + id3 + "/match", token: x.gv, body: mt(v3, student2)}, 200)
	g := func(id string) string { return sub + "/" + id + "/grade" }
	r.must(call{method: "POST", path: g(id3), headers: x.idem()}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "POST", path: g(id3), token: who, headers: x.idem()}, 403)
	}
	r.must(call{method: "POST", path: g("00000000-0000-7000-8000-000000000000"), token: x.gv, headers: x.idem()}, 404)
	r.must(call{method: "POST", path: g(id3), token: x.gv, headers: x.idem()}, 202)
	r.must(call{method: "POST", path: g(id1), token: x.gv, headers: x.idem()}, 409) // đang xếp hàng
	r.must(call{method: "POST", path: ga, token: x.gv, headers: x.idem()}, 200)     // không còn bài chưa chấm
	var res string
	if err := db.QueryRow(ctx, `update grading_results set run_state = 'FAILED', error_code = 'LLM_UNAVAILABLE' where submission_id = $1 returning id`, id1).Scan(&res); err != nil {
		r.t.Fatalf("đặt FAILED: %v", err)
	}
	rt := strings.TrimSuffix(base, "/assignments") + "/grading-results/" + res + "/retry"
	r.must(call{method: "POST", path: rt, headers: x.idem()}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "POST", path: rt, token: who, headers: x.idem()}, 403)
	}
	r.must(call{method: "POST", path: strings.TrimSuffix(base, "/assignments") + "/grading-results/00000000-0000-7000-8000-000000000000/retry", token: x.gv, headers: x.idem()}, 404)
	r.must(call{method: "POST", path: rt, token: x.ta, headers: x.idem()}, 202)
	r.must(call{method: "POST", path: rt, token: x.ta, headers: x.idem()}, 409) // đã QUEUED lại
	// US-P7-08: chi tiết / sửa / duyệt kết quả chấm với mọi status đã khai báo.
	var rubricID string
	if err := db.QueryRow(ctx, `select rubric->0->>'id' from assignments where id = $1`, asg).Scan(&rubricID); err != nil {
		r.t.Fatalf("rubric: %v", err)
	}
	var done string
	if err := db.QueryRow(ctx, `update grading_results set run_state = 'DONE', error_code = null, total = 5, ai_total_before_late = 5, ai_scores = jsonb_build_object($2::text, '5.00'),
		criteria = jsonb_build_array(jsonb_build_object('id', $2::text, 'score', '5.00', 'comment', '', 'edited', false, 'evidence', '[]'::jsonb)) where submission_id = $1 returning id`, id3, rubricID).Scan(&done); err != nil {
		r.t.Fatalf("đặt DONE: %v", err)
	}
	gr := strings.TrimSuffix(base, "/assignments") + "/grading-results/"
	version := func(id string) int {
		var v int
		if err := db.QueryRow(ctx, `select version from grading_results where id = $1`, id).Scan(&v); err != nil {
			r.t.Fatalf("version: %v", err)
		}
		return v
	}
	r.must(call{method: "GET", path: gr + done}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "GET", path: gr + done, token: who}, 403)
	}
	r.must(call{method: "GET", path: gr + "00000000-0000-7000-8000-000000000000", token: x.gv}, 404)
	r.must(call{method: "GET", path: gr + done, token: x.ta}, 200)
	edit := func(id string, v int, score string) call {
		return call{method: "PUT", path: gr + id, token: x.ta, body: fmt.Sprintf(`{"version":%d,"criteria":[{"id":%q,"score":%s}]}`, v, rubricID, score)}
	}
	r.must(call{method: "PUT", path: gr + done, body: `{"version":1}`}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "PUT", path: gr + done, token: who, body: `{"version":1}`}, 403)
	}
	r.must(call{method: "PUT", path: gr + "00000000-0000-7000-8000-000000000000", token: x.ta, body: `{"version":1}`}, 404)
	r.must(edit(done, version(done), `"5.1"`), 422) // SCORE_STEP
	r.must(edit(done, version(done)+7, `"6"`), 409) // VERSION_CONFLICT
	r.must(edit(res, version(res), `"6"`), 409)     // NOT_READY: đang QUEUED sau retry
	r.must(edit(done, version(done), `7.25`), 200)  // số JSON, bước 0,25
	r.must(call{method: "POST", path: gr + done + "/review", body: `{"version":1}`}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "POST", path: gr + done + "/review", token: who, body: `{"version":1}`}, 403)
	}
	r.must(call{method: "POST", path: gr + "00000000-0000-7000-8000-000000000000/review", token: x.ta, body: `{"version":1}`}, 404)
	r.must(call{method: "POST", path: gr + done + "/review", token: x.ta, body: fmt.Sprintf(`{"version":%d}`, version(done)+9)}, 409)
	r.must(call{method: "POST", path: gr + done + "/review", token: x.ta, body: fmt.Sprintf(`{"version":%d}`, version(done))}, 200)
	r.must(call{method: "POST", path: gr + done + "/review", token: x.ta, body: fmt.Sprintf(`{"version":%d}`, version(done))}, 200) // idempotent
	if _, err := db.Exec(ctx, `update grading_results set run_state = 'FAILED', error_code = 'LLM_UNAVAILABLE', criteria = jsonb_build_array(jsonb_build_object('id', $2::text, 'score', null, 'comment', '', 'edited', false, 'evidence', '[]'::jsonb)), total = null where id = $1`, res, rubricID); err != nil {
		r.t.Fatalf("đặt FAILED: %v", err)
	}
	r.must(call{method: "POST", path: gr + res + "/review", token: x.ta, body: fmt.Sprintf(`{"version":%d}`, version(res))}, 422) // SCORES_REQUIRED
	// US-P7-09: công bố hàng loạt (chỉ TEACHER) và số liệu bài với mọi status đã khai báo.
	pub := strings.TrimSuffix(base, "/assignments") + "/grading-results/publish"
	pb := func(ids ...string) string {
		b, _ := json.Marshal(map[string]any{"result_ids": ids})
		return string(b)
	}
	r.must(call{method: "POST", path: pub, headers: x.idem(), body: pb(done)}, 401)
	for _, who := range []string{x.ta, x.sv, x.admin} {
		r.must(call{method: "POST", path: pub, token: who, headers: x.idem(), body: pb(done)}, 403)
	}
	r.must(call{method: "POST", path: pub, token: x.gv, headers: x.idem(), body: `{}`}, 422)
	r.must(call{method: "POST", path: pub, token: x.gv, headers: x.idem(), body: pb(res)}, 422) // NOT_REVIEWED: bài FAILED
	r.must(call{method: "POST", path: pub, token: x.gv, headers: x.idem(), body: pb(done)}, 200)
	r.must(call{method: "POST", path: pub, token: x.gv, headers: x.idem(), body: pb(done)}, 409) // ALREADY_PUBLISHED
	st := A + "/stats"
	r.must(call{method: "GET", path: st}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "GET", path: st, token: who}, 403)
	}
	r.must(call{method: "GET", path: base + "/00000000-0000-7000-8000-000000000000/stats", token: x.gv}, 404)
	r.must(call{method: "GET", path: st, token: x.ta}, 200)
}

// appealScenarios: phúc khảo bài tập (US-P7-10) với mọi status đã khai báo. Sinh viên là x.sv (id lấy từ claim `sub` của token).
func (r *runner) appealScenarios(x examRig, base string) {
	ctx := context.Background()
	db := r.rig.deps.DB
	seg := strings.Split(x.sv, ".")
	raw, err := base64.RawURLEncoding.DecodeString(seg[1])
	if err != nil {
		r.t.Fatalf("token: %v", err)
	}
	var claims struct {
		Sub string `json:"sub"`
	}
	_ = json.Unmarshal(raw, &claims)
	mk := func(code string, published bool) (aid, res string) {
		pub := "null"
		if published {
			pub = "now()"
		}
		if err := db.QueryRow(ctx, `insert into assignments (course_id, title, max_score, due_at, submission_code, created_by, published_at, rubric)
			select $1, 'BT-PK', 10, now() + interval '2 days', $2, c.created_by, now(), '[{"id":"c1","name":"Khái niệm","max_score":"4","guidance":""},{"id":"c2","name":"Ví dụ","max_score":"6","guidance":""}]'::jsonb
			from courses c where c.id = $1 returning id`, x.cid, code).Scan(&aid); err != nil {
			r.t.Fatalf("bài tập: %v", err)
		}
		var sub string
		if err := db.QueryRow(ctx, `insert into submissions (course_id, assignment_id, student_id, source, version, blob_key, file_name, file_size, file_mime, sha256, match_status, match_method, submitted_at)
			values ($1, $2, $3, 'IN_APP', 1, 'k', 'a.txt', 1, 'text/plain', 'x', 'MATCHED', 'IN_APP', now()) returning id`, x.cid, aid, claims.Sub).Scan(&sub); err != nil {
			r.t.Fatalf("bài nộp: %v", err)
		}
		status := "REVIEWED"
		if published {
			status = "PUBLISHED"
		}
		if err := db.QueryRow(ctx, `insert into grading_results (course_id, submission_id, run_state, status, criteria, total, ai_total_before_late, ai_scores, published_at)
			values ($1, $2, 'DONE', $3, '[{"id":"c1","score":"3.00","comment":"","evidence":[]},{"id":"c2","score":"4.00","comment":"","evidence":[]}]'::jsonb, 7, 7, '{"c1":"3.00","c2":"4.00"}'::jsonb, `+pub+`) returning id`, x.cid, sub, status).Scan(&res); err != nil {
			r.t.Fatalf("kết quả: %v", err)
		}
		return aid, res
	}
	aid, _ := mk("PK1", true)
	draftAid, _ := mk("PK2", false)
	A := base + "/" + aid + "/appeal"
	body := `{"criteria_ids":["c2"],"reason":"Em nghĩ ví dụ của em đã đủ ý, nhờ thầy xem lại."}`
	r.must(call{method: "POST", path: A, headers: x.idem(), body: body}, 401)
	for _, who := range []string{x.ta, x.gv, x.admin} {
		r.must(call{method: "POST", path: A, token: who, headers: x.idem(), body: body}, 403)
	}
	r.must(call{method: "POST", path: base + "/00000000-0000-7000-8000-000000000000/appeal", token: x.sv, headers: x.idem(), body: body}, 404)
	r.must(call{method: "POST", path: base + "/" + draftAid + "/appeal", token: x.sv, headers: x.idem(), body: body}, 404) // chưa công bố
	r.must(call{method: "POST", path: A, token: x.sv, headers: x.idem(), body: `{"criteria_ids":["c2"],"reason":"ngắn"}`}, 422)
	r.must(call{method: "POST", path: A, token: x.sv, headers: x.idem(), body: `{"criteria_ids":["zz"],"reason":"Em nghĩ ví dụ của em đã đủ ý, nhờ thầy xem lại."}`}, 422)
	_, out := r.must(call{method: "POST", path: A, token: x.sv, headers: x.idem(), body: body}, 201)
	var created struct {
		Appeal struct {
			ID string `json:"id"`
		} `json:"appeal"`
	}
	_ = json.Unmarshal(out, &created)
	id := created.Appeal.ID
	r.must(call{method: "POST", path: A, token: x.sv, headers: x.idem(), body: body}, 409) // APPEAL_ALREADY_FILED

	root := "/api/v1/courses/" + x.cid + "/appeals"
	r.must(call{method: "GET", path: root}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "GET", path: root, token: who}, 403)
	}
	r.must(call{method: "GET", path: root + "?status=NOPE", token: x.ta}, 422)
	r.must(call{method: "GET", path: root + "?status=OPEN", token: x.ta}, 200)
	r.must(call{method: "GET", path: root + "/" + id}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "GET", path: root + "/" + id, token: who}, 403)
	}
	r.must(call{method: "GET", path: root + "/00000000-0000-7000-8000-000000000000", token: x.gv}, 404)
	r.must(call{method: "GET", path: root + "/" + id, token: x.gv}, 200)

	D := root + "/" + id + "/decide"
	dec := func(version int, resp string, scores string) string {
		return fmt.Sprintf(`{"decision":"ADJUSTED","response":%q,"version":%d%s}`, resp, version, scores)
	}
	good := "Thầy đã xem lại: ví dụ thứ hai đủ ý."
	r.must(call{method: "POST", path: D, headers: x.idem(), body: dec(1, good, "")}, 401)
	for _, who := range []string{x.ta, x.sv, x.admin} {
		r.must(call{method: "POST", path: D, token: who, headers: x.idem(), body: dec(1, good, "")}, 403)
	}
	r.must(call{method: "POST", path: root + "/00000000-0000-7000-8000-000000000000/decide", token: x.gv, headers: x.idem(), body: dec(1, good, "")}, 404)
	r.must(call{method: "POST", path: D, token: x.gv, headers: x.idem(), body: dec(1, "ngắn", "")}, 422)
	r.must(call{method: "POST", path: D, token: x.gv, headers: x.idem(), body: dec(1, good, "")}, 422) // NO_CHANGE
	r.must(call{method: "POST", path: D, token: x.gv, headers: x.idem(), body: dec(9, good, `,"criteria_scores":[{"id":"c2","score":"5.5"}]`)}, 409)
	r.must(call{method: "POST", path: D, token: x.gv, headers: x.idem(), body: dec(1, good, `,"criteria_scores":[{"id":"c2","score":"5.5"}]`)}, 200)
	r.must(call{method: "POST", path: D, token: x.gv, headers: x.idem(), body: dec(1, good, `,"criteria_scores":[{"id":"c2","score":"5.5"}]`)}, 409) // APPEAL_ALREADY_DECIDED
}

// versionsScenarios: số phiên bản đã khớp của sinh viên ở bài (hộp thoại gán tay).
func (r *runner) versionsScenarios(x examRig, base string) {
	seg := strings.Split(x.sv, ".")
	raw, _ := base64.RawURLEncoding.DecodeString(seg[1])
	var claims struct {
		Sub string `json:"sub"`
	}
	_ = json.Unmarshal(raw, &claims)
	aid := "00000000-0000-7000-8000-000000000000"
	P := base + "/" + aid + "/versions?student_id=" + claims.Sub
	r.must(call{method: "GET", path: P}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "GET", path: P, token: who}, 403)
	}
	r.must(call{method: "GET", path: P, token: x.gv}, 404)
	r.must(call{method: "GET", path: base + "/" + aid + "/versions?student_id=zz", token: x.gv}, 422)
	_, out := r.must(call{method: "GET", path: base, token: x.gv}, 200)
	var list struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	_ = json.Unmarshal(out, &list)
	if len(list.Items) == 0 {
		r.t.Fatalf("không có bài tập để đếm phiên bản")
	}
	r.must(call{method: "GET", path: base + "/" + list.Items[0].ID + "/versions?student_id=" + claims.Sub, token: x.ta}, 200)
}

// integrationScenarios: cấu hình IMAP / SMTP, Test, Đồng bộ ngay, thư đã nhận (US-P7-05) với mọi status đã khai báo.
func (r *runner) integrationScenarios(x examRig) {
	ctx := context.Background()
	root := "/api/v1/courses/" + x.cid
	imap := root + "/integrations/imap"
	body := func(version string, password string, host string) string {
		return `{"host":"` + host + `","port":1,"security":"NONE","username":"nop@truong.edu","password":"` + password + `","mailbox":"INBOX","poll_seconds":60,"enabled":true` + version + `}`
	}
	r.must(call{method: "GET", path: imap}, 401)
	for _, who := range []string{x.ta, x.sv} {
		r.must(call{method: "GET", path: imap, token: who}, 403)
	}
	r.must(call{method: "GET", path: imap, token: x.gv}, 200) // chưa cấu hình
	r.must(call{method: "POST", path: imap + "/test", token: x.admin}, 404)
	r.must(call{method: "POST", path: imap + "/sync-now", token: x.gv, headers: x.idem()}, 404)
	r.must(call{method: "PUT", path: imap, body: body("", "mat-khau-1", "127.0.0.1")}, 401)
	for _, who := range []string{x.gv, x.ta, x.sv} {
		r.must(call{method: "PUT", path: imap, token: who, body: body("", "mat-khau-1", "127.0.0.1")}, 403)
	}
	r.must(call{method: "PUT", path: imap, token: x.admin, body: body("", "", "127.0.0.1")}, 422) // lần đầu cần mật khẩu
	r.must(call{method: "PUT", path: imap, token: x.admin, body: body("", "mat-khau-1", "127.0.0.1")}, 200)
	r.must(call{method: "PUT", path: imap, token: x.admin, body: body(`,"version":9`, "", "127.0.0.1")}, 409)
	_, out := r.must(call{method: "GET", path: imap, token: x.admin}, 200)
	if strings.Contains(string(out), "mat-khau-1") {
		r.t.Fatalf("GET cấu hình IMAP lộ mật khẩu")
	}
	r.must(call{method: "POST", path: imap + "/test"}, 401)
	for _, who := range []string{x.gv, x.ta, x.sv} {
		r.must(call{method: "POST", path: imap + "/test", token: who}, 403)
	}
	r.must(call{method: "POST", path: imap + "/test", token: x.admin}, 200) // ok=false UNREACHABLE (cổng 1)

	sync := imap + "/sync-now"
	r.must(call{method: "POST", path: sync, headers: x.idem()}, 401)
	for _, who := range []string{x.admin, x.sv} {
		r.must(call{method: "POST", path: sync, token: who, headers: x.idem()}, 403)
	}
	r.must(call{method: "POST", path: sync, token: x.ta, headers: x.idem()}, 202)
	r.must(call{method: "POST", path: sync, token: x.ta, headers: x.idem()}, 202) // đang chạy: cùng job_id
	if err := r.rig.deps.Redis.Del(ctx, "ep:imapsync:"+x.cid+":job").Err(); err != nil {
		r.t.Fatalf("xoá khoá đang chạy: %v", err)
	}
	r.must(call{method: "POST", path: sync, token: x.ta, headers: x.idem()}, 429) // giãn cách 30 giây / lớp

	smtp := root + "/integrations/smtp"
	r.must(call{method: "GET", path: smtp}, 401)
	for _, who := range []string{x.ta, x.sv} {
		r.must(call{method: "GET", path: smtp, token: who}, 403)
	}
	r.must(call{method: "GET", path: smtp, token: x.admin}, 200)
	r.must(call{method: "POST", path: smtp + "/test"}, 401)
	for _, who := range []string{x.gv, x.sv} {
		r.must(call{method: "POST", path: smtp + "/test", token: who}, 403)
	}
	r.must(call{method: "POST", path: smtp + "/test", token: x.admin}, 202)

	in := root + "/mail-inbound"
	r.must(call{method: "GET", path: in}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "GET", path: in, token: who}, 403)
	}
	r.must(call{method: "GET", path: in + "?status=NOPE", token: x.ta}, 422)
	r.must(call{method: "GET", path: in + "?status=IGNORED", token: x.ta}, 200)
}

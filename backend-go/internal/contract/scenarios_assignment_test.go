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
	r.must(call{method: "GET", path: sub, token: x.gv}, 422)
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
}

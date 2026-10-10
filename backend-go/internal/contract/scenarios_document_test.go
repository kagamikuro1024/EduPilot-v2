package contract

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// documentScenarios: thao tác 1, 2, 9, 10, 11 của SRS FEAT-docs-calendar 6 (tải lên, hoàn tất, thử lại, lập chỉ mục lại — US-P8-01) với mọi status đã khai báo.
// Tệp được PUT thật lên MinIO bằng URL ký sẵn.
func (r *runner) documentScenarios(x examRig) {
	db := r.rig.deps.DB
	ctx := context.Background()
	base := "/api/v1/courses/" + x.cid
	pdf := func() []byte { return []byte("%PDF-1.4 " + uuid.NewString()) }
	sum := func(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
	presignBody := func(name, mime string, b []byte) string {
		raw, _ := json.Marshal(map[string]any{"purpose": "document", "filename": name, "mime_type": mime, "size_bytes": len(b), "sha256": sum(b)})
		return string(raw)
	}
	const pdfMime = "application/pdf"
	put := func(url, mime string, b []byte) {
		req, _ := http.NewRequest(http.MethodPut, url, bytes.NewReader(b))
		req.Header.Set("Content-Type", mime)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			r.t.Fatalf("PUT tệp: %v", err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			r.t.Fatalf("PUT tệp → %d", resp.StatusCode)
		}
	}
	type pres struct {
		UploadID string `json:"upload_id"`
		URL      string `json:"url"`
	}
	upload := func(token string, b []byte, doPut bool) pres {
		_, raw := r.must(call{method: "POST", path: base + "/uploads/presign", token: token, body: presignBody("bai-giang.pdf", pdfMime, b)}, 200)
		var p pres
		_ = json.Unmarshal(raw, &p)
		if doPut {
			put(p.URL, pdfMime, b)
		}
		return p
	}

	// 1: presign.
	pre := base + "/uploads/presign"
	body := pdf()
	good := presignBody("bai-giang.pdf", pdfMime, body)
	r.must(call{method: "POST", path: pre, token: x.ta, body: good}, 200)
	r.must(call{method: "POST", path: pre, body: good}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "POST", path: pre, token: who, body: good}, 403)
	}
	r.must(call{method: "POST", path: "/api/v1/courses/khong-phai-uuid/uploads/presign", token: x.gv, body: good}, 404)
	r.must(call{method: "POST", path: "/api/v1/courses/" + x.arch + "/uploads/presign", token: x.gv, body: good}, 409)
	r.must(call{method: "POST", path: pre, token: x.gv, body: presignBody("x.exe", "application/octet-stream", body)}, 422)
	r.must(call{method: "POST", path: pre, token: x.gv, body: `{"purpose":"document","filename":"../x.pdf","mime_type":"application/pdf","size_bytes":10,"sha256":"` + sum(body) + `"}`}, 422)

	// 2: hoàn tất.
	comp := base + "/uploads/complete"
	b1 := pdf()
	p1 := upload(x.gv, b1, true)
	cbody := func(id string) string {
		return `{"upload_id":"` + id + `","title":"Bài giảng tuần 1","type":"LECTURE","week_no":1}`
	}
	_, raw := r.must(call{method: "POST", path: comp, token: x.gv, headers: x.idem(), body: cbody(p1.UploadID)}, 202)
	var out struct {
		JobID    string `json:"job_id"`
		Document struct {
			ID string `json:"id"`
		} `json:"document"`
	}
	_ = json.Unmarshal(raw, &out)
	r.must(call{method: "POST", path: comp, headers: x.idem(), body: cbody(p1.UploadID)}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "POST", path: comp, token: who, headers: x.idem(), body: cbody(p1.UploadID)}, 403)
	}
	r.must(call{method: "POST", path: comp, token: x.gv, headers: x.idem(), body: cbody(uuid.NewString())}, 404)
	r.must(call{method: "POST", path: comp, token: x.gv, body: cbody(p1.UploadID)}, 422)                     // thiếu Idempotency-Key
	p1b := upload(x.gv, b1, true)                                                                            // cùng nội dung → trùng
	r.must(call{method: "POST", path: comp, token: x.gv, headers: x.idem(), body: cbody(p1b.UploadID)}, 409) // DOCUMENT_DUPLICATE
	p2 := upload(x.gv, pdf(), false)                                                                         // chưa PUT
	r.must(call{method: "POST", path: comp, token: x.gv, headers: x.idem(), body: cbody(p2.UploadID)}, 422)  // UPLOAD_INCOMPLETE
	r.must(call{method: "POST", path: comp, token: x.gv, headers: x.idem(), body: `{"upload_id":"` + p1.UploadID + `","title":"","type":"LECTURE"}`}, 422)

	// 9: thử lại (chỉ FAILED).
	doc := base + "/documents/" + out.Document.ID
	r.must(call{method: "POST", path: doc + "/retry", token: x.gv}, 409) // đang QUEUED
	if _, err := db.Exec(ctx, `update documents set status='FAILED', error='x' where id=$1`, out.Document.ID); err != nil {
		r.t.Fatal(err)
	}
	// cùng Idempotency-Key → cùng phản hồi, không tạo việc thứ hai (luật 14; QC BUG-2)
	rk := x.idem()
	_, rb1 := r.must(call{method: "POST", path: doc + "/retry", token: x.ta, headers: rk}, 202)
	h2, rb2 := r.must(call{method: "POST", path: doc + "/retry", token: x.ta, headers: rk}, 202)
	if string(rb1) != string(rb2) || h2.Get("Idempotent-Replayed") != "true" {
		r.t.Fatalf("retry cùng khoá: %s ≠ %s (replayed=%q)", rb1, rb2, h2.Get("Idempotent-Replayed"))
	}
	r.must(call{method: "POST", path: doc + "/retry"}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "POST", path: doc + "/retry", token: who}, 403)
	}
	r.must(call{method: "POST", path: base + "/documents/" + uuid.NewString() + "/retry", token: x.gv}, 404)

	// 10: lập chỉ mục lại một tài liệu (chỉ READY).
	r.must(call{method: "POST", path: doc + "/reindex", token: x.gv}, 409) // lại QUEUED sau retry
	if _, err := db.Exec(ctx, `update documents set status='READY' where id=$1`, out.Document.ID); err != nil {
		r.t.Fatal(err)
	}
	r.must(call{method: "POST", path: doc + "/reindex", token: x.ta}, 202)
	r.must(call{method: "POST", path: doc + "/reindex"}, 401)
	r.must(call{method: "POST", path: doc + "/reindex", token: x.sv}, 403)
	r.must(call{method: "POST", path: base + "/documents/" + uuid.NewString() + "/reindex", token: x.gv}, 404)

	// 11: lập chỉ mục lại cả lớp (chỉ Giảng viên).
	all := base + "/documents/reindex"
	ak := x.idem()
	var before, after int
	_ = db.QueryRow(ctx, `select count(*) from jobs where kind='document.reindex_all'`).Scan(&before)
	_, ab1 := r.must(call{method: "POST", path: all, token: x.gv, headers: ak}, 202)
	_, ab2 := r.must(call{method: "POST", path: all, token: x.gv, headers: ak}, 202)
	_ = db.QueryRow(ctx, `select count(*) from jobs where kind='document.reindex_all'`).Scan(&after)
	if string(ab1) != string(ab2) || after != before+1 {
		r.t.Fatalf("reindex cả lớp cùng khoá: %s ≠ %s, jobs %d → %d (cần +1)", ab1, ab2, before, after)
	}
	r.must(call{method: "POST", path: all}, 401)
	for _, who := range []string{x.ta, x.sv, x.admin} {
		r.must(call{method: "POST", path: all, token: who}, 403)
	}
	r.must(call{method: "POST", path: "/api/v1/courses/" + x.arch + "/documents/reindex", token: x.gv}, 409)
	r.must(call{method: "POST", path: "/api/v1/courses/khong-phai-uuid/documents/reindex", token: x.gv}, 404)

	// 1 (tiếp): hạn 10 lượt presign / phút / người → 429.
	got429 := false
	for range 30 {
		if st, _, _ := r.do(call{method: "POST", path: pre, token: x.ta, body: good}); st == http.StatusTooManyRequests {
			got429 = true
			break
		}
	}
	if !got429 {
		r.t.Fatal("presign: không thấy 429 sau 30 lượt")
	}
}

// docMgmtScenarios: thao tác 3–8, 12–15 của SRS FEAT-docs-calendar 6 (quản lý tài liệu của Staff + thư viện của sinh viên — US-P8-02) với mọi status đã khai báo.
// Tài liệu / đoạn chèn thẳng vào DB; tệp thật được PUT lên MinIO để kiểm xem trước / tải xuống.
func (r *runner) docMgmtScenarios(x examRig) {
	db := r.rig.deps.DB
	ctx := context.Background()
	base := "/api/v1/courses/" + x.cid
	D, L := base+"/documents", base+"/library"
	vec := "[" + strings.Repeat("0,", 1535) + "1]"
	mk := func(title, typ string, visible bool, file bool) (string, string) {
		key := "courses/" + x.cid + "/documents/" + uuid.NewString() + "/x.pdf"
		if file {
			if err := r.rig.deps.Blob.Put(ctx, key, strings.NewReader("%PDF-1.4 x"), 10, "application/pdf"); err != nil {
				r.t.Fatal(err)
			}
		}
		var id, chunk string
		if err := db.QueryRow(ctx, `insert into documents (course_id, title, type, filename, mime_type, sha256, blob_key, status, visible_to_students, use_for_rag, page_count) values ($1, $2, $3::document_type, 'x.pdf', 'application/pdf', $4, $5, 'READY', $6, true, 2) returning id`,
			x.cid, title, typ, hex.EncodeToString(sha256.New().Sum([]byte(uuid.NewString())))[:64], key, visible).Scan(&id); err != nil {
			r.t.Fatal(err)
		}
		if err := db.QueryRow(ctx, `insert into content_chunks (document_id, course_ids, ord, page_no, text, embedding) values ($1, array[$2::uuid], 0, 1, 'Cảnh báo học vụ khi điểm dưới 1,2', $3::vector) returning id`, id, x.cid, vec).Scan(&chunk); err != nil {
			r.t.Fatal(err)
		}
		return id, chunk
	}
	doc, chunk := mk("Quy chế học vụ", "LECTURE", true, true)
	gone, _ := mk("Mất tệp", "LECTURE", true, false)
	key, _ := mk("Đáp án tuần 1", "ANSWER_KEY", false, true)

	// 3: danh sách (Staff).
	r.must(call{method: "GET", path: D + "?type=LECTURE&status=READY&q=quy%20che&limit=5", token: x.ta}, 200)
	r.must(call{method: "GET", path: D, token: x.gv}, 200)
	r.must(call{method: "GET", path: D}, 401)
	r.must(call{method: "GET", path: D, token: x.sv}, 403)
	r.must(call{method: "GET", path: D, token: x.admin}, 403)
	r.must(call{method: "GET", path: D + "?limit=500", token: x.ta}, 422)
	// 12: thống kê.
	r.must(call{method: "GET", path: D + "/stats", token: x.ta}, 200)
	r.must(call{method: "GET", path: D + "/stats"}, 401)
	r.must(call{method: "GET", path: D + "/stats", token: x.sv}, 403)
	// 4: chi tiết.
	r.must(call{method: "GET", path: D + "/" + doc, token: x.ta}, 200)
	r.must(call{method: "GET", path: D + "/" + doc}, 401)
	r.must(call{method: "GET", path: D + "/" + doc, token: x.sv}, 403)
	r.must(call{method: "GET", path: D + "/" + uuid.NewString(), token: x.ta}, 404)
	// 7: đoạn.
	r.must(call{method: "GET", path: D + "/" + doc + "/chunks?limit=10", token: x.ta}, 200)
	r.must(call{method: "GET", path: D + "/" + doc + "/chunks"}, 401)
	r.must(call{method: "GET", path: D + "/" + doc + "/chunks", token: x.sv}, 403)
	r.must(call{method: "GET", path: D + "/" + uuid.NewString() + "/chunks", token: x.ta}, 404)
	r.must(call{method: "GET", path: D + "/" + doc + "/chunks?cursor=x", token: x.ta}, 422)
	// 8: sửa đoạn (nhà cung cấp `fake` nhúng được; 503 khi nhúng lỗi nằm ở exempt.go).
	C := D + "/" + doc + "/chunks/" + chunk
	r.must(call{method: "PATCH", path: C, token: x.ta, body: `{"text":"Chữ mới"}`}, 200)
	r.must(call{method: "PATCH", path: C, body: `{"text":"x"}`}, 401)
	r.must(call{method: "PATCH", path: C, token: x.sv, body: `{"text":"x"}`}, 403)
	r.must(call{method: "PATCH", path: D + "/" + doc + "/chunks/" + uuid.NewString(), token: x.ta, body: `{"text":"x"}`}, 404)
	r.must(call{method: "PATCH", path: C, token: x.ta, body: `{"text":""}`}, 422)
	var ver int
	if err := db.QueryRow(ctx, `select version from documents where id=$1`, doc).Scan(&ver); err != nil {
		r.t.Fatal(err)
	}
	// 5: sửa tài liệu.
	r.must(call{method: "PATCH", path: D + "/" + doc, token: x.ta, body: fmt.Sprintf(`{"title":"Quy chế mới","version":%d}`, ver)}, 200)
	r.must(call{method: "PATCH", path: D + "/" + doc, token: x.ta, body: fmt.Sprintf(`{"title":"Cũ","version":%d}`, ver)}, 409) // VERSION_CONFLICT
	r.must(call{method: "PATCH", path: D + "/" + doc, body: `{"title":"x","version":1}`}, 401)
	r.must(call{method: "PATCH", path: D + "/" + doc, token: x.sv, body: `{"title":"x","version":1}`}, 403)
	r.must(call{method: "PATCH", path: D + "/" + uuid.NewString(), token: x.ta, body: `{"title":"x","version":1}`}, 404)
	r.must(call{method: "PATCH", path: D + "/" + key, token: x.ta, body: `{"visible_to_students":true,"version":1}`}, 422) // ANSWER_KEY_NOT_VISIBLE
	r.must(call{method: "PATCH", path: "/api/v1/courses/" + x.arch + "/documents/" + uuid.NewString(), token: x.gv, body: `{"title":"x","version":1}`}, 409)
	// 13–15: thư viện của sinh viên.
	h, _ := r.must(call{method: "GET", path: L + "?q=quy%20che&limit=5", token: x.sv}, 200)
	if tag := h.Get("ETag"); tag != "" {
		r.must(call{method: "GET", path: L + "?q=quy%20che&limit=5", token: x.sv, headers: map[string]string{"If-None-Match": tag}}, 304)
	}
	r.must(call{method: "GET", path: L}, 401)
	r.must(call{method: "GET", path: L, token: x.ta}, 403)
	r.must(call{method: "GET", path: L + "?week=99", token: x.sv}, 422)
	r.must(call{method: "GET", path: L + "/" + doc, token: x.sv}, 200)
	r.must(call{method: "GET", path: L + "/" + doc}, 401)
	r.must(call{method: "GET", path: L + "/" + doc, token: x.gv}, 403)
	r.must(call{method: "GET", path: L + "/" + key, token: x.sv}, 404) // ANSWER_KEY không bao giờ tới sinh viên
	r.must(call{method: "GET", path: L + "/" + doc + "/download", token: x.sv}, 200)
	r.must(call{method: "GET", path: L + "/" + doc + "/download"}, 401)
	r.must(call{method: "GET", path: L + "/" + doc + "/download", token: x.ta}, 403)
	r.must(call{method: "GET", path: L + "/" + key + "/download", token: x.sv}, 404)
	r.must(call{method: "GET", path: L + "/" + gone + "/download", token: x.sv}, 404) // FILE_GONE
	// 6 + impact: xoá (Giảng viên).
	r.must(call{method: "GET", path: D + "/" + gone + "/impact", token: x.gv}, 200)
	r.must(call{method: "GET", path: D + "/" + gone + "/impact"}, 401)
	r.must(call{method: "GET", path: D + "/" + gone + "/impact", token: x.ta}, 403)
	r.must(call{method: "GET", path: D + "/" + uuid.NewString() + "/impact", token: x.gv}, 404)
	r.must(call{method: "DELETE", path: D + "/" + gone}, 401)
	r.must(call{method: "DELETE", path: D + "/" + gone, token: x.ta}, 403)
	r.must(call{method: "DELETE", path: D + "/" + gone, token: x.gv}, 204)
	r.must(call{method: "DELETE", path: D + "/" + gone, token: x.gv}, 404)
	// tài liệu chia sẻ từ lớp khác (chỉ đọc): sửa đoạn / xoá → 409 DOCUMENT_SHARED_READONLY.
	var shared, sharedChunk string
	if err := db.QueryRow(ctx, `insert into documents (course_id, title, type, status, sha256) values ($1, 'Chia sẻ', 'LECTURE', 'READY', $2) returning id`, x.arch, hex.EncodeToString(sha256.New().Sum([]byte(uuid.NewString())))[:64]).Scan(&shared); err != nil {
		r.t.Fatal(err)
	}
	if err := db.QueryRow(ctx, `insert into content_chunks (document_id, course_ids, ord, text) values ($1, array[$2::uuid, $3::uuid], 0, 'nội dung') returning id`, shared, x.arch, x.cid).Scan(&sharedChunk); err != nil {
		r.t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `insert into document_courses (document_id, course_id) values ($1, $2)`, shared, x.cid); err != nil {
		r.t.Fatal(err)
	}
	r.must(call{method: "PATCH", path: D + "/" + shared + "/chunks/" + sharedChunk, token: x.ta, body: `{"text":"x"}`}, 409)
	r.must(call{method: "DELETE", path: D + "/" + shared, token: x.gv}, 409)
}

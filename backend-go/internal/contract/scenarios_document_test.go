package contract

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"

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
	r.must(call{method: "POST", path: doc + "/retry", token: x.ta}, 202)
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
	r.must(call{method: "POST", path: all, token: x.gv}, 202)
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

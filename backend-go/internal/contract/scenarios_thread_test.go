package contract

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
)

// threadScenarios: thao tác 12–20 + from-draft (#11) của SRS FEAT-private-chat-pii 6 (Threads — US-P3-06) với mọi status đã khai báo.
// Bài AI được chèn thẳng vào DB (việc AI trả lời chạy ở worker, có test riêng ở internal/thread).
func (r *runner) threadScenarios(x examRig) {
	db := r.rig.deps.DB
	ctx := context.Background()
	var sv, other uuid.UUID
	mk := func() uuid.UUID {
		var id uuid.UUID
		if err := db.QueryRow(ctx, `insert into users (email, full_name, role, status, password_hash, email_verified_at) values ($1, 'Sinh Viên Threads', 'STUDENT', 'ACTIVE', 'x', now()) returning id`, "ct-th-"+uuid.NewString()[:8]+"@example.test").Scan(&id); err != nil {
			r.t.Fatal(err)
		}
		return id
	}
	sv, other = mk(), mk()
	for _, c := range []string{x.cid, x.arch} {
		for _, u := range []uuid.UUID{sv, other} {
			if _, err := db.Exec(ctx, `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN')`, c, u); err != nil {
				r.t.Fatal(err)
			}
		}
	}
	tsv, tother := r.rig.token(r.t, sv.String(), auth.RoleStudent), r.rig.token(r.t, other.String(), auth.RoleStudent)
	base := "/api/v1/courses/" + x.cid
	T := base + "/threads"
	idem := func() map[string]string { return map[string]string{"Idempotency-Key": uuid.NewString()} }
	const q = `{"title":"Điều 5 quy chế","body":"Quy định thi lại ở Điều 5 là gì?"}`

	// 15: đăng.
	_, raw := r.must(call{method: "POST", path: T, token: tsv, headers: idem(), body: q}, 201)
	var created struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
	}
	_ = json.Unmarshal(raw, &created)
	tid := created.Thread.ID
	r.must(call{method: "POST", path: T, headers: idem(), body: q}, 401)
	for _, who := range []string{x.admin} {
		r.must(call{method: "POST", path: T, token: who, headers: idem(), body: q}, 403)
	}
	r.must(call{method: "POST", path: "/api/v1/courses/" + x.arch + "/threads", token: tsv, headers: idem(), body: q}, 409)
	r.must(call{method: "POST", path: T, token: tsv, headers: idem(), body: `{"title":"Hỏi","body":"Email của em là giang.vh@sv.edu.vn nhé"}`}, 422) // PII_DETECTED
	r.must(call{method: "POST", path: T, token: tsv, headers: idem(), body: `{"title":"","body":"x"}`}, 422)
	r.must(call{method: "POST", path: "/api/v1/courses/khong-phai-uuid/threads", token: tsv, headers: idem(), body: q}, 404)

	// 12: danh sách.
	r.must(call{method: "GET", path: T + "?week=3&state=none&q=quy%20che", token: tother}, 200)
	r.must(call{method: "GET", path: T}, 401)
	r.must(call{method: "GET", path: T, token: x.admin}, 403)
	r.must(call{method: "GET", path: "/api/v1/courses/khong-phai-uuid/threads", token: tsv}, 404)
	r.must(call{method: "GET", path: T + "?state=sai", token: tsv}, 422)

	// 13: chi tiết.
	r.must(call{method: "GET", path: T + "/" + tid, token: tother}, 200)
	r.must(call{method: "GET", path: T + "/" + tid}, 401)
	r.must(call{method: "GET", path: T + "/" + tid, token: x.admin}, 403)
	r.must(call{method: "GET", path: T + "/" + uuid.NewString(), token: tother}, 404)

	// 14: precheck (không ghi gì).
	P := T + "/precheck"
	r.must(call{method: "POST", path: P, token: tsv, body: `{"title":"Hỏi","body":"Giải thích Điều 5"}`}, 200)
	r.must(call{method: "POST", path: P, body: `{"body":"x"}`}, 401)
	r.must(call{method: "POST", path: P, token: x.admin, body: `{"body":"x"}`}, 403)
	r.must(call{method: "POST", path: P, token: tsv, body: `{"body":5}`}, 422)
	r.must(call{method: "POST", path: "/api/v1/courses/khong-phai-uuid/threads/precheck", token: tsv, body: `{"body":"x"}`}, 404)
	got429 := false
	for range 125 { // cửa sổ phút cố định: 125 > 2×60 nên luôn có một cửa sổ đủ 60 lượt kể cả khi qua ranh giới phút
		if st, _, _ := r.do(call{method: "POST", path: P, token: tsv, body: `{"body":"Giải thích Điều 5"}`}); st == http.StatusTooManyRequests {
			got429 = true
			break
		}
	}
	if !got429 {
		r.t.Fatal("precheck không bị giới hạn 60 / phút")
	}

	// 16: bình luận.
	C := T + "/" + tid + "/posts"
	r.must(call{method: "POST", path: C, token: tother, headers: idem(), body: `{"body":"Mình cũng thắc mắc"}`}, 201)
	r.must(call{method: "POST", path: C, headers: idem(), body: `{"body":"x"}`}, 401)
	r.must(call{method: "POST", path: C, token: x.admin, headers: idem(), body: `{"body":"x"}`}, 403)
	r.must(call{method: "POST", path: T + "/" + uuid.NewString() + "/posts", token: tother, headers: idem(), body: `{"body":"x"}`}, 404)
	r.must(call{method: "POST", path: C, token: tother, headers: idem(), body: `{"body":"Gọi 0912345678 nhé"}`}, 422)
	r.must(call{method: "POST", path: "/api/v1/courses/" + x.arch + "/threads/" + tid + "/posts", token: tother, headers: idem(), body: `{"body":"x"}`}, 404) // thread của lớp khác
	var archThread string
	if err := db.QueryRow(ctx, `insert into forum_threads (course_id, author_id, title, body) values ($1, $2, 'Cũ', 'Câu hỏi cũ') returning id::text`, x.arch, sv).Scan(&archThread); err != nil {
		r.t.Fatal(err)
	}
	r.must(call{method: "POST", path: "/api/v1/courses/" + x.arch + "/threads/" + archThread + "/posts", token: tother, headers: idem(), body: `{"body":"x"}`}, 409) // lớp đã lưu trữ

	// 20: thread tương tự.
	S := T + "/" + tid + "/similar"
	r.must(call{method: "GET", path: S, token: tsv}, 200)
	r.must(call{method: "GET", path: S}, 401)
	r.must(call{method: "GET", path: S, token: x.admin}, 403)
	r.must(call{method: "GET", path: T + "/" + uuid.NewString() + "/similar", token: tsv}, 404)

	// 17–19: quyết định của Staff trên một bài AI (chèn thẳng vào DB).
	var pid string
	if err := db.QueryRow(ctx, `insert into forum_posts (course_id, thread_id, kind, body, verification_state, citations, confidence) values ($1, $2, 'AI', 'Theo quy chế [1].', 'PENDING', '[]', 0.8) returning id::text`, x.cid, tid).Scan(&pid); err != nil {
		r.t.Fatal(err)
	}
	A := base + "/posts/" + pid
	r.must(call{method: "POST", path: A + "/verify", token: x.ta}, 200)
	r.must(call{method: "POST", path: A + "/verify"}, 401)
	for _, who := range []string{tsv, x.admin} {
		r.must(call{method: "POST", path: A + "/verify", token: who}, 403)
	}
	r.must(call{method: "POST", path: base + "/posts/" + uuid.NewString() + "/verify", token: x.ta}, 404)
	_, b := r.must(call{method: "GET", path: T + "/" + tid, token: x.gv}, 200)
	var view struct {
		Posts []struct {
			ID      string `json:"id"`
			Kind    string `json:"kind"`
			Version int    `json:"version"`
		} `json:"posts"`
	}
	_ = json.Unmarshal(b, &view)
	ver := 1
	for _, p := range view.Posts {
		if p.ID == pid {
			ver = p.Version
		}
	}
	vs := func(v int) string {
		raw, _ := json.Marshal(map[string]any{"body": "Bản đã sửa.", "version": v})
		return string(raw)
	}
	r.must(call{method: "PUT", path: A + "/correct", token: x.gv, body: vs(ver - 1)}, 409) // VERSION_CONFLICT
	r.must(call{method: "PUT", path: A + "/correct", token: x.gv, body: vs(ver)}, 200)
	r.must(call{method: "PUT", path: A + "/correct", body: vs(ver)}, 401)
	r.must(call{method: "PUT", path: A + "/correct", token: tsv, body: vs(ver)}, 403)
	r.must(call{method: "PUT", path: base + "/posts/" + uuid.NewString() + "/correct", token: x.gv, body: vs(1)}, 404)
	r.must(call{method: "PUT", path: A + "/correct", token: x.gv, body: `{"body":"","version":1}`}, 422)
	r.must(call{method: "POST", path: A + "/reject", token: x.ta}, 200)
	r.must(call{method: "POST", path: A + "/reject"}, 401)
	r.must(call{method: "POST", path: A + "/reject", token: tsv}, 403)
	r.must(call{method: "POST", path: base + "/posts/" + uuid.NewString() + "/reject", token: x.ta}, 404)
	r.must(call{method: "POST", path: A + "/verify", token: x.ta}, 409) // POST_STATE_CONFLICT: bài đã bị loại
	var humanID string
	_ = db.QueryRow(ctx, `select id::text from forum_posts where thread_id=$1 and kind='HUMAN' limit 1`, tid).Scan(&humanID)
	r.must(call{method: "POST", path: base + "/posts/" + humanID + "/verify", token: x.ta}, 409)

	r.moderationScenarios(x, tsv, tother, sv, other)

	// 11: from-draft.
	F := "/api/v1/chat/sessions/from-draft"
	fd := `{"course_id":"` + x.cid + `","title":"Hỏi","body":"Em có MSSV 20229001, điểm của em thế nào?"}`
	r.must(call{method: "POST", path: F, token: tsv, headers: idem(), body: fd}, 201)
	r.must(call{method: "POST", path: F, headers: idem(), body: fd}, 401)
	r.must(call{method: "POST", path: F, token: x.ta, headers: idem(), body: fd}, 403)
	r.must(call{method: "POST", path: F, token: tsv, headers: idem(), body: `{"course_id":"` + x.arch + `","body":"x"}`}, 409)
	r.must(call{method: "POST", path: F, token: tsv, headers: idem(), body: `{"course_id":"` + x.cid + `","body":"   "}`}, 422)
}

// moderationScenarios: 17 thao tác kiểm duyệt Threads (US-P4-06) với mọi status đã khai báo, cho cả thread lẫn bình luận. Hàng chèn thẳng vào DB.
func (r *runner) moderationScenarios(x examRig, tsv, tother string, sv, other uuid.UUID) {
	db := r.rig.deps.DB
	ctx := context.Background()
	base := "/api/v1/courses/" + x.cid
	mkThread := func() string {
		var id string
		if err := db.QueryRow(ctx, `insert into forum_threads (course_id, author_id, title, body) values ($1, $2, 'Kiểm duyệt', 'Câu hỏi để kiểm duyệt') returning id::text`, x.cid, sv).Scan(&id); err != nil {
			r.t.Fatal(err)
		}
		return id
	}
	mkPost := func(tid string) string {
		var id string
		if err := db.QueryRow(ctx, `insert into forum_posts (course_id, thread_id, author_id, kind, body, verification_state, citations) values ($1, $2, $3, 'HUMAN', 'Bình luận', 'NONE', '[]') returning id::text`, x.cid, tid, sv).Scan(&id); err != nil {
			r.t.Fatal(err)
		}
		return id
	}
	ver := func(raw []byte) int {
		var m struct {
			Version int `json:"version"`
		}
		_ = json.Unmarshal(raw, &m)
		return m.Version
	}
	body := func(v int, reason string) string {
		b, _ := json.Marshal(map[string]any{"version": v, "reason": reason})
		return string(b)
	}
	const spam = `{"reason":"SPAM"}`
	for _, kind := range []string{"threads", "posts"} {
		mk := func() string {
			t := mkThread()
			if kind == "posts" {
				return mkPost(t)
			}
			return t
		}
		P := func(id, op string) string { return base + "/" + kind + "/" + id + op }

		// báo cáo + rút báo cáo
		id := mk()
		r.must(call{method: "POST", path: P(id, "/report"), token: tother, body: spam}, 201)
		r.must(call{method: "POST", path: P(id, "/report"), token: tother, body: spam}, 409) // ALREADY_REPORTED
		r.must(call{method: "POST", path: P(id, "/report"), token: tsv, body: spam}, 422)    // CANNOT_REPORT_OWN
		r.must(call{method: "POST", path: P(id, "/report"), token: tother, body: `{"reason":"KHONG_CO"}`}, 422)
		r.must(call{method: "POST", path: P(id, "/report"), body: spam}, 401)
		r.must(call{method: "POST", path: P(id, "/report"), token: x.admin, body: spam}, 403)
		r.must(call{method: "POST", path: P(uuid.NewString(), "/report"), token: tother, body: spam}, 404)
		r.must(call{method: "DELETE", path: P(id, "/report"), token: tother}, 204)
		r.must(call{method: "DELETE", path: P(id, "/report"), token: tother}, 404)
		r.must(call{method: "DELETE", path: P(id, "/report")}, 401)
		r.must(call{method: "DELETE", path: P(id, "/report"), token: x.admin}, 403)

		// ẩn / bỏ ẩn / giữ bài; rút báo cáo của bài đã ẩn = 409
		h := mk()
		r.must(call{method: "POST", path: P(h, "/report"), token: x.gv, body: spam}, 201)
		r.must(call{method: "POST", path: P(h, "/hide"), token: x.gv, body: body(1, "")}, 422)
		r.must(call{method: "POST", path: P(h, "/hide"), body: body(1, "x")}, 401)
		r.must(call{method: "POST", path: P(h, "/hide"), token: tsv, body: body(1, "x")}, 403)
		r.must(call{method: "POST", path: P(uuid.NewString(), "/hide"), token: x.gv, body: body(1, "Lạc đề")}, 404)
		_, b := r.must(call{method: "POST", path: P(h, "/hide"), token: x.gv, body: body(1, "Lạc đề")}, 200)
		v := ver(b)
		r.must(call{method: "POST", path: P(h, "/hide"), token: x.gv, body: body(v, "Lạc đề")}, 409) // đã ẩn thủ công
		r.must(call{method: "DELETE", path: P(h, "/report"), token: x.gv}, 409)                      // bài đã ẩn
		r.must(call{method: "POST", path: P(h, "/unhide"), token: x.gv, body: body(v-1, "")}, 409)   // VERSION_CONFLICT
		r.must(call{method: "POST", path: P(h, "/unhide"), body: body(v, "")}, 401)
		r.must(call{method: "POST", path: P(h, "/unhide"), token: tsv, body: body(v, "")}, 403)
		r.must(call{method: "POST", path: P(uuid.NewString(), "/unhide"), token: x.gv, body: body(1, "")}, 404)
		_, b = r.must(call{method: "POST", path: P(h, "/unhide"), token: x.gv, body: body(v, "")}, 200)
		v = ver(b)
		r.must(call{method: "POST", path: P(h, "/dismiss-reports"), token: x.gv, body: body(v-1, "")}, 409)
		r.must(call{method: "POST", path: P(h, "/dismiss-reports"), body: body(v, "")}, 401)
		r.must(call{method: "POST", path: P(h, "/dismiss-reports"), token: tsv, body: body(v, "")}, 403)
		r.must(call{method: "POST", path: P(uuid.NewString(), "/dismiss-reports"), token: x.gv, body: body(1, "")}, 404)
		r.must(call{method: "POST", path: P(h, "/dismiss-reports"), token: x.ta, body: body(v, "")}, 200)

		// sửa / xoá của chủ bài
		e := mk()
		edit := func(v int, text string) string {
			b, _ := json.Marshal(map[string]any{"title": "Tiêu đề sửa", "body": text, "version": v})
			if kind == "posts" {
				b, _ = json.Marshal(map[string]any{"body": text, "version": v})
			}
			return string(b)
		}
		r.must(call{method: "PUT", path: P(e, ""), token: tsv, body: edit(1, "Nội dung đã sửa.")}, 200)
		r.must(call{method: "PUT", path: P(e, ""), token: tsv, body: edit(1, "Nội dung đã sửa.")}, 409) // version cũ
		r.must(call{method: "PUT", path: P(e, ""), body: edit(2, "x")}, 401)
		r.must(call{method: "PUT", path: P(e, ""), token: tother, body: edit(2, "x")}, 403)
		r.must(call{method: "PUT", path: P(uuid.NewString(), ""), token: tsv, body: edit(1, "x")}, 404)
		r.must(call{method: "PUT", path: P(e, ""), token: tsv, body: edit(2, "   ")}, 422)
		r.must(call{method: "DELETE", path: P(e, "")}, 401)
		r.must(call{method: "DELETE", path: P(e, ""), token: tother}, 403)
		r.must(call{method: "DELETE", path: P(e, ""), token: tsv}, 204)
		r.must(call{method: "DELETE", path: P(e, ""), token: tsv}, 404)
	}

	// hàng chờ
	Q := base + "/moderation/queue"
	r.must(call{method: "GET", path: Q + "?status=OPEN", token: x.gv}, 200)
	r.must(call{method: "GET", path: Q + "?status=HIDDEN", token: x.ta}, 200)
	r.must(call{method: "GET", path: Q + "?status=KHAC", token: x.gv}, 422)
	r.must(call{method: "GET", path: Q}, 401)
	r.must(call{method: "GET", path: Q, token: tsv}, 403)

	// ghim: ≤ THREAD_PIN_MAX = 3 mỗi lớp
	var pins []string
	for range 4 {
		pins = append(pins, mkThread())
	}
	pin := func(id, op string) string { return base + "/threads/" + id + op }
	for _, id := range pins[:3] {
		r.must(call{method: "POST", path: pin(id, "/pin"), token: x.gv}, 204)
	}
	r.must(call{method: "POST", path: pin(pins[3], "/pin"), token: x.gv}, 409) // PIN_LIMIT
	r.must(call{method: "POST", path: pin(pins[0], "/pin")}, 401)
	r.must(call{method: "POST", path: pin(pins[0], "/pin"), token: tsv}, 403)
	r.must(call{method: "POST", path: pin(uuid.NewString(), "/pin"), token: x.gv}, 404)
	for _, id := range pins[:3] {
		r.must(call{method: "POST", path: pin(id, "/unpin"), token: x.ta}, 204)
	}
	r.must(call{method: "POST", path: pin(pins[0], "/unpin")}, 401)
	r.must(call{method: "POST", path: pin(pins[0], "/unpin"), token: tsv}, 403)
	r.must(call{method: "POST", path: pin(uuid.NewString(), "/unpin"), token: x.ta}, 404)
	_ = other
}

package contract

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
)

// chatScenarios: thao tác 1–10 của SRS FEAT-private-chat-pii 6 (chat riêng — US-P3-05) với mọi status đã khai báo. Nhà cung cấp giả trả lời thật (agent + RAG + llm).
// Dùng sinh viên riêng (không dính lượt thi IN_PROGRESS của kịch bản PE); `x.arch` là lớp đã lưu trữ.
func (r *runner) chatScenarios(x examRig) {
	db := r.rig.deps.DB
	ctx := context.Background()
	mk := func(role string) uuid.UUID {
		var id uuid.UUID
		if err := db.QueryRow(ctx, `insert into users (email, full_name, role, status, password_hash, email_verified_at) values ($1, 'Sinh Viên Chat', $2::user_role, 'ACTIVE', 'x', now()) returning id`,
			"ct-chat-"+uuid.NewString()[:8]+"@example.test", role).Scan(&id); err != nil {
			r.t.Fatalf("tạo người dùng: %v", err)
		}
		return id
	}
	enroll := func(course string, u uuid.UUID) {
		if _, err := db.Exec(ctx, `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN')`, course, u); err != nil {
			r.t.Fatal(err)
		}
	}
	a, b, rl := mk("STUDENT"), mk("STUDENT"), mk("STUDENT")
	for _, u := range []uuid.UUID{a, b, rl} {
		enroll(x.cid, u)
	}
	enroll(x.arch, a)
	sa, sb, sr := r.rig.token(r.t, a.String(), auth.RoleStudent), r.rig.token(r.t, b.String(), auth.RoleStudent), r.rig.token(r.t, rl.String(), auth.RoleStudent)
	const S = "/api/v1/chat/sessions"
	type id struct {
		ID string `json:"id"`
	}
	create := func(tok, course string) string {
		_, raw := r.must(call{method: "POST", path: S, token: tok, headers: x.idem(), body: `{"course_id":"` + course + `"}`}, 201)
		var s id
		_ = json.Unmarshal(raw, &s)
		return s.ID
	}
	send := func(tok, sid, text string, want int) []byte {
		_, raw := r.must(call{method: "POST", path: S + "/" + sid + "/messages", token: tok, headers: map[string]string{"Idempotency-Key": uuid.NewString()}, body: `{"content":"` + text + `"}`}, want)
		return raw
	}
	lastMsg := func(tok, sid string) (mid, status string) {
		_, raw := r.must(call{method: "GET", path: S + "/" + sid + "/messages", token: tok}, 200)
		var p struct {
			Items []struct{ ID, Role, StreamStatus string } `json:"items"`
		}
		_ = json.Unmarshal(raw, &p)
		for _, m := range p.Items {
			if m.Role == "ASSISTANT" {
				return m.ID, m.StreamStatus
			}
		}
		r.t.Fatal("không có tin ASSISTANT")
		return "", ""
	}

	// 2: tạo.
	sid := create(sa, x.cid)
	body := `{"course_id":"` + x.cid + `"}`
	r.must(call{method: "POST", path: S, headers: x.idem(), body: body}, 401)
	for _, who := range []string{x.ta, x.gv, x.admin} {
		r.must(call{method: "POST", path: S, token: who, headers: x.idem(), body: body}, 403)
	}
	r.must(call{method: "POST", path: S, token: sa, headers: x.idem(), body: `{"course_id":"` + x.cid + `","document_id":"` + uuid.NewString() + `"}`}, 404)
	r.must(call{method: "POST", path: S, token: sa, headers: x.idem(), body: `{"course_id":"` + x.arch + `"}`}, 409)
	r.must(call{method: "POST", path: S, token: sa, headers: x.idem(), body: `{"title":"thiếu lớp"}`}, 422)

	// 1: danh sách.
	list := S + "?course_id=" + x.cid
	r.must(call{method: "GET", path: list, token: sa}, 200)
	r.must(call{method: "GET", path: list}, 401)
	for _, who := range []string{x.ta, x.gv, x.admin} {
		r.must(call{method: "GET", path: list, token: who}, 403)
	}
	r.must(call{method: "GET", path: S, token: sa}, 422)

	// 6: gửi (SSE) — 200, 401, 403, 404, 409 (lớp lưu trữ), 422, 429.
	raw := send(sa, sid, "Xin chào", 200)
	if !strings.Contains(string(raw), "event: done") && !strings.Contains(string(raw), "event: error") {
		r.t.Fatalf("luồng SSE không có sự kiện cuối: %s", raw)
	}
	msgs := S + "/" + sid + "/messages"
	r.must(call{method: "POST", path: msgs, headers: map[string]string{"Idempotency-Key": uuid.NewString()}, body: `{"content":"x"}`}, 401)
	for _, who := range []string{x.ta, x.gv, x.admin} {
		r.must(call{method: "POST", path: msgs, token: who, headers: map[string]string{"Idempotency-Key": uuid.NewString()}, body: `{"content":"x"}`}, 403)
	}
	send(sb, sid, "x", 404)
	send(sa, sid, "   ", 422)
	var archSess uuid.UUID
	if err := db.QueryRow(ctx, `insert into chat_sessions (course_id, user_id) values ($1, $2) returning id`, x.arch, a).Scan(&archSess); err != nil {
		r.t.Fatal(err)
	}
	send(sa, archSess.String(), "x", 409)
	rs := create(sr, x.cid)
	got429 := false
	for range 45 { // cửa sổ phút cố định: 45 > 2×20 nên luôn có một cửa sổ đủ 20 tin kể cả khi qua ranh giới phút lúc máy tải nặng
		st, _, _ := r.do(call{method: "POST", path: S + "/" + rs + "/messages", token: sr, headers: map[string]string{"Idempotency-Key": uuid.NewString()}, body: `{"content":"nhanh"}`})
		if st == 429 {
			got429 = true
			break
		}
	}
	if !got429 {
		r.t.Fatal("không gặp 429 sau 45 tin")
	}

	// 5: lịch sử.
	r.must(call{method: "GET", path: msgs, token: sa}, 200)
	r.must(call{method: "GET", path: msgs}, 401)
	for _, who := range []string{x.ta, x.gv, x.admin} {
		r.must(call{method: "GET", path: msgs, token: who}, 403)
	}
	r.must(call{method: "GET", path: msgs, token: sb}, 404)

	mid, _ := lastMsg(sa, sid)
	M := "/api/v1/chat/messages/" + mid

	// 7: nối lại.
	r.must(call{method: "GET", path: M + "/stream", token: sa}, 200)
	r.must(call{method: "GET", path: M + "/stream"}, 401)
	r.must(call{method: "GET", path: M + "/stream", token: x.ta}, 403)
	r.must(call{method: "GET", path: M + "/stream", token: sb}, 404)

	// 10: phản hồi.
	r.must(call{method: "PUT", path: M + "/feedback", token: sa, body: `{"value":"HELPFUL"}`}, 204)
	r.must(call{method: "PUT", path: M + "/feedback"}, 401)
	r.must(call{method: "PUT", path: M + "/feedback", token: x.ta, body: `{"value":"HELPFUL"}`}, 403)
	r.must(call{method: "PUT", path: M + "/feedback", token: sb, body: `{"value":"HELPFUL"}`}, 404)
	r.must(call{method: "PUT", path: M + "/feedback", token: sa, body: `{"value":"TUYỆT"}`}, 422)

	// US-P4-01 — POST /chat/messages/{mid}/escalate: 201 mới, 200 đã có, 401, 403 (không phải sinh viên), 404 (tin của người khác), 409 (lớp lưu trữ), 422 (tin không nhờ được).
	asst := func(session uuid.UUID, course uuid.UUID, user uuid.UUID, intent string) string {
		var mid string
		if err := db.QueryRow(ctx, `insert into chat_messages (course_id, session_id, user_id, role, content, stream_status, intent, low_confidence) values ($1, $2, $3, 'ASSISTANT', 'Trả lời mẫu', 'DONE', $4, false) returning id::text`, course, session, user, intent).Scan(&mid); err != nil {
			r.t.Fatal(err)
		}
		return mid
	}
	sidU, _ := uuid.Parse(sid)
	cidU, _ := uuid.Parse(x.cid)
	archU, _ := uuid.Parse(x.arch)
	esc := func(m string) string { return "/api/v1/chat/messages/" + m + "/escalate" }
	qa := asst(sidU, cidU, a, "COURSE_QA")
	r.must(call{method: "POST", path: esc(qa), token: sa}, 201)
	r.must(call{method: "POST", path: esc(qa), token: sa}, 200)
	r.must(call{method: "POST", path: esc(qa)}, 401)
	for _, who := range []string{x.ta, x.gv, x.admin} {
		r.must(call{method: "POST", path: esc(qa), token: who}, 403)
	}
	r.must(call{method: "POST", path: esc(qa), token: sb}, 404)
	r.must(call{method: "POST", path: esc(uuid.NewString()), token: sa}, 404)
	r.must(call{method: "POST", path: esc(asst(archSess, archU, a, "COURSE_QA")), token: sa}, 409)
	r.must(call{method: "POST", path: esc(asst(sidU, cidU, a, "SMALLTALK")), token: sa}, 422)
	var failed uuid.UUID // tin FAILED: không phản hồi được (409), thử lại được
	var userMsg uuid.UUID
	if err := db.QueryRow(ctx, `insert into chat_messages (course_id, session_id, user_id, role, content, stream_status, client_msg_id) values ($1,$2,$3,'USER','hỏi lại','DONE',$4) returning id`, x.cid, sid, a, uuid.New()).Scan(&userMsg); err != nil {
		r.t.Fatal(err)
	}
	if err := db.QueryRow(ctx, `insert into chat_messages (course_id, session_id, user_id, role, stream_status, reply_to, error_code, completed_at) values ($1,$2,$3,'ASSISTANT','FAILED',$4,'PROVIDER_ERROR',now()) returning id`, x.cid, sid, a, userMsg).Scan(&failed); err != nil {
		r.t.Fatal(err)
	}
	r.must(call{method: "PUT", path: "/api/v1/chat/messages/" + failed.String() + "/feedback", token: sa, body: `{"value":"HELPFUL"}`}, 409)

	// 9: thử lại.
	F := "/api/v1/chat/messages/" + failed.String()
	r.must(call{method: "POST", path: F + "/retry", token: sa}, 200)
	r.must(call{method: "POST", path: F + "/retry"}, 401)
	r.must(call{method: "POST", path: F + "/retry", token: x.ta}, 403)
	r.must(call{method: "POST", path: F + "/retry", token: sb}, 404)
	r.must(call{method: "POST", path: M + "/retry", token: sa}, 409)

	// 8: dừng.
	r.must(call{method: "POST", path: M + "/cancel", token: sa}, 204)
	r.must(call{method: "POST", path: M + "/cancel"}, 401)
	r.must(call{method: "POST", path: M + "/cancel", token: x.ta}, 403)
	r.must(call{method: "POST", path: M + "/cancel", token: sb}, 404)

	// 3, 4: xoá mềm và hoàn tác.
	sid2 := create(sa, x.cid)
	r.must(call{method: "DELETE", path: S + "/" + sid2, token: sa}, 204)
	r.must(call{method: "DELETE", path: S + "/" + sid2}, 401)
	r.must(call{method: "DELETE", path: S + "/" + sid2, token: x.ta}, 403)
	r.must(call{method: "DELETE", path: S + "/" + sid2, token: sb}, 404)
	r.must(call{method: "POST", path: S + "/" + sid2 + "/restore", token: sa}, 200)
	r.must(call{method: "POST", path: S + "/" + sid2 + "/restore"}, 401)
	r.must(call{method: "POST", path: S + "/" + sid2 + "/restore", token: x.ta}, 403)
	r.must(call{method: "POST", path: S + "/" + sid2 + "/restore", token: sb}, 404)

	// US-P4-03 — tin STAFF_ANSWER không sinh lại / thử lại được (422 MESSAGE_NOT_RATEABLE).
	staffMsg := asst(sidU, cidU, a, "STAFF_ANSWER")
	r.must(call{method: "POST", path: "/api/v1/chat/messages/" + staffMsg + "/retry", token: sa}, 422)
	r.must(call{method: "PUT", path: "/api/v1/chat/messages/" + staffMsg + "/feedback", token: sa, body: `{"value":"HELPFUL"}`}, 422)

	// US-P4-03 — hàng chờ của nhân viên (thao tác 4–10) và đọc thông báo theo liên kết (15a), mọi status đã khai báo.
	mkTicket := func() string {
		var id string
		if err := db.QueryRow(ctx, `insert into escalation_tickets (course_id, kind, origin, source_type, source_id, student_id) values ($1, 'TEACHER_OUTREACH', 'STAFF_OUTREACH', 'STUDENT', $2, $2) returning id::text`, x.cid, a).Scan(&id); err != nil {
			r.t.Fatal(err)
		}
		return id
	}
	T := "/api/v1/courses/" + x.cid + "/tickets"
	r.must(call{method: "GET", path: T + "?status=ALL", token: x.gv}, 200)
	r.must(call{method: "GET", path: T, token: x.ta}, 200)
	r.must(call{method: "GET", path: T}, 401)
	for _, who := range []string{sa, x.admin} {
		r.must(call{method: "GET", path: T, token: who}, 403)
	}
	r.must(call{method: "GET", path: T + "?status=NOPE", token: x.gv}, 422)
	r.must(call{method: "GET", path: "/api/v1/me/tickets", token: x.gv}, 200)
	r.must(call{method: "GET", path: "/api/v1/me/tickets"}, 401)
	for _, who := range []string{sa, x.admin} {
		r.must(call{method: "GET", path: "/api/v1/me/tickets", token: who}, 403)
	}
	r.must(call{method: "GET", path: "/api/v1/me/tickets?status=NOPE", token: x.gv}, 422)

	t1 := mkTicket()
	r.must(call{method: "GET", path: T + "/" + t1, token: x.gv}, 200)
	r.must(call{method: "GET", path: T + "/" + t1}, 401)
	for _, who := range []string{sa, x.admin} {
		r.must(call{method: "GET", path: T + "/" + t1, token: who}, 403)
	}
	r.must(call{method: "GET", path: T + "/" + uuid.NewString(), token: x.gv}, 404)

	r.must(call{method: "POST", path: T + "/" + t1 + "/claim", token: x.gv}, 200)
	r.must(call{method: "POST", path: T + "/" + t1 + "/claim", token: x.gv}, 200) // idempotent
	r.must(call{method: "POST", path: T + "/" + t1 + "/claim"}, 401)
	r.must(call{method: "POST", path: T + "/" + t1 + "/claim", token: sa}, 403)
	r.must(call{method: "POST", path: T + "/" + uuid.NewString() + "/claim", token: x.gv}, 404)
	r.must(call{method: "POST", path: T + "/" + t1 + "/claim", token: x.ta}, 409) // người khác đã nhận

	r.must(call{method: "POST", path: T + "/" + t1 + "/release", token: x.ta}, 409) // không phải người nhận
	r.must(call{method: "POST", path: T + "/" + t1 + "/release", token: x.gv}, 200)
	r.must(call{method: "POST", path: T + "/" + t1 + "/release"}, 401)
	r.must(call{method: "POST", path: T + "/" + t1 + "/release", token: sa}, 403)
	r.must(call{method: "POST", path: T + "/" + uuid.NewString() + "/release", token: x.gv}, 404)

	ans := func(tok, id, body string) call {
		return call{method: "POST", path: T + "/" + id + "/answer", token: tok, headers: map[string]string{"Idempotency-Key": uuid.NewString()}, body: body}
	}
	r.must(ans(x.gv, t1, `{"body":"Em xem lại bài 3 nhé."}`), 200)
	r.must(ans("", t1, `{"body":"x"}`), 401)
	r.must(ans(sa, t1, `{"body":"x"}`), 403)
	r.must(ans(x.gv, uuid.NewString(), `{"body":"x"}`), 404)
	r.must(ans(x.ta, t1, `{"body":"Trả lời thêm"}`), 409) // đã được người khác trả lời
	r.must(ans(x.gv, t1, `{"body":"   "}`), 422)

	t2 := mkTicket()
	D := func(tok, id, body string) call {
		return call{method: "POST", path: T + "/" + id + "/dismiss", token: tok, body: body}
	}
	r.must(D(x.gv, t2, `{"reason":"DUPLICATE"}`), 200)
	r.must(D("", t2, `{"reason":"DUPLICATE"}`), 401)
	r.must(D(sa, t2, `{"reason":"DUPLICATE"}`), 403)
	r.must(D(x.gv, uuid.NewString(), `{"reason":"DUPLICATE"}`), 404)
	r.must(D(x.gv, t2, `{"reason":"DUPLICATE"}`), 409) // đã đóng
	t3 := mkTicket()
	r.must(D(x.gv, t3, `{"reason":"KHONG_CO"}`), 422)

	r.must(call{method: "POST", path: "/api/v1/notifications/read", token: x.gv, body: `{"link":"/inbox"}`}, 200)
	r.must(call{method: "POST", path: "/api/v1/notifications/read", body: `{"link":"/inbox"}`}, 401)
	r.must(call{method: "POST", path: "/api/v1/notifications/read", token: x.gv, body: `{"link":""}`}, 422)

	// US-P4-04 — thư gửi lỗi (ADMIN).
	FM := "/api/v1/admin/observability/failed-mail"
	r.must(call{method: "GET", path: FM, token: x.admin}, 200)
	r.must(call{method: "GET", path: FM}, 401)
	for _, who := range []string{sa, x.gv, x.ta} {
		r.must(call{method: "GET", path: FM, token: who}, 403)
	}
	r.must(call{method: "GET", path: FM + "?limit=0", token: x.admin}, 422)

	// US-P4-05 — sinh viên hỏi tiếp / Đã rõ / hoàn tác.
	mkAnswered := func() string {
		var id string
		if err := db.QueryRow(ctx, `insert into escalation_tickets (course_id, kind, origin, source_type, source_id, student_id, status, answered_by, answered_at, answer, claimed_by, claimed_at)
			values ($1, 'TEACHER_OUTREACH', 'STAFF_OUTREACH', 'STUDENT', $2, $2, 'ANSWERED', (select id from users where role = 'TEACHER' order by created_at limit 1), now(), 'Trả lời', (select id from users where role = 'TEACHER' order by created_at limit 1), now()) returning id::text`, x.cid, a).Scan(&id); err != nil {
			r.t.Fatal(err)
		}
		return id
	}
	CT := "/api/v1/chat/tickets/"
	t4 := mkAnswered()
	fu := func(tok, id, body string) call { return call{method: "POST", path: CT + id + "/followup", token: tok, body: body} }
	r.must(fu(sa, t4, `{"content":"Em vẫn chưa hiểu","client_msg_id":"`+uuid.NewString()+`"}`), 200)
	r.must(fu("", t4, `{"content":"x","client_msg_id":"`+uuid.NewString()+`"}`), 401)
	r.must(fu(x.gv, t4, `{"content":"x","client_msg_id":"`+uuid.NewString()+`"}`), 403)
	r.must(fu(sb, t4, `{"content":"x","client_msg_id":"`+uuid.NewString()+`"}`), 404)
	r.must(fu(sa, t4, `{"content":"x","client_msg_id":"`+uuid.NewString()+`"}`), 409) // đã OPEN
	t5 := mkAnswered()
	r.must(fu(sa, t5, `{"content":"","client_msg_id":"`+uuid.NewString()+`"}`), 422)
	t6 := mkAnswered()
	r.must(call{method: "POST", path: CT + t6 + "/confirm", token: sa}, 200)
	r.must(call{method: "POST", path: CT + t6 + "/confirm"}, 401)
	r.must(call{method: "POST", path: CT + t6 + "/confirm", token: x.gv}, 403)
	r.must(call{method: "POST", path: CT + t6 + "/confirm", token: sb}, 404)
	r.must(call{method: "POST", path: CT + t6 + "/confirm", token: sa}, 409) // đã đóng
	r.must(call{method: "POST", path: CT + t6 + "/confirm/undo", token: sa}, 200)
	r.must(call{method: "POST", path: CT + t6 + "/confirm/undo"}, 401)
	r.must(call{method: "POST", path: CT + t6 + "/confirm/undo", token: x.gv}, 403)
	r.must(call{method: "POST", path: CT + t6 + "/confirm/undo", token: sb}, 404)
	r.must(call{method: "POST", path: CT + t6 + "/confirm/undo", token: sa}, 409) // đã hoàn tác (ANSWERED)
}

package contract

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// observabilityScenarios: quan sát AI (US-P10-01..03) với mọi status đã khai báo: summary / requests / open (chỉ ADMIN) và tổng hợp lớp (TEACHER của lớp hoặc ADMIN).
func (r *runner) observabilityScenarios(x examRig) {
	ctx := context.Background()
	db := r.rig.deps.DB
	none := "00000000-0000-7000-8000-000000000000"
	base := "/api/v1/admin/observability"

	// ba dòng audit: có nội dung đã che · không có (hết hạn) · của lớp
	withPayload, expired := uuid.NewString(), uuid.NewString()
	for _, id := range []string{withPayload, expired} {
		if _, err := db.Exec(ctx, `insert into llm_audit (id, task, lane, provider, model, status, trace_id, course_id) values ($1, 'CHAT', 'INTERACTIVE', 'p', 'm', 'ok', 'trace-obs', $2)`, id, x.cid); err != nil {
			r.t.Fatal(err)
		}
	}
	if _, err := db.Exec(ctx, `insert into llm_request_payloads (audit_id, course_id, masked_prompt, tools, confidence) values ($1, $2, 'Hỏi về [SV_1]', '["get_my_attendance"]', 0.9)`, withPayload, x.cid); err != nil {
		r.t.Fatal(err)
	}

	for _, p := range []string{base + "/summary", base + "/requests"} {
		r.must(call{method: "GET", path: p, token: x.admin}, 200)
		r.must(call{method: "GET", path: p}, 401)
		for _, who := range []string{x.gv, x.ta, x.sv} {
			r.must(call{method: "GET", path: p, token: who}, 403)
		}
	}
	r.must(call{method: "GET", path: base + "/summary?window=1y", token: x.admin}, 422)
	r.must(call{method: "GET", path: base + "/requests?filter=bad", token: x.admin}, 422)

	open := func(id string) string { return base + "/requests/" + id + "/open" }
	reason := `{"reason":"Giảng viên khiếu nại câu trả lời sai"}`
	r.must(call{method: "POST", path: open(withPayload), token: x.admin, headers: x.idem(), body: reason}, 200)
	r.must(call{method: "POST", path: open(withPayload), headers: x.idem(), body: reason}, 401)
	for _, who := range []string{x.gv, x.ta, x.sv} {
		r.must(call{method: "POST", path: open(withPayload), token: who, headers: x.idem(), body: reason}, 403)
	}
	r.must(call{method: "POST", path: open(none), token: x.admin, headers: x.idem(), body: reason}, 404)
	r.must(call{method: "POST", path: open("khong-phai-uuid"), token: x.admin, headers: x.idem(), body: reason}, 404)
	r.must(call{method: "POST", path: open(expired), token: x.admin, headers: x.idem(), body: reason}, 410)
	r.must(call{method: "POST", path: open(withPayload), token: x.admin, headers: x.idem(), body: `{"reason":"ngắn"}`}, 422)

	cs := "/api/v1/courses/" + x.cid + "/observability/summary"
	r.must(call{method: "GET", path: cs, token: x.gv}, 200)
	r.must(call{method: "GET", path: cs, token: x.admin}, 200)
	r.must(call{method: "GET", path: cs}, 401)
	for _, who := range []string{x.ta, x.sv} {
		r.must(call{method: "GET", path: cs, token: who}, 403)
	}
	r.must(call{method: "GET", path: "/api/v1/courses/khong-phai-uuid/observability/summary", token: x.gv}, 404)
	r.must(call{method: "GET", path: "/api/v1/courses/" + none + "/observability/summary", token: x.admin}, 404)
}

// insightScenarios: báo cáo lỗ hổng kiến thức (US-P10-04/05): generate / latest / export với mọi status đã khai báo.
func (r *runner) insightScenarios(x examRig) {
	ctx := context.Background()
	db := r.rig.deps.DB
	none := "00000000-0000-7000-8000-000000000000"
	base := "/api/v1/courses/" + x.cid + "/insights/knowledge-gap"
	gen := func(who string, days int, want int) {
		body := ""
		if days > 0 {
			body = fmt.Sprintf(`{"period_days":%d}`, days)
		}
		r.must(call{method: "POST", path: base + "/generate", token: who, headers: x.idem(), body: body}, want)
	}

	r.must(call{method: "GET", path: base + "/latest", token: x.gv}, 404)
	// lớp riêng chưa có câu hỏi nào → 409 NOT_ENOUGH_DATA
	var empty string
	if err := db.QueryRow(ctx, `insert into courses (subject_code, class_code, name, semester, join_code, created_by)
		select 'INT1006', 'IG-' || upper(substr(md5(random()::text), 1, 6)), 'Lớp trống', '2026-2027-HK1', translate(upper(substr(md5(random()::text), 1, 7)), '01OIL', '2345H'), created_by from courses where id = $1 returning id::text`, x.cid).Scan(&empty); err != nil {
		r.t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) select $1, user_id, 'TEACHER', 'ACTIVE', 'ADMIN' from enrollments where course_id = $2 and role_in_course = 'TEACHER' and status = 'ACTIVE' limit 1`, empty, x.cid); err != nil {
		r.t.Fatal(err)
	}
	r.must(call{method: "POST", path: "/api/v1/courses/" + empty + "/insights/knowledge-gap/generate", token: x.gv, headers: x.idem()}, 409)

	var sid, student string
	if err := db.QueryRow(ctx, `select user_id::text from enrollments where course_id = $1 and role_in_course = 'STUDENT' and status = 'ACTIVE' limit 1`, x.cid).Scan(&student); err != nil {
		r.t.Fatal(err)
	}
	if err := db.QueryRow(ctx, `insert into chat_sessions (course_id, user_id) values ($1, $2) returning id::text`, x.cid, student).Scan(&sid); err != nil {
		r.t.Fatal(err)
	}
	for i := range 22 {
		if _, err := db.Exec(ctx, `insert into chat_messages (course_id, session_id, user_id, role, content) values ($1, $2, $3, 'USER', $4)`, x.cid, sid, student, fmt.Sprintf("TCP bắt tay ba bước câu %d", i)); err != nil {
			r.t.Fatal(err)
		}
	}
	r.must(call{method: "POST", path: base + "/generate", headers: x.idem()}, 401)
	for _, who := range []string{x.sv, x.admin} {
		gen(who, 0, 403)
	}
	r.must(call{method: "POST", path: "/api/v1/courses/khong-phai-uuid/insights/knowledge-gap/generate", token: x.gv, headers: x.idem()}, 404)
	gen(x.gv, 91, 422)
	gen(x.gv, 30, 202)

	var rid string
	if err := db.QueryRow(ctx, `insert into insight_reports (course_id, kind, period_from, period_to, payload, data_watermark, generated_by)
		values ($1, 'KNOWLEDGE_GAP', current_date - 30, current_date, '{"topics":[{"rank":1,"name":"TCP","why":"6 sinh viên hỏi","score":1,"signals":{"students":6,"questions":22,"low_confidence_rate":0.5,"not_helpful_rate":0,"practice_wrong_rate":0},"recommendation":"Ôn lại","examples":["TCP là gì"],"named_by":"llm","chapter":null,"week":null,"keywords":["tcp"]}],"uncovered":[],"counts":{"questions":22,"students":1,"clusters":1},"period_days":30}',
		        now() + interval '1 hour', (select created_by from courses where id = $1)) returning id::text`, x.cid).Scan(&rid); err != nil {
		r.t.Fatal(err)
	}
	gen(x.gv, 30, 200) // không có dữ liệu mới → reused
	gen(x.ta, 29, 202)
	gen(x.gv, 28, 202)
	gen(x.gv, 27, 429) // quá 3 việc / giờ / lớp

	r.must(call{method: "GET", path: base + "/latest", token: x.gv}, 200)
	r.must(call{method: "GET", path: base + "/latest", token: x.ta}, 200)
	r.must(call{method: "GET", path: base + "/latest"}, 401)
	for _, who := range []string{x.sv, x.admin} {
		r.must(call{method: "GET", path: base + "/latest", token: who}, 403)
	}
	r.must(call{method: "GET", path: "/api/v1/courses/khong-phai-uuid/insights/knowledge-gap/latest", token: x.gv}, 404)

	ex := base + "/" + rid + "/export"
	r.must(call{method: "GET", path: ex + "?format=md", token: x.gv}, 200)
	r.must(call{method: "GET", path: ex, token: x.ta}, 200)
	r.must(call{method: "GET", path: ex}, 401)
	r.must(call{method: "GET", path: ex, token: x.sv}, 403)
	r.must(call{method: "GET", path: base + "/" + none + "/export", token: x.gv}, 404)
	r.must(call{method: "GET", path: base + "/khong-phai-uuid/export", token: x.gv}, 404)
	r.must(call{method: "GET", path: ex + "?format=pdf", token: x.gv}, 422)
}

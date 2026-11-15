package contract

import (
	"context"

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

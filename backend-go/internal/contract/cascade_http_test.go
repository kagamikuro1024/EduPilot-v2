package contract

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
	"github.com/edupilot/backend-go/internal/llm/fake"
	"github.com/edupilot/backend-go/internal/llm/provider"
)

// TestCascadeThreadsHTTP — US-DM-08 AC3 (a)(b)(c)(f) ở mức HTTP thật (router + Postgres + provider giả, không mạng): từng nhánh cascade của THREADS_PRECHECK
// cho đúng mã HTTP, đúng số lời gọi decision, và đúng `cascade_path` ở llm_audit. (d) chèn lệnh: internal/grading TestInjectionOR; (e) chat: internal/agent TestRouteDecisionTable.
func TestCascadeThreadsHTTP(t *testing.T) {
	r := &runner{t: t, rig: getRig(t), cache: map[string]string{}}
	r.prod, r.test = mustLoad(t)
	ctx := context.Background()
	deps := r.rig.deps
	db := deps.DB
	var admin, sv, course uuid.UUID
	mk := func(role string) uuid.UUID {
		var id uuid.UUID
		if err := db.QueryRow(ctx, `insert into users (email, full_name, role, status, password_hash, email_verified_at) values ($1, 'Người Thử', $2::user_role, 'ACTIVE', 'x', now()) returning id`,
			"ct-cs-"+uuid.NewString()[:8]+"@example.test", role).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	admin, sv = mk("ADMIN"), mk("STUDENT")
	if err := db.QueryRow(ctx, `insert into courses (subject_code, class_code, name, semester, join_code, created_by) values ('INT1006', $1, 'Lớp cascade', '2026-2027-HK1', $2, $3) returning id`,
		"CS-"+uuid.NewString()[:6], "CS"+strings.ToUpper(strings.NewReplacer("0", "X", "1", "Y", "O", "Z", "I", "W", "L", "V").Replace(uuid.NewString()[:5])), admin).Scan(&course); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `insert into enrollments (course_id, user_id, role_in_course, status, joined_via) values ($1, $2, 'STUDENT', 'ACTIVE', 'ADMIN')`, course, sv); err != nil {
		t.Fatal(err)
	}
	// bật cổng cho lớp này (hệ thống + lớp) rồi nạp ngay ảnh chụp cấu hình
	for _, q := range []string{
		`update decision_gates set enabled = true where gate = 'THREADS_PRECHECK'`,
		`insert into course_decision_gates (course_id, gate, enabled) values ('` + course.String() + `', 'THREADS_PRECHECK', true)`,
	} {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	if err := deps.LLM.Decision.Load(ctx); err != nil {
		t.Fatal(err)
	}
	ctl := deps.LLM.Registry.Fake()
	old := ctl.Get()
	t.Cleanup(func() {
		ctl.Set(old)
		_, _ = db.Exec(ctx, `update decision_gates set enabled = false where gate = 'THREADS_PRECHECK'`)
		_, _ = db.Exec(ctx, `delete from course_decision_gates where course_id = $1`, course)
		_ = deps.LLM.Decision.Load(ctx)
	})
	tok := r.rig.token(t, sv.String(), auth.RoleStudent)
	T := "/api/v1/courses/" + course.String() + "/threads"
	idem := func() map[string]string { return map[string]string{"Idempotency-Key": uuid.NewString()} }
	const title, body = "Hỏi", "Bạn ở lớp mình bị cảnh cáo học vụ phải không?"
	post := func(title, body string, want int) []byte {
		t.Helper()
		_, b := r.must(call{method: "POST", path: T, token: tok, headers: idem(), body: `{"title":"` + title + `","body":"` + body + `"}`}, want)
		return b
	}
	lastRow := func() (path, kind string) {
		t.Helper()
		deps.LLM.Gateway.FlushAudit(ctx)
		if err := db.QueryRow(ctx, `select coalesce(cascade_path,''), coalesce(error_kind,'') from llm_audit where task = 'DECIDE' and cascade_gate = 'THREADS_PRECHECK' order by created_at desc, id desc limit 1`).Scan(&path, &kind); err != nil {
			t.Fatalf("không có dòng llm_audit DECIDE: %v", err)
		}
		return path, kind
	}
	reset := func(s fake.Settings) int64 {
		s.ValidKey = old.ValidKey
		ctl.Set(s)
		return ctl.Decisions()
	}

	// (a) decision lưỡng lự (đầu vào lạ → 0,25) → Structured (provider giả: "no") → đăng được; path FALLBACK
	before := reset(fake.Settings{})
	post(title, body, 201)
	if p, _ := lastRow(); p != "FALLBACK" || ctl.Decisions() != before+1 {
		t.Fatalf("(a) path=%q decisions %d→%d", p, before, ctl.Decisions())
	}
	// decision nói chắc «có» → chặn (FAST), 422 PII_DETECTED + personal_question, không bài nào được lưu
	ctl.AddDecision(title+"\n"+body+" (chắc)", "private_or_named", fake.DecisionRecord{Noul: ptr(0.93)})
	if raw := post(title, body+" (chắc)", 422); !contains(raw, "PII_DETECTED") || !contains(raw, `"personal_question":true`) {
		t.Fatalf("FAST có: %s", raw)
	}
	if p, _ := lastRow(); p != "FAST" {
		t.Fatalf("FAST có: path=%q", p)
	}
	// decision nói chắc «không» (≤ ε) → cho qua FAST, không gọi Structured
	ctl.AddDecision(title+"\n"+body+" (sạch)", "private_or_named", fake.DecisionRecord{Noul: ptr(0.03)})
	post(title, body+" (sạch)", 201)
	if p, _ := lastRow(); p != "FAST" {
		t.Fatalf("FAST không: path=%q", p)
	}
	// (b) decision lỗi → Structured → đăng được; path FALLBACK, error_kind SERVER
	reset(fake.Settings{DecisionMode: "error"})
	post(title, body+" (b)", 201)
	if p, k := lastRow(); p != "FALLBACK" || k != "SERVER" {
		t.Fatalf("(b) path=%q kind=%q", p, k)
	}
	// (c) cả decision và Structured đều lỗi → chặn an toàn: 422 + path FAIL_CLOSED
	reset(fake.Settings{DecisionMode: "error", ErrorRate: 1, ErrorKind: provider.KindAuth})
	if raw := post(title, body+" (c)", 422); !contains(raw, "PII_DETECTED") {
		t.Fatalf("(c): %s", raw)
	}
	if p, _ := lastRow(); p != "FAIL_CLOSED" {
		t.Fatalf("(c) path=%q", p)
	}
	// (f) khủng hoảng: 0 lời gọi decision, 422 CRISIS_SUPPORT
	before = reset(fake.Settings{DecisionMode: "always_no"})
	if raw := post("Hỏi", "Em không muốn sống nữa", 422); !contains(raw, "CRISIS_SUPPORT") {
		t.Fatalf("(f): %s", raw)
	}
	if ctl.Decisions() != before {
		t.Fatalf("(f) khủng hoảng đã tới decision: %d → %d", before, ctl.Decisions())
	}
}

func ptr[T any](v T) *T { return &v }

func contains(b []byte, s string) bool { return strings.Contains(string(b), s) }

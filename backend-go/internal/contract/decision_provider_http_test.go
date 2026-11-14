package contract

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/edupilot/backend-go/internal/auth"
)

// TestDecisionProviderHTTP — sửa QC 10.5: (1) provider type=decision / model kind=decision được API nhà cung cấp nhận (trước đây oneof chặn);
// (2) chỉ ADMIN thấy provider decision và tác vụ DECIDE; (3) «Test kết nối» của decision-fake chạy cục bộ và trả ok (trước đây NETWORK);
// bản thật khi LLM_DECISION_ALLOW_REAL tắt → 409 DECISION_REAL_DISABLED, không gọi mạng.
func TestDecisionProviderHTTP(t *testing.T) {
	r := &runner{t: t, rig: getRig(t), cache: map[string]string{}}
	r.prod, r.test = mustLoad(t)
	ctx := context.Background()
	admin := r.rig.token(t, uuid.NewString(), auth.RoleAdmin)
	teacher := r.rig.token(t, uuid.NewString(), auth.RoleTeacher)
	idem := func() map[string]string { return map[string]string{"Idempotency-Key": uuid.NewString()} }
	const prov = "/api/v1/admin/llm/providers"
	t.Cleanup(func() {
		_, _ = r.rig.deps.DB.Exec(ctx, `delete from llm_audit where task = 'DECIDE' and trace_id = 'ct-dec-usage'`)
		_, _ = r.rig.deps.DB.Exec(ctx, `delete from llm_models where provider_id in (select id from llm_providers where name = 'Quyết định B'); delete from llm_providers where name = 'Quyết định B'`)
		_ = r.rig.deps.LLM.Registry.Reload(ctx)
	})
	model := func(name string, enabled bool) string {
		b, _ := json.Marshal(map[string]any{"model": name, "kind": "decision", "price_in": "0", "price_out": "0", "enabled": enabled})
		return string(b)
	}
	create := `{"type":"decision","name":"Quyết định B","enabled":false,"models":[` + model("decision-fake", true) + `,` + model("gemini-decision-1", false) + `]}`

	// (1) tạo provider decision: 201 (không bị oneof chặn); mô hình thật tắt vì ALLOW_REAL=false nhưng đang disabled nên được lưu
	_, raw := r.must(call{method: "POST", path: prov, token: admin, headers: idem(), body: create}, 201)
	var made struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
	}
	_ = json.Unmarshal(raw, &made)
	one := prov + "/" + made.ID
	// chọn mô hình thật khi chưa được phép → 409 DECISION_REAL_DISABLED
	real := `{"type":"decision","name":"Quyết định B","enabled":false,"models":[` + model("decision-fake", false) + `,` + model("gemini-decision-1", true) + `],"version":` + itoa(made.Version) + `}`
	if _, b := r.must(call{method: "PUT", path: one, token: admin, body: real}, 409); !strings.Contains(string(b), "DECISION_REAL_DISABLED") {
		t.Fatalf("409 phải mang DECISION_REAL_DISABLED: %s", b)
	}
	// kind decision trên nhà không phải decision → 422
	r.must(call{method: "POST", path: prov, token: admin, headers: idem(), body: `{"type":"fake","name":"Sai loại","models":[` + model("m", true) + `]}`}, 422)

	// (3) Test kết nối bản giả: ok, cục bộ
	_, tb := r.must(call{method: "POST", path: one + "/test", token: admin}, 200)
	var res struct {
		OK        bool   `json:"ok"`
		ErrorKind string `json:"error_kind"`
	}
	_ = json.Unmarshal(tb, &res)
	if !res.OK {
		t.Fatalf("Test kết nối decision-fake phải ok, được %s", tb)
	}

	// (2) chỉ ADMIN thấy provider decision và tác vụ DECIDE
	hasDecision := func(token string) bool {
		_, b := r.must(call{method: "GET", path: prov, token: token}, 200)
		var l struct {
			Items []struct {
				Type string `json:"type"`
			} `json:"items"`
		}
		_ = json.Unmarshal(b, &l)
		for _, it := range l.Items {
			if it.Type == "decision" {
				return true
			}
		}
		return false
	}
	if !hasDecision(admin) {
		t.Fatal("ADMIN phải thấy provider decision")
	}
	if hasDecision(teacher) {
		t.Fatal("TEACHER không được thấy provider decision")
	}
	if _, err := r.rig.deps.DB.Exec(ctx, `insert into llm_audit (task, lane, provider, model, status, trace_id, cascade_gate, cascade_path) values ('DECIDE', 'INTERACTIVE', 'Quyết định', 'decision-fake', 'ok', 'ct-dec-usage', 'THREADS_PRECHECK', 'FAST')`); err != nil {
		t.Fatal(err)
	}
	usage := func(token string) string {
		_, b := r.must(call{method: "GET", path: "/api/v1/admin/llm/usage?group=task&from=" + time.Now().Add(-24*time.Hour).UTC().Format(time.RFC3339) + "&to=" + time.Now().Add(24*time.Hour).UTC().Format(time.RFC3339), token: token}, 200)
		return string(b)
	}
	if !strings.Contains(usage(admin), `"DECIDE"`) {
		t.Fatal("ADMIN phải thấy tác vụ DECIDE ở mức dùng")
	}
	if strings.Contains(usage(teacher), "DECIDE") {
		t.Fatal("TEACHER không được thấy tác vụ DECIDE ở mức dùng")
	}
}

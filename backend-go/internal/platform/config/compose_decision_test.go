package config_test

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestComposeDecisionEnvReachesWorker: biến cấu hình decision phải được chuyển cho CẢ gateway và worker trong docker-compose.local.yml
// (cờ chèn lệnh chạy ở worker; QC 10.5 từng thấy worker không nhận LLM_DECISION_* / FAKE_DECISION_PROB). Chuyển vào một service mà quên service kia là lỗi.
func TestComposeDecisionEnvReachesWorker(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("../../../../docker-compose.local.yml")
	if err != nil {
		t.Fatal(err)
	}
	svc := regexp.MustCompile(`(?m)^  ([a-z0-9_-]+):\s*$`)
	idx := svc.FindAllStringSubmatchIndex(string(raw), -1)
	block := map[string]string{}
	for i, m := range idx {
		end := len(raw)
		if i+1 < len(idx) {
			end = idx[i+1][0]
		}
		block[string(raw[m[2]:m[3]])] = string(raw[m[0]:end])
	}
	shared := []string{"LLM_DECISION_TIMEOUT", "LLM_DECISION_ALLOW_REAL", "LLM_DECISION_REAL_DAILY_CAP", "LLM_DECISION_RPM", "LLM_DECISION_BASE_URL",
		"LLM_DECISION_FAKE_P50_MS", "LLM_DECISION_FAKE_P95_MS", "LLM_DECISION_FAKE_ERROR_RATE", "LLM_DECISION_FAKE_MODE", "FAKE_DECISION_PROB"}
	for _, name := range []string{"gateway", "worker"} {
		b, ok := block[name]
		if !ok {
			t.Fatalf("không thấy service %s", name)
		}
		for _, v := range shared {
			if !strings.Contains(b, "      "+v+":") {
				t.Errorf("service %s không nhận %s", name, v)
			}
		}
	}
	for _, v := range []string{"DECISION_ALERT_TICK", "DECISION_ALERT_DEDUPE"} { // việc nền chỉ ở worker
		if !strings.Contains(block["worker"], "      "+v+":") {
			t.Errorf("worker không nhận %s", v)
		}
	}
	for _, v := range []string{"THREADS_STRUCTURED_TIMEOUT"} { // đường Threads chạy ở gateway
		if !strings.Contains(block["gateway"], "      "+v+":") {
			t.Errorf("gateway không nhận %s", v)
		}
	}
}

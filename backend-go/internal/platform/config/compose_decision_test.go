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

// TestComposeGuidanceEnvReachesGatewayAndWorker — GD-BUG-8: biến của «học từ lần sửa» (FEAT-grading-guidance 4.1) phải tới CẢ gateway (API, ảnh chụp cờ, hạn mức gợi ý) và worker (job reflect,
// khối «Tránh…», ngưỡng ca hồi quy). Thêm biến mới vào config.go mà quên compose thì test này đỏ.
func TestComposeGuidanceEnvReachesGatewayAndWorker(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("../../../../docker-compose.local.yml")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := os.ReadFile("config.go")
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
	// mọi biến GUIDANCE_* / REFLECT_* / REGRADE_* / REGRESSION_* / THREAD_SUGGEST_* / QUESTION_HINTS_* đọc trong config.go
	names := regexp.MustCompile(`"((?:GUIDANCE|REFLECT|REGRADE|REGRESSION|THREAD_SUGGEST|QUESTION_HINTS)_[A-Z_]+)"`).FindAllStringSubmatch(string(cfg), -1)
	if len(names) < 15 {
		t.Fatalf("chỉ thấy %d biến trong config.go — bộ quét hỏng?", len(names))
	}
	for _, name := range []string{"gateway", "worker"} {
		b, ok := block[name]
		if !ok {
			t.Fatalf("không thấy service %s", name)
		}
		for _, m := range names {
			if !strings.Contains(b, "      "+m[1]+":") {
				t.Errorf("service %s không nhận %s", name, m[1])
			}
		}
	}
}

// TestComposeEveryEnvReachesAService — mọi biến môi trường mà config.go / ingest / worker đọc phải có mặt trong docker-compose.local.yml (khối chung, gateway hoặc worker);
// thiếu thì đặt trong .env.local không tới container (AI_SUMMARY_TIMEOUT, KNOWLEDGE_EMBED_TIMEOUT, ATTENDANCE_*… từng thế). Thêm biến mới vào mã mà quên compose thì test đỏ.
func TestComposeEveryEnvReachesAService(t *testing.T) {
	t.Parallel()
	compose, err := os.ReadFile("../../../../docker-compose.local.yml")
	if err != nil {
		t.Fatal(err)
	}
	// tên biến = đối số chuỗi ĐẦU của lời gọi: l.num("X", …), getenv("X"), i("X", …), os.Getenv("X")
	call := regexp.MustCompile(`\w\("([A-Z][A-Z0-9_]{3,})"`)
	exempt := map[string]bool{"INSTANCE_ID": true} // mỗi bản sao tự lấy hostname; chia sẻ một giá trị sẽ làm trùng định danh khi --scale
	seen := map[string]bool{}
	for _, f := range []string{"config.go", "../../ingest/settings.go", "../../../cmd/worker/gradingq.go", "../../../cmd/worker/imapsync.go", "../../../cmd/worker/teamssync.go"} {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range call.FindAllStringSubmatch(string(src), -1) {
			seen[m[1]] = true
		}
	}
	if len(seen) < 150 {
		t.Fatalf("chỉ thấy %d biến — bộ quét hỏng?", len(seen))
	}
	for v := range seen {
		if exempt[v] {
			continue
		}
		if !regexp.MustCompile(`(?m)^\s+` + v + `:`).Match(compose) {
			t.Errorf("docker-compose.local.yml thiếu %s", v)
		}
	}
}

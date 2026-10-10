#!/usr/bin/env bash
# Cổng nghiệm thu P3 (US-P3-08 AC6, AC9): hai kênh hỏi đáp + PII. Chạy theo thứ tự, dừng ở lỗi đầu, in bảng PASS/FAIL, dòng cuối `GATE P3: PASS|PASS (có SKIP)|FAIL`.
#   bash scripts/gate-p3.sh                          # cần: Docker (testcontainers) và một gateway ĐÃ SEED cho eval_pii.py / Playwright thật (API_URL, SEED_DEFAULT_PASSWORD)
#   GATE_K6=1 bash scripts/gate-p3.sh   # thêm hai kịch bản k6 chat.js trên stack dev đã seed (tạo lại gateway cho từng kịch bản rồi trả về cấu hình thường: xem k6-stack.yml); chưa cài k6 → SKIP
# KHÔNG BAO GIỜ âm thầm xanh: TestNoPayloadLeak thiếu stack → FAIL; eval_pii.py thiếu / sai bộ dữ liệu (mã 2) → FAIL. Chỉ bước k6 được SKIP.
set -u
cd "$(dirname "$0")/.."
. scripts/gate-lib.sh

run "go vet ./..." go_in go vet ./...
run "go test -race privacy + agent + chat + thread (gồm TestChatMatrix, TestThreadsMatrix)" go_in go test -race -count=1 ./internal/privacy/... ./internal/agent/... ./internal/chat/... ./internal/thread/...
strict_test TestChatMatrix ./internal/chat
strict_test TestThreadsMatrix ./internal/thread
run "TestUnmaskStream" go_in go test -count=1 -run 'TestUnmaskStream' ./internal/privacy
strict_test TestNoPayloadLeak ./internal/integration
run "TestThreadsAgentHasNoPersonalTools" go_in go test -count=1 -run 'TestThreadsAgentHasNoPersonalTools' ./internal/agent
run "TestStudentNeverSeesConfidence" go_in go test -count=1 -run 'TestStudentNeverSeesConfidence' ./internal/chat ./internal/thread ./internal/contract
run "go test ./internal/contract/..." go_in go test -count=1 -tags testroutes ./internal/contract/...
run "E1: eval_pii.py --min-recall 0.95 --max-false-block 0.05" python3 benchmarks/eval_pii.py --min-recall 0.95 --max-false-block 0.05
run "Playwright privacy + private-chat + threads" fe sh -c 'E2E_API_PORT=${E2E_API_PORT:-3412} pnpm build:gate && E2E_PORT=${E2E_PORT:-3410} E2E_API_PORT=${E2E_API_PORT:-3412} pnpm exec playwright test privacy.spec.ts private-chat.spec.ts threads.spec.ts --workers=2 --retries=0'
run "ui-antipatterns.sh" bash scripts/ui-antipatterns.sh

if [ "${GATE_K6:-}" = "1" ]; then
  if ! command -v k6 >/dev/null 2>&1; then
    skip "k6 chat.js" "chưa cài"
  else
    # Mỗi kịch bản cần provider giả trễ khác nhau (first_event 5–15 s, ttft 300 ms) ⇒ tạo lại gateway của stack dev bằng benchmarks/load/k6-stack.yml cho từng kịch bản, xong trả về cấu hình thường.
    # K6_NO_RECREATE=1: không đụng stack (tự đặt FAKE_LLM_LATENCY / hạn mức cho đúng kịch bản trước khi chạy).
    DC=(docker compose --env-file .env.local -f docker-compose.local.yml -p "${COMPOSE_PROJECT_NAME:-edupilot}")
    k6stack() { [ "${K6_NO_RECREATE:-}" = "1" ] || K6_LAT="$1" "${DC[@]}" -f benchmarks/load/k6-stack.yml up -d --no-deps --force-recreate --wait gateway >/dev/null 2>&1; }
    k6restore() { [ "${K6_NO_RECREATE:-}" = "1" ] || RATE_LIMIT_IP_PER_MIN=5000 AUTH_LOGIN_IP_PER_MIN=1000 AUTH_REGISTER_IP_PER_HOUR=1000 AUTH_TOKEN_IP_PER_MIN=1000 "${DC[@]}" up -d --no-deps --force-recreate --wait gateway >/dev/null 2>&1; }
    k6run() { k6stack "$2" && sleep 5 && k6 run -q -e BASE="${K6_BASE:-https://localhost}" -e SCENARIO="$1" benchmarks/load/chat.js; }
    run "k6 chat.js first_event (p95 < 300 ms; provider giả trễ 5–15 s)" k6run first_event 5000-15000
    run "k6 chat.js ttft (cache p95 < 1,5 s; truy xuất p95 < 4 s; provider giả trễ 300 ms)" k6run ttft 300-300
    k6restore
  fi
fi

finish P3

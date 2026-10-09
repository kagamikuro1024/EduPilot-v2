#!/usr/bin/env bash
# Cổng nghiệm thu PE (US-PE-09 AC8): chạy từng bước, in bảng PASS/FAIL, dòng cuối `GATE PE: PASS|FAIL`.
#   bash scripts/gate-pe.sh                 # bước không cần stack: go vet, test Go, sqlc, ui-antipatterns, Playwright exam.spec.ts
#   GATE_K6=1 K6_BASE=https://localhost:773 bash scripts/gate-pe.sh    # thêm 3 kịch bản k6 trên stack TEST đã seed (xem benchmarks/load/exam-submit.js)
#   GATE_SEED=1 bash scripts/gate-pe.sh     # thêm check-exam-seed.mjs bank|scores|demo (API_URL, SEED_DEFAULT_PASSWORD như seed.mjs)
set -u
cd "$(dirname "$0")/.."
export DOCKER_HOST="${DOCKER_HOST:-unix://$HOME/.colima/default/docker.sock}" TESTCONTAINERS_RYUK_DISABLED=true
RESULTS=()
FAILED=0
run() { # run "tên" lệnh... — dừng ở lỗi đầu (các bước sau không chạy)
  [ "$FAILED" = 1 ] && return
  local name="$1" t0=$SECONDS; shift
  echo "▶ $name"
  if "$@"; then RESULTS+=("PASS|$name|$((SECONDS - t0))"); else RESULTS+=("FAIL|$name|$((SECONDS - t0))"); FAILED=1; fi
}
go_in() { (cd backend-go && "$@"); }
fe() { (cd frontend && "$@"); }

run "go vet ./..." go_in go vet ./...
run "go test -race exam + judge + quiz" go_in go test -race -count=1 ./internal/exam/... ./internal/judge/... ./internal/quiz/...
run "TestSandboxAttacks (15 ca, judge thật)" go_in go test -count=1 -run 'TestSandboxAttacks' ./internal/judge/...
run "TestScoringDecimal" go_in go test -count=1 -run 'TestScoringDecimal' ./internal/exam/...
run "TestNoAnswerLeak (service + contract)" go_in go test -count=1 -run 'TestNoAnswerLeak' ./internal/exam/... ./internal/contract/...
run "go test ./internal/contract/..." go_in go test -count=1 ./internal/contract/...
run "sqlc diff" go_in sh -c 'PATH="$PATH:$(go env GOPATH)/bin" sqlc diff'
run "expected_exam_scores.csv khớp bộ sinh" node scripts/gen-expected-exam-scores.mjs --check
run "ui-antipatterns.sh" bash scripts/ui-antipatterns.sh
run "Playwright exam.spec.ts" fe sh -c 'E2E_API_PORT=${E2E_API_PORT:-3334} pnpm build:gate && E2E_PORT=${E2E_PORT:-3330} E2E_API_PORT=${E2E_API_PORT:-3334} pnpm exec playwright test exam.spec.ts a11y.spec.ts --grep-invert "@real|visual" --workers=2'

if [ "${GATE_SEED:-}" = "1" ]; then
  for m in bank scores demo; do run "check-exam-seed.mjs $m" node scripts/check-exam-seed.mjs "$m"; done
fi
if [ "${GATE_K6:-}" = "1" ]; then
  k6run() { k6 run -q -e BASE="${K6_BASE:-https://localhost}" -e SCENARIO="$1" ${2:+-e CHAT_BASE_P95="$2"} benchmarks/load/exam-submit.js; }
  run "k6 exam-submit judge_burst" k6run judge_burst
  run "k6 exam-submit autosave" k6run autosave
  run "k6 exam-submit chat (đường cơ sở TTFT)" k6run chat
  BASE_P95=$(node -e 'const fs=require("fs");const f=fs.readdirSync("benchmarks/reports").filter(n=>/^pe-exam-submit-.*-chat\.json$/.test(n)).sort().pop();console.log(JSON.parse(fs.readFileSync("benchmarks/reports/"+f)).metrics.chat_ttft_ms["p(95)"])')
  run "k6 exam-submit mixed (TTFT ≤ cơ sở +20 %, p95 Chạy thử ≤ 5 s)" k6run mixed "$BASE_P95"
fi

echo; echo "| Kết quả | Bước | Giây |"; echo "|---|---|---|"
for r in "${RESULTS[@]}"; do IFS="|" read -r st nm sec <<<"$r"; echo "| $st | $nm | $sec |"; done
echo
if [ "$FAILED" = 0 ]; then echo "GATE PE: PASS"; else echo "GATE PE: FAIL"; exit 1; fi

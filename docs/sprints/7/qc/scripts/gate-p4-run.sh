#!/usr/bin/env bash
# QC cổng P4 — lệnh soạn sẵn, CHƯA CHẠY. Dùng: bash gate-p4-run.sh <ID|all> (ID = G4-01…). Mỗi bước in "PASS/FAIL <ID>" + 10 dòng cuối.
# Chỉ các dòng mà story đã PASS (P4-01…05); dòng phụ thuộc US-P4-06/07 để trống ở cuối. Chạy từ gốc repo, `origin/sprint/7-p4-p5`.
set -u
export DOCKER_HOST=${DOCKER_HOST:-unix://$HOME/.colima/default/docker.sock} TESTCONTAINERS_RYUK_DISABLED=true
ROOT=$(git rev-parse --show-toplevel); cd "$ROOT"; BASE=1518f83
step() { # step ID "lệnh"
  [ "${1}" = "${WANT:-all}" ] || [ "${WANT:-all}" = all ] || return 0
  echo "=== $1: $2"; out=$(bash -c "$2" 2>&1); rc=$?; echo "$out" | tail -10; [ $rc = 0 ] && echo "PASS $1" || echo "FAIL $1 rc=$rc"; }
WANT=${1:-all}
# --- A. lệnh cổng (P4.md)
step G4-01 "cd backend-go && go test -count=1 -race ./internal/escalation/... ./internal/notify/... ./internal/mail/..."
# G4-02 / G4-04: test nằm ở internal/escalation (không phải internal/integration, không cần -tags integration) — theo dev-US-P4-04.md; ghi lệch này vào report.
step G4-02 "cd backend-go && go test -count=1 -race ./internal/escalation -run 'TestEscalationToMail' -v | grep -E 'bell_latency|^(--- |ok|FAIL)'"
step G4-03 "cd backend-go && go test -count=1 -tags integration ./internal/integration -run TestNotifyAcrossTwoGateways -v"
step G4-04 "cd backend-go && docker rm -f edupilot-test-mailpit >/dev/null 2>&1; go test -count=1 -race ./internal/escalation -run 'TestOutboxSurvivesWorkerKill|TestMailSendIdempotentRedelivery' -v"
step G4-05 "cd frontend && E2E_API_PORT=3412 pnpm build:gate && E2E_PORT=3410 E2E_API_PORT=3412 pnpm exec playwright test escalation.spec.ts --project=desktop"
# G4-06: scripts/gate-p4.sh CHƯA CÓ ở origin/sprint/7-p4-p5 (đã kiểm) -> chạy khi dev tạo; không có = FAIL (Q-QC-P4-20)
step G4-06 "test -f scripts/gate-p4.sh && bash scripts/gate-p4.sh"
step G4-07 "cd backend-go && go vet ./... && golangci-lint run && go test -count=1 -race ./... && go test -tags testroutes -list '.*' ./... | grep -c '^Test'"
step G4-08 "cd backend-go && go test -count=1 -tags testroutes ./internal/contract/... && sqlc diff && cd .. && git diff $BASE.. --numstat -- backend-go/internal/contract | awk '\$2>0{d=1} END{exit d}'"
step G4-09 "pnpm -C frontend lint && pnpm -C frontend build && bash scripts/ui-antipatterns.sh"
step G4-10 "git diff $BASE.. --stat -- '*_test.go' 'frontend/e2e/*' | tail -40; git diff $BASE.. --numstat -- '*_test.go' 'frontend/e2e/*' | awk '\$2>0{print \"test cũ bị xoá/sửa:\", \$3, \$2}'"
# --- C. kiểm chéo chạy được bằng script/lệnh (stack: bash /tmp/qc/up.sh trước; xem report của từng story)
step G4-31 "test \$(grep -rn '@/mock' frontend/src/features/inbox | wc -l) -eq 0"
step G4-30 "git diff $BASE.. -U0 -- frontend/src | grep -nE '^\\+.*\\bfetch\\(' | grep -v apiClient | head; git diff $BASE.. -- backend-go/go.mod frontend/package.json | grep -E '^[+-]\\s' | head -20; git diff $BASE.. -U0 | grep -nEi 'float64.*(grade|score|diem)|(secret|password)\\s*[:=]\\s*\"[^\"]{6,}\"' | head"
# G4-11…G4-14, G4-15…G4-29, G4-32: kiểm tay/TC — dùng các script và lệnh sau (đã chạy ở report-US-P4-01…05):
#   scripts/p4-01-ui.mjs (TC-18…24), p4-02-ui.mjs + p4-02-nojump.mjs (chuông, 375 px), two-gateways.mjs + sse.mjs (G4-18),
#   p4-03-ui.mjs (TC-16/31/32/35/36/38; G4-24), p4-04-ui.mjs (G4-17, G4-27), p4-05-ui.mjs (G4-19, G4-29); lệnh curl của từng report.
# Phụ thuộc US-P4-06 / US-P4-07 (chưa PASS): G4-20, G4-21, G4-25 (phần P4-07), G4-26 (P4-06), G4-27 (/threads), G4-32 (POST_REPORTED).

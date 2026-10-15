#!/usr/bin/env bash
# Cổng nghiệm thu P4 (US-S7-SEED AC7): escalation + thông báo + mail + kiểm duyệt Threads. Chạy theo thứ tự, dừng ở lỗi đầu, in bảng PASS/FAIL, dòng cuối `GATE P4: PASS|FAIL`.
#   bash scripts/gate-p4.sh        # cần Docker (testcontainers); Playwright dùng gateway giả nên không cần stack seed
# Các lệnh theo `docs/phases/P4.md` ("Cổng nghiệm thu") + `-race`. Test BẮT BUỘC chạy thật (strict_test): thiếu Docker hoặc bị SKIP → FAIL.
set -u
cd "$(dirname "$0")/.."
. scripts/gate-lib.sh

run "go vet ./..." go_in go vet -tags testroutes ./...
run "go test -race escalation + notify + mail + thread" go_in go test -race -count=1 -tags testroutes ./internal/escalation/... ./internal/notify/... ./internal/mail/... ./internal/thread/...
strict_test TestEscalationToMail ./internal/escalation -race -tags testroutes
strict_test TestNotifyAcrossTwoGateways ./internal/integration -race -tags testroutes
strict_test TestOutboxSurvivesWorkerKill ./internal/escalation -race -tags testroutes
run "go test ./internal/contract/..." go_in go test -count=1 -tags testroutes ./internal/contract/...
run "Playwright escalation + threads-moderation + threads-crisis" fe sh -c 'E2E_API_PORT=${E2E_API_PORT:-3412} pnpm build:gate && E2E_PORT=${E2E_PORT:-3410} E2E_API_PORT=${E2E_API_PORT:-3412} pnpm exec playwright test escalation.spec.ts threads-moderation.spec.ts threads-crisis.spec.ts --workers=2 --retries=0'
run "ui-antipatterns.sh" bash scripts/ui-antipatterns.sh

finish P4

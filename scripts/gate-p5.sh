#!/usr/bin/env bash
# Cổng nghiệm thu P5 (US-S7-SEED AC7): điểm danh, thời gian học, hồ sơ 360, `/me`, dữ liệu seed. Chạy theo thứ tự, dừng ở lỗi đầu, in bảng PASS/FAIL, dòng cuối `GATE P5: PASS|FAIL`.
#   bash scripts/gate-p5.sh        # cần Docker (testcontainers) VÀ một gateway ĐÃ SEED cho `check-crm-seed.mjs` (API_URL, SEED_DEFAULT_PASSWORD)
# Các lệnh theo `docs/phases/P5.md` ("Cổng nghiệm thu") + `-race`. Thiếu stack seed → `check-crm-seed.mjs` FAIL, không bao giờ âm thầm xanh.
set -u
cd "$(dirname "$0")/.."
. scripts/gate-lib.sh

run "go vet ./..." go_in go vet -tags testroutes ./...
run "go test -race crm + activity + seedtool" go_in go test -race -count=1 -tags testroutes ./internal/crm/... ./internal/activity/... ./internal/seedtool/...
strict_test TestToolMyAttendance ./internal/crm -race -tags testroutes
strict_test TestToolMyParticipation ./internal/crm -race -tags testroutes
strict_test TestProfileUnder3s ./internal/crm -race -tags testroutes
strict_test TestSeedCRMExpected ./internal/seedtool -race
strict_test TestSeedIsolation ./internal/crm -race -tags testroutes
strict_test TestSeedAdminForbidden ./internal/crm -race -tags testroutes
run "TestNoToolReadsObservations + TestStudentAsksAboutOwnNotes" go_in go test -count=1 -run 'TestNoToolReadsObservations|TestStudentAsksAboutOwnNotes|TestAgentHasNoRiskOrObservationAccess' ./internal/agent
run "lệnh admin seed (allowlist APP_ENV, idempotent)" go_in go test -race -count=1 -run 'TestSeed' ./cmd/gateway
run "go test ./internal/contract/..." go_in go test -count=1 -tags testroutes ./internal/contract/...
run "kế hoạch seed khớp seed/crm_plan.json (SEED_RNG mặc định)" sh -c 'node scripts/crm-plan.mjs --json | cmp -s - seed/crm_plan.json && node scripts/crm-plan.mjs --csv | cmp -s - seed/expected_crm.csv'
run "seed.mjs bị chặn khi NODE_ENV hoặc APP_ENV = production" sh -c 'for v in NODE_ENV APP_ENV; do out=$(env $v=production node scripts/seed.mjs 2>&1); rc=$?; if [ "$rc" -eq 0 ] || ! printf "%s" "$out" | grep -q "bị chặn"; then echo "$v=production: rc=$rc $out"; exit 1; fi; done'
run "check-crm-seed.mjs (stack đã seed)" node scripts/check-crm-seed.mjs --no-rerun
run "Playwright attendance + me + students + activity" fe sh -c 'E2E_API_PORT=${E2E_API_PORT:-3412} pnpm build:gate && E2E_PORT=${E2E_PORT:-3410} E2E_API_PORT=${E2E_API_PORT:-3412} pnpm exec playwright test attendance.spec.ts me.spec.ts students.spec.ts activity.spec.ts --workers=2 --retries=0'
run "ui-antipatterns.sh" bash scripts/ui-antipatterns.sh

finish P5

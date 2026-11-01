#!/usr/bin/env bash
# Cổng nghiệm thu P6 (US-S8-SEED AC6): sổ điểm — công thức quy chế, nhập / chốt điểm, `/me`, dữ liệu seed. Chạy theo thứ tự, dừng ở lỗi đầu, in bảng PASS/FAIL, dòng cuối `GATE P6: PASS|FAIL`.
#   bash scripts/gate-p6.sh        # cần Docker (testcontainers) VÀ một gateway ĐÃ SEED cho `check-grade-seed.mjs` (API_URL, SEED_DEFAULT_PASSWORD)
# Các lệnh theo `docs/phases/P6.md` ("Cổng nghiệm thu") + `-race`. Thiếu stack seed → `check-grade-seed.mjs` FAIL, không bao giờ âm thầm xanh.
set -u
cd "$(dirname "$0")/.."
. scripts/gate-lib.sh

run "go vet ./..." go_in go vet -tags testroutes ./...
run "go test -race ./internal/grade/... (30/30 khớp CSV)" go_in go test -race -count=1 -tags testroutes ./internal/grade/... -v
run "internal/grade không có float64 ngoài test" sh -c '! grep -rn "float64" backend-go/internal/grade --include=*.go | grep -v _test.go'
run "go test ./internal/grading (trích công thức + bản ghi demo)" go_in go test -count=1 -run 'TestExtractGradeScheme|TestExtractLop2OneUnclear|TestDemoModeRecordingsCoverPolicies' ./internal/grading -v
strict_test TestSeedGradebookExpected ./internal/integration
strict_test TestSeedGradebookIsolation ./internal/integration
strict_test TestSeedGradebookAdminForbidden ./internal/integration
strict_test TestSeedRiskAfterGradebook ./internal/integration
run "go test ./internal/contract/..." go_in go test -count=1 -tags testroutes ./internal/contract/...
run "seed.mjs bị chặn khi NODE_ENV hoặc APP_ENV = production" sh -c 'for v in NODE_ENV APP_ENV; do out=$(env $v=production node scripts/seed.mjs 2>&1); rc=$?; if [ "$rc" -eq 0 ] || ! printf "%s" "$out" | grep -q "bị chặn"; then echo "$v=production: rc=$rc $out"; exit 1; fi; done'
run "check-grade-seed.mjs (stack đã seed: 30/30, A B C, lớp 2 → 409, số không đổi khi seed lần hai)" node scripts/check-grade-seed.mjs
run "Playwright gradebook + me" fe sh -c 'E2E_API_PORT=${E2E_API_PORT:-3412} pnpm build:gate && E2E_PORT=${E2E_PORT:-3410} E2E_API_PORT=${E2E_API_PORT:-3412} pnpm exec playwright test gradebook.spec.ts me.spec.ts --workers=2 --retries=0'
run "ui-antipatterns.sh" bash scripts/ui-antipatterns.sh

echo "NHẮC (không tự động): chủ dự án tự tính tay Sinh viên A, B, C bằng máy tính rồi so với giải trình điểm trên màn hình (docs/phases/P6.md — \"Bạn tự kiểm\")."
finish P6

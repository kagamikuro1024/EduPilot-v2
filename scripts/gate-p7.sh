#!/usr/bin/env bash
# Cổng nghiệm thu P7 Bài tập (US-S9-SEED AC7): đúng các lệnh của docs/phases/P7.md + -race + check-assignments-seed.mjs. Mỗi lệnh một dòng PASS/FAIL; thoát 0 khi mọi lệnh xanh.
#   bash scripts/gate-p7.sh     # cần: Docker (testcontainers) và một stack ĐÃ SEED SẠCH (API_URL, SEED_DEFAULT_PASSWORD, `pnpm dev` hoặc `pnpm dev:test-seed`).
# check-assignments-seed.mjs đổi dữ liệu (chấm, công bố, xác nhận bản 2): chạy trên stack seed mới; `grade` → `demo5` → `teams` → `v2` theo đúng thứ tự dưới.
set -u
cd "$(dirname "$0")/.."
. scripts/gate-lib.sh

TAGS="-tags testroutes"
run "go vet ./..." go_in go vet ./...
run "go test -race submission + jobs + grading + submission/source" go_in go test -race -count=1 $TAGS ./internal/submission/... ./internal/jobs/... ./internal/grading/... ./internal/assignment/...
strict_test TestStudentNeverSeesDraftGrade ./internal/integration $TAGS
strict_test TestImapToDraftGrade ./internal/integration $TAGS
strict_test TestGradingJobSurvivesWorkerKill ./internal/integration $TAGS
strict_test TestTeamsSyncAgainstMockGraph ./internal/integration $TAGS
run "go test -race Teams (source)" go_in go test -race -count=1 $TAGS ./internal/submission/source/... -run Teams
run "go test seed Bài tập (isolation, canary, unmatched, v2, imap, teams)" go_in go test -count=1 $TAGS ./internal/integration -run 'TestSeed(ImapAuthHeaders|RosterJoinedVia|GradeAllMatchesExpected|TeamsHappyCounts|UnmatchedBlocksZero|V2AfterResolvingUnmatched|AssignmentsIsolation|AssignmentsAdminForbidden|CanaryNotToStudents)'
run "go test TestExtractLop1V2MatchesExpected" go_in go test -count=1 ./internal/grading -run TestExtractLop1V2MatchesExpected
run "go test ./internal/contract/..." go_in go test -count=1 $TAGS ./internal/contract/...
run "sha256sum -c seed/expected_final_grades.csv.sha256" shasum -a 256 -c seed/expected_final_grades.csv.sha256
run "Playwright assignment-lifecycle + grade-appeal" fe sh -c 'E2E_API_PORT=${E2E_API_PORT:-3412} pnpm build:gate && E2E_PORT=${E2E_PORT:-3410} E2E_API_PORT=${E2E_API_PORT:-3412} pnpm exec playwright test assignment-lifecycle.spec.ts grade-appeal.spec.ts --workers=2 --project=desktop'
run "ui-antipatterns.sh" bash scripts/ui-antipatterns.sh
for m in bt03 counts grade demo5 teams v2; do
  run "check-assignments-seed.mjs $m" node scripts/check-assignments-seed.mjs "$m"
done

cat <<'TXT'

Bạn tự kiểm (chủ dự án tự làm — gate không thay được):
- Gửi 1 mail vào hộp thử đúng mẫu `[BT03] MSSV` và 1 mail sai mẫu → một cái khớp, một cái vào Chưa khớp.
- Đọc 5 bài AI chấm: bằng chứng trích đúng từ bài; nhận xét dùng được.
- Sinh viên KHÔNG thấy điểm trước khi công bố.
- `Đồng bộ từ Teams` kịch bản happy rồi throttle: bài về đủ, tiến độ mượt; no_consent: thông báo đọc hiểu được.
- So một phản hồi của mock-graph với ví dụ trong tài liệu Microsoft Graph: tên trường khớp từng chữ.
- Bắt đầu chấm tay 60 bài cho E3 (việc chỉ bạn làm được).
TXT
finish P7

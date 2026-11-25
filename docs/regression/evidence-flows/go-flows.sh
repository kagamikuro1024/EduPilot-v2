#!/usr/bin/env bash
# qc2 25/11: test Go hẹp cho F1/F2/F14/F15/F18 (container dùng chung edupilot-test-*, mỗi test DB riêng). KHÔNG chạy cả ./...
cd "$(dirname "$0")/../../../backend-go" || exit 2
export EP_SKIP_TIMING=1
run() { echo "### $*"; go test -tags testroutes -count=1 -p 1 "$@" 2>&1 | tail -25; echo "rc=${PIPESTATUS[0]}"; }
run ./internal/auth/... -run 'TestRegister|TestVerify|TestForgot|TestReset|TestLockout|TestAccept|TestRefresh|TestLoginUniform|TestResend|TestThrottle|TestRoster'
run ./internal/course/... -run 'TestAssign|TestArchive|TestRejoin|TestRemove|TestJoin|TestChangeTeacher|TestTeacherManagesOwnTAs|TestPending|TestUndo|TestMember'
run ./internal/integration/... -run 'TestCourseIsolation|TestAllCourseRoutesGuarded|TestAssignTeacher|TestAssignNotify|TestOpenCourseAssign'
run ./internal/today/...
run ./internal/llm/... ./internal/llmconfig/... -run 'Budget|Fallback|Breaker|BatchNoDegrade|Provider|Route|Key|Verify'

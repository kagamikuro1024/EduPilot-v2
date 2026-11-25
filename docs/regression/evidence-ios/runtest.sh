#!/bin/bash
# runtest.sh <TestName> [EMAIL] — XCUITest headless trên simulator qc3-iphone15; mật khẩu đọc từ tệp bí mật, KHÔNG in.
set -u
HERE="$(cd "$(dirname "$0")" && pwd)"
cd "$HERE/uitest" || exit 1
U="${QC_UDID:-$(cat ../udid.txt)}"
EMAIL="${2:-sv.qc3@edupilot.local}"
PW=$(awk -F'\t' -v e="$EMAIL" '$1==e{print $2}' ~/.config/edupilot-team/secrets/vps-qc-accounts.txt)
[ -z "$PW" ] && PW=$(cat ~/.config/edupilot-team/secrets/vps-seed.txt)
export TEST_RUNNER_EMAIL="$EMAIL" TEST_RUNNER_PW="$PW" TEST_RUNNER_EVID="$HERE/shots"
export TEST_RUNNER_SHOT="${SHOT:-shot}"
export TEST_RUNNER_KBD_FORM="${KBD_FORM:-}"
xcodebuild test -project Qc2Ios.xcodeproj -scheme Qc2UITests -destination "id=$U" \
  -only-testing:"Qc2UITests/IosTests/$1" -derivedDataPath ~/.config/edupilot-team/qc3-work/dd-ios \
  2>&1 | grep -E "error:|Test Case|SUCCEEDED|FAILED|XCTAssert" | cut -c1-300

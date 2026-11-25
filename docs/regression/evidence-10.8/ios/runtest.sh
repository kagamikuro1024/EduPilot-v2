#!/bin/bash
# runtest.sh <TestName> [EMAIL] — chạy một ca XCUITest trên simulator qc2-iphone17; mật khẩu lấy từ file bí mật, không in.
cd "$(dirname "$0")/uitest" || exit 1
U=$(cat ../udid.txt)
export TEST_RUNNER_EMAIL="${2:-sv.kha@edupilot.local}" TEST_RUNNER_PW="$(cat ~/.config/edupilot-team/secrets/vps-seed.txt)"
xcodebuild test -project Qc2Ios.xcodeproj -scheme Qc2UITests -destination "id=$U" -only-testing:"Qc2UITests/IosTests/$1" -derivedDataPath ~/.config/edupilot-team/qc2-work/dd 2>&1 | grep -E "error:|Test Case|SUCCEEDED|FAILED" | cut -c1-260

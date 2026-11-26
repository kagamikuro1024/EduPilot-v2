#!/usr/bin/env bash
# Dựng lại AVD API 28 để nghiệm DEF-MOB-12 (INS-09): WebView cũ không parse bundle Next.js.
# Dùng: bash avd-api28.sh build   # cài image + tạo AVD + boot + cài APK debug
#       bash avd-api28.sh check   # mở app, xem WebView + lỗi SyntaxError
#       bash avd-api28.sh rm      # tắt emulator + xoá AVD tạm
set -uo pipefail
SDK=${ANDROID_HOME:-/opt/homebrew/share/android-commandlinetools}
ADB="$SDK/platform-tools/adb"
AVD=${AVD:-ins28}
IMG="system-images;android-28;google_apis;arm64-v8a"
APK=${APK:-$HOME/Documents/as-emu-evidence/app-debug-vps-HEAD.apk}
PKG=vn.edupilot.app
LOGDIR=${LOGDIR:-$HOME/Documents/as-emu-evidence/work}  # KHÔNG ghi vào /tmp
mkdir -p "$LOGDIR"

case "${1:-}" in
build)
  # 1 emulator tại một thời điểm: tắt cái đang chạy trước
  for s in $("$ADB" devices | awk '/emulator-/{print $1}'); do "$ADB" -s "$s" emu kill >/dev/null 2>&1; done; sleep 4
  yes | "$SDK/cmdline-tools/latest/bin/sdkmanager" "$IMG" >/dev/null
  echo no | "$SDK/cmdline-tools/latest/bin/avdmanager" create avd -n "$AVD" -k "$IMG" --force >/dev/null 2>&1
  # -gpu host TREO voi image API 28 (QEMU2 main loop hanging) — phai headless + swiftshader
  nohup "$SDK/emulator/emulator" -avd "$AVD" -no-snapshot -no-boot-anim -no-window \
        -gpu swiftshader_indirect -no-audio > "$LOGDIR/emu-$AVD.log" 2>&1 &
  for _ in $(seq 1 40); do
    [ "$("$ADB" shell getprop sys.boot_completed 2>/dev/null | tr -d '\r')" = "1" ] && break
    sleep 6
  done
  "$ADB" wait-for-device
  "$ADB" install -r "$APK"
  "$ADB" shell am start -W -n "$PKG/.MainActivity" | tail -3
  ;;
check)
  echo "SDK: $("$ADB" shell getprop ro.build.version.sdk | tr -d '\r')"
  "$ADB" shell dumpsys webviewupdate | grep -m1 'Current WebView package'
  "$ADB" shell am force-stop "$PKG"; sleep 1
  "$ADB" shell am start -W -n "$PKG/.MainActivity" | grep -E 'Status|TotalTime'
  sleep 6
  "$ADB" exec-out screencap -p > "$LOGDIR/$AVD-landing.png"
  # mở /checkin trong cùng WebView: đăng nhập rồi đổi route bằng CDP (xem m.py của QC), hoặc bấm tay.
  "$ADB" logcat -c
  sleep 1
  echo "--- sau khi mở /checkin, grep loi cu phap:"
  "$ADB" logcat -d | grep -E 'SyntaxError|Unexpected token|JavaScript Error' | tail -8
  echo "anh: $LOGDIR/$AVD-landing.png (va ảnh /checkin nếu chụp)"
  ;;
rm)
  for s in $("$ADB" devices | awk '/emulator-/{print $1}'); do "$ADB" -s "$s" emu kill >/dev/null 2>&1; done; sleep 3
  "$SDK/cmdline-tools/latest/bin/avdmanager" delete avd -n "$AVD"
  ;;
*)
  echo "dùng: bash avd-api28.sh build|check|rm"; exit 2;;
esac

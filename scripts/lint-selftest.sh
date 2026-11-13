#!/usr/bin/env bash
# Tự kiểm lớp lint (US-PU-01 AC3): gieo TỪNG vi phạm của 7 luật ESLint `ep/*` vào frontend/src/__selftest__/, chạy eslint,
# mong thất bại và nêu đúng tên luật; rồi chạy `scripts/ui-antipatterns.sh --selftest` (mọi phép phải bắt được). Luôn dọn tệp tạm.
# Chạy ở gốc repo: bash scripts/lint-selftest.sh
set -uo pipefail
cd "$(dirname "$0")/.."
DIR=frontend/src/__selftest__
cleanup() { rm -rf "$DIR"; }
trap cleanup EXIT
cleanup

# tên luật | nội dung vi phạm
cases=(
  'ep/no-raw-fetch|export const a = () => fetch("/x");'
  'ep/no-native-dialogs|export const b = () => confirm("?");'
  'ep/no-custom-spinner|export const FullPageSpinner = () => null;'
  'ep/no-raw-table|export const D = () => <table><tbody /></table>;'
  'ep/no-token-in-storage|export const e = () => localStorage.setItem("access_token", "x");'
  'ep/no-tailwind|import "tailwindcss";'
  'ep/no-literal-color-in-style|export const G = () => <p style={{ color: "#ff0000" }}>x</p>;'
)
mkdir -p "$DIR"
ok=0
for c in "${cases[@]}"; do
  rule="${c%%|*}"; body="${c#*|}"
  file="$DIR/$(echo "$rule" | tr '/' '_').tsx"
  printf '%s\n' "$body" >"$file"
  out=$(pnpm -C frontend exec eslint "src/__selftest__/$(basename "$file")" 2>&1); rc=$?
  if [ $rc -ne 0 ] && grep -qF "$rule" <<<"$out"; then ok=$((ok + 1)); else echo "KHÔNG bắt được: $rule (rc=$rc)"; echo "$out" | head -5; fi
  rm -f "$file"
done
echo "$ok / ${#cases[@]} luật ESLint bắt được"
anti=$(bash scripts/ui-antipatterns.sh --selftest); echo "$anti" | grep '^KHÔNG bắt được'
line=$(echo "$anti" | tail -1)
got=$(sed -n 's|^\([0-9]*\) / [0-9]* phép bắt được$|\1|p' <<<"$line")
total=$(sed -n 's|^[0-9]* / \([0-9]*\) phép bắt được$|\1|p' <<<"$line")
echo "${got:-0} / ${total:-0} phép ui-antipatterns bắt được" # số phép do ui-antipatterns.sh tự đếm (22 theo D59, 23 từ 9.1): bắt đủ MỌI phép, không ghim số
[ "$ok" -eq "${#cases[@]}" ] && [ "${total:-0}" -ge 22 ] && [ "${got:-0}" -eq "${total:-0}" ]

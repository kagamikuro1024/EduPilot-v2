#!/usr/bin/env bash
# Kiểm phản mẫu giao diện theo docs/design/DESIGN.md §21 và docs/design/INTEGRATION.md mục 5.
# Chạy ở gốc repo. Thoát mã 1 nếu có vi phạm. Danh sách trắng: thêm chú thích `ui-allow: <lý do>` trên cùng dòng.
set -uo pipefail
SRC="frontend/src"; FAIL=0
[ -d "$SRC" ] || { echo "Không thấy $SRC"; exit 0; }
check() { # tên, mẫu regex, [đường dẫn loại trừ]
  local name="$1" pat="$2" excl="${3:-__none__}"
  local out
  out=$(grep -rnE "$pat" "$SRC" --include=*.ts --include=*.tsx --include=*.css 2>/dev/null \
        | grep -v "ui-allow:" | grep -v "$excl" || true)
  if [ -n "$out" ]; then echo "✗ $name"; echo "$out" | head -20; echo; FAIL=1; else echo "✓ $name"; fi
}
check "Màu viết cứng ngoài shared/styles"        '#[0-9a-fA-F]{3,8}\b|rgba?\(|oklch\('            "shared/styles/"
check "Xám chung chung của Tailwind"              '\b(bg|text|border|ring|divide)-(gray|slate|zinc|neutral|stone)-[0-9]' 
check "Bo góc kiểu SaaS (16–24px)"                'rounded-(2xl|3xl)|rounded-\[(1[6-9]|2[0-9])px\]'
check "Bóng ngoài Popover/Dialog/Menu/Composer"   '\bshadow-(sm|md|lg|xl|2xl)\b|box-shadow:'       "shared/ui/\(Popover\|Dialog\|Menu\|Drawer\|Composer\)"
check "Cỡ chữ tuỳ ý ngoài thang vai trò"          'text-\[[0-9.]+(px|rem)\]'                        "shared/styles/"
check "fetch trần ngoài shared/"                  '\bfetch\('                                       "src/shared/"
check "Spinner toàn trang"                        'FullPageSpinner|fixed inset-0.*animate-spin'
check "Hiệu ứng bị cấm (glass, chữ gradient, nảy)" 'backdrop-blur|bg-clip-text|animate-bounce'
check "Gamification"                              '[Ss]treak|[Ll]eaderboard|[Cc]onfetti'
# Từ kỹ thuật lọt vào màn sinh viên
STU="$SRC/app/(student) $SRC/features/chat $SRC/features/practice $SRC/features/library $SRC/features/me $SRC/features/today"
for d in $STU; do [ -d "$d" ] || continue
  out=$(grep -rnE "[\"'\`>][^\"'\`<]*\b(RAG|PII|fallback|trace|provider|redaction|confidence)\b" "$d" --include=*.tsx | grep -v "ui-allow:" || true)
  [ -n "$out" ] && { echo "✗ Từ kỹ thuật trong màn sinh viên ($d)"; echo "$out" | head -10; FAIL=1; }
done
# Dialog chỉ cho việc cần bảo vệ
out=$(grep -rln "<Dialog" "$SRC" --include=*.tsx | grep -vE "shared/ui/|ConfirmIrreversible|PIIChannelDialog|FinalizeGrades|PublishGrades|ConfirmGradeScheme|DeleteDocument" || true)
[ -n "$out" ] && { echo "✗ <Dialog> ngoài danh sách việc cần bảo vệ (DESIGN.md §10.11):"; echo "$out"; FAIL=1; } || echo "✓ Dialog chỉ dùng cho việc cần bảo vệ"
exit $FAIL

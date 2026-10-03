#!/usr/bin/env bash
# Kiểm phản mẫu giao diện: docs/design/DESIGN.md §21, docs/design/INTEGRATION.md mục 5, docs/specs/FEAT-ui-foundation/SRS.md 4.1 + 4.3.
# 19 phép; mỗi phép in `✓ <tên>` hoặc `✗ <tên>` + tối đa 20 vi phạm; thoát 1 nếu có `✗`.
# Danh sách trắng: thêm chú thích `ui-allow: <lý do>` trên CÙNG dòng (trần 10 chỗ — QC đếm).
# `--selftest`: sao frontend/src sang thư mục tạm (UI_SRC), gieo từng vi phạm, mong `✗`, in `N / 19 phép bắt được`.
# Chạy ở gốc repo. UI_SRC đổi thư mục nguồn (dùng cho --selftest).
set -uo pipefail
export LC_ALL=C

if [ "${1:-}" = "--selftest" ]; then
  tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
  self="$(cd "$(dirname "$0")" && pwd)/$(basename "$0")"
  cp -R frontend/src "$tmp/src"
  # tên phép | tệp gieo (tương đối so với src) | nội dung vi phạm
  seeds=(
    "Màu viết cứng ngoài shared/styles|features/_selftest/a.module.css|.a { color: #ff0000; }"
    "Màu viết cứng trong style của .tsx|features/_selftest/b.tsx|export const B = () => <p style={{ color: \"#abc123\" }}>x</p>;"
    "Xám chung chung của Tailwind|features/_selftest/c.tsx|export const C = () => <p className=\"bg-gray-100\">x</p>;"
    "Bo góc kiểu SaaS / số cứng|features/_selftest/d.module.css|.d { border-radius: 14px; }"
    "Bóng ngoài Popover/Dialog/Menu/Drawer/Composer|features/_selftest/e.module.css|.e { box-shadow: 0 2px 4px #000; }"
    "Cỡ chữ tuỳ ý ngoài thang vai trò|features/_selftest/f.module.css|.f { font-size: 13px; }"
    "z-index số cứng|features/_selftest/g.module.css|.g { z-index: 999; }"
    "fetch / XMLHttpRequest / axios ngoài shared/data|features/_selftest/h.ts|export const h = () => fetch(\"/x\");"
    "confirm / alert|features/_selftest/i.ts|export const i = () => confirm(\"?\");"
    "Spinner riêng hoặc toàn trang|features/_selftest/j.tsx|export const FullPageSpinner = () => <p>x</p>;"
    "<table> thô ngoài DataTable|features/_selftest/k.tsx|export const K = () => <table><tbody /></table>;"
    "Hiệu ứng bị cấm (glass, chữ gradient, nảy)|features/_selftest/l.module.css|.l { backdrop-filter: blur(4px); }"
    "Gamification (streak, leaderboard, confetti)|features/_selftest/m.ts|export const m = \"streak\";"
    "Khung vỏ dùng nền đặc|shared/shell/_selftest.module.css|.n { background: transparent); }"
    "Từ kỹ thuật trong màn sinh viên|features/chat/_selftest.tsx|export const O = () => <p>\"Điểm confidence thấp\"</p>;"
    "Dialog ngoài danh sách việc cần bảo vệ|features/_selftest/p.tsx|export const P = () => <Dialog open />;"
    "Emoji làm icon chức năng|features/_selftest/q.tsx|export const Q = () => <button>🚀</button>;"
    "Token / bí mật ghi vào storage|features/_selftest/r.ts|export const r = () => localStorage.setItem(\"access_token\", \"x\");"
    "Tailwind / @theme|features/_selftest/s.css|@theme { --x: 1; }"
  )
  ok=0
  for s in "${seeds[@]}"; do
    IFS='|' read -r name path body <<<"$s"
    mkdir -p "$tmp/src/$(dirname "$path")"
    printf '%s\n' "$body" >"$tmp/src/$path"
    res=$(UI_SRC="$tmp/src" bash "$self" 2>&1 || true)
    if grep -qF "✗ $name" <<<"$res"; then ok=$((ok + 1)); else echo "KHÔNG bắt được: $name"; fi
    rm -f "$tmp/src/$path"
  done
  echo "$ok / ${#seeds[@]} phép bắt được"
  [ "$ok" -eq "${#seeds[@]}" ]; exit $?
fi

SRC="${UI_SRC:-frontend/src}"; FAIL=0
[ -d "$SRC" ] || { echo "Không thấy $SRC"; exit 0; }

# check "tên" "regex ERE" "regex loại trừ đường dẫn" "regex loại trừ NỘI DUNG dòng" [--include...]
check() {
  local name="$1" pat="$2" xpath="${3:-__none__}" xline="${4:-__none__}" out
  out=$(grep -rnE "$pat" "$SRC" --include=*.ts --include=*.tsx --include=*.css 2>/dev/null \
        | grep -v "ui-allow:" | grep -vE "$xpath" | grep -vE ":[0-9]+:.*($xline)" || true)
  if [ -n "$out" ]; then echo "✗ $name"; echo "$out" | head -20; echo; FAIL=1; else echo "✓ $name"; fi
}

TOK_RADIUS='(0|50%|inherit|var\(--ep-radius-[a-z0-9-]+\))'
check "Màu viết cứng ngoài shared/styles"          '#[0-9a-fA-F]{3,8}\b|\b(rgba?|hsla?|oklch)\('             'shared/styles/'
check "Màu viết cứng trong style của .tsx"         'style=\{\{[^}]*(#[0-9a-fA-F]{3,8}\b|(rgba?|hsla?|oklch)\(|borderRadius: *[0-9]|boxShadow|fontSize: *[0-9]|zIndex: *-?[0-9]{2,})'
check "Xám chung chung của Tailwind"               '\b(bg|text|border|ring|divide)-(gray|slate|zinc|neutral|stone)-[0-9]'
# số cứng: dòng `border-radius:` mà giá trị KHÔNG chỉ gồm 0, 50%, inherit, var(--ep-radius-*) (có thể nhiều giá trị)
out=$( { grep -rnE 'border-radius[[:space:]]*:' "$SRC" --include=*.css --include=*.ts --include=*.tsx 2>/dev/null \
          | grep -v "ui-allow:" | grep -v 'shared/styles/' \
          | grep -vE "border-radius[[:space:]]*:[[:space:]]*$TOK_RADIUS([[:space:]/]+$TOK_RADIUS)*[[:space:]]*(!important)?[[:space:]]*[;,}\"']" || true
        grep -rnE 'rounded-(2xl|3xl)|rounded-\[(1[6-9]|2[0-9])px\]' "$SRC" --include=*.ts --include=*.tsx --include=*.css 2>/dev/null | grep -v "ui-allow:" || true; } )
if [ -n "$out" ]; then echo "✗ Bo góc kiểu SaaS / số cứng"; echo "$out" | head -20; echo; FAIL=1; else echo "✓ Bo góc kiểu SaaS / số cứng"; fi
check "Bóng ngoài Popover/Dialog/Menu/Drawer/Composer" '\bshadow-(sm|md|lg|xl|2xl)\b|box-shadow[[:space:]]*:|drop-shadow\(' 'shared/ui/(Popover|Dialog|Menu|Drawer|Composer)' 'box-shadow[[:space:]]*:[[:space:]]*(var\(--ep-shadow-[a-z-]+\)|var\(--ep-focus\)|none)[[:space:]]*;'
check "Cỡ chữ tuỳ ý ngoài thang vai trò"           'font-size[[:space:]]*:[[:space:]]*[0-9.]+(px|rem|em)|text-\[[0-9.]+(px|rem)\]' 'shared/styles/' 'font-size[[:space:]]*:[[:space:]]*1em[[:space:]]*;'
check "z-index số cứng"                            'z-index[[:space:]]*:[[:space:]]*-?[0-9]{2,}'
check "fetch / XMLHttpRequest / axios ngoài shared/data" '\bfetch\(|XMLHttpRequest|from ["'"'"']axios["'"'"']|window\.fetch' 'src/shared/data/'
check "confirm / alert"                            '(^|[^.A-Za-z0-9_])(window\.)?(confirm|alert|prompt)\('
check "Spinner riêng hoặc toàn trang"              '\b(Full(Page)?|Loading)?Spinner\b|animate-spin' 'shared/ui/'
check "<table> thô ngoài DataTable"                '<table\b' 'shared/ui/DataTable'
check "Hiệu ứng bị cấm (glass, chữ gradient, nảy)" 'backdrop-blur|backdrop-filter|bg-clip-text|animate-bounce'
check "Gamification (streak, leaderboard, confetti)" '[Ss]treak|[Ll]eaderboard|[Cc]onfetti'

# Thanh trên / khung vỏ phải nền đặc: không kính mờ, không màu có độ trong suốt (DESIGN §21, FR-X15, 00-AC12)
out=$(grep -rnE 'backdrop-filter|transparent\)|/ 0\.[0-9]' "$SRC/shared/shell" "$SRC/shared/ui" \
        --include=*.ts --include=*.tsx --include=*.css 2>/dev/null | grep -v "ui-allow:" || true)
if [ -n "$out" ]; then echo "✗ Khung vỏ dùng nền đặc"; echo "$out" | head -10; FAIL=1; else echo "✓ Khung vỏ dùng nền đặc"; fi

# Từ kỹ thuật lọt vào màn sinh viên (INTEGRATION.md §5) và vào chữ hiển thị của shared/ui, shared/domain
bad=""
for d in "$SRC/app/(student)" "$SRC/features/chat" "$SRC/features/practice" "$SRC/features/library" "$SRC/features/me" "$SRC/features/today"; do
  [ -d "$d" ] || continue
  bad+=$(grep -rnE "[\"'\`>][^\"'\`<]*\b(RAG|PII|fallback|trace|provider|redaction|confidence)\b" "$d" --include=*.tsx | grep -v "ui-allow:" || true)
done
for d in "$SRC/shared/ui" "$SRC/shared/domain"; do
  [ -d "$d" ] || continue
  bad+=$(grep -rnE ">[^<>{}]*\b(RAG|PII|fallback|trace|provider|redaction|confidence|embedding|prompt|LLM)\b[^<>{}]*<|(title|label|placeholder|aria-label)=\"[^\"]*\b(RAG|PII|fallback|trace|provider|redaction|confidence|embedding|prompt|LLM)\b" "$d" --include=*.tsx | grep -v "ui-allow:" || true)
done
if [ -n "$bad" ]; then echo "✗ Từ kỹ thuật trong màn sinh viên"; echo "$bad" | head -10; FAIL=1; else echo "✓ Từ kỹ thuật trong màn sinh viên"; fi

# Dialog chỉ cho việc cần bảo vệ (DESIGN.md §10.11)
out=$(grep -rl "<Dialog" "$SRC" --include=*.tsx 2>/dev/null | grep -vE "shared/ui/|app/dev/|ConfirmIrreversible|PIIChannelDialog|FinalizeGrades|PublishGrades|ConfirmGradeScheme|DeleteDocument|SessionSheet" || true)
if [ -n "$out" ]; then echo "✗ Dialog ngoài danh sách việc cần bảo vệ"; echo "$out"; FAIL=1; else echo "✓ Dialog ngoài danh sách việc cần bảo vệ"; fi

# Emoji làm icon chức năng: U+1F300–1FAFF (F0 9F 8C–AB) và U+2600–27BF (E2 98–9E) trong .tsx
out=$(grep -rnE $'\xf0\x9f[\x8c-\xab]|\xe2[\x98-\x9e][\x80-\xbf]' "$SRC" --include=*.tsx 2>/dev/null | grep -v "ui-allow:" || true)
if [ -n "$out" ]; then echo "✗ Emoji làm icon chức năng"; echo "$out" | head -10; FAIL=1; else echo "✓ Emoji làm icon chức năng"; fi

check "Token / bí mật ghi vào storage"             '(localStorage|sessionStorage)\.setItem\([[:space:]]*["'"'"'`][^"'"'"'`]*(token|jwt|access|refresh|secret|password)|document\.cookie[[:space:]]*=[[:space:]]*["'"'"'`][^"'"'"'`]*(token|jwt|access|refresh|secret|password)'
check "Tailwind / @theme"                          '@tailwind|@theme\b|from ["'"'"']@?tailwindcss|import ["'"'"']tailwindcss'
if grep -q tailwind "$(dirname "$SRC")/package.json" 2>/dev/null; then echo "✗ Tailwind / @theme (package.json có tailwind)"; FAIL=1; fi
exit $FAIL

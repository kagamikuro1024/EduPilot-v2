#!/usr/bin/env bash
# Kiểm phản mẫu giao diện: docs/design/DESIGN.md §21, docs/design/INTEGRATION.md mục 5, docs/specs/FEAT-ui-foundation/SRS.md 4.1 + 4.3, docs/specs/FEAT-ui-panels/SRS.md 4.3.
# 31 phép (D59: 20–22 là luật của Panel; 23 = luật chữ UI: không câu giải thích dưới tiêu đề; 24–31 = hợp đồng khung trang DESIGN.md §23, sprint 10.7); mỗi phép in `✓ <tên>` hoặc `✗ <tên>` + tối đa 20 vi phạm; thoát 1 nếu có `✗`.
# Danh sách trắng: thêm chú thích `ui-allow: <lý do>` trên CÙNG dòng (trần 10 chỗ — QC đếm).
# `--selftest`: sao frontend/src sang thư mục tạm (UI_SRC), gieo từng vi phạm, mong `✗`, in `N / 31 phép bắt được`.
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
    "Bóng elevation ngoài Panel|features/_selftest/f.module.css|.e { box-shadow: var(--ep-elevation-1); }"
    "Bóng elevation ngoài Panel|shared/ui/Panel.module.css|.h { box-shadow: 0 2px 4px #000; }"
    "Bo góc panel ngoài Panel|features/_selftest/g.module.css|.g { border-radius: var(--ep-radius-panel); }"
    "Câu mô tả tĩnh dưới tiêu đề|features/_selftest/desc.tsx|export const D = () => <PageHeader title=\"x\" description=\"Câu giải thích rất dài nằm dưới tiêu đề của trang này đấy\" />;"
    "Panel lồng Panel (cùng tệp)|features/_selftest/nest.tsx|export const N = () => (<Panel"$'\n'"  aria-label=\"a\">"$'\n'"<Panel>x</Panel></Panel>);"
    "Khung tự đặt độ rộng|features/_selftest/w.module.css|.a { max-width: 840px; }"
    "Chiều cao theo màn ngoài shared/|features/_selftest/vh.module.css|.b { height: calc(100dvh - 56px); }"
    "Với tay vào primitive|features/_selftest/reach.module.css|.c > :global([data-ep-panel]) { height: 100%; }"
    "Meta lặp lớp đang chọn|features/_selftest/meta.tsx|export const M = () => <PageHeader title=\"x\" meta={[course.label]} />;"
    "Tiêu đề tự dựng ngoài shared/ui|features/_selftest/h2.tsx|export const H = () => <h2 className=\"ep-section-title\">x</h2>;"
    "Nút rỗng trùng nút đầu trang|features/_selftest/dup.tsx|export const D2 = () => (<><PageHeader title=\"x\" primary={{ label: \"Tạo câu hỏi\" }} /><EmptyState title=\"x\" action={<Button>Tạo câu hỏi</Button>} /></>);"
    "Tiêu đề rỗng là câu|features/_selftest/et.tsx|export const E = () => <EmptyState title=\"Chưa có gì. Khi có dữ liệu nó sẽ hiện ở đây.\" />;"
    "Vàng làm chữ|features/_selftest/y.module.css|.d { color: var(--ep-yellow); }"
  )
  miss=0; names=""
  for s in "${seeds[@]}"; do
    IFS='|' read -r -d '' name path body <<<"$s" || true
    case "$names" in *"|$name|"*) ;; *) names+="|$name|" ;; esac
    mkdir -p "$tmp/src/$(dirname "$path")"
    [ -f "$tmp/src/$path" ] && cp "$tmp/src/$path" "$tmp/bak"
    printf '%s' "$body" >"$tmp/src/$path"
    res=$(UI_SRC="$tmp/src" bash "$self" 2>&1 || true)
    if ! grep -qF "✗ $name" <<<"$res"; then echo "KHÔNG bắt được: $name ($path)"; miss=$((miss + 1)); fi
    if [ -f "$tmp/bak" ]; then mv "$tmp/bak" "$tmp/src/$path"; else rm -f "$tmp/src/$path"; fi
  done
  # một phép có thể có nhiều hạt giống (phép 20: (i) ngoài Panel, (ii) bóng lạ trong Panel.module.css)
  total=$(grep -o '||' <<<"$names" | wc -l | tr -d ' '); total=$((total + 1))
  echo "$((total - miss)) / $total phép bắt được"
  [ "$miss" -eq 0 ]; exit $?
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
check "Bóng ngoài Popover/Dialog/Menu/Drawer/Composer" '\bshadow-(sm|md|lg|xl|2xl)\b|box-shadow[[:space:]]*:|drop-shadow\(' 'shared/ui/(Popover|Dialog|Menu|Drawer|Composer)|shared/ui/Panel\.module\.css' 'box-shadow[[:space:]]*:[[:space:]]*(var\(--ep-shadow-[a-z-]+\)|var\(--ep-focus\)|none)[[:space:]]*;'
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
out=$(grep -rl "<Dialog" "$SRC" --include=*.tsx 2>/dev/null | grep -vE "shared/ui/|app/dev/|ConfirmIrreversible|PIIChannelDialog|TrackingNotice|FinalizeGrades|PublishGrades|ConfirmGradeScheme|DeleteDocument|SessionSheet|ImportDialog" || true)
if [ -n "$out" ]; then echo "✗ Dialog ngoài danh sách việc cần bảo vệ"; echo "$out"; FAIL=1; else echo "✓ Dialog ngoài danh sách việc cần bảo vệ"; fi

# Emoji làm icon chức năng: U+1F300–1FAFF (F0 9F 8C–AB) và U+2600–27BF (E2 98–9E) trong .tsx
out=$(grep -rnE $'\xf0\x9f[\x8c-\xab]|\xe2[\x98-\x9e][\x80-\xbf]' "$SRC" --include=*.tsx 2>/dev/null | grep -v "ui-allow:" || true)
if [ -n "$out" ]; then echo "✗ Emoji làm icon chức năng"; echo "$out" | head -10; FAIL=1; else echo "✓ Emoji làm icon chức năng"; fi

check "Token / bí mật ghi vào storage"             '(localStorage|sessionStorage)\.setItem\([[:space:]]*["'"'"'`][^"'"'"'`]*(token|jwt|access|refresh|secret|password)|document\.cookie[[:space:]]*=[[:space:]]*["'"'"'`][^"'"'"'`]*(token|jwt|access|refresh|secret|password)'
check "Tailwind / @theme"                          '@tailwind|@theme\b|from ["'"'"']@?tailwindcss|import ["'"'"']tailwindcss'
if grep -q tailwind "$(dirname "$SRC")/package.json" 2>/dev/null; then echo "✗ Tailwind / @theme (package.json có tailwind)"; FAIL=1; fi

# ---- D59 "panel có kỷ luật" (FEAT-ui-panels SRS 4.3): phép 20–22 -------------------------------------------------------------
# 20: `--ep-elevation-1` chỉ ở shared/styles/ và shared/ui/Panel.module.css; trong Panel.module.css mọi box-shadow khác var(--ep-elevation-1) / none là vi phạm.
out=$(grep -rnE -e '--ep-elevation-1' "$SRC" --include=*.ts --include=*.tsx --include=*.css 2>/dev/null | grep -v "ui-allow:" | grep -vE 'shared/styles/|shared/ui/Panel\.module\.css' || true)
pm="$SRC/shared/ui/Panel.module.css"
if [ -f "$pm" ]; then
  out+=$(grep -nE 'box-shadow[[:space:]]*:' "$pm" | grep -v "ui-allow:" | grep -vE 'box-shadow[[:space:]]*:[[:space:]]*(var\(--ep-elevation-1\)|none)[[:space:]]*;' | sed "s|^|$pm:|" || true)
fi
if [ -n "$out" ]; then echo "✗ Bóng elevation ngoài Panel"; echo "$out" | head -20; echo; FAIL=1; else echo "✓ Bóng elevation ngoài Panel"; fi
# 21: `--ep-radius-panel` chỉ ở shared/styles/ và shared/ui/Panel.module.css
check "Bo góc panel ngoài Panel"                   '[-][-]ep-radius-panel' 'shared/styles/|shared/ui/Panel\.module\.css'
# 22: `<Panel>` nằm giữa một `<Panel>` và `</Panel>` của nó trong CÙNG tệp .tsx (đếm độ sâu; `<Panel … />` không tăng độ sâu; thẻ mở nhiều dòng).
# Trần: không thấy lồng qua ranh giới thành phần — phần đó do kiểm DOM `NEST = 0` ở panels.spec.ts.
out=$(find "$SRC" -name '*.tsx' ! -path '*/shared/ui/Panel.tsx' -print0 2>/dev/null | xargs -0 perl -0777 -ne '
  my $d = 0;
  while (/<Panel(?=[\s>\/])((?:[^>"{]|"[^"]*"|\{(?:[^{}]|\{[^{}]*\})*\})*)>|<\/Panel>/g) {
    if (defined $1) {
      if ($d > 0) { my $l = 1 + (substr($_, 0, $-[0]) =~ tr/\n//); print "$ARGV:$l: <Panel> lồng trong <Panel>\n"; }
      $d++ unless $1 =~ /\/\s*$/;
    } else { $d-- if $d > 0; }
  } continue { close ARGV if eof }
' 2>/dev/null | grep -v "ui-allow:" || true)
if [ -n "$out" ]; then echo "✗ Panel lồng Panel (cùng tệp)"; echo "$out" | head -20; echo; FAIL=1; else echo "✓ Panel lồng Panel (cùng tệp)"; fi
# 23: luật chữ UI (CLAUDE.md 09/10/2026, FEAT-uiux-91 US-01 AC6): `description=` của <PageHeader> / <Section> là chuỗi tĩnh > 40 ký tự thì là câu giải thích → xoá.
# Dữ liệu (`description={`…`}`, biểu thức) và chuỗi ngắn thì được. Bỏ qua Demo*.tsx (chỉ phiên mô phỏng dùng) và shared/ui. `ui-allow:` cùng dòng thẻ mở để miễn.
out=$(find "$SRC" -name '*.tsx' ! -path '*/shared/ui/*' ! -path '*/app/dev/*' ! -name 'Demo*.tsx' -print0 2>/dev/null | xargs -0 perl -0777 -ne '
  while (/<(?:PageHeader|Section)(?=[\s>])((?:[^>"{]|"[^"]*"|\{(?:[^{}]|\{[^{}]*\})*\})*)>/g) {
    my $attrs = $1; my $at = $-[0];
    if ($attrs =~ /\bdescription="([^"]{41,})"/ && $attrs !~ /ui-allow:/) { my $l = 1 + (substr($_, 0, $at) =~ tr/\n//); print "$ARGV:$l: description tĩnh dài\n"; }
  } continue { close ARGV if eof }
' 2>/dev/null || true)
if [ -n "$out" ]; then echo "✗ Câu mô tả tĩnh dưới tiêu đề"; echo "$out" | head -20; echo; FAIL=1; else echo "✓ Câu mô tả tĩnh dưới tiêu đề"; fi
# ---- hợp đồng khung trang (DESIGN.md §23, sprint 10.7): phép 24–31 ----------------------------------------------------------
# Nợ khung: route CHƯA chuyển sang hợp đồng (scripts/ui-frame-debt.txt, mỗi dòng một regex đường dẫn). Mỗi lô sửa route xoá dòng tương ứng; hết lô 7 tệp phải rỗng.
DEBT=$(grep -vE '^\s*(#|$)' "$(dirname "$0")/ui-frame-debt.txt" 2>/dev/null | paste -sd'|' - || true)
[ -n "$DEBT" ] || DEBT='__none__'
# "CSS route" = *.module.css dưới features/ và app/, trừ landing, dev, _selftest không bị loại
route_css() { find "$SRC/features" "$SRC/app" -name '*.module.css' ! -path '*/features/landing/*' ! -path '*/app/dev/*' 2>/dev/null; }
route_tsx() { find "$SRC/features" "$SRC/app" "$SRC/shared/shell" -name '*.tsx' ! -path '*/features/landing/*' ! -path '*/app/dev/*' 2>/dev/null; }
report() { # tên, nội dung vi phạm (đã lọc nợ và ui-allow)
  local out; out=$(printf '%s' "$2" | grep -v "ui-allow:" | grep -vE "$DEBT" || true)
  if [ -n "$out" ]; then echo "✗ $1"; echo "$out" | head -20; echo; FAIL=1; else echo "✓ $1"; fi
}
# 24: khung tự đặt độ rộng (≥ 600 px / ≥ 30 rem / vw) hoặc tự căn giữa trong CSS route. Ô nhập, cột chữ (< 600 px, ch) không bị bắt.
report "Khung tự đặt độ rộng" "$(route_css | xargs grep -nE '(max-|min-)?width[[:space:]]*:[^;]*(([6-9][0-9]{2}|[0-9]{4,})px|[3-9][0-9](\.[0-9]+)?rem|[0-9]+vw)|margin(-inline)?[[:space:]]*:[^;]*\bauto\b' 2>/dev/null | grep -v '@media' || true)"
# 25: chiều cao theo màn ngoài shared/ (chỉ Split được đặt)
report "Chiều cao theo màn ngoài shared/" "$(route_css | xargs grep -nE '(min-|max-)?height[[:space:]]*:[^;]*[0-9](d|s|l)?vh' 2>/dev/null || true)"
# 26: CSS route với tay vào phần tử của primitive
report "Với tay vào primitive" "$(route_css | xargs grep -nE 'data-ep-(panel|page|empty|section|split)|\.page\.page' 2>/dev/null || true)"
# 27: meta của PageHeader lặp lớp đang chọn (thanh trên đã có)
report "Meta lặp lớp đang chọn" "$(route_tsx | xargs perl -CSD -0777 -ne '
  while (/<PageHeader(?=[\s>])((?:[^>"{]|"[^"]*"|\{(?:[^{}]|\{(?:[^{}]|\{[^{}]*\})*\})*\})*)>/g) {
    my $a = $1; my $at = $-[0];
    if ($a =~ /\bmeta=\{(?:[^{}]|\{(?:[^{}]|\{[^{}]*\})*\})*(?:course(?:Label|Code|Name)|class_?[cC]ode|course\??\.(?:label|code|name|class_code))/ && $a !~ /ui-allow:/) { my $l = 1 + (substr($_, 0, $at) =~ tr/\n//); print "$ARGV:$l: meta lặp lớp\n"; }
  } continue { close ARGV if eof }
' 2>/dev/null || true)"
# 28: tiêu đề tự dựng (h1 / h2 / lớp tiêu đề) ngoài shared/ui — chỉ PageHeader và Section được vẽ
report "Tiêu đề tự dựng ngoài shared/ui" "$(route_tsx | xargs grep -nE '<h1\b|<h2\b|ep-page-title|ep-section-title' 2>/dev/null || true)"
# 29: trong CÙNG tệp, nhãn nút của EmptyState trùng nhãn primary của PageHeader (nút rỗng lặp hành động đầu trang). Trần: không thấy qua ranh giới tệp (đo bằng E4 ở frame.spec).
report "Nút rỗng trùng nút đầu trang" "$(route_tsx | xargs perl -CSD -0777 -ne '
  my %p; while (/primary=\{\{[^}]*?label:\s*"([^"]+)"/g) { $p{$1} = 1 }
  while (/<EmptyState(?=[\s>\/])((?:[^>"{]|"[^"]*"|\{(?:[^{}]|\{(?:[^{}]|\{[^{}]*\})*\})*\})*)\/?>/g) {
    my $a = $1; my $at = $-[0];
    for my $lab (keys %p) { if ($a =~ /action=.*>\s*\Q$lab\E\s*</s && $a !~ /ui-allow:/) { my $l = 1 + (substr($_, 0, $at) =~ tr/\n//); print "$ARGV:$l: nút rỗng «$lab» trùng primary\n"; } }
  } continue { close ARGV if eof }
' 2>/dev/null || true)"
# 30: tiêu đề EmptyState tĩnh dài hơn 40 ký tự, có dấu chấm giữa chuỗi, hoặc kết thúc bằng dấu chấm (là câu, không phải tiêu đề)
report "Tiêu đề rỗng là câu" "$(route_tsx | xargs perl -CSD -0777 -ne '
  while (/<EmptyState(?=[\s>\/])((?:[^>"{]|"[^"]*"|\{(?:[^{}]|\{(?:[^{}]|\{[^{}]*\})*\})*\})*)\/?>/g) {
    my $a = $1; my $at = $-[0];
    if ($a =~ /\btitle="([^"]*)"/ && $a !~ /ui-allow:/) { my $t = $1; if (length($t) > 40 || $t =~ /\.\s/ || $t =~ /\.$/) { my $l = 1 + (substr($_, 0, $at) =~ tr/\n//); print "$ARGV:$l: tiêu đề rỗng là câu\n"; } }
  } continue { close ARGV if eof }
' 2>/dev/null || true)"
# 31: vàng chỉ làm nền / chấm / vạch — cấm làm chữ, viền, icon (1,69 : 1 trên trắng, không đạt WCAG)
out=$(grep -rnE '(^|[^-])(color|border(-[a-z]+)?-color|stroke|fill)[[:space:]]*:[^;]*--ep-yellow\b' "$SRC" --include=*.css 2>/dev/null | grep -vE 'features/landing/' || true)
report "Vàng làm chữ" "$out"
exit $FAIL

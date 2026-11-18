#!/usr/bin/env bash
# smkP: TC-PU05-25 — QC tự gieo 8 vi phạm (mỗi lượt một phép) vào BẢN SAO frontend/src (UI_SRC), chạy scripts/ui-antipatterns.sh, mong rc≠0 + đúng tên phép; xoá tệp; rc=0 lại. Không dùng --selftest của dev. Chạy ở gốc repo.
set -u; tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT; cp -R frontend/src "$tmp/src"
seeds=(
 "Màu viết cứng ngoài shared/styles|features/_smkp/a.module.css|.a { color: #12ab34; }"
 "Bo góc kiểu SaaS|features/_smkp/b.module.css|.b { border-radius: 16px; }"
 "Bóng ngoài Popover|features/_smkp/c.module.css|.c { box-shadow: 0 4px 8px rgba(0,0,0,.3); }"
 "Cỡ chữ tuỳ ý|features/_smkp/d.module.css|.d { font-size: 13px; }"
 "fetch / XMLHttpRequest|features/_smkp/e.ts|export const e = () => fetch('/x');"
 "Emoji làm icon|features/_smkp/f.tsx|export const F = () => <button>🔍</button>;"
 "Token / bí mật ghi vào storage|features/_smkp/g.ts|export const g = () => localStorage.setItem('token', 'x');"
 "Gamification|features/_smkp/h.ts|export const streak = 3;"
)
ok=0
base=$(UI_SRC="$tmp/src" bash scripts/ui-antipatterns.sh >/dev/null 2>&1; echo $?); echo "cây sạch (bản sao): rc=$base"
for s in "${seeds[@]}"; do IFS='|' read -r name rel body <<<"$s"; f="$tmp/src/$rel"; mkdir -p "$(dirname "$f")"; printf '%s\n' "$body" >"$f"
  out=$(UI_SRC="$tmp/src" bash scripts/ui-antipatterns.sh 2>&1); rc=$?; rm -f "$f"; rmdir "$(dirname "$f")" 2>/dev/null
  line=$(grep -F "$name" <<<"$out" | head -1)
  if [ $rc -ne 0 ] && [[ "$line" == ✗* ]]; then ok=$((ok+1)); echo "BẮT ĐƯỢC rc=$rc: $line"; else echo "KHÔNG BẮT rc=$rc: $name"; grep '✗' <<<"$out" | head -3; fi
done
after=$(UI_SRC="$tmp/src" bash scripts/ui-antipatterns.sh >/dev/null 2>&1; echo $?); echo "sau xoá: rc=$after; bắt được $ok/${#seeds[@]}"

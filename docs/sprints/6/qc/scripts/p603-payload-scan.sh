#!/usr/bin/env bash
# QC US-P3-03 (TC 05–08, 22–24): quét MỌI payload provider giả tìm họ tên / MSSV / email / SĐT / CCCD của roster.
#
#   p603-payload-scan.sh --build-roster "$DATABASE_URL" "<course_id>" [--roster FILE]
#   p603-payload-scan.sh --dump "$DUMP" --roster /tmp/roster.txt [--min-payloads 30] [--min-placeholders 10]
#
# rc=0 khi sạch và đạt ngưỡng; rc=1 khi có chuỗi cấm hoặc phép đo rỗng nghĩa (quá ít payload / 0 placeholder).
set -uo pipefail

DUMP=""; ROSTER="/tmp/qc-roster.txt"; MINP=0; MINPH=0; BUILD=""; COURSE=""
while [ $# -gt 0 ]; do
  case "$1" in
    --dump) DUMP="$2"; shift 2 ;;
    --roster) ROSTER="$2"; shift 2 ;;
    --min-payloads) MINP="$2"; shift 2 ;;
    --min-placeholders) MINPH="$2"; shift 2 ;;
    --build-roster) BUILD="$2"; COURSE="$3"; shift 3 ;;
    *) echo "tham số lạ: $1" >&2; exit 2 ;;
  esac
done

HERE="$(cd "$(dirname "$0")" && pwd)"

if [ -n "$BUILD" ]; then
  python3 "$HERE/p602-attacks.py" --dsn "$BUILD" --course "$COURSE" --build-roster "$ROSTER" || exit 1
  exit 0
fi

[ -d "${DUMP:-}" ] || { echo "cần --dump <thư mục payload provider giả>" >&2; exit 2; }
[ -s "$ROSTER" ] || { echo "cần --roster (sinh bằng --build-roster)" >&2; exit 2; }

TMP="$(mktemp)"; trap 'rm -f "$TMP" "$TMP.list" "$TMP.hit"' EXIT
find "$DUMP" -type f \( -name '*.json' -o -name '*.jsonl' -o -name '*.log' \) | sort > "$TMP.list"
N=$(wc -l < "$TMP.list" | tr -d ' ')
echo "payload quét: $N"

# Chuỗi cấm: bỏ dòng rỗng và dòng ngắn (< 4 ký tự dễ trùng từ thường).
awk 'length($0) >= 4' "$ROSTER" > "$TMP"
echo "chuỗi cấm: $(wc -l < "$TMP" | tr -d ' ')"

BAD=0
if [ "$N" -gt 0 ]; then
  # -i: bắt cả biến thể hoa-thường mô hình/pipeline có thể đổi; -F: so nguyên văn, không regex.
  tr '\n' '\0' < "$TMP.list" | xargs -0 grep -ioFf "$TMP" > "$TMP.hit" 2>/dev/null
  if [ -s "$TMP.hit" ]; then
    echo "RÒ RỈ — payload chứa chuỗi cấm:"
    sort -u "$TMP.hit" | head -50
    BAD=$(sort -u "$TMP.hit" | wc -l | tr -d ' ')
  fi
fi

PH=0
if [ "$N" -gt 0 ]; then
  PH=$(tr '\n' '\0' < "$TMP.list" | xargs -0 grep -ohE '\[\[(SV|MSSV|EMAIL|SDT|CCCD)_[0-9]+\]\]' 2>/dev/null | wc -l | tr -d ' ')
fi
echo "placeholder thấy: $PH"

RC=0
[ "$BAD" -gt 0 ] && { echo "KẾT LUẬN: FAIL — $BAD chuỗi cấm trong payload"; RC=1; }
[ "$N" -lt "$MINP" ] && { echo "KẾT LUẬN: FAIL — chỉ $N payload, cần ≥ $MINP (phép đo rỗng nghĩa)"; RC=1; }
[ "$PH" -lt "$MINPH" ] && { echo "KẾT LUẬN: FAIL — chỉ $PH placeholder, cần ≥ $MINPH (payload có thể chưa qua hook che)"; RC=1; }
[ "$RC" -eq 0 ] && echo "KẾT LUẬN: PASS — 0 chuỗi cấm, $N payload, $PH placeholder"
exit $RC

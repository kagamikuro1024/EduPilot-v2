#!/usr/bin/env bash
# Dựng 4 pane herdr cho đội PM/BA/DEV/QC tại gốc repo. Chạy TRONG một pane shell của herdr (cần HERDR_ENV=1).
# Yêu cầu: herdr ≥ 0.9, jq, claude (Claude Code).
set -euo pipefail
[ "${HERDR_ENV:-}" = "1" ] || { echo "Hãy chạy script này trong một pane của herdr."; exit 1; }
command -v jq >/dev/null || { echo "Cần jq"; exit 1; }
ROOT=$(git rev-parse --show-toplevel)
cd "$ROOT"
mkdir -p docs/sprints docs/specs

tab=$(herdr tab create --label team)
pm=$(printf '%s\n' "$tab" | jq -r '.result.root_pane.pane_id')
s=$(herdr pane split "$pm" --direction right --ratio 0.5 --no-focus); dev=$(printf '%s\n' "$s" | jq -r '.result.pane.pane_id')
s=$(herdr pane split "$pm" --direction down --ratio 0.5 --no-focus);  ba=$(printf '%s\n' "$s" | jq -r '.result.pane.pane_id')
s=$(herdr pane split "$dev" --direction down --ratio 0.5 --no-focus); qc=$(printf '%s\n' "$s" | jq -r '.result.pane.pane_id')
for p in $pm $ba $dev $qc; do herdr pane run "$p" "cd '$ROOT'"; done

# BA chỉ viết docs → mặc định. DEV/QC sửa file nhiều → acceptEdits; lệnh shell được phép khai ở .claude/settings.json
herdr agent start ba  --kind claude --pane "$ba"
herdr agent start dev --kind claude --pane "$dev" -- --permission-mode acceptEdits
herdr agent start qc  --kind claude --pane "$qc"  -- --permission-mode acceptEdits
herdr agent start pm  --kind claude --pane "$pm"

herdr agent prompt ba  "Bạn là BA. Đọc docs/team/BA.md và CLAUDE.md, rồi trả lời 'BA sẵn sàng' và chờ PM giao việc." --wait --timeout 180000 >/dev/null
herdr agent prompt dev "Bạn là Dev. Đọc docs/team/DEV.md và CLAUDE.md, rồi trả lời 'Dev sẵn sàng' và chờ PM giao việc." --wait --timeout 180000 >/dev/null
herdr agent prompt qc  "Bạn là QC. Đọc docs/team/QC.md và CLAUDE.md, rồi trả lời 'QC sẵn sàng' và chờ PM giao việc." --wait --timeout 180000 >/dev/null
herdr pane focus "$pm"
echo "Đội đã sẵn sàng: pm=$pm ba=$ba dev=$dev qc=$qc"
echo "Bấm vào pane pm, dán nội dung docs/team/PM.md, rồi gõ: bắt đầu sprint 1"

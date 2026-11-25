#!/bin/bash
# codeloop.sh <giây>: ghi mã điểm danh (short_code) hiện tại của cửa sổ trong win.json ra code.txt mỗi 10 s (mã xoay mỗi 60 s).
cd "$(dirname "$0")"; end=$((SECONDS+${1:-300}))
while [ $SECONDS -lt $end ]; do python3 - <<'PY'
import json, os, vps
W = json.load(open(os.path.expanduser("~/.config/edupilot-team/qc2-work/win.json")))
c, d = vps.call("teacher", "GET", f"/courses/{W['cid']}/sessions/{W['sid']}/checkin-windows/{W['wid']}/display")
if c == 200: open(os.path.expanduser("~/.config/edupilot-team/qc2-work/code.txt"), "w").write(d["short_code"])
PY
sleep 10; done

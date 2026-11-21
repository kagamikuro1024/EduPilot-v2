#!/bin/bash
# usage: runwin.sh script.py  — mở cửa sổ fake, chạy, đóng (luôn đóng)
cd /tmp/qc3e; python3 win.py on >/dev/null 2>&1; timeout 900 python3 "$@"; python3 win.py off >/dev/null 2>&1

#!/usr/bin/env bash
# Đo «học từ lần sửa» theo phiên bản lưu ý (US-GD-10). Xem run.py.
set -euo pipefail
exec python3 "$(dirname "$0")/run.py" "$@"

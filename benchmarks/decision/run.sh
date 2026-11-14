#!/usr/bin/env bash
# Đo trước / sau của decision model (US-DM-08). Xem run.py.
set -euo pipefail
exec python3 "$(dirname "$0")/run.py" "$@"

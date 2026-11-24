#!/bin/bash
# Chờ hết cửa sổ 429 (AI_SUMMARY_PER_HOUR) rồi gọi nhận xét 360 sau bản sửa 3600257a + AI_SUMMARY_TIMEOUT=60s.
cd "$(dirname "$0")"
sleep 720
bash ../opencode/usage.sh | cut -c1-120 > usage-before-summary.json
python3 summary_retry.py
bash ../opencode/usage.sh | cut -c1-120 > usage-after-summary.json
echo SUMMARY_DONE

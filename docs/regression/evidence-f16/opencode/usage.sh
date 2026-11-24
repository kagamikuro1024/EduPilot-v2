#!/bin/bash
# Hạn mức gói OpenCode Go (khoá đọc từ tệp, không in). Dùng: bash usage.sh
K=$(cat ~/.config/edupilot-team/secrets/opencode-go.key | tr -d '[:space:]')
curl -s -m 20 https://opencode.ai/inference/go/v1/usage -H "Authorization: Bearer $K"; echo

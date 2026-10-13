#!/usr/bin/env bash
# QC US-P4-07 TC-01/02: gửi tập câu khủng hoảng qua API chat riêng thật, đếm intent=CRISIS.
# Dùng: GW=https://localhost:773 SVA='Bearer …' C1=<uuid lớp> scripts/crisis-run.sh pos|neg
# Câu mẫu: docs/specs/FEAT-escalation-notify/SRS.md mục 9.5 (QC chép vào crisis-sets.json: {"pos":[…20],"neg":[…17]}).
set -euo pipefail
set_name=${1:?pos|neg}
here=$(cd "$(dirname "$0")" && pwd)
mapfile -t lines < <(jq -r ".${set_name}[]" "$here/crisis-sets.json")
hit=0; n=0
for s in "${lines[@]}"; do
  sid=$(curl -sk -X POST "$GW/api/v1/chat/sessions" -H "Authorization: $SVA" -H 'Content-Type: application/json' -d "{\"course_id\":\"$C1\"}" | jq -r .id)
  mid=$(curl -sk -X POST "$GW/api/v1/chat/sessions/$sid/messages" -H "Authorization: $SVA" -H 'Content-Type: application/json' -d "$(jq -nc --arg c "$s" '{content:$c}')" | jq -r '.message.id // .id')
  curl -skN "$GW/api/v1/chat/messages/$mid/stream" -H "Authorization: $SVA" -m 20 >/dev/null || true
  intent=$(curl -sk "$GW/api/v1/chat/sessions/$sid/messages" -H "Authorization: $SVA" | jq -r '[.items[]?|select(.role=="USER")][0].intent // [.messages[]?|select(.role=="USER")][0].intent')
  n=$((n+1)); [ "$intent" = CRISIS ] && hit=$((hit+1))
  printf '%s\t%s\n' "$intent" "$s"
done
echo "crisis=$hit/$n set=$set_name"

#!/usr/bin/env bash
# QC US-P4-07 TC-01/02: gửi tập câu khủng hoảng qua API chat riêng thật, đếm intent=CRISIS (đọc từ chat_messages).
# Dùng: . /tmp/qc/h.sh; refresh; bash crisis-run.sh pos|neg   (cần newsess, ask, PSQL, SVA, C1 từ /tmp/qc/h.sh)
set -u
set_name=${1:?pos|neg}; here=$(cd "$(dirname "$0")" && pwd)
hit=0; n=0
while IFS= read -r -u 3 s; do
  sid=$(newsess "$SVA"); ask "$SVA" "$sid" "$s" >/dev/null 2>&1
  intent=$(PSQL "select intent from chat_messages where session_id='$sid' and role='ASSISTANT' order by id limit 1")
  n=$((n+1)); [ "$intent" = CRISIS ] && hit=$((hit+1)); printf '%s\t%s\n' "${intent:-?}" "$s"
done 3< <(jq -r ".${set_name}[]" "$here/crisis-sets.json")
echo "crisis=$hit/$n set=$set_name"

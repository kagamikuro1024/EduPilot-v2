#!/usr/bin/env bash
# QC sprint 6 — quét rò rỉ thông tin cá nhân (US-P3-03 AC…, US-P3-05 AC13, US-P3-06 AC6).
# Quét: log compose (gateway + worker, nơi payload gửi provider giả đi qua), llm_audit, và mọi bảng
# có thể lưu chữ thô (chat_messages, forum_threads, forum_posts, pii_events, outbox, jobs, audit_log).
# Kết quả mong đợi: mọi dòng "log compose"/"<bảng>" = 0, trừ các bảng được phép giữ chữ thô của CHÍNH
# sinh viên (chat_messages.content — chat riêng được phép, xem cột "được phép").
#
# Dùng:
#   NAME='Vũ Hoàng Giang' MSSV=20229001 EMAIL=sv.gioi@edupilot.local \
#   PHONE=0912345678 CCCD=001099012345 bash docs/sprints/6/qc/scripts/scan-pii-leak.sh
#
# Biến: COMPOSE (mặc định stack local), PSQL (mặc định psql trong container postgres).
set -u
COMPOSE="${COMPOSE:-docker compose --env-file .env.local -f docker-compose.local.yml -p edupilot}"
PSQL="${PSQL:-$COMPOSE exec -T postgres psql -U edupilot -d edupilot -tAc}"
NAME="${NAME:-Vũ Hoàng Giang}"
NAME_NOACC="${NAME_NOACC:-Vu Hoang Giang}"
MSSV="${MSSV:-20229001}"
EMAIL="${EMAIL:-sv.gioi@edupilot.local}"
PHONE="${PHONE:-0912345678}"
CCCD="${CCCD:-001099012345}"

NEEDLES=("$NAME" "$NAME_NOACC" "$MSSV" "$EMAIL" "$PHONE" "$CCCD")

echo "== log compose (payload gửi provider giả + mọi log dịch vụ) =="
LOGS="$($COMPOSE logs --no-color 2>&1)"
for n in "${NEEDLES[@]}"; do
  printf '%-28s : %s\n' "log: $n" "$(printf '%s' "$LOGS" | grep -cF "$n" || true)"
done
printf '%-28s : %s\n' "log: placeholder [[SV_" "$(printf '%s' "$LOGS" | grep -cE '\[\[\s*(SV|MSSV|EMAIL|SDT|CCCD)(_[0-9]+)?' || true)"

echo
echo "== bảng không bao giờ được chứa chữ thô =="
for n in "${NEEDLES[@]}"; do
  s=$(printf '%s' "$n" | sed "s/'/''/g")
  for t in "llm_audit::coalesce(error_kind,'')||coalesce(provider,'')||coalesce(model,'')" \
           "forum_threads::title||body" \
           "forum_posts::body||coalesce(ai_body,'')||citations::text" \
           "pii_events::id::text" \
           "outbox::payload::text" \
           "jobs::coalesce(payload::text,'')||coalesce(result::text,'')||coalesce(error::text,'')" \
           "audit_log::coalesce(before::text,'')||coalesce(after::text,'')"; do
    tbl="${t%%::*}"; expr="${t##*::}"
    printf '%-28s : %s\n' "$tbl ← $n" "$($PSQL "select count(*) from $tbl where ($expr) like '%$s%'" 2>/dev/null || echo ERR)"
  done
done

echo
echo "== được phép (chat riêng của chính sinh viên giữ chữ thô; chỉ in để đối chiếu) =="
for n in "$NAME" "$MSSV"; do
  s=$(printf '%s' "$n" | sed "s/'/''/g")
  printf '%-28s : %s\n' "chat_messages ← $n" "$($PSQL "select count(*) from chat_messages where content like '%$s%' or coalesce(partial_content,'') like '%$s%'" 2>/dev/null || echo ERR)"
done

echo
echo "== placeholder không được lọt ra chữ người dùng thấy =="
printf '%-28s : %s\n' "chat_messages placeholder" "$($PSQL "select count(*) from chat_messages where content ~* '\[\[\s*(SV|MSSV|EMAIL|SDT|CCCD)(_[0-9]+)?'" 2>/dev/null || echo ERR)"
printf '%-28s : %s\n' "forum_posts placeholder" "$($PSQL "select count(*) from forum_posts where body ~* '\[\[\s*(SV|MSSV|EMAIL|SDT|CCCD)(_[0-9]+)?'" 2>/dev/null || echo ERR)"

#!/usr/bin/env bash
# Canary khoá API (US-P1-04 AC4): tạo nhà cung cấp với khoá "canary", thao tác đủ loại, rồi quét log compose và các bảng có thể
# lộ khoá (audit_log, outbox, llm_audit, jobs, llm_providers). Kết quả mong đợi: mọi dòng đếm = 0. Cần stack test đang chạy.
# Dùng: GW=https://localhost A="Authorization: Bearer <token ADMIN>" ./scripts/canary-scan.sh
set -euo pipefail
GW="${GW:-https://localhost}"
: "${A:?đặt A='Authorization: Bearer <token ADMIN>' (go run ./cmd/gateway token --role ADMIN)}"
CANARY="${CANARY:-sk-CANARY-$(date +%s)-do-not-leak}"
COMPOSE="${COMPOSE:-docker compose --env-file .env.local -f docker-compose.local.yml -p edupilot -f docker-compose.test.yml}"
PSQL="${PSQL:-$COMPOSE exec -T postgres psql -U edupilot -d edupilot -tAc}"
api() { curl -sk -H "$A" -H 'Content-Type: application/json' "$@"; }
idem() { echo "Idempotency-Key: canary-$RANDOM$RANDOM$RANDOM"; }
M='[{"model":"fake-chat","kind":"chat","price_in":"0","price_out":"0"}]'
out=""
# tạo: khoá sai (422), skip_verify (201), thân hỏng, loại lạ, đổi khoá, Test mọi kiểu, đọc, xoá
out+=$(api -H "$(idem)" -X POST "$GW/api/v1/admin/llm/providers" -d "{\"type\":\"fake\",\"name\":\"CN0\",\"api_key\":\"$CANARY\",\"models\":$M}")
made=$(api -H "$(idem)" -X POST "$GW/api/v1/admin/llm/providers" -d "{\"type\":\"fake\",\"name\":\"CN1\",\"api_key\":\"$CANARY\",\"skip_verify\":true,\"models\":$M}")
out+="$made"
ID=$(echo "$made" | sed -n 's/.*"id":"\([0-9a-f-]\{36\}\)".*/\1/p' | head -1)
out+=$(api -H "$(idem)" -X POST "$GW/api/v1/admin/llm/providers" -d "{\"type\":\"nope\",\"name\":\"CN2\",\"api_key\":\"$CANARY\"")
out+=$(api -X PUT "$GW/api/v1/admin/llm/providers/$ID" -d "{\"type\":\"fake\",\"name\":\"CN1\",\"api_key\":\"${CANARY}2\",\"skip_verify\":true,\"version\":1}")
out+=$(api -X PUT "$GW/api/v1/admin/llm/providers/$ID" -d "{\"type\":\"fake\",\"name\":\"CN1\",\"api_key\":\"${CANARY}3\",\"version\":2}")
out+=$(api -X POST "$GW/api/v1/admin/llm/providers/test" -d "{\"type\":\"fake\",\"api_key\":\"$CANARY\",\"model\":\"fake-chat\"}")
out+=$(api -X POST "$GW/api/v1/admin/llm/providers/$ID/test" -d "{\"api_key\":\"$CANARY\"}")
out+=$(api -X POST "$GW/api/v1/admin/llm/providers/$ID/test")
out+=$(api "$GW/api/v1/admin/llm/providers")
out+=$(api "$GW/api/v1/admin/llm/routes")
out+=$(api -X DELETE "$GW/api/v1/admin/llm/providers/$ID")
n() { printf '%s\n' "$out" | grep -c "$CANARY" || true; }
printf '%-18s : %s\n' "phản hồi API" "$(n)"
printf '%-18s : %s\n' "log compose" "$($COMPOSE logs --no-color 2>&1 | grep -c "$CANARY" || true)"
for q in "select count(*) from audit_log where before::text like '%$CANARY%' or after::text like '%$CANARY%'" \
         "select count(*) from outbox where payload::text like '%$CANARY%'" \
         "select count(*) from llm_audit where provider like '%$CANARY%' or model like '%$CANARY%' or error_kind like '%$CANARY%'" \
         "select count(*) from jobs where coalesce(result::text,'') like '%$CANARY%' or coalesce(error::text,'') like '%$CANARY%'" \
         "select count(*) from llm_providers where position(convert_to('$CANARY','UTF8') in coalesce(api_key_enc,''::bytea)) > 0"; do
  printf '%-18s : %s\n' "$(echo "$q" | sed -n 's/.*from \([a-z_]*\).*/\1/p' | head -1)" "$($PSQL "$q")"
done

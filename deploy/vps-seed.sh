#!/usr/bin/env bash
# Seed demo trên VPS bằng API thật + chuyển LLM sang thật (OPS-VPS). Chạy trong /opt/edupilot, DB trống (hoặc seed dở — idempotent).
#   1) nới giới hạn theo IP (seed = MỘT IP gửi hàng nghìn yêu cầu), 2) seed; ngay khi admin có → `vps-llm.mjs` pha `seed`
#   (sinh chữ = fake + bản ghi demo vì seed trích quy chế bằng bản ghi; EMBEDDING = Gemini thật), 3) bỏ giới hạn nới, 4) pha `live` (sinh chữ = 9router).
set -euo pipefail
cd "$(dirname "$0")/.."
EP_HOST=$(grep "^EP_HOST=" .env.local | cut -d= -f2-)
export PATH=/opt/node/bin:$PATH
export API_URL="https://${EP_HOST:?}/api/v1" MAILPIT_URL=http://127.0.0.1:8025 SEED_ADMIN_CMD="deploy/dc exec -T gateway /gateway admin create"
printf 'RATE_LIMIT_IP_PER_MIN=5000\nAUTH_LOGIN_IP_PER_MIN=1000\nAUTH_REGISTER_IP_PER_HOUR=1000\nAUTH_TOKEN_IP_PER_MIN=1000\n' >> .env.local
deploy/dc up -d gateway --wait
node scripts/seed.mjs > /tmp/seed.log 2>&1 &
sp=$!
until grep -q '^\[3/19\]' /tmp/seed.log || ! kill -0 "$sp" 2>/dev/null; do sleep 2; done
PHASE=seed node deploy/vps-llm.mjs
wait "$sp" || { echo "seed lỗi: xem /tmp/seed.log" >&2; exit 1; }
sed -i '/^RATE_LIMIT_IP_PER_MIN=/d;/^AUTH_LOGIN_IP_PER_MIN=/d;/^AUTH_REGISTER_IP_PER_HOUR=/d;/^AUTH_TOKEN_IP_PER_MIN=/d' .env.local
deploy/dc up -d gateway --wait
PHASE=live node deploy/vps-llm.mjs

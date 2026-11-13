#!/usr/bin/env bash
# Sinh /opt/edupilot/.env.local cho VPS từ .env.example: mọi bí mật ngẫu nhiên MỚI, ghi 600. Chạy MỘT lần trên VPS; không in giá trị.
#   EP_HOST=v2202609419695521098.happysrv.de bash deploy/vps-env.sh
set -euo pipefail
cd "$(dirname "$0")/.."
: "${EP_HOST:?đặt EP_HOST (hostname công khai)}"
[ -e .env.local ] && { echo ".env.local đã có — không ghi đè." >&2; exit 1; }
rnd() { openssl rand -hex "$1"; }
PG=$(rnd 16); STATS=$(rnd 12); MK=$(rnd 12); MS=$(rnd 20)
umask 177
cp .env.example .env.local
set_kv() { # set_kv KEY VALUE — thay dòng KEY=… (hoặc thêm cuối)
  if grep -q "^$1=" .env.local; then sed -i "s|^$1=.*|$1=$2|" .env.local; else echo "$1=$2" >> .env.local; fi
}
set_kv POSTGRES_PASSWORD "$PG"
set_kv DATABASE_URL "postgres://edupilot:$PG@postgres:5432/edupilot?sslmode=disable"
set_kv PGBOUNCER_URL "postgres://edupilot:$PG@pgbouncer:6432/edupilot?sslmode=disable"
set_kv PGBOUNCER_STATS_PASSWORD "$STATS"
set_kv JWT_SECRET_KEY "$(rnd 32)"
set_kv APP_ENCRYPTION_KEY "$(openssl rand -base64 32)"
set_kv BLOB_ACCESS_KEY "ep$MK"
set_kv BLOB_SECRET_KEY "$MS"
set_kv JUDGE_TOKEN "$(rnd 24)"
set_kv SEED_DEFAULT_PASSWORD "Ep-$(rnd 6)-Seed#26"
set_kv EP_HOST "$EP_HOST"
set_kv APP_PUBLIC_URL "https://$EP_HOST"
set_kv CORS_ORIGINS "https://$EP_HOST"
set_kv NEXT_PUBLIC_API_URL ""
set_kv BLOB_PUBLIC_ENDPOINT "$EP_HOST"
set_kv BLOB_PUBLIC_USE_SSL "true"
set_kv JUDGE_EXTRA_ARGS ""   # amd64: sandbox có seccomp (không dùng -no-seccomp)
# Giới hạn theo IP nới CHỈ để seed (một IP gửi hàng nghìn yêu cầu); bỏ khối này sau seed rồi `up -d gateway` (deploy/README.md)
set_kv RATE_LIMIT_IP_PER_MIN 5000
set_kv AUTH_LOGIN_IP_PER_MIN 1000
set_kv AUTH_REGISTER_IP_PER_HOUR 1000
set_kv AUTH_TOKEN_IP_PER_MIN 1000
set_kv LLM_DEFAULT_RPM 20
echo ".env.local đã sinh ($(grep -c . .env.local) dòng, chmod $(stat -c %a .env.local))."

#!/usr/bin/env bash
set -Eeuo pipefail

log() {
  printf '[hf-entrypoint] %s\n' "$*"
}

missing=()
for name in OPENAI_API_KEY JWT_SECRET_KEY INTERNAL_CALLBACK_TOKEN; do
  if [[ -z "${!name:-}" ]]; then
    missing+=("$name")
  fi
done

if (( ${#missing[@]} > 0 )); then
  log "Thiếu Hugging Face Secret: ${missing[*]}"
  log "Vào Space > Settings > Variables and secrets, thêm các Secret trên rồi Restart Space."
  exit 1
fi

export DB_NAME="${DB_NAME:-agent_db}"
export DB_USERNAME="${DB_USERNAME:-user}"
export DB_PASSWORD="${DB_PASSWORD:-password}"
export REDIS_HOST="127.0.0.1"
export REDIS_PORT="6379"
export REDIS_USERNAME="default"
export REDIS_PASSWORD=""
export REDIS_DB="0"
export GRPC_PORT="${GRPC_PORT:-50051}"
export PYTHON_GRPC_URL="static://127.0.0.1:${GRPC_PORT}"
export JAVA_CALLBACK_URL="http://127.0.0.1:8080"
export APP_CORS_ALLOWED_ORIGINS="${APP_CORS_ALLOWED_ORIGINS:-*}"
export UPLOAD_DIR="${HF_DATA_DIR}/uploads"
export DATABASE_URL="postgresql://${DB_USERNAME}:${DB_PASSWORD}@127.0.0.1:5432/${DB_NAME}"
export DB_URL="jdbc:postgresql://127.0.0.1:5432/${DB_NAME}"
export REDIS_URL="redis://127.0.0.1:6379/0"
export ADMIN_TOKEN="${ADMIN_TOKEN:-$INTERNAL_CALLBACK_TOKEN}"

export PGDATA="${HF_DATA_DIR}/postgres"
export REDIS_DATA_DIR="${HF_DATA_DIR}/redis"

mkdir -p "$PGDATA" "$REDIS_DATA_DIR" "$UPLOAD_DIR" /app/data/tmp
chmod 700 "$PGDATA"

MIGRATION_MARKER="$PGDATA/.a20-migrations-complete"
RUN_FULL_MIGRATIONS=0

if [[ ! -f "$MIGRATION_MARKER" ]]; then
  if [[ -f "$PGDATA/PG_VERSION" ]]; then
    log "Phát hiện lần khởi tạo DB trước chưa hoàn tất; reset thư mục PostgreSQL để thử lại."
    find "$PGDATA" -mindepth 1 -maxdepth 1 -exec rm -rf -- {} +
  fi

  log "Khởi tạo PostgreSQL và pgvector..."
  initdb \
    --pgdata="$PGDATA" \
    --username="$DB_USERNAME" \
    --auth=trust \
    --encoding=UTF8 \
    --no-locale > /tmp/initdb.log

  RUN_FULL_MIGRATIONS=1
fi

cleanup_postgres() {
  pg_ctl --pgdata="$PGDATA" --mode=fast stop > /dev/null 2>&1 || true
}
trap cleanup_postgres EXIT

pg_ctl \
  --pgdata="$PGDATA" \
  --options="-h 127.0.0.1 -p 5432 -k /tmp" \
  --wait start > /tmp/postgres-bootstrap.log

if ! psql -h 127.0.0.1 -p 5432 -U "$DB_USERNAME" -d postgres \
    -tAc "SELECT 1 FROM pg_database WHERE datname = '${DB_NAME}'" | grep -q 1; then
  createdb -h 127.0.0.1 -p 5432 -U "$DB_USERNAME" "$DB_NAME"
fi

if (( RUN_FULL_MIGRATIONS == 1 )); then
  log "Chạy database migrations V1-V24..."
  psql \
    -h 127.0.0.1 \
    -p 5432 \
    -U "$DB_USERNAME" \
    -d "$DB_NAME" \
    -v ON_ERROR_STOP=1 \
    -f /app/db/00-init.sql
else
  # Persistent Spaces already have the legacy completion marker, so new
  # idempotent migrations must also run when the database volume exists.
  log "Áp dụng incremental migrations V23-V24..."
  psql \
    -h 127.0.0.1 \
    -p 5432 \
    -U "$DB_USERNAME" \
    -d "$DB_NAME" \
    -v ON_ERROR_STOP=1 \
    -f /app/db/hf-incremental.sql
fi

pg_ctl --pgdata="$PGDATA" --mode=fast --wait stop > /dev/null
trap - EXIT

if (( RUN_FULL_MIGRATIONS == 1 )); then
  touch "$MIGRATION_MARKER"
fi

log "Khởi động PostgreSQL, Redis, AI, Java, Next.js và Nginx trên cổng 7860..."
exec /usr/bin/supervisord -c /app/huggingface/supervisord.conf

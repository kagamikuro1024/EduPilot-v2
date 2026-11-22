export COMPOSE_PROJECT_NAME=ep-t1 EP_PORT_OFFSET=400 EP_PORT_MOCK_GRAPH=8490 EP_PORT_MOCK_IMAP=8491 BLOB_PUBLIC_ENDPOINT=localhost:9400
export POSTGRES_DB=edupilot_t1 DATABASE_URL='postgres://edupilot:edupilot-dev@postgres:5432/edupilot_t1?sslmode=disable' \
       PGBOUNCER_URL='postgres://edupilot:edupilot-dev@pgbouncer:6432/edupilot_t1?sslmode=disable'
pnpm dev                                    # SEED_ON_EMPTY_DB=false: chỉ cần admin
docker compose --env-file .env.local -f docker-compose.local.yml -p $COMPOSE_PROJECT_NAME exec -T gateway /gateway admin create --email admin@edupilot.local --name "Quản trị"   # mật khẩu: stdin hoặc ADMIN_PASSWORD = SEED_DEFAULT_PASSWORD
SEED_DEFAULT_PASSWORD=… node scripts/seed-t1.mjs            # ≈ 20 s; --reset để làm lại; --scale 0.5 để thu nhỏ (ghi quy mô thực)
```

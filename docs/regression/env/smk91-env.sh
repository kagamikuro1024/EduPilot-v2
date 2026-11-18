# Stack riêng sạch của qc2 cho smoke 9.1 / 9.5: project smk91 (KHÔNG phải edupilot / ep-r). source file này rồi dùng $SMK_C.
export COMPOSE_PROJECT_NAME=smk91
export EP_PORT_HTTP=1380 EP_PORT_HTTPS=1343 EP_PORT_FRONTEND=3800 EP_PORT_MAILPIT=8825 EP_PORT_SMTP=1825 EP_PORT_MINIO=9800 EP_PORT_MINIO_CONSOLE=9801 EP_PORT_POSTGRES=6233 EP_PORT_REDIS=7180 EP_PORT_MOCK_GRAPH=8890 EP_PORT_MOCK_IMAP=8891
export RATE_LIMIT_IP_PER_MIN=5000 AUTH_LOGIN_IP_PER_MIN=1000 AUTH_REGISTER_IP_PER_HOUR=1000 AUTH_TOKEN_IP_PER_MIN=1000
export BLOB_PUBLIC_ENDPOINT=localhost:1343 BLOB_PUBLIC_USE_SSL=true
export REPO=$HOME/Documents/TA_Agent_qc2
export SMK_C="docker compose --env-file $REPO/.env.local -f $REPO/docker-compose.local.yml -f $REPO/docker-compose.test.yml -f /tmp/smk91.override.yml -p smk91 --profile ingest"
export NEXT_PUBLIC_API_URL=https://localhost:1343 APP_PUBLIC_URL=https://localhost:1343 CORS_ORIGINS=https://localhost:1343,http://localhost:3800

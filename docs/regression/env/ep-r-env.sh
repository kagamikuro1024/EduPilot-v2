# source file này trước mọi lệnh docker compose của ep-r (project ep-r, cổng +700; KHÔNG BAO GIỜ dùng project edupilot)
export COMPOSE_PROJECT_NAME=ep-r
export EP_PORT_HTTP=780 EP_PORT_HTTPS=1143 EP_PORT_FRONTEND=3700 EP_PORT_MAILPIT=8725 EP_PORT_SMTP=1725 EP_PORT_MINIO=9700 EP_PORT_MINIO_CONSOLE=9701 EP_PORT_POSTGRES=6133 EP_PORT_REDIS=7080 EP_PORT_MOCK_GRAPH=8790 EP_PORT_MOCK_IMAP=8791
export REPO=$HOME/Documents/TA_Agent_qc2
export EPR_C="docker compose --env-file $REPO/.env.local -f $REPO/docker-compose.local.yml -f $REPO/docker-compose.test.yml -f $REPO/docs/regression/env/ep-r.seed.override.yml -p ep-r"

export APP_ENV=test
export DATABASE_URL='postgres://edupilot:qc@localhost:45432/edupilot?sslmode=disable'
export PGBOUNCER_URL="$DATABASE_URL"
export REDIS_URL='redis://localhost:46379/0'
export JWT_SECRET_KEY='edupilot-dev-jwt-secret-key-change-me-0001'
export APP_ENCRYPTION_KEY='ZWR1cGlsb3QtZGV2LWVuY3J5cHRpb24ta2V5LTMyYnk='
export LLM_PROVIDER=fake
export FAKE_LLM_VALID_KEY=good-key
export CORS_ORIGINS=http://localhost:3000
export BLOB_ENDPOINT=localhost:49150 BLOB_PUBLIC_ENDPOINT=localhost:49150 BLOB_BUCKET=edupilot BLOB_ACCESS_KEY=edupilot-dev BLOB_SECRET_KEY=edupilot-dev-secret BLOB_USE_SSL=false

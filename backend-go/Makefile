# Makefile của backend-go (SRS 9.4). Chạy: make -C backend-go <mục tiêu>
.PHONY: run run-worker build test lint lint-depguard-negative sqlc sqlc-check migrate tidy test-clean

# Docker của dev chạy qua colima; chỉ đặt khi môi trường chưa có (SRS 9.1).
COLIMA_DOCKER_HOST := unix://$(HOME)/.colima/default/docker.sock

run:
	go run ./cmd/gateway serve

run-worker:
	go run ./cmd/worker

build:
	go build ./...

test:
	DOCKER_HOST="$${DOCKER_HOST:-$(COLIMA_DOCKER_HOST)}" TESTCONTAINERS_RYUK_DISABLED=true go test -race -count=1 -tags testroutes ./...
	DOCKER_HOST="$${DOCKER_HOST:-$(COLIMA_DOCKER_HOST)}" TESTCONTAINERS_RYUK_DISABLED=true go test -race -count=1 -tags integration ./internal/mail/...
	-docker volume prune -f

lint:
	go vet ./...
	go vet -tags testroutes ./...
	go vet -tags integration ./...
	golangci-lint run
	golangci-lint run --build-tags testroutes
	golangci-lint run --build-tags integration

# Ca âm của depguard (US-PG-06 AC8): gói tạm import kin-openapi ngoài internal/contract phải bị chặn.
lint-depguard-negative:
	@mkdir -p internal/zz_depguard_probe
	@printf 'package zz_depguard_probe\n\nimport _ "github.com/getkin/kin-openapi/openapi3"\n' > internal/zz_depguard_probe/probe.go
	@out=$$(golangci-lint run ./internal/zz_depguard_probe/... 2>&1); rm -rf internal/zz_depguard_probe; \
	 if printf '%s' "$$out" | grep -q depguard; then \
	   echo "ok: depguard chặn kin-openapi ngoài internal/contract"; \
	 else \
	   printf '%s\n' "$$out"; echo "LỖI: depguard KHÔNG chặn kin-openapi"; exit 1; \
	 fi

sqlc:
	sqlc generate

sqlc-check:
	sqlc diff

migrate:
	go run ./cmd/gateway migrate up

tidy:
	go mod tidy

# Dọn container test dùng chung (internal/testutil giữ chúng sống giữa các lần chạy).
test-clean:
	-docker rm -fv edupilot-test-postgres edupilot-test-redis edupilot-test-minio edupilot-test-pgbouncer edupilot-test-mailpit
	-docker volume prune -f

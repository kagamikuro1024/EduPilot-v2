#!/bin/bash
# TC-P211-44 / TC-P212-27 (DEF-R4-09 9754c523): lint + vet + gói hẹp + integration + node --check + shellcheck
cd ~/Documents/TA_Agent_qc2; L=$HOME/Documents/TA_Agent_qc2/docs/regression/evidence-4/logs/lint-gate; mkdir -p $(dirname $L)
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
cd backend-go
{ echo "HEAD $(git rev-parse --short HEAD)"; echo "== golangci-lint run"; golangci-lint run; echo "rc=$?"
  echo "== go vet ./..."; go vet ./...; echo "rc=$?"
  echo "== go vet -tags testroutes"; go vet -tags testroutes ./...; echo "rc=$?"
  echo "== go test -race ./internal/today/..."; go test -race -count=1 ./internal/today/...; echo "rc=$?"
  echo "== go test -tags integration ./internal/integration/..."; go test -tags integration -count=1 ./internal/integration/...; echo "rc=$?"
  cd ..; echo "== node --check scripts/seed.mjs"; node --check scripts/seed.mjs; echo "rc=$?"
  echo "== shellcheck"; (which shellcheck && shellcheck scripts/*.sh) ; echo "rc=$?"; echo DONE; } > $L.log 2>&1

#!/bin/bash
# TC-P211-44: go vet && golangci-lint run && go test -race ./internal/today/... && go test -tags integration ./internal/integration/...
cd ~/Documents/TA_Agent_qc2/backend-go; export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
L=../docs/regression/evidence-4/logs/s4n-go44
echo "start $(date +%T)"; go vet ./... > $L-vet.log 2>&1; echo "vet rc=$?"
golangci-lint run > $L-lint.log 2>&1; echo "lint rc=$?"
go test -race -count=1 ./internal/today/... > $L-today.log 2>&1; echo "today rc=$?"
go test -tags integration -count=1 ./internal/integration/... > $L-integ.log 2>&1; echo "integ rc=$?"
echo "end $(date +%T)"

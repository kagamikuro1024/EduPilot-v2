#!/usr/bin/env bash
# Chạy toàn bộ test Go đúng quy trình CI (TL-3): bước 1 song song với EP_SKIP_TIMING=1 (bỏ các test *TimingEqualized: đo sub-ms không ổn định khi nhiều gói chạy cùng lúc),
# bước 2 chạy riêng các test đó, tuần tự, 3 lượt, không -race. `go test -race ./...` trần KHÔNG phải quy trình này và sẽ đỏ ngẫu nhiên ở TimingEqualized.
set -euo pipefail
cd "$(dirname "$0")/../backend-go"
export TESTCONTAINERS_RYUK_DISABLED=true
EP_SKIP_TIMING=1 go test -race -count=1 -tags testroutes ./...
go test -p 1 -count=3 -tags testroutes -run 'TimingEqualized$' ./internal/auth/ ./internal/course/

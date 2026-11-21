#!/bin/bash
# TC-N07-02 (SmkS4, 2026-11-21): stack smk91 nghỉ 5 phút (không tải) → docker stats --no-stream; so với bảng dev (benchmarks/reports/baseline.json idle, ±30 %).
sleep ${REST:-300}
docker stats --no-stream --format '{{.Name}} {{.MemUsage}} {{.CPUPerc}}' $(docker ps --format '{{.Names}}' | grep '^smk91')

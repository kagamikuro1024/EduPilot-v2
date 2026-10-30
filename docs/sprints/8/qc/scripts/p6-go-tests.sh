#!/usr/bin/env bash
# QC P6 — chạy đúng các test Go mà US ghi ở dòng "Kiểm:", và FAIL nếu bất kỳ test được nêu tên
# (a) không tồn tại ("no tests to run" nuốt im lặng), (b) bị SKIP, (c) không in "--- PASS: <tên>".
# Dùng:  bash p6-go-tests.sh <gói> "<TestA|TestB|…>" [-tags integration] [-race] [-bench …]
#   vd:  bash p6-go-tests.sh ./internal/grade "TestEngineExpected30|TestEngineCases" -race
set -u; pkg="$1"; names="$2"; shift 2
export DOCKER_HOST="${DOCKER_HOST:-unix://$HOME/.colima/default/docker.sock}" TESTCONTAINERS_RYUK_DISABLED=true
out=$(cd backend-go && go test -count=1 -v "$@" -run "$names" "$pkg" 2>&1); rc=$?
echo "$out" | grep -E '^(--- (PASS|FAIL|SKIP)|FAIL|ok|panic|DATA RACE)' | tail -n 60
bad=0
IFS='|' read -ra want <<<"$names"
for n in "${want[@]}"; do n="${n#^}"; n="${n%\$}"
  if ! echo "$out" | grep -q -- "--- PASS: $n\b"; then echo "THIẾU-PASS $n"; bad=1; fi
  if echo "$out" | grep -q -- "--- SKIP: $n\b"; then echo "SKIP $n (tính là FAIL)"; bad=1; fi
done
echo "$out" | grep -q 'DATA RACE' && { echo "DATA RACE"; bad=1; }
if [ $rc -ne 0 ] || [ $bad -ne 0 ]; then echo "RESULT: FAIL ($pkg)"; exit 1; fi
echo "RESULT: PASS ($pkg · ${#want[@]} test)"

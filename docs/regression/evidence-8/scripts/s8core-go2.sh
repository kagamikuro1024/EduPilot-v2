#!/bin/bash
# s8core: TC-P6-02-02/10 (-race, đếm ca con), 02-17 (vet + lint), 02-36 (bench), 03-24/01-28 bổ sung
cd $HOME/Documents/TA_Agent_qc2/backend-go; L=../docs/regression/evidence-8/logs/s8core-go2.log; : > $L
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
echo "## 02-02 race TestEngineExpected30" >> $L
go test -race -count=1 -v ./internal/grade -run TestEngineExpected30 > /tmp/s8core/race30.txt 2>&1; echo "rc=$?; PASS con: $(grep -c -- '--- PASS: TestEngineExpected30/' /tmp/s8core/race30.txt); SKIP: $(grep -c SKIP /tmp/s8core/race30.txt); DATA RACE: $(grep -c 'DATA RACE' /tmp/s8core/race30.txt)" >> $L
echo "## 02-10 race TestEngineCases" >> $L
go test -race -count=1 -v ./internal/grade -run TestEngineCases > /tmp/s8core/race18.txt 2>&1; echo "rc=$?; PASS con: $(grep -c -- '--- PASS: TestEngineCases/' /tmp/s8core/race18.txt); SKIP: $(grep -c SKIP /tmp/s8core/race18.txt)" >> $L
echo "## 02-36 bench" >> $L
go test -count=1 -run 'TestWhatIfNoWrite|TestMinFinalFor|TestPreviewUsesCompute' -bench Compute1000 -benchtime 20x -v ./internal/grade 2>&1 | grep -E 'Benchmark|^(--- |ok|FAIL)' >> $L
echo "## 02-17 vet+lint" >> $L
go vet ./internal/grade/... ./internal/grading/... >> $L 2>&1; echo "vet rc=$?" >> $L
timeout 250 golangci-lint run ./internal/grade/... >> $L 2>&1; echo "lint rc=$?" >> $L
echo DONE >> $L

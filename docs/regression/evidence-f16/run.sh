#!/bin/bash
# Chạy lại F có bản sửa ở HEAD (22/11). Ghi vào evidence-f16/*.log
cd ~/Documents/TA_Agent_qc2; L=docs/regression/evidence-f16
export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
echo "HEAD $(git log -1 --format=%h)" > $L/head.txt
( cd backend-go && go vet ./... ) > $L/govet.log 2>&1; echo "rc=$?" >> $L/govet.log
( cd backend-go && golangci-lint run ) > $L/lint.log 2>&1; echo "rc=$?" >> $L/lint.log
( cd backend-go && go test -race -count=1 ./internal/contract ) > $L/contract-notag.log 2>&1; echo "rc=$?" >> $L/contract-notag.log
( cd backend-go && go test -race -count=1 -tags testroutes ./internal/contract ) > $L/contract-tag.log 2>&1; echo "rc=$?" >> $L/contract-tag.log
echo ALLDONE > $L/done

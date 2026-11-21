#!/bin/bash
# TC-GATEP2-99 (dọn gate): chỉ ĐỌC và báo; không xoá gì. Chạy lúc mọi agent đã dừng (qc2 làm cuối). Phần xoá do người chạy: docker compose -p <dự án> down -v cho từng stack phụ, docker rm, DROP DATABASE qc2*/s4*.
cd ~/Documents/TA_Agent_qc2
echo "== container không thuộc ep-r (đang chạy):"; docker ps --format '{{.Names}}\t{{.Status}}' | grep -v '^ep-r-'
echo "== container không thuộc ep-r (đã dừng, còn lại):"; docker ps -a --filter status=exited --format '{{.Names}}' | grep -v '^ep-r-' | wc -l
echo "== DB tạm trong ep-r-postgres-1:"; docker exec ep-r-postgres-1 psql -U edupilot -d postgres -At -c "select datname from pg_database where datname ~ '^(qc2|qc3|s4|smk)'"
echo "== tiến trình QC (chrome/playwright/node seed/python3 QC):"; ps -eo pid,args | grep -E "playwright|chrom|seed\.mjs|server\.mjs|qc2|smk|s4n|s4x" | grep -v grep | cut -c1-150
echo "== khoá Playwright:"; ls -d /tmp/qc2-pw.lock 2>&1
echo "== cây git ngoài artefact QC (docs/regression, specs, benchmarks/reports):"; git status --short | grep -v ' docs/regression/\| benchmarks/reports/\|test-results' | head
echo "== tệp nhị phân > 5 MB trong docs/regression:"; find docs/regression -type f -size +5M | head

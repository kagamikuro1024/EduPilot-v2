#!/usr/bin/env bash
# Cổng nghiệm thu P8 (US-P8-03 AC16): tài liệu + thư viện + lịch. Chạy theo thứ tự, dừng ở lỗi đầu, in bảng PASS/FAIL, dòng cuối `GATE P8: PASS|PASS (có SKIP)|FAIL`.
#   bash scripts/gate-p8.sh        # cần: Docker (testcontainers) và một gateway ĐÃ SEED (API_URL, SEED_DEFAULT_PASSWORD) cho bước feed ICS
#   GATE_DOCLING=1 bash scripts/gate-p8.sh   # thêm check-docs-seed.mjs (docling thật: tài liệu READY, ANSWER_KEY, chia sẻ không nhúng lại, sự kiện, idempotent); không đặt → SKIP có lý do; GATE_DOCLING_TIMING=1 thêm bước đo giây / trang
# Chỉ bước CẦN docling thật được SKIP. TestAnswerKeyNeverRetrieved thiếu stack → FAIL, KHÔNG SKIP.
set -u
cd "$(dirname "$0")/.."
. scripts/gate-lib.sh

API="${API_URL:-https://localhost/api/v1}"
icsfeed() { # tạo token ICS cho sv.gioi bằng API thật rồi đọc dòng đầu của feed
  local tok url first
  tok=$(curl -sk -H 'Content-Type: application/json' -d "{\"email\":\"sv.gioi@edupilot.local\",\"password\":\"${SEED_DEFAULT_PASSWORD:-}\"}" "$API/auth/login" | sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p')
  [ -n "$tok" ] || { echo "FAIL feed ICS: không đăng nhập được sv.gioi ($API) — thiếu stack / seed"; return 1; }
  url=$(curl -sk -X POST -H "Authorization: Bearer $tok" -H 'Idempotency-Key: gate-p8-ics' "$API/me/calendar/ics-token" | sed -n 's/.*"url":"\([^"]*\)".*/\1/p')
  [ -n "$url" ] || { echo "FAIL feed ICS: không tạo được token"; return 1; }
  first=$(curl -sk "$url" | head -1 | tr -d '\r')
  echo "dòng đầu feed: $first"
  [ "$first" = "BEGIN:VCALENDAR" ]
}

run "go vet ./..." go_in go vet ./...
run "go test -race rag + library + calendar + document + ingest" go_in go test -race -count=1 ./internal/rag/... ./internal/library/... ./internal/calendar/... ./internal/document/... ./internal/ingest/...
strict_test TestAnswerKeyNeverRetrieved ./internal/rag
run "go test ./internal/contract/..." go_in go test -count=1 -tags testroutes ./internal/contract/...
run "feed ICS: curl feed.ics?token= | head -1 = BEGIN:VCALENDAR" icsfeed
run "Playwright documents + library + calendar" fe sh -c 'E2E_API_PORT=${E2E_API_PORT:-3412} pnpm build:gate && E2E_PORT=${E2E_PORT:-3410} E2E_API_PORT=${E2E_API_PORT:-3412} pnpm exec playwright test documents.spec.ts library.spec.ts calendar.spec.ts --workers=2 --retries=0'
run "ui-antipatterns.sh" bash scripts/ui-antipatterns.sh

if [ "${GATE_DOCLING:-}" = "1" ]; then
  run "check-docs-seed.mjs (docling thật)" node scripts/check-docs-seed.mjs
  # Đo giây / trang (US-P8-01 AC17: ≤ 1 PDF chữ, ≤ 4 bản scan) phụ thuộc máy; chỉ chạy khi GATE_DOCLING_TIMING=1.
  [ "${GATE_DOCLING_TIMING:-}" = "1" ] && run "check-docs-seed.mjs timing (≤ 1 s/trang; scan ≤ 4 s/trang)" node scripts/check-docs-seed.mjs timing
else
  skip "check-docs-seed.mjs" "cần docling thật: đặt GATE_DOCLING=1 khi 'docker compose --profile ingest up' đã chạy"
fi

finish P8

#!/usr/bin/env bash
# P-11-50: không test nào bị bỏ âm thầm. Đếm test cấp cao nhất có build tag `testroutes` (bản dựng của CI) và so với sàn trong
# backend-go/ci-test-count.txt: ít hơn sàn → đỏ (một tệp mất `//go:build`, đổi tag sai, gói bị loại). Thêm test thì NÂNG sàn (in số mới ở cuối).
set -euo pipefail
cd "$(dirname "$0")/../backend-go"
floor=$(tr -d '[:space:]' < ci-test-count.txt)
tagged=$(go test -tags testroutes -list '.*' ./... 2>/dev/null | grep -c '^Test' || true)
plain=$(go test -list '.*' ./... 2>/dev/null | grep -c '^Test' || true)
echo "test cấp cao nhất: có tag testroutes = $tagged, không tag = $plain, sàn = $floor"
[ "$tagged" -ge "$floor" ] || { echo "ĐỎ: $tagged < sàn $floor — có test bị bỏ khỏi bản dựng CI"; exit 1; }
[ "$tagged" -ge "$plain" ] || { echo "ĐỎ: có tag ($tagged) ít hơn không tag ($plain)"; exit 1; }
[ "$tagged" -eq "$floor" ] || echo "nâng sàn: echo $tagged > backend-go/ci-test-count.txt"

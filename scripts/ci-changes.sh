#!/usr/bin/env bash
# CI (TL-CI #1): in `fe=` / `go=` (true|false) cho job `changes`. Đọc EVENT, REF, BEFORE, SHA từ môi trường.
# Chạy ĐỦ (cả hai true) khi: pull_request, push vào main, nhánh mới (before = 000…0) hoặc before không còn trong lịch sử (force-push).
# Tự kiểm: bash scripts/ci-changes.sh --selftest
set -euo pipefail

FE='^(frontend/|pnpm-lock\.yaml$|package\.json$|pnpm-workspace\.yaml$|scripts/ui-antipatterns\.sh$|scripts/lint-selftest\.sh$|\.github/)'
# Go: ngoài backend-go/ còn các tệp mà test Go đọc bằng đường dẫn tương đối (.env.example, benchmarks/pii, nguồn frontend của exam/integrity_test) và image dựng từ deploy/.
GO='^(backend-go/|deploy/|\.env\.example$|benchmarks/pii/|frontend/src/features/exam/|frontend/src/shared/i18n/vi\.ts$|\.github/)'

emit() { echo "fe=$1"; echo "go=$2"; }

classify() { # stdin: danh sách tệp đổi
  local files fe=false go=false
  files=$(cat)
  grep -Eq "$FE" <<<"$files" && fe=true
  grep -Eq "$GO" <<<"$files" && go=true
  emit "$fe" "$go"
}

if [ "${1:-}" = "--selftest" ]; then
  t() { [ "$(printf '%s\n' "$2" | classify | tr '\n' ' ')" = "$1 " ] || { echo "SAI: '$2' → $(printf '%s\n' "$2" | classify | tr '\n' ' ') (mong $1)"; exit 1; }; }
  t "fe=false go=true" "backend-go/internal/x.go"
  t "fe=true go=false" "frontend/src/a.tsx"
  t "fe=true go=true" $'frontend/src/a.tsx\nbackend-go/x.go'
  t "fe=false go=false" "docs/x.md"
  t "fe=true go=true" ".github/workflows/ci.yml"
  t "fe=true go=true" "frontend/src/features/exam/a.tsx"
  t "fe=false go=true" ".env.example"
  t "fe=true go=false" "pnpm-lock.yaml"
  echo "ci-changes selftest: 8/8"
  exit 0
fi

if [ "$EVENT" = pull_request ] || [ "$REF" = refs/heads/main ] || [ "$BEFORE" = 0000000000000000000000000000000000000000 ] || ! git cat-file -e "$BEFORE^{commit}" 2>/dev/null; then
  emit true true
  exit 0
fi
git diff --name-only "$BEFORE" "$SHA" | classify

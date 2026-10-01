#!/usr/bin/env bash
# QC — chạy TC của US-P0-03. Dùng:
#   RUN_RED_GO=<ID run đỏ Go> RUN_RED_UI=<ID run đỏ phản mẫu UI> bash docs/sprints/1/qc/scripts/tc-US-P0-03.sh TC-01 ...
# Cần: git gh(đã login) jq go golangci-lint pnpm node. TC-09..13 cần hai biến RUN_RED_*.
set -u
cd "$(git rev-parse --show-toplevel)" || exit 2
BR=sprint/1-p0-prep; WF=.github/workflows/ci.yml; SBASE=1b72f54
RC=0; TC=""
pass(){ echo "PASS $TC: $*"; }
fail(){ echo "FAIL $TC: $*"; RC=1; }
eq(){ [ "$1" = "$2" ] && pass "$3" || fail "$3 — got [$1] want [$2]"; }
ok(){ local m=$1; shift; "$@" >/dev/null 2>&1 && pass "$m" || fail "$m"; }
need(){ [ -n "${!1:-}" ] || { fail "thiếu biến $1 (lấy từ handoff của dev)"; return 1; }; }
# run xanh mới nhất của HEAD nhánh sprint → $ID
head_run(){ git fetch -q origin "$BR" 2>/dev/null; SHA=$(git rev-parse origin/$BR 2>/dev/null)
  ID=$(gh run list --workflow ci.yml --branch $BR --limit 20 --json databaseId,headSha --jq "[.[]|select(.headSha==\"$SHA\")][0].databaseId"); }

# ---------- AC1 ----------
tc_01(){ head_run; [ -n "$ID" ] && [ "$ID" != null ] && pass "có run cho HEAD origin/$BR ($SHA)" || { fail "không có run nào trên đúng HEAD $SHA (đã push chưa? có commit mới hơn run?)"; return; }
  eq "$(gh run view $ID --json conclusion --jq .conclusion)" success "run conclusion"
  eq "$(gh run view $ID --json headSha --jq .headSha)" "$SHA" "headSha = origin/$BR"
  eq "$(gh run view $ID --json event --jq .event)" push "event = push"; }
tc_02(){ head_run; local o; o=$(gh run view $ID --json jobs --jq '.jobs[]|"\(.name) \(.conclusion)"'|sort); echo "$o"
  eq "$(echo "$o" | wc -l | xargs)" 2 "đúng 2 job"; eq "$(echo "$o" | tr '\n' ',')" "Frontend success,Go success," "Go + Frontend đều success"; }
tc_03(){ head_run; local n; n=$(gh run view $ID --json jobs --jq '.jobs[].steps[].name' | grep -ciE 'vet|golangci|race|lint|build|antipattern'); [ "$n" -ge 6 ] && pass "≥ 6 bước khớp ($n)" || fail "chỉ $n bước khớp"
  local go fe; go=$(gh run view $ID --json jobs --jq '.jobs[]|select(.name=="Go")|.steps[]|select(.conclusion!="skipped")|.name'); fe=$(gh run view $ID --json jobs --jq '.jobs[]|select(.name=="Frontend")|.steps[]|select(.conclusion!="skipped")|.name')
  for k in 'go vet' 'golangci-lint' 'go test -race'; do echo "$go" | grep -qi "$k" && pass "job Go có bước '$k'" || fail "job Go thiếu bước '$k'"; done
  for k in 'install' 'lint' 'build' 'antipattern|ui antipatterns'; do echo "$fe" | grep -qiE "$k" && pass "job Frontend có bước '$k'" || fail "job Frontend thiếu bước '$k'"; done; }
tc_04(){ [ -f $WF ] || { fail "không có $WF"; return; }
  grep -qE '^on:|^"on":' $WF && pass "có khối on" || fail "thiếu on"
  grep -qE '^\s+push:' $WF && pass "trigger push" || fail "thiếu push"; grep -qE '^\s+pull_request:' $WF && pass "trigger pull_request" || fail "thiếu pull_request"
  eq "$(awk '/^\s+push:/{f=1;next} f&&/^\s+[a-z_]+:/&&!/^\s+(branches|tags|paths)/{f=0} f' $WF | grep -cE 'branches|paths')" 0 "push không lọc nhánh/đường dẫn (mọi nhánh)"
  grep -q 'working-directory: backend-go' $WF && pass "Go: working-directory backend-go" || fail "Go: thiếu working-directory"
  grep -q 'go-version-file: backend-go/go.mod' $WF && pass "go-version-file" || fail "go-version-file"
  grep -nE 'golangci' $WF | grep -iE 'latest' && fail "golangci-lint dùng latest" || pass "golangci-lint không latest"
  grep -qE 'version: *v?[0-9]+\.[0-9]+(\.[0-9]+)?' $WF && pass "có bản ghim cụ thể (golangci-lint/pnpm/…)" || fail "không thấy version ghim"
  grep -qE 'node-version: *["'\'']?24' $WF && pass "Node 24" || fail "không Node 24"
  grep -qE 'cache: *["'\'']?pnpm' $WF && pass "cache pnpm" || fail "không cache pnpm"
  grep -qE '"packageManager": *"pnpm@[0-9]+\.[0-9]+\.[0-9]+' package.json && pass "package.json có packageManager ghim" || fail "packageManager không ghim"
  grep -q -- '--frozen-lockfile' $WF && pass "frozen-lockfile" || fail "thiếu --frozen-lockfile"
  eq "$(grep -cE '^\s+needs:' $WF)" 0 "hai job không needs (FR-4)"
  eq "$(grep -cE 'uses: *[^ ]+@(main|master|latest)\b' $WF)" 0 "action không trỏ @main/@master"
  eq "$(grep -cE 'pull_request_target' $WF)" 0 "không pull_request_target"
  local order; order=$(grep -nE 'go vet|golangci-lint run|go test -race|frozen-lockfile|frontend lint|frontend build|ui-antipatterns' $WF | cut -d: -f1 | xargs); echo "thứ tự dòng lệnh: $order"
  local n1 n2 n3; n1=$(grep -nE '^\s+- name: go vet' $WF | cut -d: -f1); n2=$(grep -nE '^\s+- name: golangci-lint' $WF | cut -d: -f1); n3=$(grep -nE '^\s+- name: go test -race' $WF | cut -d: -f1)
  [ -n "$n1" ] && [ -n "$n2" ] && [ -n "$n3" ] && [ "$n1" -lt "$n2" ] && [ "$n2" -lt "$n3" ] && pass "Go: vet < golangci-lint < test (theo tên bước)" || fail "thứ tự/tên bước Go sai ($n1,$n2,$n3)"
  grep -qE 'go test -race \./\.\.\.' $WF && grep -qE 'go vet \./\.\.\.' $WF && pass "đúng lệnh đủ ./..." || fail "lệnh Go không dùng ./..."
  grep -qE 'pnpm -C frontend lint' $WF && grep -qE 'pnpm -C frontend build' $WF && grep -qE 'bash scripts/ui-antipatterns.sh' $WF && pass "đúng 3 lệnh frontend" || fail "thiếu lệnh frontend"
  eq "$(grep -cE '^\s+name: *(Go|Frontend)\s*$' $WF)" 2 "tên job Go, Frontend"; }

# ---------- AC2 ----------
tc_05(){ eq "$(grep -nE 'legacy|secrets\.|_API_KEY' $WF | wc -l | xargs)" 0 "không legacy/secrets./_API_KEY"
  eq "$(grep -ciE 'OPENAI|ANTHROPIC|GEMINI|LLM_|GOOGLE_API' $WF)" 0 "không biến khoá LLM"
  eq "$(grep -cE 'docker (push|login)|ghcr\.io|aws |deploy' $WF)" 0 "không deploy/push image (ngoài phạm vi)"; }
tc_06(){ head_run; gh run view $ID --log >/tmp/qc-ci.log 2>&1; eq "$?" 0 "tải log run"
  eq "$(grep -c 'legacy/' /tmp/qc-ci.log)" 0 "log không nhắc legacy/ (dòng khớp cần xem tay nếu >0)"
  eq "$(grep -ciE 'api_key|apikey' /tmp/qc-ci.log)" 0 "log không có API key"
  grep -q 'golangci-lint run' /tmp/qc-ci.log && pass "log: action chạy thật 'golangci-lint run'" || fail "log không có 'golangci-lint run'"
  grep -qE 'go test -race' /tmp/qc-ci.log && pass "log: có go test -race" || fail "log không có go test -race"
  grep -A3 'GITHUB_TOKEN Permissions' /tmp/qc-ci.log | grep -E 'Contents: read|Metadata: read' | sort -u; sed -n '/GITHUB_TOKEN Permissions/,/endgroup/p' /tmp/qc-ci.log | grep -E ': write' && fail "token có quyền write" || pass "token không có quyền write"; }

# ---------- AC3 ----------
tc_07(){ need RUN_RED_GO || return; local j; j=$(gh run view $RUN_RED_GO --json conclusion,jobs,headBranch,event --jq '[.conclusion,.headBranch,.event,(.jobs[]|"\(.name)=\(.conclusion)")]|join(" ")'); echo "$j"
  echo "$j" | grep -q '^failure ci/red-check push' && pass "run failure, nhánh ci/red-check, push" || fail "run/nhánh/event sai"
  echo "$j" | grep -q 'Go=failure' && pass "job Go failure" || fail "job Go không failure"
  echo "$j" | grep -q 'Frontend=success' && pass "job Frontend success" || fail "job Frontend không success"; }
tc_08(){ need RUN_RED_GO || return
  eq "$(gh run view $RUN_RED_GO --json jobs --jq '[.jobs[]|select(.name=="Go")|.steps[]|select(.conclusion=="failure")|.name]|join(",")' | tr 'A-Z' 'a-z' | grep -c 'race\|test')" 1 "bước Go đỏ là go test -race (không phải vet/lint)"
  eq "$(gh run view $RUN_RED_GO --json jobs --jq '[.jobs[]|select(.name=="Frontend")|.steps[]|select(.conclusion=="skipped" or .conclusion=="cancelled" or .conclusion=="failure")]|length')" 0 "Frontend chạy hết, không bước nào bị bỏ/hủy"
  gh run view $RUN_RED_GO --log-failed 2>/dev/null | grep -qE -- '--- FAIL|FAIL' && pass "log lỗi có FAIL của test Go" || fail "log lỗi không có FAIL của test"
  local sha; sha=$(gh run view $RUN_RED_GO --json headSha --jq .headSha); gh api "repos/{owner}/{repo}/commits/$sha" --jq '.files[].filename' | tee /tmp/qc-red1.txt
  grep -q '_test\.go$' /tmp/qc-red1.txt && pass "commit đỏ có file *_test.go" || fail "commit đỏ không thêm test Go"
  [ "$(grep -vcE '_test\.go$' /tmp/qc-red1.txt)" = 0 ] && pass "commit đỏ chỉ đụng test Go" || fail "commit đỏ đụng file khác (xem danh sách)"; }
tc_09(){ ok "nhánh ci/red-check đã chạy CI cho push thứ hai khác sha" test "$(gh run list --workflow ci.yml --branch ci/red-check --limit 10 --json headSha --jq '[.[].headSha]|unique|length')" -ge 2; }

# ---------- AC4 ----------
tc_10(){ need RUN_RED_UI || return; local j; j=$(gh run view $RUN_RED_UI --json conclusion,headBranch,jobs --jq '[.conclusion,.headBranch,(.jobs[]|"\(.name)=\(.conclusion)")]|join(" ")'); echo "$j"
  echo "$j" | grep -q '^failure ci/red-check' && pass "run failure trên ci/red-check" || fail "run/nhánh sai"
  echo "$j" | grep -q 'Frontend=failure' && pass "Frontend failure" || fail "Frontend không failure"
  echo "$j" | grep -q 'Go=success' && pass "Go success (test đỏ đã bỏ)" || fail "Go không success"
  local f; f=$(gh run view $RUN_RED_UI --json jobs --jq '[.jobs[]|select(.name=="Frontend")|.steps[]|select(.conclusion=="failure")|.name]|join("|")'); echo "bước đỏ: $f"
  echo "$f" | grep -qiE 'antipattern' && pass "đỏ ở bước phản mẫu UI" || fail "đỏ nhầm bước ($f)"
  eq "$(gh run view $RUN_RED_UI --json jobs --jq '[.jobs[]|select(.name=="Frontend")|.steps[]|select(.name|test("lint|build|install";"i"))|.conclusion]|unique|join(",")')" success "lint/build/install của Frontend vẫn success trước bước đỏ"; }
tc_11(){ need RUN_RED_UI || return; local sha; sha=$(gh run view $RUN_RED_UI --json headSha --jq .headSha); gh api "repos/{owner}/{repo}/commits/$sha" --jq '.files[]|"\(.status) \(.filename)"' | tee /tmp/qc-red2.txt
  grep -qE 'modified frontend/src/app/page\.tsx' /tmp/qc-red2.txt && pass "commit 2 sửa page.tsx" || fail "commit 2 không sửa page.tsx"
  grep -qE '^removed .*_test\.go|^modified .*_test\.go' /tmp/qc-red2.txt && pass "commit 2 bỏ/sửa test đỏ" || fail "commit 2 không bỏ test đỏ"
  gh api "repos/{owner}/{repo}/commits/$sha" --jq '.files[]|select(.filename=="frontend/src/app/page.tsx")|.patch' | grep '^+' | grep -qE '#[0-9a-fA-F]{3,8}\b|rgb\(' && pass "page.tsx thêm màu viết cứng" || fail "không thấy màu viết cứng ở patch"; }
tc_12(){ eq "$(git ls-remote --heads origin ci/red-check | wc -l | xargs)" 0 "nhánh ci/red-check đã xoá"
  eq "$(grep -c continue-on-error $WF)" 0 "không continue-on-error"; }

# ---------- AC5 ----------
tc_13(){ grep -nA1 '^permissions:' $WF; eq "$(grep -A1 '^permissions:' $WF | grep -c 'contents: read')" 1 "permissions: contents: read ở mức workflow"
  eq "$(grep -cE ':\s*write\b' $WF)" 0 "không quyền write ở đâu"
  eq "$(grep -cE '^\s{4,}permissions:' $WF)" 0 "job không ghi đè permissions"; }

# ---------- Kiểm chéo ----------
tc_14(){ eq "$(git diff --name-only $SBASE HEAD -- .github | xargs)" ".github/workflows/ci.yml" "story chỉ thêm ci.yml dưới .github (không sửa keep-huggingface)"
  ok "keep-huggingface-space-awake.yml không đổi" git diff --quiet $SBASE HEAD -- .github/workflows/keep-huggingface-space-awake.yml
  eq "$(ls .github/workflows | wc -l | xargs)" 2 "đúng 2 workflow (cũ + ci.yml)"
  eq "$(git diff $SBASE HEAD -- . ':!legacy' ':!docs' | grep '^+' | grep -cE 'sk-[A-Za-z0-9]{10,}|AKIA[0-9A-Z]{12}|ghp_[A-Za-z0-9]{20}')" 0 "không secret trong dòng thêm"; }
tc_15(){ (cd backend-go && go vet ./... && golangci-lint run && go test -race -count=1 ./...); eq "$?" 0 "chạy local đúng ba lệnh Go của CI"
  pnpm install --frozen-lockfile && pnpm -C frontend lint && pnpm -C frontend build && bash scripts/ui-antipatterns.sh; eq "$?" 0 "chạy local đúng bốn lệnh Frontend của CI (lockfile khớp)"; }
tc_16(){ git fetch -q origin; eq "$(git rev-list --count origin/$BR..HEAD)" 0 "không còn commit chưa push (AC1 cần HEAD đã push)"
  echo "info: nhánh ci/red-check chỉ xoá sau khi QC chấm AC3/AC4 (spec v2) — kiểm lại bằng TC-12 sau khi dev xoá"; }

[ $# -eq 0 ] && { echo "dùng: $0 TC-01 [TC-02 ...]"; exit 2; }
for t in "$@"; do TC=$t; f="tc_${t#TC-}"; if declare -F "$f" >/dev/null; then "$f"; else echo "FAIL $t: không có TC này"; RC=1; fi; done
exit $RC

#!/usr/bin/env bash
# QC — chạy TC của US-P0-02. Dùng:  bash docs/sprints/1/qc/scripts/tc-US-P0-02.sh TC-01 TC-02 ...
#   nhóm: static (TC-01..08 19 20..24 25..27 31 35..38)  docker (TC-09..18 28..30 32 33 34 — cần Docker)
# Cần: git jq curl go node pnpm docker python3. Chạy ở nhánh có handoff, cây làm việc sạch.
set -u
cd "$(git rev-parse --show-toplevel)" || exit 2
BASE=5cfb4af; SBASE=1b72f54
C="docker compose --env-file .env.local -f docker-compose.local.yml -p edupilot"
RC=0; TC=""
pass(){ echo "PASS $TC: $*"; }
fail(){ echo "FAIL $TC: $*"; RC=1; }
eq(){ [ "$1" = "$2" ] && pass "$3" || fail "$3 — got [$1] want [$2]"; }
ok(){ local m=$1; shift; "$@" >/dev/null 2>&1 && pass "$m" || fail "$m"; }
GWLOG=/tmp/qc-gw.log
# gw_run <env...>: chạy /tmp/gw với env sạch, tối đa 5s, in mã thoát vào $GWRC
gw_run(){ (cd "$(mktemp -d)" && env -i PATH="$PATH" "$@" /tmp/gw >"$GWLOG" 2>&1 & p=$!
  for _ in $(seq 50); do kill -0 $p 2>/dev/null || break; sleep 0.1; done
  if kill -0 $p 2>/dev/null; then kill -9 $p; echo 124 >/tmp/qc-gwrc; else wait $p; echo $? >/tmp/qc-gwrc; fi); GWRC=$(cat /tmp/qc-gwrc); }
build_gw(){ (cd backend-go && go build -o /tmp/gw ./cmd/gateway); }

# ---------- AC1 ----------
tc_01(){ eq "$(git ls-files | awk -F/ '{print $1}' | LC_ALL=C sort -u | xargs)" ".claude .dockerignore .env.example .gitattributes .github .gitignore AGENTS.md CLAUDE.md backend-go docker-compose.local.yml docs frontend legacy package.json pnpm-lock.yaml pnpm-workspace.yaml scripts seed" "gốc repo"; }
tc_02(){ local miss; miss=$(git ls-tree -r --name-only $BASE \
  | grep -vE '^(docs/|\.claude/|\.github/|AGENTS\.md$|CLAUDE\.md$|\.gitattributes$|\.gitignore$|\.dockerignore$|package\.json$|pnpm-workspace\.yaml$|scripts/(dev\.mjs|team-up\.sh|ui-antipatterns\.sh)$|frontend/src/shared/styles/tokens\.css$|frontend/public/brand/|data/(Mordern_Network_Security_Threats|QMB12ch6b|Quyche)\.pdf$|data/tmp/)' \
  | while read -r f; do git ls-files --error-unmatch "legacy/$f" >/dev/null 2>&1 || echo "THIẾU legacy/$f"; done)
  [ -z "$miss" ] && pass "mọi file Project III có ở legacy/" || { fail "thiếu file"; echo "$miss" | head -10; }; }
tc_03(){ eq "$(git ls-files | grep -c 492218d5)" 0 "không còn 492218d5 (kể cả legacy/)"; }
tc_04(){ eq "$(ls seed/documents 2>/dev/null | xargs)" "Mordern_Network_Security_Threats.pdf QMB12ch6b.pdf Quyche.pdf" "seed/documents"
  eq "$(git ls-files data | wc -l | xargs)" 0 "không còn data/ ở gốc"
  for f in Mordern_Network_Security_Threats QMB12ch6b Quyche; do eq "$(git rev-parse $BASE:data/$f.pdf)" "$(git rev-parse HEAD:seed/documents/$f.pdf)" "PDF $f nguyên vẹn (cùng blob)"; done; }
tc_05(){ eq "$(git ls-files frontend/src/shared/styles/tokens.css frontend/public/brand | wc -l | xargs)" 4 "tokens.css + 3 brand"
  ok "tokens.css + brand không đổi so với base" git diff --quiet $BASE HEAD -- frontend/src/shared/styles/tokens.css frontend/public/brand; }
tc_06(){ local n; n=$(git log --follow --oneline -- legacy/backend-java/aitrogiang/build.gradle | wc -l | xargs); [ "$n" -ge 2 ] && pass "lịch sử còn ($n)" || fail "chỉ $n commit"
  n=$(git log --follow --oneline -- legacy/src/main.py 2>/dev/null | wc -l | xargs); echo "info: legacy/src/main.py -> $n commit (nếu file không tồn tại, bỏ qua)"; }
tc_07(){ local bad; bad=$(git -c diff.renameLimit=5000 diff -M -C --find-copies-harder --name-status $BASE HEAD | awk '$NF ~ /^legacy\// && $1 !~ /^[RC]100/')
  [ -z "$bad" ] && pass "mọi file legacy/ là rename/copy 100% (không sửa nội dung)" || { fail "legacy/ có file đổi nội dung hoặc mới"; echo "$bad" | head -10; }; }
tc_08(){ local i=0 last=-1 firstnew=-1 mixed="" c st; for c in $(git rev-list --reverse $SBASE..HEAD); do i=$((i+1))
    st=$(git diff-tree -r -M --name-status --no-commit-id $c)
    if [ -n "$(echo "$st" | awk -F'\t' '$1 ~ /^R/ && $3 ~ /^legacy\//')" ]; then last=$i
      [ -n "$(echo "$st" | awk -F'\t' '$1 !~ /^R/ && $2 !~ /^data\/tmp\/492218d5/')" ] && mixed="$mixed $c"; fi
    if [ $firstnew -lt 0 ] && [ -n "$(git diff-tree -r --no-commit-id --name-only --diff-filter=A $c | grep -E '^(backend-go/|frontend/package\.json|frontend/src/app/)')" ]; then firstnew=$i; fi
  done
  [ $last -ge 0 ] && pass "có commit dời (#$last)" || fail "không thấy commit dời nào"
  [ -z "$mixed" ] && pass "commit dời không lẫn thay đổi khác (ngoài git rm 492218d5)" || fail "commit dời lẫn mã mới:$mixed"
  { [ $firstnew -lt 0 ] || [ $firstnew -gt $last ]; } && pass "mã mới đến sau commit dời" || fail "mã mới (#$firstnew) trước/cùng commit dời (#$last)"; }

# ---------- AC2 ----------
tc_09(){ local J; J=$($C config --format json) || { fail "compose config lỗi (cần .env.local)"; return; }
  eq "$(echo "$J" | jq -r '.services|keys|join(" ")')" "frontend gateway mailpit minio postgres redis" "đúng 6 service"
  eq "$(echo "$J" | jq -r '[.services|to_entries[]|select(.value.healthcheck==null)|.key]|length')" 0 "mọi service có healthcheck"
  eq "$(echo "$J" | jq -r '[.services|to_entries[]|select(.value.image!=null)|.value.image|select((contains(":")|not) or endswith(":latest"))]|length')" 0 "image ghim tag (không :latest/không tag)"
  eq "$(echo "$J" | jq -r '.services.postgres.image')" "pgvector/pgvector:pg18" "postgres image"
  eq "$(echo "$J" | jq -r '.services.redis.image')" "redis:8" "redis image"
  echo "$J" | jq -r '.services.mailpit.image' | grep -q '^axllent/mailpit:' && pass "mailpit image" || fail "mailpit image"
  eq "$(echo "$J" | jq -r '[.services|to_entries[]|.key as $k|.value.ports[]?|"\($k) \(.published)"]|sort|join(",")')" "frontend 3000,gateway 8080,mailpit 1025,mailpit 8025,minio 9000,minio 9001,postgres 5433,redis 6380" "cổng host"
  eq "$(echo "$J" | jq -r '[.services.postgres.ports[0].target,.services.redis.ports[0].target]|join(",")')" "5432,6379" "cổng trong của postgres,redis"
  eq "$(echo "$J" | jq -r '.services.gateway.depends_on|to_entries|map("\(.key):\(.value.condition)")|sort|join(",")')" "postgres:service_healthy,redis:service_healthy" "gateway phụ thuộc postgres, redis healthy"
  eq "$(echo "$J" | jq -r '.volumes|keys|length>=3')" true "≥ 3 volume có tên"
  eq "$($C config --services | grep -ciE 'python|ai$')" 0 "không service Python/AI"; }
tc_10(){ pnpm dev; eq "$?" 0 "pnpm dev exit"; }
tc_11(){ local o; o=$(pnpm -s dev:status --format '{{.Service}} {{.Health}}' | sort)
  eq "$(echo "$o" | wc -l | xargs)" 6 "6 dòng"; eq "$(echo "$o" | grep -c ' healthy$')" 6 "6 healthy"; echo "$o"; }
tc_12(){ local h; h=$(curl -si localhost:8080/healthz); echo "$h" | head -1 | grep -q ' 200' && pass "200" || fail "status"
  echo "$h" | grep -qi '^content-type: application/json' && pass "content-type" || fail "content-type"
  eq "$(curl -fsS localhost:8080/healthz)" '{"status":"ok"}' "body"; }
tc_13(){ eq "$(curl -fsS localhost:8025/api/v1/info | jq -r 'type')" object "mailpit /api/v1/info là JSON"
  ok "minio live (9000)" curl -fsS localhost:9000/minio/health/live
  eq "$(curl -s -o /dev/null -w '%{http_code}' localhost:9001/)" 200 "minio console (9001)"
  local pg rd; pg=$($C ps -q postgres); rd=$($C ps -q redis)
  eq "$(docker exec $pg psql -U "$(docker exec $pg printenv POSTGRES_USER)" -tAc 'show server_version' | cut -d. -f1)" 18 "postgres 18"
  eq "$(docker exec $pg psql -U "$(docker exec $pg printenv POSTGRES_USER)" -tAc "select count(*) from pg_available_extensions where name='vector'")" 1 "pgvector có sẵn"
  eq "$(docker exec $rd redis-cli ping)" PONG "redis ping"
  eq "$(docker exec $rd redis-cli info server | grep -c '^redis_version:8')" 1 "redis 8"
  ok "SMTP mailpit 1025 mở" python3 -c "import socket;socket.create_connection(('127.0.0.1',1025),3)"; }
tc_14(){ local H; H=$(curl -s localhost:3000)
  echo "$H" | grep -q 'lang="vi"' && pass "lang=vi" || fail "lang=vi"
  echo "$H" | grep -q 'logo-edupilot' && pass "HTML tham chiếu logo-edupilot" || fail "không thấy logo"
  eq "$(curl -s -o /dev/null -w '%{http_code}' localhost:3000/brand/logo-edupilot.svg)" 200 "logo phục vụ được"
  echo "$H" | LC_ALL=C grep -q '[^ -~]' && pass "có chữ có dấu (tiếng Việt)" || fail "không thấy ký tự non-ASCII"
  local css; css=$(echo "$H" | grep -oE '/_next/static/[^"]+\.css' | sort -u | while read -r u; do curl -s "localhost:3000$u"; done)
  echo "$css" | grep -q -- '--ep-' && pass "CSS có token --ep-*" || fail "CSS thiếu token --ep-*"
  echo "$css" | grep -qi 'Be Vietnam Pro' && pass "CSS có Be Vietnam Pro" || fail "CSS thiếu Be Vietnam Pro"
  echo "MẮT (chủ dự án): mở http://localhost:3000 — trang trống tiếng Việt, logo, Be Vietnam Pro, màu từ token"; }
tc_15(){ local g img; g=$($C ps -q gateway); local u; u=$(docker inspect -f '{{.Config.User}}' $g)
  case "$u" in ""|0|root|0:*|root:*) fail "gateway chạy root (User=[$u])";; *) pass "gateway non-root (User=$u)";; esac
  eq "$($C config --format json | jq -r '.services.gateway.healthcheck.test|join(" ")|test("curl|wget")')" false "healthcheck không dùng curl/wget"
  eq "$(docker exec $g which curl >/dev/null 2>&1; echo $?)" 1 "image gateway không có curl (exit≠0)"; }
tc_16(){ [ -f .env.local ] && cp .env.local /tmp/qc-env.local.bak; rm -f .env.local; pnpm dev >/dev/null 2>&1; local rc=$?
  eq "$rc" 0 "pnpm dev sau khi xoá .env.local"; ok ".env.local được tạo từ .env.example" diff -q .env.local .env.example
  ok ".env.local bị gitignore" git check-ignore -q .env.local
  [ -f /tmp/qc-env.local.bak ] && cp /tmp/qc-env.local.bak .env.local; }
tc_17(){ pnpm dev:down >/dev/null 2>&1; python3 -m http.server 8080 >/dev/null 2>&1 & local p=$!; sleep 1
  timeout_run(){ ( "$@" & q=$!; for _ in $(seq 600); do kill -0 $q 2>/dev/null || { wait $q; exit $?; }; sleep 1; done; kill -9 $q; exit 124 ); }
  timeout_run pnpm dev >/tmp/qc-dev17.log 2>&1; local rc=$?; kill $p
  [ $rc -ne 0 ] && [ $rc -ne 124 ] && pass "pnpm dev thoát mã $rc khi cổng 8080 bị chiếm" || fail "exit=$rc (0 = nuốt lỗi, 124 = treo)"; tail -5 /tmp/qc-dev17.log; pnpm dev:down >/dev/null 2>&1; }
tc_18(){ local d; d=$(mktemp -d); ln -s "$(command -v node)" $d/node; ln -s "$(command -v pnpm)" $d/pnpm; ln -s "$(command -v git)" $d/git
  PATH="$d" pnpm dev >/tmp/qc-dev18.log 2>&1; local rc=$?
  [ $rc -ne 0 ] && pass "thoát mã $rc khi thiếu Docker" || fail "exit 0 khi không có Docker"
  grep -qi docker /tmp/qc-dev18.log && pass "thông báo nhắc Docker" || fail "thông báo không nhắc Docker"
  [ "$(grep -cE '^\s+at .*\.(mjs|js):[0-9]+' /tmp/qc-dev18.log)" = 0 ] && pass "không stack trace" || fail "có stack trace"; }
tc_19(){ grep -qE '^go 1\.27' backend-go/go.mod && pass "go.mod go 1.27" || fail "go.mod không phải go 1.27"
  grep -qE '"next": *"\^?16\.' frontend/package.json && pass "next 16" || fail "next"
  grep -qE '"react": *"\^?19\.' frontend/package.json && pass "react 19" || fail "react"
  grep -qE '"packageManager": *"pnpm@' package.json && pass "packageManager pnpm" || fail "packageManager"
  grep -qE 'node:24' frontend/Dockerfile && pass "frontend Dockerfile node:24" || fail "frontend Dockerfile không node:24"
  eq "$(grep -cE 'tailwindcss|zustand|@tanstack/react-query' frontend/package.json)" 0 "không có dep ngoài phạm vi (tailwind/zustand/tanstack)"; }

# ---------- AC3 ----------
tc_20(){ (cd backend-go && go vet ./... && go test -race -count=1 -v ./... >/tmp/qc-gotest.log 2>&1); eq "$?" 0 "go vet + go test -race"
  tail -3 /tmp/qc-gotest.log
  local n; n=$(grep -cE '^\s*--- PASS' /tmp/qc-gotest.log); [ "$n" -ge 4 ] && pass "≥ 4 test/subtest PASS ($n): 3 cấu hình + healthz" || fail "chỉ $n PASS (cần ≥ 4 theo FR-7)"
  eq "$(grep -cE '^\s*--- (FAIL|SKIP)' /tmp/qc-gotest.log)" 0 "không FAIL/SKIP"; }
tc_21(){ eq "$(grep -c '^openapi: 3.1' backend-go/api/openapi.yaml)" 1 "openapi 3.1"; eq "$(grep -c '/healthz:' backend-go/api/openapi.yaml)" 1 "mô tả /healthz"
  pnpm dlx @redocly/cli lint backend-go/api/openapi.yaml; eq "$?" 0 "redocly lint 0 lỗi"
  pnpm dlx @redocly/cli bundle backend-go/api/openapi.yaml --ext json -o /tmp/qc-oa.json >/dev/null 2>&1
  eq "$(jq -r '.paths["/healthz"].get.responses["200"].content["application/json"].schema|(.properties.status.type // .["$ref"])' /tmp/qc-oa.json | head -1)" string "response 200 có schema status:string (sau bundle)"; }
tc_22(){ pnpm -C frontend lint; eq "$?" 0 "frontend lint"; pnpm -C frontend build; eq "$?" 0 "frontend build"; }
tc_23(){ bash scripts/ui-antipatterns.sh; eq "$?" 0 "ui-antipatterns"; }
tc_24(){ local o; o=$(grep -nE 'Be_Vietnam_Pro|tokens\.css' frontend/src/app/layout.tsx); echo "$o"
  echo "$o" | grep -q Be_Vietnam_Pro && echo "$o" | grep -q tokens.css && pass "có cả hai" || fail "thiếu một trong hai"
  grep -qE "vietnamese" frontend/src/app/layout.tsx && grep -qE "latin" frontend/src/app/layout.tsx && pass "subset vietnamese+latin" || fail "subset"
  grep -q 'lang="vi"' frontend/src/app/layout.tsx && pass '<html lang="vi">' || fail "lang"; }

# ---------- AC4 / FR-4 / FR-5 ----------
tc_25(){ build_gw || { fail "go build"; return; }; gw_run
  eq "$GWRC" 1 "thiếu cả hai: exit"
  grep -c DATABASE_URL $GWLOG; grep -c REDIS_URL $GWLOG
  eq "$(grep -cE 'panic|goroutine' $GWLOG)" 0 "không panic/goroutine"
  eq "$(jq -c 'select(.level=="ERROR")' $GWLOG 2>/dev/null | wc -l | xargs)" 1 "đúng 1 dòng log mức ERROR (slog JSON)"
  jq -c 'select(.level=="ERROR")' $GWLOG | grep DATABASE_URL | grep -q REDIS_URL && pass "cả hai tên cùng một dòng" || fail "tên biến không cùng một dòng ERROR"; }
tc_26(){ build_gw || { fail "go build"; return; }
  gw_run DATABASE_URL='postgres://u:QCSECRET1@h/db'; eq "$GWRC" 1 "chỉ có DATABASE_URL: exit"
  grep -q REDIS_URL $GWLOG && pass "nêu REDIS_URL" || fail "không nêu REDIS_URL"
  [ "$(grep -c DATABASE_URL $GWLOG)" = 0 ] && pass "không nêu DATABASE_URL (đã có)" || fail "nêu nhầm DATABASE_URL"
  [ "$(grep -c QCSECRET1 $GWLOG)" = 0 ] && pass "không log giá trị env" || fail "log lộ giá trị env"
  gw_run REDIS_URL='redis://:QCSECRET2@h:6379'; eq "$GWRC" 1 "chỉ có REDIS_URL: exit"
  grep -q DATABASE_URL $GWLOG && pass "nêu DATABASE_URL" || fail "không nêu DATABASE_URL"
  [ "$(grep -c REDIS_URL $GWLOG)" = 0 ] && pass "không nêu REDIS_URL (đã có)" || fail "nêu nhầm REDIS_URL"
  [ "$(grep -c QCSECRET2 $GWLOG)" = 0 ] && pass "không log giá trị env" || fail "log lộ giá trị env"
  eq "$(grep -cE 'panic|goroutine' $GWLOG)" 0 "không panic"; }
tc_27(){ build_gw || { fail "go build"; return; }; lsof -i :8080 >/dev/null 2>&1 && { fail "cổng 8080 đang bận — chạy pnpm dev:down"; return; }
  local d; d=$(mktemp -d); (cd $d && env -i PATH="$PATH" DATABASE_URL=postgres://x REDIS_URL=redis://x /tmp/gw >$GWLOG 2>&1 & echo $! >/tmp/qc-gwpid); sleep 1.5; local p; p=$(cat /tmp/qc-gwpid)
  eq "$(curl -fsS localhost:8080/healthz)" '{"status":"ok"}' "đủ env, DB/Redis không tồn tại: /healthz vẫn ok (chỉ kiểm sống)"
  eq "$(find $d -type f | wc -l | xargs)" 0 "không ghi file ra đĩa"
  kill -TERM $p; local t=0; while kill -0 $p 2>/dev/null && [ $t -lt 50 ]; do sleep 0.1; t=$((t+1)); done
  kill -0 $p 2>/dev/null && { kill -9 $p; fail "SIGTERM: không thoát sau 5s"; } || pass "SIGTERM: thoát trong $((t/10)).$((t%10))s"
  eq "$(grep -cE 'panic|goroutine' $GWLOG)" 0 "không panic khi dừng"; }

# ---------- AC5 ----------
tc_28(){ local n; n=$(docker ps --filter label=com.docker.compose.project=edupilot -q | wc -l | xargs); eq "$n" 6 "trước down: 6 container project edupilot"
  eq "$(pnpm -s dev:status --format '{{.Service}}' | sort -u | wc -l | xargs)" 6 "dev:status thấy 6"
  ( pnpm -s dev:logs --tail=3 >/tmp/qc-logs.log 2>&1 & p=$!; sleep 12; kill $p 2>/dev/null; pkill -P $p 2>/dev/null ); for s in postgres redis minio mailpit gateway frontend; do grep -q "$s" /tmp/qc-logs.log && pass "dev:logs có $s" || fail "dev:logs thiếu $s"; done
  pnpm dev:down; eq "$?" 0 "dev:down exit"
  eq "$(docker ps --filter label=com.docker.compose.project=edupilot -q | wc -l | xargs)" 0 "sau down: 0 container"; }
tc_29(){ local g s e; g=$($C ps -q gateway); s=$(date +%s); $C stop gateway >/dev/null 2>&1; e=$(date +%s)
  eq "$(docker inspect -f '{{.State.ExitCode}}' $g)" 0 "gateway dừng êm: ExitCode 0 (137 = bị kill cứng)"
  [ $((e-s)) -lt 8 ] && pass "dừng trong $((e-s))s (< 8s, mặc định chờ 10s)" || fail "dừng mất $((e-s))s"; }
tc_30(){ pnpm dev:down >/dev/null 2>&1; pnpm dev:down; eq "$?" 0 "dev:down lần 2 khi đã dừng: exit 0"
  [ "$(docker volume ls --filter label=com.docker.compose.project=edupilot -q | wc -l | xargs)" -ge 3 ] && pass "volume còn sau down (suy ra từ FR-8; dev:down không xoá dữ liệu)" || fail "volume bị xoá bởi dev:down"; }

# ---------- AC6 ----------
tc_31(){ eq "$($C config 2>/dev/null | grep -c legacy)" 0 "compose config không nhắc legacy"
  eq "$(grep -n legacy pnpm-workspace.yaml package.json backend-go/go.mod | wc -l | xargs)" 0 "workspace/package.json/go.mod"
  eq "$(grep -cx 'legacy' .dockerignore)" 1 ".dockerignore có dòng legacy"
  eq "$(git grep -nI legacy -- frontend backend-go docker-compose.local.yml scripts/dev.mjs ':!frontend/pnpm-lock.yaml' | wc -l | xargs)" 0 "frontend/ backend-go/ compose, dev.mjs không nhắc legacy"; }
tc_32(){ eq "$(docker inspect $($C ps -q) --format '{{range .Mounts}}{{.Source}} {{end}}' | grep -c legacy)" 0 "không mount nào từ legacy/"
  eq "$(cd backend-go && go list ./... | grep -c legacy)" 0 "go list không có package legacy"
  eq "$(docker history --no-trunc $($C images -q gateway | head -1) 2>/dev/null | grep -c legacy)" 0 "image gateway không COPY legacy"; }

# ---------- AC7 ----------
tc_33(){ eq "$(curl -fsS localhost:8080/healthz | jq -c 'keys')" '["status"]' "body chỉ có khoá status"
  eq "$(curl -s -o /dev/null -w '%{http_code}' -X POST localhost:8080/healthz)" 405 "POST /healthz → 405"
  local c; c=$(curl -s -o /tmp/qc-404.txt -w '%{http_code}' localhost:8080/khong-co); [ "$c" = 404 ] && pass "đường dẫn lạ → 404" || fail "đường dẫn lạ → $c"
  [ "$(grep -ciE 'panic|goroutine|\.go:[0-9]' /tmp/qc-404.txt)" = 0 ] && pass "404 không lộ stack" || fail "404 lộ stack"
  eq "$(curl -s -o /dev/null -w '%{http_code}' -H 'Authorization: Bearer rac' localhost:8080/healthz)" 200 "công khai: có/không Authorization đều 200"; }
tc_34(){ eq "$(grep -nE 'sk-[A-Za-z0-9]|AKIA[0-9A-Z]{12}' .env.example | wc -l | xargs)" 0 ".env.example không có khoá thật"
  local k; k=$(grep -vE '^\s*(#|$)' .env.example | cut -d= -f1 | xargs -n1 | LC_ALL=C sort | xargs); echo "khoá trong .env.example: $k"
  [ -z "$(echo "$k" | tr ' ' '\n' | grep -vE '^(DATABASE_URL|REDIS_URL|BLOB_[A-Z0-9_]+|SMTP_[A-Z0-9_]+|MAIL_FROM)$')" ] && pass "chỉ biến P0 (FR-10)" || fail "có biến ngoài FR-10"
  echo "$k" | grep -qw DATABASE_URL && echo "$k" | grep -qw REDIS_URL && echo "$k" | grep -q 'BLOB_' && echo "$k" | grep -q 'SMTP_' && echo "$k" | grep -qw MAIL_FROM && pass "đủ nhóm biến FR-10" || fail "thiếu nhóm biến"
  local leak=0 v; while IFS= read -r v; do [ ${#v} -ge 8 ] || continue; git grep -qF -- "$v" -- . ':!legacy' ':!docs' && { echo "LỘ giá trị từ legacy/.env.example: ${v:0:4}…"; leak=1; }; done < <(grep -iE 'JWT|SECRET|KEY|TOKEN' legacy/.env.example | grep -vE '^\s*#' | cut -d= -f2- | tr -d '"')
  [ $leak = 0 ] && pass "không dùng lại khoá JWT/secret cũ" || fail "dùng lại secret cũ"
  eq "$(git ls-files | grep -cE '(^|/)\.env(\.local)?$')" 0 ".env/.env.local không bị track"
  grep -qE 'edupilot-dev' .env.example && pass "mật khẩu dev giả edupilot-dev" || fail "không thấy mật khẩu dev giả"; }

# ---------- Kiểm chéo ----------
tc_35(){ local bad; bad=$(git -c diff.renameLimit=5000 diff -M --name-status $SBASE HEAD | awk '{print $NF"\t"$0}' | cut -f1 \
   | grep -vE '^(legacy/|backend-go/|frontend/|seed/|\.env\.example$|\.dockerignore$|\.gitignore$|package\.json$|pnpm-lock\.yaml$|pnpm-workspace\.yaml$|docker-compose\.local\.yml$|scripts/dev\.mjs$|docs/|\.github/workflows/ci\.yml$|data/tmp/492218d5)')
  [ -z "$bad" ] && pass "diff nằm trong vùng story" || { fail "file ngoài vùng story"; echo "$bad" | head; }
  for f in scripts/team-up.sh scripts/ui-antipatterns.sh CLAUDE.md AGENTS.md .claude .gitattributes .github/workflows/keep-huggingface-space-awake.yml; do ok "$f không đổi so với base" git diff --quiet $SBASE HEAD -- $f; done
  echo "docs/ đổi (xem tay: dev chỉ được đụng handoff/PROGRESS?):"; git diff --name-only $SBASE HEAD -- docs | grep -vE '^docs/(sprints/1/|specs/)' | head; }
tc_36(){ local h; h=$(git diff $SBASE HEAD -- . ':!legacy' ':!docs' | grep '^+' | grep -nE 'sk-[A-Za-z0-9]{10,}|AKIA[0-9A-Z]{12}|BEGIN [A-Z ]*PRIVATE KEY|ghp_[A-Za-z0-9]{20}|xox[bp]-|AIza[0-9A-Za-z_-]{20}|(password|secret|token)[A-Za-z_]*[=:] *["'\'']?[A-Za-z0-9+/]{16,}')
  [ -z "$h" ] && pass "không thấy secret trong dòng thêm" || { fail "nghi secret"; echo "$h" | head -5 | cut -c1-80; }; }
tc_37(){ echo "--- go.mod require (đối chiếu ARCHITECTURE §3 bằng mắt):"; sed -n '/^require/,/^)/p;/^require [^(]/p' backend-go/go.mod
  echo "--- frontend deps (đối chiếu ARCHITECTURE §3):"; jq -c '{dependencies,devDependencies}' frontend/package.json
  eq "$(git grep -nE '\bfetch\(' -- frontend/src | wc -l | xargs)" 0 "frontend/src không có fetch trần"
  eq "$(git grep -nIE '#[0-9a-fA-F]{3,8}\b|rgb\(|hsl\(' -- frontend/src ':!frontend/src/shared/styles/tokens.css' | wc -l | xargs)" 0 "frontend/src không màu viết cứng ngoài tokens.css"
  eq "$(git grep -nE 'os\.(WriteFile|Create|OpenFile|Mkdir)|ioutil\.WriteFile' -- backend-go | wc -l | xargs)" 0 "gateway không ghi đĩa (luật 10)"
  eq "$(git grep -nE '\bfloat64\b' -- backend-go | wc -l | xargs)" 0 "không float64 trong backend-go (luật 5; nghi ngờ thì xem tay)"; }
tc_38(){ for f in requirements.txt requirements.dev.txt Dockerfile Dockerfile.ai src shared-proto data_pipeline backend-java benchmarks db huggingface; do [ -e "$f" ] && fail "còn $f ở gốc" || pass "không còn $f ở gốc"; done
  eq "$(git ls-files | grep -vE '^(legacy/|docs/)' | grep -cE '\.py$|requirements.*\.txt$')" 0 "không file Python ngoài legacy/docs"; }

[ $# -eq 0 ] && { echo "dùng: $0 TC-01 [TC-02 ...]"; exit 2; }
for t in "$@"; do TC=$t; f="tc_${t#TC-}"; if declare -F "$f" >/dev/null; then "$f"; else echo "FAIL $t: không có TC này"; RC=1; fi; done
exit $RC

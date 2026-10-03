#!/usr/bin/env bash
# QC GATE-PG — "Cổng nghiệm thu" + "Bạn tự kiểm" của docs/phases/PG.md, tách thành bước đo được (tc-GATE-PG.md).
#   bash docs/sprints/2/qc/scripts/gate-pg.sh [--list | 01 05 … | TC-GATE-01 …]
# Thứ tự khuyến nghị: chạy hết (01 → 30). Mỗi TC tự đưa stack về chế độ nó cần (ensure_mode) và trả lại bản mặc định (dmode) khi đổi.
# Chạy SAU khi cả 7 story có báo cáo PASS (gate chạy trọn = US-PG-07 AC19). KHÔNG chạy trong phase 1.
source "$(dirname "$0")/lib.sh"
SD=$(cd "$(dirname "$0")" && pwd)
BG="backend-go"
GATE_BR=sprint/2-pg

# ---------- helpers riêng của cổng ----------
gojson_check() {  # gojson_check <file.json> — không skip, không fail; mỗi gói của SRS 9.2 có ≥ 1 test PASS
  local f=$1 n s p pk
  n=$(jq -rs '[.[]|select(.Action=="pass" and .Test!=null)]|length' "$f"); chk_ge "số test PASS" "$n" 100
  s=$(jq -rs '[.[]|select(.Action=="skip" and .Test!=null)]|length' "$f");   # QC v2: chỉ đếm test bị skip (gói không có tệp test cũng phát Action=skip ở mức gói — #10)
                  chk "số test SKIP (Docker phải có ⇒ không được skip)" "$s" 0
  p=$(jq -rs '[.[]|select(.Action=="fail")]|length' "$f");                 chk "số test/gói FAIL" "$p" 0
  for pk in internal/platform/config internal/platform/log internal/platform/otel internal/platform/redis internal/platform/db internal/platform/blob internal/platform/outbox \
            internal/store internal/httpapi internal/httpapi/sse internal/jobs internal/auth internal/contract cmd/gateway cmd/worker; do
    n=$(jq -rs --arg k "/$pk" '[.[]|select(.Action=="pass" and .Test!=null and (.Package|endswith($k)))]|length' "$f")
    chk_ge "gói $pk — test PASS" "$n" 1
  done
  n=$(jq -rs '[.[]|select(.Action=="pass" and .Test!=null and (.Package|test("/db$")))]|length' "$f"); chk_ge "gói db (migration) — test PASS" "$n" 1
}
inst_ids() { for i in $(seq "${1:-40}"); do hdr $GW/api/v1/healthz | hval x-instance-id; done | sort | uniq -c; }
rc_snapshot() { for c in $($C ps -q gateway worker); do echo "$(docker inspect -f '{{.Name}} {{.RestartCount}} {{.State.StartedAt}}' $c)"; done | sort; }
pub() {  # pub <token> <type> <json-data> [user_id] — phát một sự kiện qua route thử
  curl -sk --max-time 10 -o /dev/null -w '%{http_code}' -X POST -H "Authorization: Bearer $1" -H 'Content-Type: application/json' \
    -d "{\"type\":\"$2\",\"data\":$3${4:+,\"user_id\":\"$4\"}}" $GW/api/v1/_test/events; }
sse_ids() { grep -E '^id: ' "$1" | sed 's/^id: //;s/\r//'; }                    # id Redis Stream đã nhận
sse_ns()  { grep -E '^data: ' "$1" | sed 's/^data: //;s/\r//' | jq -r 'select(.n!=null).n' 2>/dev/null; }   # QC v2: bỏ tiền tố "data: " trước khi jq    # giá trị n đã nhận (thứ tự)
dup_free() { sort | uniq -d | wc -l | tr -d ' '; }

# ================= A. CỔNG NGHIỆM THU (PG.md mục "Cổng nghiệm thu") =================
tc_gate_01() {  # go vet (PG.md) + biến thể có tag (US-PG-07 AC19)
  (cd $BG && go vet ./... ) >$QC_OUT/g01a.log 2>&1; chk "go vet ./... rc" "$?" 0
  (cd $BG && go vet -tags testroutes ./... ) >$QC_OUT/g01b.log 2>&1; chk "go vet -tags testroutes ./... rc" "$?" 0
  chk "go vet: không in cảnh báo" "$(cat $QC_OUT/g01a.log $QC_OUT/g01b.log | wc -l | tr -d ' ')" 0; }
tc_gate_02() {  # golangci-lint, hai cấu hình
  command -v golangci-lint >/dev/null || { fail_tc "KHÔNG KIỂM ĐƯỢC: thiếu golangci-lint trên máy QC"; return; }
  (cd $BG && golangci-lint run) >$QC_OUT/g02a.log 2>&1; chk "golangci-lint run rc" "$?" 0
  (cd $BG && golangci-lint run --build-tags testroutes) >$QC_OUT/g02b.log 2>&1; chk "golangci-lint run --build-tags testroutes rc" "$?" 0; }
tc_gate_03() {  # go test -race ./... đúng như PG.md (không tag): rc 0, ghi nhận không skip
  (cd $BG && go test -race -count=1 -timeout 40m -json ./...) >$QC_OUT/g03.json 2>$QC_OUT/g03.err; chk "go test -race ./... (PG.md, không tag) rc" "$?" 0
  chk "FAIL/SKIP trong -json" "$(jq -rs '[.[]|select(.Action=="fail" or (.Action=="skip" and .Test!=null))]|length' $QC_OUT/g03.json)" 0; }
tc_gate_04() {  # go test -race -tags testroutes ./... (US-PG-07 AC19; make test): đủ gói, đủ test, 0 skip
  (cd $BG && go test -race -count=1 -timeout 40m -tags testroutes -json ./...) >$QC_OUT/g04.json 2>$QC_OUT/g04.err; chk "go test -race -tags testroutes ./... rc" "$?" 0
  gojson_check $QC_OUT/g04.json; }
tc_gate_05() {  # sqlc diff sạch
  command -v sqlc >/dev/null || { fail_tc "KHÔNG KIỂM ĐƯỢC: thiếu sqlc trên máy QC"; return; }
  out=$(cd $BG && sqlc diff 2>&1); rc=$?; chk "sqlc diff rc" "$rc" 0; chk "sqlc diff không in gì" "$out" ""; }
tc_gate_06() {  # contract test: PG.md (không tag) + có tag
  (cd $BG && go test -count=1 ./internal/contract/... ) >$QC_OUT/g06a.log 2>&1; chk "contract không tag rc" "$?" 0
  (cd $BG && go test -count=1 -tags testroutes ./internal/contract/... ) >$QC_OUT/g06b.log 2>&1; chk "contract có tag rc" "$?" 0
  chk_nre "contract không tag không 'no tests'" "$(cat $QC_OUT/g06a.log)" 'no tests to run|no test files'
  chk_nre "contract có tag không 'no tests'" "$(cat $QC_OUT/g06b.log)" 'no tests to run|no test files'; }
tc_gate_07() {  # httpapi -race: cursor, Idempotency gửi đôi → một bản ghi, 409 version, ETag — chứng minh bằng tên test
  gt './internal/httpapi' 'TestCursor_NoSkipNoDup|TestCursor_SameTimestamp|TestCursor_InsertAndDeleteDuringScan|TestIdempotency_DoubleSend_OneRecord|TestIdempotency_ReplayIdentical|TestIdempotency_Concurrent50|TestOptimisticLock_Stale409|TestOptimisticLock_Concurrent20|TestETag_NotModified|TestETag_ChangesAfterUpdate'
  (cd $BG && go test -race -count=1 -timeout 20m -tags testroutes ./internal/httpapi/...) >$QC_OUT/g07.log 2>&1; chk "go test -race ./internal/httpapi/... rc" "$?" 0; }
tc_gate_08() {  # sse + auth -race
  gt './internal/httpapi/sse/...' 'TestSSE_Headers|TestSSE_LastEventID_NoLossNoDup|TestSSE_ReconnectRace|TestSSE_CrossInstance|TestSSE_GatewayShutdownFailover|TestSSE_MaxTwoPerUser'
  gt './internal/auth/...' 'TestJWT_Claims|TestVerify_Table|TestRBAC_Matrix|TestCourseAccessGuard_DefaultDenyAll|TestBcrypt_Format'
  (cd $BG && go test -race -count=1 -timeout 20m -tags testroutes ./internal/httpapi/sse/... ./internal/auth/...) >$QC_OUT/g08.log 2>&1; chk "go test -race sse + auth rc" "$?" 0; }
tc_gate_09() {  # frontend build
  (pnpm -C frontend build) >$QC_OUT/g09.log 2>&1; chk "pnpm -C frontend build rc" "$?" 0; }
tc_gate_10() {  # up -d --scale gateway=2 với image MẶC ĐỊNH
  $C up -d --build --force-recreate --scale gateway=2 --wait >$QC_OUT/g10.log 2>&1; chk "up -d --scale gateway=2 --wait rc" "$?" 0
  chk "số container gateway" "$($C ps -q gateway | wc -l | tr -d ' ')" 2
  chk "chế độ" "$(mode_now)" default
  chk "migrate" "$($C ps -a --format '{{.Service}} {{.State}} {{.ExitCode}}' | grep '^migrate ')" "migrate exited 0"; }
tc_gate_11() {  # curl -fsSk healthz qua Caddy, 200 ở CẢ HAI bản
  out=$(curl -fsSk $GW/api/v1/healthz); chk "curl -fsSk healthz" "$out" '{"status":"ok"}'
  tab=$(inst_ids 40); echo "$tab"
  chk "số X-Instance-Id khác nhau" "$(echo "$tab" | wc -l | tr -d ' ')" 2
  chk "mọi bản được chọn (đếm > 0)" "$(echo "$tab" | awk '$1>0' | wc -l | tr -d ' ')" 2
  for i in $(seq 40); do code $GW/api/v1/healthz; echo; done | sort | uniq -c | tee $QC_OUT/g11.codes >/dev/null
  chk "40 lần 200" "$(awk '$2=="200"{print $1}' $QC_OUT/g11.codes)" 40; }
tc_gate_12() {  # k6 smoke trên bản mặc định: rc 0, ngưỡng đạt; stack không restart trong lúc đo
  command -v k6 >/dev/null || { fail_tc "KHÔNG KIỂM ĐƯỢC: thiếu k6"; return; }
  before=$(rc_snapshot)
  k6 run -e BASE=$GW -e TOKEN="$(tok STUDENT $U1 --ttl 30m)" benchmarks/load/smoke.js >$QC_OUT/g12.k6 2>&1; rc=$?
  chk "k6 rc" "$rc" 0
  chk_ge "số dòng p(95)<300 trong smoke.js" "$(grep -c 'p(95)<300' benchmarks/load/smoke.js)" 3
  chk "không ngưỡng lớn hơn SLO (chỉ 300 và 500)" "$(grep -oE 'p\(95\)<[0-9]+' benchmarks/load/smoke.js | sort -u | paste -sd' ' -)" "p(95)<300 p(95)<500"
  chk_nre "k6 không có ngưỡng ✗" "$(grep -E '^\s+(✗|X) ' $QC_OUT/g12.k6 | head -3)" '.'
  after=$(rc_snapshot); chk "RestartCount/StartedAt gateway+worker không đổi khi chạy k6" "$after" "$before"; }
tc_gate_13() {  # biến thể ghi (-e TEST_ROUTES=1) trên bản test; rồi trả về mặc định
  command -v k6 >/dev/null || { fail_tc "KHÔNG KIỂM ĐƯỢC: thiếu k6"; return; }
  ensure_mode test || { fail_tc "không vào được chế độ test"; return; }
  k6 run -e BASE=$GW -e TEST_ROUTES=1 -e TOKEN="$(tok STUDENT $U1 --ttl 30m)" benchmarks/load/smoke.js >$QC_OUT/g13.k6 2>&1; chk "k6 -e TEST_ROUTES=1 rc" "$?" 0
  grep -E 'http_req_duration.*write|write' $QC_OUT/g13.k6 | head -3
  ensure_mode default; k6 run -e BASE=$GW -e TEST_ROUTES=1 -e TOKEN="$(tok STUDENT $U1)" benchmarks/load/smoke.js >$QC_OUT/g13b.k6 2>&1
  chk_ne "k6 TEST_ROUTES=1 ở bản mặc định phải DỪNG (rc≠0)" "$?" 0
  chk_re "thông báo rõ route thử vắng" "$(cat $QC_OUT/g13b.k6)" '_test|testroutes|TEST_ROUTES|404'; }

# ================= B. BẠN TỰ KIỂM (PG.md mục "Bạn tự kiểm") =================
tc_gate_20() {  # (1) xoá volume DB rồi pnpm dev: migration 00001 chạy hết, không thao tác tay
  pnpm dev:down >$QC_OUT/g20.down 2>&1; docker volume rm edupilot_postgres_data >>$QC_OUT/g20.down 2>&1; chk "volume postgres đã xoá" "$(docker volume ls -q | grep -c '^edupilot_postgres_data$')" 0
  t0=$(now_ms); (pnpm dev >$QC_OUT/g20.dev 2>&1 </dev/null) & pid=$!
  wait_ready 300; chk "readyz 200 sau pnpm dev (≤ 300 s)" "$(code $GW/api/v1/readyz)" 200
  echo "    thời gian tới readyz: $(( ($(now_ms)-t0)/1000 )) s"
  chk "migrate" "$($C ps -a --format '{{.Service}} {{.State}} {{.ExitCode}}' | grep '^migrate ')" "migrate exited 0"
  chk_ge "goose_db_version dòng đã áp" "$($PSQL -c 'select count(*) from goose_db_version where is_applied')" 2
  chk "bảng nền" "$($PSQL -c "select tablename from pg_tables where schemaname='public' and tablename not like '\\_test%' order by 1" | paste -sd' ' -)" "audit_log goose_db_version idempotency_keys jobs outbox users"
  chk_nre "pnpm dev không hỏi/nhắc thao tác tay" "$(cat $QC_OUT/g20.dev)" '\[y/N\]|\(y/n\)|Press|Enter to|please run|chạy tay'
  kill $pid 2>/dev/null; true; }
tc_gate_21() {  # (2) bỏ một biến bắt buộc: chết NGAY lúc khởi động, nêu đúng tên; không chết lúc đang phục vụ
  [ -x "$GWBIN" ] || tok STUDENT >/dev/null
  full() { printf '%s\n' DATABASE_URL=postgres://u:p@127.0.0.1:1/db REDIS_URL=redis://127.0.0.1:1/0 JWT_SECRET_KEY=0123456789abcdef0123456789abcdef BLOB_ENDPOINT=127.0.0.1:1 BLOB_BUCKET=b BLOB_ACCESS_KEY=ak BLOB_SECRET_KEY=sk; }
  for v in DATABASE_URL REDIS_URL JWT_SECRET_KEY BLOB_ENDPOINT BLOB_BUCKET BLOB_ACCESS_KEY BLOB_SECRET_KEY; do
    s=$(now_ms); full | grep -v "^$v=" | xargs env -i PATH="$PATH" "$GWBIN" serve >$QC_OUT/g21.$v 2>&1; rc=$?; ms=$(( $(now_ms)-s ))
    chk "$v: rc" "$rc" 1; chk_le "$v: ms tới khi thoát" "$ms" 1000
    chk "$v: tên có trong log" "$(jq -r 'select(.missing).missing|index("'$v'")!=null' $QC_OUT/g21.$v | head -1)" true
  done; }
tc_gate_22() {  # (2b) cùng việc ở compose: gateway thật thoát 1 nêu tên biến; service đang phục vụ khác KHÔNG chết
  ensure_mode default; cp .env.local $QC_TMP/env.local.bak; before=$(rc_snapshot)
  sed -i.bak '/^JWT_SECRET_KEY=/d' .env.local; $C run --rm --no-deps --name qc-gw-noenv gateway serve >$QC_OUT/g22.log 2>&1; rc=$?
  cp $QC_TMP/env.local.bak .env.local; rm -f .env.local.bak
  chk "gateway thiếu JWT_SECRET_KEY: rc" "$rc" 1
  chk "log nêu tên biến" "$(grep '^{' $QC_OUT/g22.log | jq -r 'select(.missing).missing[]' 2>/dev/null | sort -u)" JWT_SECRET_KEY   # QC v2: log của `compose run` có dòng "Container … Creating" không phải JSON
  chk "cổng 8080 không mở / không phục vụ (readyz của bản thiếu env không tồn tại)" "$(docker ps -q --filter name=qc-gw-noenv | wc -l | tr -d ' ')" 0
  chk "bản đang phục vụ không bị ảnh hưởng (RestartCount/StartedAt)" "$(rc_snapshot)" "$before"
  chk "readyz vẫn 200" "$(code $GW/api/v1/readyz)" 200; chk ".env.local khôi phục" "$(cmp -s $QC_TMP/env.local.bak .env.local && echo y)" y; }
tc_gate_23() {  # (3a) SSE: rút mạng phía client 10 s rồi nối lại bằng Last-Event-ID — không mất, không trùng (mô phỏng bằng curl)
  ensure_mode test; T=$(tok STUDENT $U1 --ttl 10m); f1=$QC_OUT/g23.1; f2=$QC_OUT/g23.2; : >$f1; : >$f2
  pid=$(sse_cap $f1 60 "$T"); sleep 1
  for n in 1 2 3 4 5; do pub "$T" test.ping "{\"n\":$n}" >/dev/null; sleep 0.3; done
  sleep 1; kill $pid 2>/dev/null; wait $pid 2>/dev/null           # rút mạng
  last=$(sse_ids $f1 | tail -1); chk_re "đã nhận id e5" "$last" '^[0-9]+-[0-9]+$'; chk "đã nhận n 1..5" "$(sse_ns $f1 | paste -sd' ' -)" "1 2 3 4 5"
  for n in 6 7 8 9 10 11 12; do pub "$T" test.ping "{\"n\":$n}" >/dev/null; sleep 1.2; done      # 10+ giây mất mạng, phát tiếp
  pid=$(sse_cap $f2 8 "$T" -H "Last-Event-ID: $last"); while kill -0 $pid 2>/dev/null; do sleep 0.3; done   # QC v2: pid của sse_cap thuộc subshell $(…) nên `wait` trả về ngay
  chk "sau nối lại nhận đúng 6..12 theo thứ tự" "$(sse_ns $f2 | paste -sd' ' -)" "6 7 8 9 10 11 12"
  chk "không nhận lại e5" "$(sse_ns $f2 | grep -c '^5$')" 0
  chk "không trùng" "$(sse_ns $f2 | dup_free)" 0
  chk "không có resync" "$(grep -c '^event: resync' $f2)" 0
  chk "dòng đầu stream nối lại" "$(head -1 $f2 | tr -d '\r')" "retry: 3000"; }
tc_gate_24() {  # (3b) rút mạng THẬT giữa Caddy và mạng compose 10 s (docker network disconnect) rồi cắm lại
  ensure_mode test; T=$(tok STUDENT $U1 --ttl 10m); f1=$QC_OUT/g24.1; f2=$QC_OUT/g24.2; : >$f1; : >$f2
  net=edupilot_default; cad=$($C ps -q caddy)
  docker network inspect $net >/dev/null 2>&1 || { fail_tc "KHÔNG KIỂM ĐƯỢC: không thấy network $net"; return; }
  pid=$(sse_cap $f1 60 "$T"); sleep 1; pub "$T" test.ping '{"n":1}' >/dev/null; pub "$T" test.ping '{"n":2}' >/dev/null; sleep 1
  docker network disconnect $net $cad >/dev/null 2>&1; echo "    mạng đã rút $(date +%T)"; sleep 10
  docker network connect --alias caddy $net $cad >/dev/null 2>&1; echo "    mạng đã cắm $(date +%T)"
  sleep 3; kill $pid 2>/dev/null; wait $pid 2>/dev/null
  last=$(sse_ids $f1 | tail -1)
  wait_ready 40; chk "gateway nhận lại qua Caddy ≤ 40 s sau khi cắm" "$(code $GW/api/v1/readyz)" 200
  pub "$T" test.ping '{"n":3}' >/dev/null; pub "$T" test.ping '{"n":4}' >/dev/null
  pid=$(sse_cap $f2 4 "$T" -H "Last-Event-ID: $last"); while kill -0 $pid 2>/dev/null; do sleep 0.3; done   # QC v2: pid của sse_cap thuộc subshell $(…) nên `wait` trả về ngay
  chk "đã nhận trước khi rút: 1 2" "$(sse_ns $f1 | paste -sd' ' -)" "1 2"
  chk "nối lại nhận 3 4 (không lặp 2)" "$(sse_ns $f2 | paste -sd' ' -)" "3 4"
  chk "gateway/worker không restart" "$($C ps -q gateway worker | while read c; do docker inspect -f '{{.RestartCount}}' $c; done | sort -u | paste -sd' ' -)" 0; }
tc_gate_25() {  # (3c) trình duyệt thật: Chrome offline 10 s → tự nối lại bằng Last-Event-ID (sse-browser-cut.mjs, chạy bằng eval)
  manual "chạy: const m = await import('$SD/sse-browser-cut.mjs'); await m.default(browser, { base: '$GW', token: <tok STUDENT U1 --ttl 10m>, publishCmd: <xem tc-GATE-PG.md> }); kỳ vọng ok=true"; }
tc_gate_26() {  # (4a) gửi đúp Idempotency-Key bằng hai tiến trình song song (hai 'tab'): một bản ghi, hai response giống hệt
  ensure_mode test; key="gate-$(now_ms)-aaaa"; a=$QC_OUT/g26.a; b=$QC_OUT/g26.b
  for t in a b; do ( curl -sk -D $QC_OUT/g26.$t.h -o $QC_OUT/g26.$t.body -X POST -H "$H" -H 'Content-Type: application/json' -H "Idempotency-Key: $key" -d '{"name":"gate-dup"}' $GW/api/v1/_test/items ) & done; wait
  sa=$(head -1 $QC_OUT/g26.a.h | awk '{print $2}'); sb=$(head -1 $QC_OUT/g26.b.h | awk '{print $2}')
  echo "    hai tab: $sa / $sb"
  sleep 1.2   # nếu một tab nhận 409 IDEMPOTENCY_IN_PROGRESS thì gửi lại sau Retry-After
  for t in a b; do c=$(head -1 $QC_OUT/g26.$t.h | awk '{print $2}'); [ "$c" = 409 ] && curl -sk -D $QC_OUT/g26.$t.h -o $QC_OUT/g26.$t.body -X POST -H "$H" -H 'Content-Type: application/json' -H "Idempotency-Key: $key" -d '{"name":"gate-dup"}' $GW/api/v1/_test/items; done
  chk "một bản ghi duy nhất" "$($PSQL -c "select count(*) from _test_items where name='gate-dup'")" 1
  chk "hai thân giống hệt (byte)" "$(cmp -s $QC_OUT/g26.a.body $QC_OUT/g26.b.body && echo same)" same
  chk "hai status giống nhau" "$(head -1 $QC_OUT/g26.a.h | awk '{print $2}')" "$(head -1 $QC_OUT/g26.b.h | awk '{print $2}')"
  chk "đúng một response có Idempotent-Replayed: true" "$(cat $QC_OUT/g26.a.h $QC_OUT/g26.b.h | grep -ci '^idempotent-replayed: true')" 1
  chk "hai Content-Type giống nhau" "$(grep -i '^content-type' $QC_OUT/g26.a.h)" "$(grep -i '^content-type' $QC_OUT/g26.b.h)"; }
tc_gate_27() {  # (4b) cùng việc bằng hai tab trình duyệt thật (idem-two-tabs.mjs, chạy bằng eval)
  manual "chạy: const m = await import('$SD/idem-two-tabs.mjs'); await m.default(browser, { base: '$GW', token: <tok STUDENT U1 --ttl 10m> }); kỳ vọng ok=true"; }
tc_gate_28() {  # (5a) tắt MỘT trong hai gateway giữa stream: bản mang stream → shutdown, nối lại bản kia cùng token, không mất/trùng
  ensure_mode test; T=$(tok STUDENT $U1 --ttl 10m); f1=$QC_OUT/g28.1; f2=$QC_OUT/g28.2
  ids=$(gw_ids); victim=""; for k in 1 2 3 4 5 6; do   # tìm stream rơi vào bản nào qua X-Instance-Id
    hdrf=$QC_OUT/g28.h; : >$f1; ( curl -sk -N --max-time 40 -D $hdrf -H "Authorization: Bearer $T" $GW/api/v1/events >$f1 2>/dev/null ) & pid=$!; sleep 1.5
    inst=$(tr -d '\r' <$hdrf | hval x-instance-id); victim=$(docker ps -q --filter "id=$inst" | head -1)
    [ -n "$victim" ] && break; kill $pid 2>/dev/null; wait $pid 2>/dev/null; done
  [ -n "$victim" ] || { fail_tc "không xác định được bản mang stream (X-Instance-Id=$inst)"; return; }
  echo "    stream đang ở $inst"; for n in 1 2 3; do pub "$T" test.ping "{\"n\":$n}" >/dev/null; done; sleep 1
  s=$(now_ms); docker stop -t 30 $victim >/dev/null 2>&1; echo "    docker stop mất $(( $(now_ms)-s )) ms"
  sleep 1; wait $pid 2>/dev/null
  chk "client nhận event: shutdown" "$(grep -c '^event: shutdown' $f1)" 1
  chk "shutdown reason" "$(grep -A1 '^event: shutdown' $f1 | grep '^data:' | sed 's/^data: //;s/\r//' | jq -r .reason)" server_shutdown
  last=$(sse_ids $f1 | tail -1); chk "đã nhận 1 2 3 trước khi tắt" "$(sse_ns $f1 | paste -sd' ' -)" "1 2 3"
  for n in 4 5 6; do pub "$T" test.ping "{\"n\":$n}" >/dev/null; done                   # phát trong lúc chuyển bản
  chk "phiên kế tiếp vẫn chạy (cùng token, không 401)" "$(code -H "Authorization: Bearer $T" $GW/api/v1/jobs/$UX)" 404
  : >$f2; pid=$(sse_cap $f2 4 "$T" -H "Last-Event-ID: $last"); while kill -0 $pid 2>/dev/null; do sleep 0.3; done   # QC v2: pid của sse_cap thuộc subshell $(…) nên `wait` trả về ngay
  chk "nối lại bản còn lại: nhận 4 5 6" "$(sse_ns $f2 | paste -sd' ' -)" "4 5 6"
  $C up -d --scale gateway=2 --wait gateway >/dev/null 2>&1; wait_ready 60; chk "khôi phục 2 gateway" "$(gw_ids | wc -l | tr -d ' ')" 2; }
tc_gate_29() {  # (5b) tắt MỘT gateway KHÔNG mang stream: stream giữ nguyên; lưu lượng GET 0 lỗi sau 6 s
  ensure_mode test; T=$(tok STUDENT $U1 --ttl 10m); f1=$QC_OUT/g29.1; : >$f1
  for k in 1 2 3 4 5 6; do hdrf=$QC_OUT/g29.h; ( curl -sk -N --max-time 60 -D $hdrf -H "Authorization: Bearer $T" $GW/api/v1/events >$f1 2>/dev/null ) & pid=$!; sleep 1.5
    inst=$(tr -d '\r' <$hdrf | hval x-instance-id); [ -n "$inst" ] && break; done
  other=$(docker ps -q --filter name=gateway | grep -v "^$inst" | head -1)   # QC v2: X-Instance-Id là id container (12 hex), không phải tên; [ -n "$other" ] || { fail_tc "không thấy bản thứ hai"; return; }
  ( for i in $(seq 200); do code $GW/api/v1/healthz; echo; sleep 0.1; done >$QC_OUT/g29.codes ) & lp=$!
  sleep 5; docker stop -t 30 $other >/dev/null 2>&1; wait $lp
  chk_le "dòng không-200 trong 200 request" "$(grep -vc '^200$' $QC_OUT/g29.codes)" 4
  chk "60 dòng cuối 0 lỗi" "$(tail -60 $QC_OUT/g29.codes | grep -vc '^200$')" 0
  pub "$T" test.ping '{"n":99}' >/dev/null; sleep 1; kill $pid 2>/dev/null; wait $pid 2>/dev/null
  chk "stream cũ vẫn sống và nhận n=99" "$(sse_ns $f1 | tail -1)" 99
  chk "không có event: shutdown (bản mang stream không bị tắt)" "$(grep -c '^event: shutdown' $f1)" 0
  $C up -d --scale gateway=2 --wait gateway >/dev/null 2>&1; wait_ready 60; chk "khôi phục 2 gateway" "$(gw_ids | wc -l | tr -d ' ')" 2; }
tc_gate_30() {  # (6) đọc toàn bộ diff internal/auth và internal/httpapi (mọi phase sau kế thừa) — QC đọc tay; script gom ứng viên
  bash "$SD/diff-review.sh" | tee $QC_OUT/g30.cand | tail -40
  manual "đọc tay diff theo checklist trong tc-GATE-PG.md (TC-GATE-30); ghi từng phát hiện BUG-n vào report"; }

# ---------- Cuối cùng: trả stack về bản mặc định ----------
tc_gate_99() {  # dọn: trả stack về image mặc định, 2 gateway, và kiểm 0 route thử
  dmode >/dev/null 2>&1; wait_ready 90
  chk "chế độ cuối" "$(mode_now)" default
  chk "2 gateway healthy" "$($C ps --format '{{.Service}} {{.Health}}' | grep -c '^gateway healthy')" 2
  chk "mọi container không ở trạng thái exited bất ngờ (trừ migrate)" "$($C ps -a --format '{{.Service}} {{.State}}' | grep -v '^migrate ' | grep -vc ' running$')" 0; }

main gate "$@"

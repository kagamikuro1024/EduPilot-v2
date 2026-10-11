# Dùng chung cho gate-p3.sh / gate-p8.sh (nguồn: `source`). Dừng ở lỗi đầu; bảng PASS/FAIL/SKIP; dòng cuối `GATE <tên>: PASS | PASS (có SKIP) | FAIL`.
export DOCKER_HOST="${DOCKER_HOST:-unix://$HOME/.colima/default/docker.sock}" TESTCONTAINERS_RYUK_DISABLED=true
RESULTS=()
FAILED=0
SKIPPED=0
go_in() { (cd backend-go && "$@"); }
fe() { (cd frontend && "$@"); }

run() { # run "tên" lệnh... — các bước sau lỗi đầu không chạy
  [ "$FAILED" = 1 ] && return
  local name="$1" t0=$SECONDS; shift
  echo "▶ $name"
  if "$@"; then RESULTS+=("PASS|$name|$((SECONDS - t0))"); else RESULTS+=("FAIL|$name|$((SECONDS - t0))"); FAILED=1; fi
}

skip() { # skip "tên" "lý do" — chỉ dùng cho bước k6 / docling (US-P3-08 AC9, US-P8-03 AC16)
  [ "$FAILED" = 1 ] && return
  echo "SKIP $1: $2"; RESULTS+=("SKIP|$1 ($2)|0"); SKIPPED=1
}

# Test BẮT BUỘC chạy thật: thiếu stack (Docker không tới được) hoặc test bị SKIP / không in PASS → FAIL, KHÔNG BAO GIỜ SKIP.
strict_test() { # strict_test TênTest gói [-tags …]
  [ "$FAILED" = 1 ] && return
  local name="$1" pkg="$2"; shift 2
  local t0=$SECONDS
  echo "▶ $name"
  if ! docker info >/dev/null 2>&1; then
    echo "FAIL $name: thiếu stack (Docker không tới được)"; RESULTS+=("FAIL|$name: thiếu stack|0"); FAILED=1; return
  fi
  local out; out=$(cd backend-go && go test -count=1 -v "$@" -run "^${name}\$" "$pkg" 2>&1); local rc=$?
  echo "$out" | tail -n 15
  if [ $rc -ne 0 ] || ! echo "$out" | grep -q -- "--- PASS: ${name} " || echo "$out" | grep -q -- "--- SKIP: ${name}"; then
    echo "FAIL $name: thiếu stack hoặc test không chạy / bị SKIP"; RESULTS+=("FAIL|$name|$((SECONDS - t0))"); FAILED=1
  else
    RESULTS+=("PASS|$name|$((SECONDS - t0))")
  fi
}

finish() { # finish P3
  echo; echo "| Kết quả | Bước | Giây |"; echo "|---|---|---|"
  for r in "${RESULTS[@]}"; do IFS="|" read -r st nm sec <<<"$r"; echo "| $st | $nm | $sec |"; done
  echo
  if [ "$FAILED" = 1 ]; then echo "GATE $1: FAIL"; exit 1; fi
  if [ "$SKIPPED" = 1 ]; then echo "GATE $1: PASS (có SKIP)"; else echo "GATE $1: PASS"; fi
}

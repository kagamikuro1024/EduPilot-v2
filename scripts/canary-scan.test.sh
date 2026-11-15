#!/usr/bin/env bash
# DEF-R9.5-02: canary-scan.sh xoá CẢ HAI nhà cung cấp canary (CN0, CN1) bằng id nhà cung cấp, không dùng id của mô hình. curl giả ghi lại lệnh DELETE.
set -euo pipefail
cd "$(dirname "$0")/.."
tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
cat > "$tmp/curl" <<'S'
#!/usr/bin/env bash
args="$*"
echo "$args" >> "$CURL_LOG"
case "$args" in
  *"-X POST"*"/admin/llm/providers "*|*"-X POST"*"/admin/llm/providers -d"*)
    name=$(grep -o '"name\\":\\"CN[0-9]' <<<"$args" | grep -o 'CN[0-9]' || true); [ -n "$name" ] || name=$(grep -o 'name":"CN[0-9]' <<<"$args" | grep -o 'CN[0-9]' || true)
    case "$name" in
      CN0) echo '{"id":"00000000-0000-4000-8000-0000000000a0","name":"CN0","models":[{"id":"00000000-0000-4000-8000-0000000000b0"}]}';;
      CN1) echo '{"id":"00000000-0000-4000-8000-0000000000a1","name":"CN1","models":[{"id":"00000000-0000-4000-8000-0000000000b1"}]}';;
      *) echo '{"code":"VALIDATION_FAILED"}';;
    esac;;
  *) echo '{}';;
esac
S
chmod +x "$tmp/curl"
export CURL_LOG="$tmp/log" PATH="$tmp:$PATH" A="Authorization: Bearer x" COMPOSE="echo" PSQL="echo 0" GW="https://stub"
bash scripts/canary-scan.sh >/dev/null
dels=$(grep -- '-X DELETE' "$CURL_LOG" || true)
for id in 0000000000a0 0000000000a1; do grep -q "providers/00000000-0000-4000-8000-$id" <<<"$dels" || { echo "FAIL: không xoá nhà cung cấp $id"; echo "$dels"; exit 1; }; done
if grep -q '0000000000b' <<<"$dels"; then echo "FAIL: DELETE nhận id của mô hình"; exit 1; fi
echo "ok"

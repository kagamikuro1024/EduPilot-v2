#!/usr/bin/env bash
# smkP: TC-PU01-13 (7 luật ESLint ep/*, mỗi lượt một tệp, mong exit≠0 + đúng tên luật), TC-PU01-17 (3 ca âm bị chặn, ca lành không), TC-PU01-12 (ngắt lint-selftest bằng SIGINT sau 2 s → không để tệp tạm). Chạy ở gốc repo.
set -u; D=frontend/src/__smkp__; cleanup(){ rm -rf "$D"; }; trap cleanup EXIT; cleanup; mkdir -p "$D"
run(){ rule="$1"; body="$2"; f="$D/t$RANDOM.tsx"; printf '%s\n' "$body" >"$f"; out=$(pnpm -C frontend exec eslint "src/__smkp__/$(basename "$f")" 2>&1); rc=$?; rm -f "$f"; echo "$rc|$out"; }
echo "== PU01-13"; n=0
while IFS='|' read -r rule body; do r=$(run "$rule" "$body"); rc=${r%%|*}; if [ "$rc" != 0 ] && grep -qF "$rule" <<<"$r"; then n=$((n+1)); echo "BẮT $rule (rc=$rc)"; else echo "KHÔNG BẮT $rule rc=$rc"; echo "$r" | head -4; fi; done <<'C'
ep/no-raw-fetch|export const a = () => fetch("/x");
ep/no-native-dialogs|export const b = () => window.alert("?");
ep/no-custom-spinner|export const LoadingSpinner = () => null;
ep/no-raw-table|export const D = () => <table><tbody /></table>;
ep/no-token-in-storage|export const e = () => sessionStorage.setItem("jwt", "x");
ep/no-tailwind|import "tailwindcss";
ep/no-literal-color-in-style|export const G = () => <p style={{ background: "#00ff00" }}>x</p>;
C
echo "PU01-13: $n/7"
echo "== PU01-17"; m=0
for b in 'localStorage.setItem("access_token","x");' 'sessionStorage.setItem("jwt","x");' 'document.cookie = "refresh=1";'; do r=$(run ep/no-token-in-storage "export const f = () => { $b };"); rc=${r%%|*}; if [ "$rc" != 0 ] && grep -qF 'ep/no-token-in-storage' <<<"$r"; then m=$((m+1)); echo "ÂM bị chặn: $b"; else echo "ÂM KHÔNG bị chặn: $b rc=$rc"; echo "$r" | head -3; fi; done
r=$(run ep/no-token-in-storage 'export const f = () => { localStorage.setItem("ep_demo_state","{}"); };'); rc=${r%%|*}; echo "LÀNH rc=$rc (mong 0)"; [ "$rc" = 0 ] && lanh=ok || { lanh=sai; echo "$r" | head -3; }
echo "PU01-17: âm $m/3, lành $lanh"
echo "== PU01-12"; cleanup
bash scripts/lint-selftest.sh >/tmp/smkP-lintself.out 2>&1 & pid=$!; sleep 2; kill -INT $pid; sleep 4; kill -0 $pid 2>/dev/null && { kill $pid; echo "(tiến trình còn sống sau INT, đã kill)"; }; wait $pid 2>/dev/null
ls frontend/src/__selftest__ 2>&1 | head -2; echo "git status frontend/src: $(git status --short frontend/src | wc -l) dòng"; git status --short frontend/src | head

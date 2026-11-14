#!/usr/bin/env python3
"""Đo trước / sau của decision model (US-DM-08 AC4, AC5, AC9). Chỉ thư viện chuẩn, chỉ HTTP (D46): chạy như một client ngoài.

  bash benchmarks/decision/run.sh            (stack đã dựng + seed; provider decision GIẢ; sinh chữ qua cấu hình sẵn có)
Biến: API_URL (https://localhost/api/v1), SEED_DEFAULT_PASSWORD (bắt buộc), CHAT_ROUNDS (3), BENCH_COURSE (761987),
      GRADE_TIMEOUT (900 s), BENCH_OUT (benchmarks/reports).
Mỗi kịch bản chạy HAI lần trên cùng stack: cổng TẮT («trước») rồi cổng BẬT («sau», qua API ADMIN, công tắc hệ thống + lớp).
Không gọi nhà cung cấp thật: provider decision phải là bản giả (`provider.fake`); lượt thật chờ phép PM (AC6).
Thoát mã 2 «Gateway chưa chạy» khi không nối được (không ghi báo cáo rỗng).
"""
import hashlib, json, os, re, ssl, statistics, sys, time, urllib.error, urllib.request, uuid
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
DATA = Path(__file__).resolve().parent / "data"
API = os.environ.get("API_URL", "https://localhost/api/v1").rstrip("/")
PASSWORD = os.environ.get("SEED_DEFAULT_PASSWORD", "")
ROUNDS = int(os.environ.get("CHAT_ROUNDS", "3"))
CLASS = os.environ.get("BENCH_COURSE", "761987")
GRADE_TIMEOUT = int(os.environ.get("GRADE_TIMEOUT", "900"))
OUT = Path(os.environ.get("BENCH_OUT", ROOT / "benchmarks" / "reports"))
DOMAIN = "@edupilot.local"
GATES = ["THREADS_PRECHECK", "INJECTION_FLAG", "CHAT_ROUTING"]
CTX = ssl._create_unverified_context() if re.match(r"https://(localhost|127\.0\.0\.1)", API) else None  # chứng chỉ nội bộ của Caddy (dev)
ORIGIN = re.match(r"(https?://[^/]+)", API).group(1)


def die(msg, code=1):
    print(msg, file=sys.stderr)
    sys.exit(code)


def call(method, path, token=None, body=None, headers=None, raw=False):
    h = {"Accept": "application/json", "Origin": ORIGIN, **(headers or {})}
    if token:
        h["Authorization"] = f"Bearer {token}"
    data = None
    if body is not None:
        data = json.dumps(body).encode()
        h["Content-Type"] = "application/json"
    if method == "POST":
        h.setdefault("Idempotency-Key", str(uuid.uuid4()))
    for attempt in range(6):
        req = urllib.request.Request(API + path, data=data, method=method, headers=h)
        try:
            with urllib.request.urlopen(req, context=CTX, timeout=120) as r:
                txt = r.read()
                return r.status, (txt if raw else (json.loads(txt) if txt else None))
        except urllib.error.HTTPError as e:
            txt = e.read()
            if e.code == 429 and attempt < 5:
                time.sleep(min(30, float(e.headers.get("Retry-After") or 3)))
                continue
            try:
                return e.code, json.loads(txt)
            except ValueError:
                return e.code, None


def must(status, ok, what):
    if status not in ok:
        die(f"{what} → HTTP {status}")


def pct(xs, q):
    if not xs:
        return None
    xs = sorted(xs)
    return round(xs[min(len(xs) - 1, int(q * len(xs)))], 1)


def lat(xs):
    return {"n": len(xs), "p50_ms": pct(xs, 0.5), "p95_ms": pct(xs, 0.95)}


def jsonl(name):
    return [json.loads(l) for l in (DATA / name).read_text(encoding="utf-8").splitlines() if l.strip()]


# ---- đăng nhập, cấu hình --------------------------------------------------------------------------------------------
_tokens = {}


def token(account):
    if account not in _tokens:
        s, b = call("POST", "/auth/login", body={"email": account + DOMAIN, "password": PASSWORD})
        must(s, (200,), f"đăng nhập {account}")
        _tokens[account] = b["access_token"]
    return _tokens[account]


def course_id():
    s, b = call("GET", "/me/courses?limit=100", token("teacher"))
    must(s, (200,), "GET /me/courses")
    for m in b["items"]:
        if m["course"]["class_code"] == CLASS:
            return m["course"]["id"]
    die(f"không thấy lớp {CLASS} (đã seed chưa?)")


def set_mode(c1, on):
    """Bật / tắt cả ba cổng: công tắc hệ thống + công tắc lớp (API ADMIN, như giao diện)."""
    adm = token("admin")
    s, ov = call("GET", "/admin/llm/decision", adm)
    must(s, (200,), "GET /admin/llm/decision")
    for g in ov["gates"]:
        if g["enabled"] != on:
            s, _ = call("PUT", f"/admin/llm/decision/gates/{g['gate']}", adm, {"enabled": on, "version": g["version"]})
            must(s, (200,), f"PUT cổng {g['gate']}")
    s, cg = call("GET", f"/courses/{c1}/decision-gates", adm)
    must(s, (200,), "GET decision-gates của lớp")
    for g in cg["items"]:
        if g["enabled"] != on:
            s, _ = call("PUT", f"/courses/{c1}/decision-gates/{g['gate']}", adm, {"enabled": on, "version": g["version"]})
            must(s, (200,), f"PUT cổng lớp {g['gate']}")
    time.sleep(1.5)  # snapshot cấu hình ở gateway / worker làm mới (Redis pub/sub)


def obs():
    s, b = call("GET", "/admin/observability/decision?window=24h", token("admin"))
    must(s, (200,), "GET /admin/observability/decision")
    return {g["gate"]: g for g in b["gates"]}


def path_delta(before, after, gate):
    """Số lời gọi FAST / FALLBACK / FAIL_CLOSED của một cổng giữa hai lần đọc (từ calls × %)."""
    def counts(o):
        g = o.get(gate) or {}
        n = g.get("calls", 0)
        return {k: round(n * g.get(f"{k}_pct", 0) / 100) for k in ("fast", "fallback", "fail_closed")}, n
    (b, nb), (a, na) = counts(before), counts(after)
    return {"calls": na - nb, **{k: a[k] - b[k] for k in a}}


# ---- (1) chat ------------------------------------------------------------------------------------------------------
def ask(sv, c1, text):
    s, sess = call("POST", "/chat/sessions", sv, {"course_id": c1})
    must(s, (200, 201), "tạo phiên chat")
    h = {"Accept": "text/event-stream", "Authorization": f"Bearer {sv}", "Content-Type": "application/json", "Origin": ORIGIN, "Idempotency-Key": str(uuid.uuid4())}
    req = urllib.request.Request(f"{API}/chat/sessions/{sess['id']}/messages", data=json.dumps({"content": text}).encode(), method="POST", headers=h)
    t0 = time.perf_counter()
    first = tok = None
    stages, ev = set(), ""
    try:
        with urllib.request.urlopen(req, context=CTX, timeout=120) as r:
            for line in r:
                line = line.decode().rstrip("\n")
                if line.startswith("event:"):
                    ev = line[6:].strip()
                elif line.startswith("data:"):
                    now = (time.perf_counter() - t0) * 1000
                    first = first if first is not None else now
                    if ev == "token" and tok is None:
                        tok = now
                    if ev == "status":
                        stages.add(json.loads(line[5:]).get("stage"))
    except urllib.error.HTTPError as e:
        return {"error": e.code}
    s, msgs = call("GET", f"/chat/sessions/{sess['id']}/messages?limit=20", sv)
    kinds = []
    for m in (msgs or {}).get("items", []):
        if m["role"] == "ASSISTANT":
            kinds = [b.get("kind") for b in m.get("blocks", [])]
    call("DELETE", f"/chat/sessions/{sess['id']}", sv)
    # câu trả lời từ cache không có giai đoạn «searching» và không có block: không đại diện cho đường cần đo → loại khỏi TTFT
    cached = "searching" not in stages and not kinds
    return {"first_ms": first, "ttft_ms": tok, "kinds": kinds, "cached": cached}


def scenario_chat(c1):
    sv, rows = token("sv.gioi"), jsonl("chat.jsonl")
    res = []
    for rnd in range(ROUNDS):
        for q in rows:
            r = ask(sv, c1, q["text"])
            r.update(stratum=q["stratum"], expect=q["expect_block"], round=rnd)
            res.append(r)
    ok = [r for r in res if "error" not in r]
    correct = lambda r: (r["expect"] is None and not r["kinds"]) or (r["expect"] is not None and r["expect"] in r["kinds"])
    live = [r for r in ok if not r["cached"] and r["ttft_ms"] is not None]
    by = {}
    for st in ("rules", "course_qa", "paraphrase"):
        xs = [r for r in ok if r["stratum"] == st]
        by[st] = {"n": len(xs), "accuracy": round(sum(map(correct, xs)) / len(xs), 3) if xs else None}
    return {"requests": len(res), "errors": len(res) - len(ok), "cached_excluded": len(ok) - len(live),
            "ttft": lat([r["ttft_ms"] for r in live]), "first_event": lat([r["first_ms"] for r in ok if r["first_ms"] is not None]),
            "accuracy": round(sum(map(correct, ok)) / len(ok), 3) if ok else None, "by_stratum": by}


# ---- (2) Threads ---------------------------------------------------------------------------------------------------
THREAD_ACCOUNTS = [f"sv{n:02d}" for n in range(4, 14)]


def scenario_threads(c1, tag):
    rows, res = jsonl("threads.jsonl"), []
    for i, r in enumerate(rows):
        sv = token(THREAD_ACCOUNTS[i % len(THREAD_ACCOUNTS)])
        t0 = time.perf_counter()
        s, b = call("POST", f"/courses/{c1}/threads", sv, {"title": f"{r['title']} ({tag})", "body": r["body"]})
        ms = (time.perf_counter() - t0) * 1000
        blocked = s == 422
        if s not in (201, 422):
            die(f"đăng bài Threads → HTTP {s} {json.dumps(b, ensure_ascii=False)[:200]}")
        res.append({"stratum": r["stratum"], "blocked": blocked, "ms": ms, "expect": r["expect_block"]})
    clean = [r for r in res if r["stratum"] == "clean"]
    bad = [r for r in res if r["expect"]]
    return {"posts": len(res), "latency": lat([r["ms"] for r in res]),
            "false_block": sum(r["blocked"] for r in clean), "false_block_rate": round(sum(r["blocked"] for r in clean) / len(clean), 3),
            "caught": sum(r["blocked"] for r in bad), "missed": sum(not r["blocked"] for r in bad),
            "caught_by_stratum": {st: sum(r["blocked"] for r in res if r["stratum"] == st) for st in ("private", "rules")}}


# ---- (3) chấm bài: chèn lệnh ---------------------------------------------------------------------------------------
STUDENTS = ["sv.gioi", "sv.kha", "sv.nguyco"] + [f"sv{n:02d}" for n in range(4, 13)]


def strip_comments(s):
    return re.sub(r"<!--.*?-->", "", s, flags=re.S)


def grading_corpus():
    inj = sorted((ROOT / "benchmarks/real-llm/data/injection").glob("*.md"))
    fx = sorted((ROOT / "docs/specs/FEAT-assignments/fixtures/submissions").glob("*.md"))
    rows = [(p.stem, p.read_text(encoding="utf-8"), True) for p in inj] + [(p.stem, p.read_text(encoding="utf-8"), p.stem == "s6-injection") for p in fx]
    assert len(rows) == 12, len(rows)
    return rows


def put_blob(pre, data):
    h = {k: v for k, v in (pre.get("headers") or {}).items() if k.lower() != "content-length"}
    req = urllib.request.Request(pre["url"], data=data, method=pre["method"], headers=h)
    with urllib.request.urlopen(req, context=CTX, timeout=60):
        pass


def scenario_grading(c1, tag):
    t = token("teacher")
    body = json.loads((ROOT / "docs/specs/FEAT-assignments/fixtures/create-bt03.json").read_text(encoding="utf-8"))
    body["description"], body["reference_answer"] = strip_comments(body["description"]), strip_comments(body["reference_answer"])
    if isinstance(body.get("rubric"), list):
        body["rubric"] = [{**c, "name": strip_comments(c["name"]), "guidance": strip_comments(c["guidance"])} for c in body["rubric"]]
    body.update(submission_code=f"DM{tag}", channels=["IN_APP"], title=f"Đo decision {tag}",
                due_at=time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time() + 3600)))
    s, a = call("POST", f"/courses/{c1}/assignments", t, body)
    must(s, (200, 201), "tạo bài tập đo")
    s, _ = call("POST", f"/courses/{c1}/assignments/{a['id']}/publish", t)
    must(s, (200, 202), "công bố bài tập đo")
    corpus = grading_corpus()
    expect = {}
    for acct, (name, text, yes) in zip(STUDENTS, corpus):
        sv, data = token(acct), text.encode()
        s, pre = call("POST", f"/courses/{c1}/assignments/{a['id']}/submit/presign", sv,
                      {"filename": "bai-lam.md", "size_bytes": len(data), "sha256": hashlib.sha256(data).hexdigest()})
        must(s, (200, 201), f"presign {acct}")
        put_blob(pre, data)
        s, _ = call("POST", f"/courses/{c1}/assignments/{a['id']}/submit", sv, {"upload_id": pre["upload_id"]})
        must(s, (200, 201, 202), f"nộp bài {acct}")
        expect[acct] = (name, yes)
    t0 = time.perf_counter()
    s, g = call("POST", f"/courses/{c1}/assignments/{a['id']}/grade-all", t, {})
    must(s, (200, 202), "grade-all")
    rows, deadline = [], time.time() + GRADE_TIMEOUT
    while time.time() < deadline:
        rows = []
        for st in ("DRAFT", "REVIEWED", "PUBLISHED", "FAILED"):
            s, b = call("GET", f"/courses/{c1}/submissions?assignment_id={a['id']}&status={st}&limit=100", t)
            rows += (b or {}).get("items", [])
        if len(rows) >= len(corpus):
            break
        time.sleep(3)
    wall = time.perf_counter() - t0
    flagged = {}
    for r in rows:
        flags = [f.get("code", f) if isinstance(f, dict) else f for f in ((r.get("result") or {}).get("flags") or [])]
        flagged[r["student"]["id"]] = "INJECTION_SUSPECTED" in flags
    # student.id của hàng nộp = id người dùng: lấy bằng GET /me của chính tài khoản
    ids = {}
    for acct in STUDENTS:
        s, me = call("GET", "/me", token(acct))
        ids[acct] = me["id"]
    tp = sum(flagged.get(ids[a_], False) for a_, (_, y) in expect.items() if y)
    fp = sum(flagged.get(ids[a_], False) for a_, (_, y) in expect.items() if not y)
    return {"graded": len(rows), "of": len(corpus), "flag_correct": tp, "flag_expected": sum(y for _, y in expect.values()), "flag_false": fp,
            "controls": sum(not y for _, y in expect.values()), "wall_s": round(wall, 1), "avg_s_per_essay": round(wall / max(1, len(rows)), 1)}


# ---- báo cáo -------------------------------------------------------------------------------------------------------
def verdicts(off, on):
    v = {}
    c0, c1 = off["chat"], on["chat"]
    ratio = (c1["ttft"]["p95_ms"] or 0) / (c0["ttft"]["p95_ms"] or 1)
    ok = ratio <= 1.2 and (c1["accuracy"] or 0) >= (c0["accuracy"] or 0)
    v["CHAT_ROUTING"] = ("Bật được" if ok else "Giữ tắt", f"TTFT p95 ×{ratio:.2f} (≤ 1,20), độ chính xác {c0['accuracy']} → {c1['accuracy']}")
    t0, t1 = off["threads"], on["threads"]
    ok = t1["false_block_rate"] <= 0.05 and t1["missed"] <= t0["missed"]
    v["THREADS_PRECHECK"] = ("Bật được" if ok else "Giữ tắt", f"chặn nhầm {t1['false_block_rate']} (≤ 0,05), lọt {t0['missed']} → {t1['missed']}")
    g1 = on["grading"]
    ok = g1["flag_correct"] == g1["flag_expected"] and g1["flag_false"] == 0
    v["INJECTION_FLAG"] = ("Bật được" if ok else "Giữ tắt", f"cờ đúng {g1['flag_correct']}/{g1['flag_expected']}, cờ nhầm {g1['flag_false']}/{g1['controls']}")
    return v


def svg(off, on):
    bars = [("Chat TTFT p95 (ms)", off["chat"]["ttft"]["p95_ms"], on["chat"]["ttft"]["p95_ms"]),
            ("Threads đăng p95 (ms)", off["threads"]["latency"]["p95_ms"], on["threads"]["latency"]["p95_ms"]),
            ("Chấm 12 bài (s)", off["grading"]["wall_s"], on["grading"]["wall_s"])]
    h, y, parts = 40 + 70 * len(bars), 30, []
    for name, a, b in bars:
        m = max(a or 1, b or 1, 1)
        for i, (lab, v, col) in enumerate((("tắt", a, "var(--muted)"), ("bật", b, "var(--accent)"))):
            w = 300 * (v or 0) / m
            parts.append(f'<rect x="170" y="{y + i * 22}" width="{w:.0f}" height="18" fill="{col}"/><text x="{176 + w:.0f}" y="{y + i * 22 + 13}" font-size="12" fill="currentColor">{lab} {v}</text>')
        parts.append(f'<text x="8" y="{y + 24}" font-size="13" fill="currentColor">{name}</text>')
        y += 70
    return f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 620 {h}" font-family="sans-serif">' + "".join(parts) + "</svg>\n"


def markdown(off, on, v, paths, meta):
    L = ["# Decision model: đo trước / sau", "",
         f"Cấu hình: MacBook Apple Silicon, Colima 6 CPU / 10 GB · provider decision GIẢ (p50 {meta['fake_p50']} / p95 {meta['fake_p95']} ms) · lượt chat {ROUNDS} · {meta['when']} · lớp {CLASS}.", "",
         "## Chat (30 câu × lượt)", "", "| | tắt | bật |", "|---|---|---|",
         f"| TTFT p50 / p95 (ms) | {off['chat']['ttft']['p50_ms']} / {off['chat']['ttft']['p95_ms']} | {on['chat']['ttft']['p50_ms']} / {on['chat']['ttft']['p95_ms']} |",
         f"| Sự kiện đầu p50 / p95 (ms) | {off['chat']['first_event']['p50_ms']} / {off['chat']['first_event']['p95_ms']} | {on['chat']['first_event']['p50_ms']} / {on['chat']['first_event']['p95_ms']} |",
         f"| Độ chính xác (rules / course_qa / paraphrase) | {off['chat']['accuracy']} ({' / '.join(str(off['chat']['by_stratum'][k]['accuracy']) for k in ('rules', 'course_qa', 'paraphrase'))}) | {on['chat']['accuracy']} ({' / '.join(str(on['chat']['by_stratum'][k]['accuracy']) for k in ('rules', 'course_qa', 'paraphrase'))}) |",
         f"| Loại khỏi TTFT (trả từ cache) / lỗi | {off['chat']['cached_excluded']} / {off['chat']['errors']} | {on['chat']['cached_excluded']} / {on['chat']['errors']} |",
         f"| FAST / FALLBACK / FAIL_CLOSED | — | {paths['CHAT_ROUTING']['fast']} / {paths['CHAT_ROUTING']['fallback']} / {paths['CHAT_ROUTING']['fail_closed']} |",
         "", "## Threads (40 bài)", "", "| | tắt | bật |", "|---|---|---|",
         f"| Đăng p50 / p95 (ms) | {off['threads']['latency']['p50_ms']} / {off['threads']['latency']['p95_ms']} | {on['threads']['latency']['p50_ms']} / {on['threads']['latency']['p95_ms']} |",
         f"| Chặn nhầm (trên 20 bài sạch) | {off['threads']['false_block']} ({off['threads']['false_block_rate']}) | {on['threads']['false_block']} ({on['threads']['false_block_rate']}) |",
         f"| Chặn đúng / lọt (20 bài cần chặn) | {off['threads']['caught']} / {off['threads']['missed']} | {on['threads']['caught']} / {on['threads']['missed']} |",
         f"| FAST / FALLBACK / FAIL_CLOSED | — | {paths['THREADS_PRECHECK']['fast']} / {paths['THREADS_PRECHECK']['fallback']} / {paths['THREADS_PRECHECK']['fail_closed']} |",
         "", "## Chấm 12 bài (6 biến thể chèn lệnh + 6 đối chứng)", "", "| | tắt | bật |", "|---|---|---|",
         f"| Cờ đúng / cờ nhầm | {off['grading']['flag_correct']}/{off['grading']['flag_expected']} · {off['grading']['flag_false']}/{off['grading']['controls']} | {on['grading']['flag_correct']}/{on['grading']['flag_expected']} · {on['grading']['flag_false']}/{on['grading']['controls']} |",
         f"| Thời gian chấm cả lô (s) / mỗi bài | {off['grading']['wall_s']} / {off['grading']['avg_s_per_essay']} | {on['grading']['wall_s']} / {on['grading']['avg_s_per_essay']} |",
         f"| FAST / FALLBACK / FAIL_CLOSED | — | {paths['INJECTION_FLAG']['fast']} / {paths['INJECTION_FLAG']['fallback']} / {paths['INJECTION_FLAG']['fail_closed']} |",
         "", "## Kết luận từng cổng", "", "| Cổng | Kết luận | Số liệu |", "|---|---|---|"]
    L += [f"| {g} | **{k}** | {why} |" for g, (k, why) in v.items()]
    L += ["", "## Giới hạn", "",
          "- Độ trễ decision là mẫu log-chuẩn của provider giả (p50 / p95 ở trên), không phải trễ mạng từ Việt Nam; nhãn trả theo bản phát lại `llm/testdata/decision-replay.json` dựng từ chính bộ nhãn này (mô phỏng, không phải độ chính xác của Gemini).",
          "- Độ chính xác chat đo bằng block dữ liệu của trợ lý (điểm danh / điểm / cộng điểm) trong câu trả lời, không đọc nhãn ý định nội bộ.",
          "- Câu trả lời trả từ cache bị loại khỏi TTFT; cache là theo lớp nên lượt «bật» có thể có nhiều câu cache hơn lượt «tắt» (xem hàng «Loại khỏi TTFT»).",
          "- Cột «thật» (AC6): chờ phép PM."]
    return "\n".join(L) + "\n"


def main():
    if not PASSWORD:
        die("Thiếu SEED_DEFAULT_PASSWORD.")
    try:
        urllib.request.urlopen(urllib.request.Request(ORIGIN + "/healthz"), context=CTX, timeout=5)
    except urllib.error.HTTPError:
        pass  # có phản hồi HTTP = gateway đang chạy
    except OSError:
        die(f"Gateway chưa chạy ({ORIGIN}). Dựng stack và seed trước (pnpm dev).", 2)
    c1 = course_id()
    s, ov = call("GET", "/admin/llm/decision", token("admin"))
    must(s, (200,), "GET /admin/llm/decision")
    prov = ov.get("provider") or {}
    if not prov.get("fake"):
        die("Provider decision không phải bản giả: lượt thật cần phép PM (AC6). Dừng.")
    tag = time.strftime("%m%d%H%M%S")
    results, paths = {}, {}
    for mode in ("off", "on"):
        set_mode(c1, mode == "on")
        before = obs()
        print(f"[{mode}] chat…", flush=True)
        chat = scenario_chat(c1)
        print(f"[{mode}] threads…", flush=True)
        threads = scenario_threads(c1, f"{mode}-{tag}")
        print(f"[{mode}] chấm bài…", flush=True)
        grading = scenario_grading(c1, f"{mode.upper()}{tag}")
        results[mode] = {"chat": chat, "threads": threads, "grading": grading}
        if mode == "on":
            after = obs()
            paths = {g: path_delta(before, after, g) for g in GATES}
    set_mode(c1, False)  # trả stack về trạng thái ban đầu của seed mặc định
    v = verdicts(results["off"], results["on"])
    meta = {"when": time.strftime("%Y-%m-%d %H:%M"), "fake_p50": os.environ.get("LLM_DECISION_FAKE_P50_MS", "426"), "fake_p95": os.environ.get("LLM_DECISION_FAKE_P95_MS", "682")}
    OUT.mkdir(parents=True, exist_ok=True)
    (OUT / "decision-before-after.json").write_text(json.dumps({"meta": meta, **results, "paths_on": paths, "verdict": {g: {"result": k, "why": why} for g, (k, why) in v.items()}}, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    (OUT / "decision-before-after.md").write_text(markdown(results["off"], results["on"], v, paths, meta), encoding="utf-8")
    (OUT / "decision-before-after.svg").write_text(svg(results["off"], results["on"]), encoding="utf-8")
    for g, (k, why) in v.items():
        print(f"{g}: {k} — {why}")
    print(f"Báo cáo: {OUT}/decision-before-after.{{json,md,svg}}")


if __name__ == "__main__":
    main()

"""Dùng chung cho `make eval` (US-P10-10, FR-20): client NGOÀI, chỉ thư viện chuẩn, chỉ gọi HTTP API của gateway (D46, AC2).

Ngoại lệ có ghi rõ trong báo cáo: `sql()` đọc Postgres của stack đo trong giao dịch READ ONLY cho những số API không trả
(cos_top1 của truy xuất cho P-11-19, `llm_audit` cho độ trễ / token / số lời gọi nhúng, quét rò rỉ `llm_request_payloads`).
Không có khoá API nào đi qua tệp này.
"""
import json
import math
import os
import re
import shlex
import ssl
import subprocess
import sys
import time
import urllib.error
import urllib.request
import uuid
from datetime import datetime, timezone
from pathlib import Path
from xml.sax.saxutils import escape

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[1]
API = os.environ.get("API_URL", "https://localhost:943/api/v1").rstrip("/")
PASSWORD = os.environ.get("SEED_DEFAULT_PASSWORD", "")
DOMAIN = "@edupilot.local"
CLASS1 = "761987"  # lớp 1 của seed T0
MACHINE = "MacBook Apple Silicon 18 GB, Colima 6 CPU / 10 GB, k6 trong VM 1,5 CPU"  # = benchmarks/lib/meta.mjs
PSQL = os.environ.get("EVAL_PSQL", "docker exec -i ep-qc-postgres-1 psql -U edupilot -d edupilot -tA -v ON_ERROR_STOP=1")
OUT = Path(os.environ.get("EVAL_OUT", ROOT / "benchmarks/reports" / f"eval-{os.environ.get('EVAL_DATE') or datetime.now().strftime('%Y-%m-%d')}"))
RAW = OUT / "raw"
CTX = ssl.create_default_context()
if re.match(r"https://(localhost|127\.0\.0\.1)", API):
    CTX.check_hostname, CTX.verify_mode = False, ssl.CERT_NONE  # chứng chỉ nội bộ của Caddy
ORIGIN = re.match(r"(https?://[^/]+)", API).group(1)


class Fail(Exception):
    """Lỗi của bộ đo (không phải kết quả đo)."""


class Stop(Exception):
    """Nhà cung cấp bắt chờ quá lâu (429 Retry-After > 60 s): dừng lượt nhẹ nhàng, ghi lý do (AC6, AC10)."""


def refuse_demo():
    port = re.match(r"https?://[^/:]+:?(\d*)", API).group(1)
    if port in ("", "80", "443", "3000"):
        raise Fail(f"Từ chối: {API} là stack demo. Dùng stack -p ep-qc (cổng lệch).")


def ready():
    """AC12: gateway phải trả /readyz 200, không thì thoát ≠ 0 với một dòng «Gateway chưa chạy»."""
    try:
        with urllib.request.urlopen(urllib.request.Request(API + "/readyz"), context=CTX, timeout=5) as r:
            return r.status == 200
    except (urllib.error.URLError, OSError):
        return False


def call(method, path, token=None, body=None, idem=None, ok=(200, 201, 202, 204), tries=6, timeout=120):
    """→ (status, json). 429 / 503 lùi theo Retry-After (tối đa `tries` lần); Retry-After > 60 s → Stop. Mã ngoài `ok` → (mã, thân) cho bên gọi xử lý."""
    h = {"Accept": "application/json", "Origin": ORIGIN}
    if token:
        h["Authorization"] = "Bearer " + token
    if idem:
        h["Idempotency-Key"] = re.sub(r"[^A-Za-z0-9._:-]", "-", "eval-" + idem)[:120]  # 8–128 ký tự
    data = None
    if body is not None:
        data, h["Content-Type"] = json.dumps(body).encode(), "application/json"
    for i in range(tries):
        try:
            with urllib.request.urlopen(urllib.request.Request(API + path, data=data, headers=h, method=method), context=CTX, timeout=timeout) as r:
                b = r.read()
                return r.status, (json.loads(b) if b else None)
        except urllib.error.HTTPError as e:
            b = e.read()
            try:
                j = json.loads(b) if b else None
            except ValueError:
                j = {"raw": b[:200].decode("utf8", "replace")}
            if e.code in (429, 503) and i < tries - 1:
                ra = float(e.headers.get("Retry-After") or 0) or 2.0 * (2 ** i)
                if ra > 60:
                    raise Stop(f"{method} {path}: {e.code} Retry-After {ra:.0f}s") from None
                time.sleep(ra + 0.5)
                continue
            return e.code, j
        except (urllib.error.URLError, OSError) as e:
            raise Fail(f"Gateway chưa chạy ({API}): {e}") from None
    raise Fail(f"{method} {path}: hết {tries} lượt thử (429/503)")


_tokens = {}


def login(who):
    """Tài khoản seed `who@edupilot.local` → access token (cache; token 15 phút nên bên gọi dài gọi `relogin`)."""
    if who not in _tokens:
        if not PASSWORD:
            raise Fail("thiếu SEED_DEFAULT_PASSWORD")
        st, b = call("POST", "/auth/login", body={"email": who + DOMAIN, "password": PASSWORD})
        if st != 200:
            raise Fail(f"đăng nhập {who} → HTTP {st}")
        _tokens[who] = (b["access_token"], time.time())
    tok, at = _tokens[who]
    if time.time() - at > 600:
        _tokens.pop(who)
        return login(who)
    return tok


def list_all(path, token):
    out, cur = [], ""
    for _ in range(100):
        sep = "&" if "?" in path else "?"
        st, b = call("GET", f"{path}{sep}limit=100" + (f"&cursor={cur}" if cur else ""), token)
        if st != 200:
            raise Fail(f"GET {path} → {st} {b}")
        out += b["items"]
        cur = b.get("next_cursor") or ""
        if not cur:
            return out
    raise Fail(f"{path}: quá dài")


def course_by_code(token, code):
    for m in list_all("/me/courses", token):
        if m["course"]["class_code"] == code:
            return m["course"]["id"]
    raise Fail(f"tài khoản không thuộc lớp {code}")


def wait_job(token, job_id, timeout=1800):
    end = time.time() + timeout
    while time.time() < end:
        st, j = call("GET", f"/jobs/{job_id}", token)
        if st == 200 and j["status"] in ("SUCCEEDED", "FAILED", "CANCELLED"):
            return j
        time.sleep(2)
    raise Fail(f"việc {job_id} quá {timeout}s")


def new_key(prefix):
    return f"{prefix}-{uuid.uuid4().hex[:12]}"


# ---- đọc DB chỉ-đọc ---------------------------------------------------------------------------------------------------------
def sql(q):
    wrapped = f"begin read only; select coalesce(jsonb_agg(t), '[]'::jsonb) from ({q}) t; rollback;"
    r = subprocess.run(shlex.split(PSQL), input=wrapped, capture_output=True, text=True, timeout=900)
    if r.returncode != 0:
        raise Fail(f"EVAL_PSQL lỗi: {r.stderr[:300]}")
    return json.loads(next((x for x in r.stdout.splitlines() if x.startswith("[")), "[]"))


def lit(s):
    return "'" + str(s).replace("'", "''") + "'"


def audit(t0, t1, task=None, course=None):
    """Dòng llm_audit trong cửa sổ [t0, t1] (ISO UTC), không nội dung."""
    w = [f"created_at between {lit(t0)} and {lit(t1)}"]
    if task:
        w.append(f"task = {lit(task)}")
    if course:
        w.append(f"course_id = {lit(course)}")
    return sql(f"""select task, lane, provider, model, status, attempts, fallback_index, latency_ms, queue_wait_ms, tokens_in, tokens_out, created_at
        from llm_audit where {' and '.join(w)} order by created_at""")


def now_iso():
    return datetime.now(timezone.utc).isoformat(timespec="milliseconds").replace("+00:00", "Z")


# ---- số --------------------------------------------------------------------------------------------------------------------
def pct(xs):
    xs = sorted(xs)
    if not xs:
        return {"n": 0}
    q = lambda p: xs[min(len(xs) - 1, max(0, math.ceil(p * len(xs)) - 1))]  # noqa: E731 — phân vị "nearest rank"
    return {"n": len(xs), "p50": q(0.5), "p95": q(0.95), "max": xs[-1], "mean": round(sum(xs) / len(xs), 1)}


# ---- ghi báo cáo -------------------------------------------------------------------------------------------------------------
def git_commit():
    try:
        return subprocess.run(["git", "rev-parse", "--short", "HEAD"], capture_output=True, text=True, cwd=ROOT).stdout.strip() or "unknown"
    except OSError:
        return "unknown"


def meta(config, runs=1):
    return {"commit": git_commit(), "timestamp": now_iso(), "machine": MACHINE, "runs": runs, "config": config}


def save_raw(name, obj):
    """Kết quả thô; tệp nào cũng có khối `meta` (benchmarks/check-meta.mjs kiểm mọi json dưới reports/)."""
    RAW.mkdir(parents=True, exist_ok=True)
    if isinstance(obj, dict) and "meta" not in obj:
        obj = {"meta": meta({"stack": "ep-qc", "raw": name}), **obj}
    (RAW / name).write_text(json.dumps(obj, ensure_ascii=False, indent=1, default=str) + "\n", encoding="utf-8")


def load_raw(name):
    p = RAW / name
    return json.loads(p.read_text(encoding="utf-8")) if p.exists() else None


def write_report(name, obj, md_lines):
    OUT.mkdir(parents=True, exist_ok=True)
    (OUT / f"{name}.json").write_text(json.dumps(obj, ensure_ascii=False, indent=2, default=str) + "\n", encoding="utf-8")
    (OUT / f"{name}.md").write_text("\n".join(md_lines) + "\n", encoding="utf-8")
    print(f"→ {OUT / name}.json, .md")


# ---- SVG bằng thư viện chuẩn (P-11-2) -----------------------------------------------------------------------------------------
COLORS = ["#b3261e", "#1f5aa6", "#2e7d32", "#8a6d00", "#6a1b9a", "#455a64"]


def _frame(w, h, title, xlabel, ylabel):
    return [f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {w} {h}" width="{w}" height="{h}" font-family="sans-serif" font-size="12">',
            f'<text x="{w / 2}" y="18" text-anchor="middle" font-size="14">{escape(title)}</text>',
            f'<text x="{w / 2}" y="{h - 6}" text-anchor="middle">{escape(xlabel)}</text>',
            f'<text x="14" y="{h / 2}" text-anchor="middle" transform="rotate(-90 14 {h / 2})">{escape(ylabel)}</text>']


def svg_lines(path, series, title, xlabel, ylabel, xmin, xmax, ymin=0.0, ymax=1.0, xticks=None, vline=None):
    """series: [(nhãn, [(x, y)…])]. Trục tuyến tính, lưới 5 vạch, chú giải góc phải trên."""
    w, h, l, r, t, b = 720, 420, 60, 20, 30, 50
    X = lambda x: l + (x - xmin) / (xmax - xmin) * (w - l - r)  # noqa: E731
    Y = lambda y: h - b - (y - ymin) / (ymax - ymin) * (h - t - b)  # noqa: E731
    s = _frame(w, h, title, xlabel, ylabel)
    for i in range(6):
        yv = ymin + (ymax - ymin) * i / 5
        s.append(f'<line x1="{l}" y1="{Y(yv):.1f}" x2="{w - r}" y2="{Y(yv):.1f}" stroke="#ddd"/><text x="{l - 6}" y="{Y(yv) + 4:.1f}" text-anchor="end">{yv:.2f}</text>')
    for xv in xticks or [xmin + (xmax - xmin) * i / 6 for i in range(7)]:
        s.append(f'<line x1="{X(xv):.1f}" y1="{h - b}" x2="{X(xv):.1f}" y2="{h - b + 4}" stroke="#333"/><text x="{X(xv):.1f}" y="{h - b + 18}" text-anchor="middle">{xv:.2f}</text>')
    s.append(f'<line x1="{l}" y1="{h - b}" x2="{w - r}" y2="{h - b}" stroke="#333"/><line x1="{l}" y1="{t}" x2="{l}" y2="{h - b}" stroke="#333"/>')
    if vline is not None:
        s.append(f'<line x1="{X(vline[0]):.1f}" y1="{t}" x2="{X(vline[0]):.1f}" y2="{h - b}" stroke="#888" stroke-dasharray="4 3"/><text x="{X(vline[0]) + 4:.1f}" y="{t + 12}">{escape(vline[1])}</text>')
    for i, (name, pts) in enumerate(series):
        c = COLORS[i % len(COLORS)]
        s.append(f'<polyline fill="none" stroke="{c}" stroke-width="2" points="' + " ".join(f"{X(x):.1f},{Y(y):.1f}" for x, y in pts) + '"/>')
        s += [f'<circle cx="{X(x):.1f}" cy="{Y(y):.1f}" r="2.5" fill="{c}"/>' for x, y in pts]
        s.append(f'<rect x="{w - r - 230}" y="{t + 6 + i * 18}" width="12" height="3" fill="{c}"/><text x="{w - r - 212}" y="{t + 11 + i * 18}">{escape(name)}</text>')
    (OUT / path).write_text("\n".join(s + ["</svg>"]) + "\n", encoding="utf-8")


def svg_bars(path, groups, series_names, title, ylabel, ymax=None, fmt="{:.2f}"):
    """groups: [(nhãn nhóm, [giá trị theo series…])]; cột nhóm, giá trị in trên đầu cột. None = không có số (vẽ chữ «–»)."""
    w, h, l, r, t, b = 720, 420, 60, 20, 30, 70
    vals = [v for _, vs in groups for v in vs if v is not None]
    ymax = ymax or (max(vals) * 1.15 if vals else 1.0) or 1.0
    s = _frame(w, h, title, "", ylabel)
    Y = lambda y: h - b - y / ymax * (h - t - b)  # noqa: E731
    for i in range(6):
        yv = ymax * i / 5
        s.append(f'<line x1="{l}" y1="{Y(yv):.1f}" x2="{w - r}" y2="{Y(yv):.1f}" stroke="#ddd"/><text x="{l - 6}" y="{Y(yv) + 4:.1f}" text-anchor="end">{fmt.format(yv)}</text>')
    gw = (w - l - r) / max(1, len(groups))
    bw = gw * 0.8 / max(1, len(series_names))
    for gi, (gname, vs) in enumerate(groups):
        x0 = l + gi * gw + gw * 0.1
        for si, v in enumerate(vs):
            x = x0 + si * bw
            if v is None:
                s.append(f'<text x="{x + bw / 2:.1f}" y="{h - b - 4}" text-anchor="middle">–</text>')
                continue
            s.append(f'<rect x="{x:.1f}" y="{Y(v):.1f}" width="{bw - 2:.1f}" height="{h - b - Y(v):.1f}" fill="{COLORS[si % len(COLORS)]}"/>'
                     f'<text x="{x + bw / 2:.1f}" y="{Y(v) - 3:.1f}" text-anchor="middle" font-size="10">{fmt.format(v)}</text>')
        s.append(f'<text x="{l + gi * gw + gw / 2:.1f}" y="{h - b + 16}" text-anchor="middle" font-size="{10 if len(gname) * 6 > gw else 12}">{escape(gname)}</text>')
    s.append(f'<line x1="{l}" y1="{h - b}" x2="{w - r}" y2="{h - b}" stroke="#333"/>')
    for si, n in enumerate(series_names):
        s.append(f'<rect x="{l + si * 170}" y="{h - 26}" width="12" height="10" fill="{COLORS[si % len(COLORS)]}"/><text x="{l + si * 170 + 16}" y="{h - 17}">{escape(n)}</text>')
    (OUT / path).write_text("\n".join(s + ["</svg>"]) + "\n", encoding="utf-8")


def die(msg, code=1):
    print(msg, file=sys.stderr)
    sys.exit(code)

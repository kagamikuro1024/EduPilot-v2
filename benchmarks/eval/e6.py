#!/usr/bin/env python3
"""E6 — nền Go (US-P10-10 AC8): RAM nghỉ, khởi động lạnh, kích thước image, p95 ≥ 5 endpoint. Không gọi LLM. Không đối chiếu hệ cũ (D45).

  e6.py run --tag r1    đo một lượt → raw/e6-<tag>.json  (AC11: chạy 2 lượt, sai số ghi ở báo cáo; Q-S11-13: ±15 % RAM / khởi động)
  e6.py report          → e6.json / e6.md / e6.svg
Stack đo `ep-qc` (docker compose -p ep-qc). Khởi động lạnh = `docker start` → `/api/v1/readyz` 200 (gateway, thăm dò từ caddy cùng mạng) /
`/worker -healthcheck` 0 (worker), mỗi tiến trình 3 lần.
"""
import argparse
import json
import re
import subprocess
import time

import lib
from lib import Fail, call, login

P = "ep-qc"
SERVICES = ["gateway", "worker", "postgres", "pgbouncer", "redis", "caddy", "frontend", "minio", "docling", "judge"]
N_REQ, N_WARM = 200, 20


def sh(*cmd, check=True):
    r = subprocess.run(cmd, capture_output=True, text=True, timeout=600)
    if check and r.returncode:
        raise Fail(f"{' '.join(cmd[:4])}…: {r.stderr[:200]}")
    return r.stdout


def mib(s):
    m = re.match(r"([\d.]+)\s*([KMG]i?B|B)", s)
    v, u = float(m.group(1)), m.group(2)
    return round(v * {"B": 1 / 2**20, "KiB": 1 / 1024, "KB": 1 / 1024, "MiB": 1, "MB": 1, "GiB": 1024, "GB": 1024}[u], 1)


def idle_ram():
    time.sleep(60)  # nghỉ 60 s, không yêu cầu nào
    out = {}
    for line in sh("docker", "stats", "--no-stream", "--format", "{{.Name}}\t{{.MemUsage}}\t{{.CPUPerc}}").splitlines():
        name, mem, cpu = line.split("\t")
        m = re.match(rf"{P}-(.+)-\d+$", name)
        if m and m.group(1) in SERVICES:
            out[m.group(1)] = {"mem_mib": mib(mem.split("/")[0].strip()), "cpu": cpu}
    return out


def cold_start(svc, probe, n=3):
    times = []
    for _ in range(n):
        sh("docker", "stop", f"{P}-{svc}-1")
        t0 = time.perf_counter()
        sh("docker", "start", f"{P}-{svc}-1")
        end = time.time() + 120
        while time.time() < end:
            if subprocess.run(probe, capture_output=True, timeout=10).returncode == 0:
                break
            time.sleep(0.05)
        else:
            raise Fail(f"{svc} không sẵn sàng sau 120 s")
        times.append(round((time.perf_counter() - t0) * 1000))
        time.sleep(5)
    return times


def image_sizes():
    out = {}
    for svc in ("gateway", "worker"):
        img = sh("docker", "inspect", f"{P}-{svc}-1", "--format", "{{.Config.Image}}").strip()
        out[svc] = {"image": img, "mib": round(int(sh("docker", "image", "inspect", img, "--format", "{{.Size}}").strip()) / 2**20, 1)}
    for tgt in ("gateway", "worker"):  # bản production (target không tag test) dựng từ cùng commit — số này mới là kích thước giao hàng
        tag = f"ep-eval-{tgt}:{lib.git_commit()}"
        sh("docker", "build", "-q", "--target", tgt, "-t", tag, str(lib.ROOT / "backend-go"))
        out[f"{tgt}_production"] = {"image": tag, "mib": round(int(sh("docker", "image", "inspect", tag, "--format", "{{.Size}}").strip()) / 2**20, 1)}
    return out


def latency():
    """Mỗi endpoint một tài khoản (seed chỉ có 2 tài khoản nhân viên lớp 1: teacher, ta → token nào vừa dùng trong 61 s thì nghỉ NGOÀI vùng đo
    cho giới hạn 600 yêu cầu / phút / người hồi lại). Trong vùng đo `tries=1`: 429 đếm riêng và loại khỏi độ trễ, không ngủ Retry-After."""
    cid = lib.course_by_code(login("teacher"), lib.CLASS1)
    eps = [("GET /readyz", "/readyz", None), ("GET /me/courses (SV)", "/me/courses?limit=20", "sv.kha"), ("GET threads lớp 1 (SV)", f"/courses/{cid}/threads?limit=20", "sv.gioi"),
           ("GET documents lớp 1 (GV)", f"/courses/{cid}/documents?limit=20", "teacher"), ("GET students lớp 1 (TA)", f"/courses/{cid}/students?limit=50", "ta"),
           ("GET gradebook lớp 1 (GV)", f"/courses/{cid}/gradebook", "teacher"), ("GET today lớp 1 (TA)", f"/courses/{cid}/today", "ta")]
    out, last = {}, {}
    for name, path, who in eps:
        if who in last:
            time.sleep(max(0.0, 61 - (time.time() - last[who])))
        tok = login(who) if who else None
        if who and call("GET", path, tok, tries=1)[0] != 200:
            raise Fail(f"{name}: tài khoản {who} không đọc được ({path})")
        for _ in range(N_WARM):
            call("GET", path, tok, tries=1)
        ms, bad, limited = [], 0, 0
        for _ in range(N_REQ):
            t = time.perf_counter()
            st, _ = call("GET", path, tok, tries=1)
            if st == 429:
                limited += 1
                continue
            ms.append(round((time.perf_counter() - t) * 1000, 1))
            bad += st != 200
        out[name] = {**lib.pct(ms), "non_200": bad, "rate_limited_excluded": limited, "account": who}
        if who:
            last[who] = time.time()
    return out


def run(tag):
    caddy_probe = ["docker", "exec", f"{P}-caddy-1", "wget", "-q", "-O", "/dev/null", "http://gateway:8080/api/v1/readyz"]
    worker_probe = ["docker", "exec", f"{P}-worker-1", "/worker", "-healthcheck"]
    images = image_sizes() if tag == "r1" else (lib.load_raw("e6-r1.json") or {}).get("images")  # dựng image trước: lỗi thì chưa tốn lượt đo
    raw = {"tag": tag, "idle_ram": idle_ram(), "cold_start_ms": {"gateway": cold_start("gateway", caddy_probe), "worker": cold_start("worker", worker_probe)}}
    time.sleep(10)
    raw["latency_ms"] = latency()
    raw["images"] = images
    raw["colima"] = sh("colima", "list", "--json", check=False).strip()[:400]
    lib.save_raw(f"e6-{tag}.json", raw)
    print(json.dumps({k: raw[k] for k in ("cold_start_ms",)}, ensure_ascii=False))


def report():
    runs = {p.stem[3:]: lib.load_raw(p.name) for p in sorted(lib.RAW.glob("e6-*.json"))}
    if not runs:
        raise Fail("chưa có raw/e6-*.json")
    a = runs[sorted(runs)[0]]
    go = lambda r: round(r["idle_ram"]["gateway"]["mem_mib"] + r["idle_ram"]["worker"]["mem_mib"], 1)  # noqa: E731
    med = lambda xs: sorted(xs)[len(xs) // 2]  # noqa: E731
    repro = None
    if len(runs) >= 2:
        b = runs[sorted(runs)[1]]
        d = lambda x, y: round(abs(x - y) / x * 100, 1) if x else None  # noqa: E731
        repro = {"ram_go_pct": d(go(a), go(b)), "cold_gateway_pct": d(med(a["cold_start_ms"]["gateway"]), med(b["cold_start_ms"]["gateway"])),
                 "cold_worker_pct": d(med(a["cold_start_ms"]["worker"]), med(b["cold_start_ms"]["worker"])), "tolerance_pct": 15}
        repro["within"] = all(v is not None and v <= 15 for k, v in repro.items() if k.endswith("_pct") and k != "tolerance_pct")
    obj = {"meta": lib.meta({"stack": "ep-qc (1 gateway, 1 worker, seed T0)", "requests_per_endpoint": N_REQ, "warmup": N_WARM, "client": "Python urllib tuần tự, qua Caddy HTTPS"}, runs=len(runs)),
           "status": "OK", "runs": runs, "reproducibility": repro}
    lib.svg_bars("e6.svg", [(k.replace(" lớp 1", ""), [v["p50"], v["p95"]]) for k, v in a["latency_ms"].items()], ["p50", "p95"], "E6 — độ trễ endpoint (ms, tuần tự, 200 yêu cầu)", "ms", fmt="{:.0f}")
    f = lambda x: str(x).replace(".", ",")  # noqa: E731
    L = ["# E6 — nền Go", "", f"Trạng thái: **OK**. Stack `ep-qc` @ `{obj['meta']['commit']}`, {obj['meta']['timestamp'][:10]}. Máy: {lib.MACHINE}. {len(runs)} lượt.", "",
         "## RAM khi nghỉ (sau 60 s không yêu cầu, `docker stats`)", "", "| tiến trình | " + " | ".join(sorted(runs)) + " |", "| --- |" + " --- |" * len(runs)]
    for s in SERVICES:
        if s in a["idle_ram"]:
            L.append(f"| {s} | " + " | ".join(f(r['idle_ram'].get(s, {}).get('mem_mib', '–')) + " MiB" for r in runs.values()) + " |")
    L += ["", "## Khởi động lạnh (`docker start` → sẵn sàng, ms, 3 lần)", "", "| tiến trình | " + " | ".join(sorted(runs)) + " |", "| --- |" + " --- |" * len(runs)]
    for s in ("gateway", "worker"):
        L.append(f"| {s} | " + " | ".join(", ".join(map(str, r["cold_start_ms"][s])) for r in runs.values()) + " |")
    L += ["", "## Kích thước image", "", "| image | MiB |", "| --- | --- |"] + [f"| `{v['image']}` | {f(v['mib'])} |" for v in (a.get("images") or {}).values()]
    rs = sorted(runs)
    L += ["", "## Độ trễ endpoint (ms, " + " / ".join(rs) + ")", "", "| endpoint | n | p50 | p95 | max | ≠ 200 | 429 (đã loại) |", "| --- | --- | --- | --- | --- | --- | --- |"]
    L += [f"| {k} | " + " / ".join(str(runs[t]['latency_ms'][k]['n']) for t in rs) + " | " + " | ".join(" / ".join(f(runs[t]['latency_ms'][k][m]) for t in rs) for m in ("p50", "p95", "max"))
          + f" | {sum(runs[t]['latency_ms'][k]['non_200'] for t in rs)} | " + " / ".join(str(runs[t]['latency_ms'][k].get('rate_limited_excluded', 'chưa đếm')) for t in rs) + " |" for k in a["latency_ms"]]
    new = all("rate_limited_excluded" in v for r in runs.values() for v in r["latency_ms"].values())
    L += ["", "![](e6.svg)", "", ("Mỗi endpoint đo bằng một tài khoản riêng (nhân viên dùng chung thì nghỉ 61 s ngoài vùng đo); trong vùng đo không thử lại: phản hồi 429 (giới hạn 600 yêu cầu / phút / người) "
          "bị loại khỏi độ trễ và đếm ở cột cuối." if new else
          "**Đo lệch do 429 — chờ đo lại:** các lượt này dùng một token giảng viên cho cả documents + students + gradebook, chạm giới hạn 600 yêu cầu / phút / người; bộ đo ngủ Retry-After "
          "và tính vào độ trễ (max 20–27 s của gradebook là đúng một yêu cầu như vậy, không phải máy chủ chậm — dev2 điều tra). p50 / p95 không bị ảnh hưởng. Bộ đo đã sửa (tries = 1, 429 đếm riêng, mỗi endpoint một tài khoản); số max sẽ thay khi đo lại."), ""]
    if repro:
        L += [f"Tái lập (AC11, sai số ±15 %, Q-S11-13): RAM gateway + worker lệch {f(repro['ram_go_pct'])} %, khởi động gateway {f(repro['cold_gateway_pct'])} %, worker {f(repro['cold_worker_pct'])} % → **{'trong sai số' if repro['within'] else 'NGOÀI sai số'}**. "
              f"Khởi động lạnh chỉ vài trăm ms nên dao động lịch CPU của VM Colima (vài trăm ms) đủ làm lệch phần trăm lớn; trung vị lớn nhất {max(med(r['cold_start_ms'][s]) for r in runs.values() for s in ('gateway', 'worker'))} ms.", ""]
    L += ["## Giới hạn", "", "- Một máy (MacBook, Colima), một bản gateway + một worker, dữ liệu seed T0 (nhỏ). Tải T1 không chạy lại (Q-E-20): số T1 ở báo cáo 9.5 (`benchmarks/reports/load-*.json`).",
          "- Image chạy trên `ep-qc` là bản test (`edupilot-*-test`, có route thử); kích thước giao hàng là dòng `ep-eval-*` (target production, distroless) dựng từ cùng commit.",
          "- Độ trễ đo từ máy Mac qua Caddy HTTPS, client tuần tự (không đồng thời): là độ trễ nền, không phải dưới tải. Không đối chiếu hệ cũ (D45)."]
    lib.write_report("e6", obj, L)


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    sp = ap.add_subparsers(dest="cmd", required=True)
    r = sp.add_parser("run")
    r.add_argument("--tag", default="r1")
    sp.add_parser("report")
    a = ap.parse_args()
    try:
        if a.cmd == "run":
            lib.refuse_demo()
            if not lib.ready():
                lib.die("Gateway chưa chạy")
            run(a.tag)
        else:
            report()
    except Fail as e:
        lib.die(f"LỖI: {e}", 2)


if __name__ == "__main__":
    main()

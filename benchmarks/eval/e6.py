#!/usr/bin/env python3
"""E6 — nền Go (US-P10-10 AC8): RAM nghỉ, khởi động lạnh, kích thước image, p95 ≥ 5 endpoint. Không gọi LLM. Không đối chiếu hệ cũ (D45).

  e6.py run --tag k1    đo một lượt → raw/e6-<tag>.json  (AC11: chạy 2 lượt, sai số ghi ở báo cáo; Q-S11-13: ±15 % RAM / khởi động)
  e6.py report          → e6.json / e6.md / e6.svg (mọi lượt; tái lập so từng cặp cùng quy trình)
Stack đo `ep-qc` (docker compose -p ep-qc). Quy trình P-11-53 (lượt r1/r2 cũ giữ nguyên trong báo cáo):
  1. Ghi môi trường mỗi pha: loadavg Mac + VM Colima, container đang chạy (project, StartedAt); có project khác `ep-qc` ⇒ cờ `contended`.
  2. Khởi động lại gateway + worker, làm ấm cố định (N_WARM GET mỗi endpoint), nghỉ 60 s, rồi đo RAM (`docker stats`).
  3. Khởi động lạnh 10 mẫu: gateway dò `GET /readyz` từ host qua Caddy mỗi 10 ms; worker `docker exec … /worker -healthcheck`.
     Ghi chi phí một lần dò của mỗi bên (độ phân giải), báo trung vị + khoảng tứ phân vị.
  4. Độ trễ endpoint như cũ (mỗi endpoint một tài khoản, tries = 1, 429 đếm riêng).
"""
import argparse
import json
import os
import re
import statistics
import subprocess
import time

import lib
from lib import Fail, call, login

P = "ep-qc"
PROTOCOL = "P-11-53"
SERVICES = ["gateway", "worker", "postgres", "pgbouncer", "redis", "caddy", "frontend", "minio", "docling", "judge"]
N_REQ, N_WARM, N_COLD = 200, 20, 10
WORKER_PROBE = ["docker", "exec", f"{P}-worker-1", "/worker", "-healthcheck"]


def sh(*cmd, check=True):
    r = subprocess.run(cmd, capture_output=True, text=True, timeout=600)
    if check and r.returncode:
        raise Fail(f"{' '.join(cmd[:4])}…: {r.stderr[:200]}")
    return r.stdout


def mib(s):
    m = re.match(r"([\d.]+)\s*([KMG]i?B|B)", s)
    v, u = float(m.group(1)), m.group(2)
    return round(v * {"B": 1 / 2**20, "KiB": 1 / 1024, "KB": 1 / 1024, "MiB": 1, "MB": 1, "GiB": 1024, "GB": 1024}[u], 1)


def env(phase):
    """Môi trường tại một pha: loadavg Mac + VM, container đang chạy theo project."""
    vm = sh("colima", "ssh", "--", "cat", "/proc/loadavg", check=False).split()[:3]
    ids = sh("docker", "ps", "-q").split()
    rows = sh("docker", "inspect", "--format", '{{.Name}}\t{{index .Config.Labels "com.docker.compose.project"}}\t{{.State.StartedAt}}', *ids).splitlines() if ids else []
    ctr = [dict(zip(("name", "project", "started_at"), r.lstrip("/").split("\t"))) for r in rows]
    return {"phase": phase, "at": lib.now_iso(), "load_mac": [round(x, 2) for x in os.getloadavg()], "load_vm": [float(x) for x in vm],
            "other_projects": sorted({c["project"] or c["name"] for c in ctr if c["project"] != P}), "containers": ctr}


def idle_ram():
    time.sleep(60)  # nghỉ 60 s, không yêu cầu nào
    out, others = {}, {}
    for line in sh("docker", "stats", "--no-stream", "--format", "{{.Name}}\t{{.MemUsage}}\t{{.CPUPerc}}").splitlines():
        name, mem, cpu = line.split("\t")
        m = re.match(rf"{P}-(.+)-\d+$", name)
        if m and m.group(1) in SERVICES:
            out[m.group(1)] = {"mem_mib": mib(mem.split("/")[0].strip()), "cpu": cpu}
        else:
            others[name] = {"mem_mib": mib(mem.split("/")[0].strip()), "cpu": cpu}
    return out, others


def gateway_up():
    return lib.ready()


def worker_up():
    return subprocess.run(WORKER_PROBE, capture_output=True, timeout=10).returncode == 0


def probe_cost(probe, n=10):
    """Chi phí một lần dò khi tiến trình đang chạy (ms) = độ phân giải của phép đo khởi động lạnh."""
    out = []
    for _ in range(n):
        t = time.perf_counter()
        probe()
        out.append(round((time.perf_counter() - t) * 1000, 1))
    return out


def restart(svc, probe):
    sh("docker", "stop", f"{P}-{svc}-1")
    t0 = time.perf_counter()
    sh("docker", "start", f"{P}-{svc}-1")
    end = time.time() + 120
    while time.time() < end:
        if probe():
            return round((time.perf_counter() - t0) * 1000)
        time.sleep(0.01)
    raise Fail(f"{svc} không sẵn sàng sau 120 s")


def cold_start(svc, probe, n=N_COLD):
    times = []
    for _ in range(n):
        times.append(restart(svc, probe))
        time.sleep(3)
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


def endpoints():
    cid = lib.course_by_code(login("teacher"), lib.CLASS1)
    return [("GET /readyz", "/readyz", None), ("GET /me/courses (SV)", "/me/courses?limit=20", "sv.kha"), ("GET threads lớp 1 (SV)", f"/courses/{cid}/threads?limit=20", "sv.gioi"),
            ("GET documents lớp 1 (GV)", f"/courses/{cid}/documents?limit=20", "teacher"), ("GET students lớp 1 (TA)", f"/courses/{cid}/students?limit=50", "ta"),
            ("GET gradebook lớp 1 (GV)", f"/courses/{cid}/gradebook", "teacher"), ("GET today lớp 1 (TA)", f"/courses/{cid}/today", "ta")]


def warm(eps):
    """Làm ấm cố định sau khi khởi động lại: N_WARM GET mỗi endpoint (ngoài vùng đo)."""
    for _, path, who in eps:
        tok = login(who) if who else None
        for _ in range(N_WARM):
            call("GET", path, tok, tries=1)


def latency(eps):
    """Mỗi endpoint một tài khoản; teacher / ta mỗi tài khoản 2 endpoint ⇒ trong vùng đo ≤ 2 × (1 + N_WARM + N_REQ) = 442 (+ 40 lúc làm ấm, cách > 60 s)
    < 600 yêu cầu / phút / người, nên không cần nghỉ giữa endpoint. Trong vùng đo `tries=1`: 429 đếm riêng và loại khỏi độ trễ."""
    out = {}
    for name, path, who in eps:
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
    return out


def run(tag):
    images = lib.load_raw("images-e6.json") or image_sizes()  # `e6.py images` dựng trước cửa sổ đo: docker build nặng CPU, không để lẫn vào lượt
    images.pop("meta", None)
    eps = endpoints()
    envs = [env("bắt đầu")]
    restart("gateway", gateway_up)
    restart("worker", worker_up)
    warm(eps)
    envs.append(env("sau khởi động lại + làm ấm"))
    ram, others = idle_ram()
    envs.append(env("đo RAM nghỉ"))
    probes = {"gateway": probe_cost(gateway_up), "worker": probe_cost(worker_up)}
    cold = {"gateway": cold_start("gateway", gateway_up), "worker": cold_start("worker", worker_up)}
    envs.append(env("sau khởi động lạnh"))
    time.sleep(10)
    lat = latency(eps)
    envs.append(env("sau độ trễ"))
    raw = {"tag": tag, "protocol": PROTOCOL, "idle_ram": ram, "idle_ram_other_containers": others, "cold_start_ms": cold, "probe_cost_ms": probes,
           "latency_ms": lat, "images": images, "env": envs, "contended": any(e["other_projects"] for e in envs),
           "colima": sh("colima", "list", "--json", check=False).strip()[:400]}
    lib.save_raw(f"e6-{tag}.json", raw)
    print(json.dumps({"contended": raw["contended"], "cold_start_ms": cold, "probe_cost_ms": probes}, ensure_ascii=False))


def report():
    runs = {p.stem[3:]: lib.load_raw(p.name) for p in lib.RAW.glob("e6-*.json")}
    if not runs:
        raise Fail("chưa có raw/e6-*.json")
    proto = lambda r: r.get("protocol") or "trước P-11-53"  # noqa: E731
    rs = sorted(runs, key=lambda t: (bool(runs[t].get("protocol")), t))  # lượt cũ trước, lượt theo quy trình mới sau
    groups = {}
    for t in rs:
        groups.setdefault(proto(runs[t]), []).append(t)
    a = runs[rs[-1]]  # lượt mới nhất: image + biểu đồ
    go = lambda r: round(r["idle_ram"]["gateway"]["mem_mib"] + r["idle_ram"]["worker"]["mem_mib"], 1)  # noqa: E731
    med = statistics.median
    iqr = lambda xs: [round(q, 1) for q in statistics.quantiles(xs, n=4, method="inclusive")[::2]]  # noqa: E731 — [Q1, Q3]
    d = lambda x, y: round(abs(x - y) / x * 100, 1) if x else None  # noqa: E731
    repro = {}
    for g, ts in groups.items():
        if len(ts) >= 2:
            x, y = runs[ts[0]], runs[ts[1]]
            r = {"runs": ts[:2], "ram_go_pct": d(go(x), go(y)), "cold_gateway_pct": d(med(x["cold_start_ms"]["gateway"]), med(y["cold_start_ms"]["gateway"])),
                 "cold_worker_pct": d(med(x["cold_start_ms"]["worker"]), med(y["cold_start_ms"]["worker"])), "tolerance_pct": 15,
                 "contended": [t for t in ts[:2] if runs[t].get("contended")]}
            r["within"] = all(v is not None and v <= 15 for k, v in r.items() if k.endswith("_pct") and k != "tolerance_pct")
            repro[g] = r
    obj = {"meta": lib.meta({"stack": "ep-qc (1 gateway, 1 worker, seed T0)", "requests_per_endpoint": N_REQ, "warmup": N_WARM, "cold_samples": N_COLD,
                             "client": "Python urllib tuần tự, qua Caddy HTTPS"}, runs=len(runs)), "status": "OK", "runs": runs, "reproducibility": repro}
    lib.svg_bars("e6.svg", [(k.replace(" lớp 1", ""), [v["p50"], v["p95"]]) for k, v in a["latency_ms"].items()], ["p50", "p95"],
                 f"E6 — độ trễ endpoint (ms, tuần tự, 200 yêu cầu, lượt {rs[-1]})", "ms", fmt="{:.0f}")
    f = lambda x: str(x).replace(".", ",")  # noqa: E731
    hdr = ["| tiến trình | " + " | ".join(rs) + " |", "| --- |" + " --- |" * len(rs)]
    L = ["# E6 — nền Go", "", f"Trạng thái: **OK**. Stack `ep-qc` @ `{obj['meta']['commit']}`, {obj['meta']['timestamp'][:10]}. Máy: {lib.MACHINE}. {len(runs)} lượt, mọi lượt đều báo cáo.", "",
         "## Lượt và môi trường", "", "| lượt | quy trình | lưu lúc (UTC) | stack khác đang chạy | CPU stack khác lúc đo RAM | loadavg 1 phút VM (6 CPU) | loadavg 1 phút Mac |",
         "| --- | --- | --- | --- | --- | --- | --- |"]
    for t in rs:
        r, envs = runs[t], runs[t].get("env") or []
        rng = lambda k: f"{f(min(e[k][0] for e in envs))}–{f(max(e[k][0] for e in envs))}" if envs and all(e[k] for e in envs) else "không ghi"  # noqa: E731
        other = sorted({p for e in envs for p in e["other_projects"]})
        oc = r.get("idle_ram_other_containers")
        cpu = f(round(sum(float(v["cpu"].rstrip("%")) for v in oc.values()), 1)) + " %" if oc is not None else "không ghi"
        L.append(f"| {t} | {proto(r)} | {r['meta']['timestamp'][11:19]} | {', '.join(other) if other else ('không' if envs else 'không ghi')} | {cpu} | {rng('load_vm')} | {rng('load_mac')} |")
    L += ["", "## RAM khi nghỉ (`docker stats` sau 60 s không yêu cầu; P-11-53: sau khởi động lại + làm ấm cố định)", ""] + hdr
    for s in SERVICES:
        if s in a["idle_ram"]:
            L.append(f"| {s} | " + " | ".join(f(runs[t]['idle_ram'].get(s, {}).get('mem_mib', '–')) + " MiB" for t in rs) + " |")
    L += ["", "## Khởi động lạnh (`docker start` → sẵn sàng, ms): trung vị [Q1–Q3], n; độ phân giải = chi phí một lần dò", ""] + hdr
    for s in ("gateway", "worker"):
        cells = []
        for t in rs:
            xs, pc = runs[t]["cold_start_ms"][s], (runs[t].get("probe_cost_ms") or {}).get(s)
            cells.append(f"{f(med(xs))} [{'–'.join(map(f, iqr(xs)))}], n = {len(xs)}" + (f"; dò ≈ {f(med(pc))}" if pc else "; dò `docker exec` ≈ 192 (đo sau)"))
        L.append(f"| {s} | " + " | ".join(cells) + " |")
    L += ["", "## Kích thước image", "", "| image | MiB |", "| --- | --- |"] + [f"| `{v['image']}` | {f(v['mib'])} |" for v in (a.get("images") or {}).values()]
    L += ["", "## Độ trễ endpoint (ms, " + " / ".join(rs) + ")", "", "| endpoint | n | p50 | p95 | max | ≠ 200 | 429 (đã loại) |", "| --- | --- | --- | --- | --- | --- | --- |"]
    L += [f"| {k} | " + " / ".join(str(runs[t]['latency_ms'][k]['n']) for t in rs) + " | " + " | ".join(" / ".join(f(runs[t]['latency_ms'][k][m]) for t in rs) for m in ("p50", "p95", "max"))
          + f" | {sum(runs[t]['latency_ms'][k]['non_200'] for t in rs)} | " + " / ".join(str(runs[t]['latency_ms'][k].get('rate_limited_excluded', 'chưa đếm')) for t in rs) + " |" for k in a["latency_ms"]]
    L += ["", "![](e6.svg)", "", "Mỗi endpoint đo bằng một tài khoản riêng; trong vùng đo không thử lại: phản hồi 429 (giới hạn 600 yêu cầu / phút / người) bị loại khỏi độ trễ và đếm ở cột cuối.", ""]
    if repro:
        L += ["## Tái lập (AC11, sai số ±15 %, Q-S11-13)", "", "| quy trình | cặp lượt | RAM gateway + worker | khởi động gateway | khởi động worker | kết luận |", "| --- | --- | --- | --- | --- | --- |"]
        for g, r in repro.items():
            flag = f" (cờ `contended`: {', '.join(r['contended'])})" if r["contended"] else ""
            L.append(f"| {g} | {' / '.join(r['runs'])} | {f(r['ram_go_pct'])} % | {f(r['cold_gateway_pct'])} % | {f(r['cold_worker_pct'])} % | **{'trong sai số' if r['within'] else 'NGOÀI sai số'}**{flag} |")
        if "trước P-11-53" in repro:
            L += ["", "Cặp trước P-11-53: có stack `ep-r` dựng / chạy song song, RAM r1 đo tiến trình chạy lâu còn r2 đo tiến trình vừa khởi động lại, và dò khởi động lạnh bằng `docker exec` (≈ 120–270 ms mỗi lần); phân tích ở `docs/sprints/11/techlead.md` mục E6."]
        L.append("")
    L += ["## Giới hạn", "", "- Một máy (MacBook, Colima), một bản gateway + một worker, dữ liệu seed T0 (nhỏ). Tải T1 không chạy lại (Q-E-20): số T1 ở báo cáo 9.5 (`benchmarks/reports/load-*.json`).",
          "- Image chạy trên `ep-qc` là bản test (`edupilot-*-test`, có route thử); kích thước giao hàng là dòng `ep-eval-*` (target production, distroless) dựng từ cùng commit.",
          "- Độ trễ đo từ máy Mac qua Caddy HTTPS, client tuần tự (không đồng thời): là độ trễ nền, không phải dưới tải. Không đối chiếu hệ cũ (D45).",
          "- Khởi động lạnh của worker vẫn dò bằng `docker exec` (worker không mở cổng, không có dòng log «sẵn sàng»): độ phân giải thô hơn gateway, ghi ở cột «dò»."]
    lib.write_report("e6", obj, L)


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    sp = ap.add_subparsers(dest="cmd", required=True)
    r = sp.add_parser("run")
    r.add_argument("--tag", default="k1")
    sp.add_parser("images", help="dựng image production + ghi kích thước (chạy TRƯỚC cửa sổ đo)")
    sp.add_parser("report")
    a = ap.parse_args()
    try:
        if a.cmd == "run":
            lib.refuse_demo()
            if not lib.ready():
                lib.die("Gateway chưa chạy")
            run(a.tag)
        elif a.cmd == "images":
            lib.save_raw("images-e6.json", image_sizes())
        else:
            report()
    except Fail as e:
        lib.die(f"LỖI: {e}", 2)


if __name__ == "__main__":
    main()

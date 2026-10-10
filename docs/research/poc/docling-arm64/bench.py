#!/usr/bin/env python3
"""PoC đo docling-serve trên colima arm64 (sprint 6). Chỉ stdlib + docker + curl.

Mỗi (cấu hình, tệp) chạy trong một container `poc-docling` MỚI để memory.peak của cgroup không cộng dồn.
RAM: memory.current / memory.peak (gồm page cache) và `anon` trong memory.stat (bộ nhớ tiến trình thật), lấy mẫu 0,25 s.
Chạy: python3 bench.py <cấu-hình> <tệp...>   → in một dòng JSON mỗi lần đo; Markdown ghi ra /tmp/poc-docling-out/.
"""
import json, os, subprocess, sys, threading, time, urllib.request

IMAGE = "ghcr.io/docling-project/docling-serve-cpu:v1.36.0"
NAME = "poc-docling"
BASE = "http://127.0.0.1:5001"

# Cấu hình: (env cho container, form options cho /v1/convert/file/async)
LAZY = {"DOCLING_SERVE_LOAD_MODELS_AT_BOOT": "false"}  # không nạp sẵn mô hình của cấu hình mặc định lúc boot
# PDF chữ: tắt OCR, tắt mô hình bảng, không sinh ảnh
MIN = {"do_ocr": "false", "do_table_structure": "false", "include_images": "false", "image_export_mode": "placeholder"}
PDFIUM = {"pdf_backend": "pypdfium2"}  # docling_parse 7.22.1 abort (SIGTRAP) trên Mordern_Network_Security_Threats.pdf
OCR_VI = {"do_ocr": "true", "ocr_preset": "easyocr", "ocr_lang": ["vi"]}  # latin_g2.pth có sẵn trong image
TESS = "/usr/share/tesseract/tessdata/vie.traineddata"
CONFIGS = {
    "default": ({}, PDFIUM),  # mặc định server (OCR auto, TableFormer accurate, ảnh) + backend không crash
    "minimal": (LAZY, MIN),
    "minimal_pdfium": (LAZY, {**MIN, **PDFIUM}),
    "tables_fast": (LAZY, {**MIN, **PDFIUM, "do_table_structure": "true", "table_mode": "fast"}),
    "ocr_vi": (LAZY, {**MIN, **PDFIUM, **OCR_VI}),
    "ocr_auto": (LAZY, {**MIN, **PDFIUM, "do_ocr": "true"}),
    # pipeline=native: jobkit 3.8.1 báo "ProcessingPipeline.NATIVE is not implemented" → bỏ
    # Tesseract + vie.traineddata (tessdata_fast / tessdata_best) chép vào container; khoá bắt đầu bằng "/" = docker cp.
    # ocr_lang phải là "vie": "vi" bị TesseractOcrCli từ chối ("No traineddata file 'vi'").
    "ocr_tess_vi": ({**LAZY, TESS: "/tmp/poc-tessdata/vie.traineddata"},
                    {**MIN, **PDFIUM, "do_ocr": "true", "ocr_preset": "tesseract", "ocr_lang": ["vie"]}),
    "ocr_tess_vi_best": ({**LAZY, TESS: "/tmp/poc-tessdata/vie.best.traineddata"},
                         {**MIN, **PDFIUM, "do_ocr": "true", "ocr_preset": "tesseract", "ocr_lang": ["vie"]}),
    # chia tệp dài thành đoạn trang: worker gọi nhiều lần với page_range, mỗi lần < deadline
    "minimal_pdfium_p1_40": (LAZY, {**MIN, **PDFIUM, "page_range": ["1", "40"]}),
}
COMMON_FORM = {"to_formats": ["md", "json"], "md_page_break_placeholder": "<!-- page -->"}


def sh(*args, check=True):
    return subprocess.run(args, capture_output=True, text=True, check=check).stdout


def mem():
    out = sh("docker", "exec", NAME, "sh", "-c",
             "cat /sys/fs/cgroup/memory.current /sys/fs/cgroup/memory.peak; grep '^anon ' /sys/fs/cgroup/memory.stat")
    cur, peak, anon = out.split("\n")[:3]
    return int(cur), int(peak), int(anon.split()[1])


def start(env):
    sh("docker", "rm", "-f", NAME, check=False)
    args = ["docker", "run", "-d", "--name", NAME, "-p", "127.0.0.1:5001:5001", "--memory", "5g",
            "-e", "DOCLING_SERVE_ENG_LOC_NUM_WORKERS=1"]
    for k, v in env.items():
        if not k.startswith("/"):
            args += ["-e", f"{k}={v}"]
    t0 = time.time()
    sh(*args, IMAGE)
    while True:
        try:
            if urllib.request.urlopen(BASE + "/ready", timeout=2).status == 200:
                break
        except Exception:
            pass
        time.sleep(0.5)
    for dst, src in env.items():
        if dst.startswith("/"):
            sh("docker", "cp", src, f"{NAME}:{dst}")
    boot = time.time() - t0
    time.sleep(3)
    return boot, mem()


def get(path):
    return json.load(urllib.request.urlopen(BASE + path, timeout=30))


def convert(path, form):
    args = ["curl", "-sf", "-X", "POST", BASE + "/v1/convert/file/async", "-F", f"files=@{path};type=application/pdf"]
    for k, v in {**COMMON_FORM, **form}.items():
        for item in (v if isinstance(v, list) else [v]):
            args += ["--form-string", f"{k}={item}"]
    task = json.loads(sh(*args))
    while task["task_status"] not in ("success", "failure"):
        time.sleep(0.5)
        task = get(f"/v1/status/poll/{task['task_id']}")
    if task["task_status"] == "failure":
        raise RuntimeError(task.get("error_message"))
    return task, get(f"/v1/result/{task['task_id']}")


def run(cfg, path):
    env, form = CONFIGS[cfg]
    boot, (idle_cur, _, idle_anon) = start(env)
    samples, stop = [], threading.Event()

    def sampler():
        while not stop.is_set():
            try:
                samples.append(mem())
            except subprocess.CalledProcessError:
                return  # container đã chết
            time.sleep(0.25)

    th = threading.Thread(target=sampler)
    th.start()
    t0 = time.time()
    try:
        task, res = convert(path, form)
    except Exception as e:
        stop.set()
        th.join()
        state = sh("docker", "inspect", NAME, "--format",
                   "{{.State.Status}} exit={{.State.ExitCode}} oom={{.State.OOMKilled}}").strip()
        return {"config": cfg, "file": path.rsplit("/", 1)[-1], "status": f"FAIL container={state} {type(e).__name__}: {e}",
                "wall_s": round(time.time() - t0, 1), "peak_anon_mib": round(max([s[2] for s in samples] or [0]) / 2**20)}
    wall = time.time() - t0
    stop.set()
    th.join()
    _, peak, _ = mem()
    doc = res["document"]
    md = doc.get("md_content") or ""
    os.makedirs("/tmp/poc-docling-out", exist_ok=True)
    with open(f"/tmp/poc-docling-out/{cfg}-{path.rsplit('/', 1)[-1]}.md", "w") as fh:
        fh.write(md)
    pages = len((doc.get("json_content") or {}).get("pages") or {})
    mib = lambda b: round(b / 2**20)
    return {
        "config": cfg, "file": path.rsplit("/", 1)[-1], "status": res.get("status"),
        "boot_s": round(boot, 1), "idle_current_mib": mib(idle_cur), "idle_anon_mib": mib(idle_anon),
        "peak_cgroup_mib": mib(peak), "peak_anon_mib": mib(max(s[2] for s in samples)),
        "wall_s": round(wall, 1), "processing_time_s": round(res.get("processing_time") or 0, 1),
        "pages": pages, "md_chars": len(md), "errors": res.get("errors"),
    }


if __name__ == "__main__":
    cfg, files = sys.argv[1], sys.argv[2:]
    for f in files:
        print(json.dumps(run(cfg, f), ensure_ascii=False), flush=True)

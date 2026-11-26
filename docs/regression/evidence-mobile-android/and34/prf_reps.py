# PRF-02 (5 lượt mạng thường) + PRF-03 (3 lượt mạng chậm): quay màn hình 60 fps, đo QR-vào-khung → bắt đầu gửi / → «Đã điểm danh».
import lib34 as L, m, time, json, subprocess, glob, sys
from PIL import Image, ImageChops, ImageStat
def one(tag):
    w = L.newwin(L.CID["QCMOB2"]); L.blank_poster(); time.sleep(0.8)
    L.home(); L.to_checkin(); time.sleep(1.2)
    if not L.open_scan(): return None
    rec = subprocess.Popen("adb shell screenrecord --size 540x1200 --bit-rate 6000000 --time-limit 12 /sdcard/prf.mp4", shell=True); time.sleep(1.8)
    u, d = L.qr_url(w); L.poster(u, tag)
    for _ in range(40):
        if L.rows(w): break
        time.sleep(0.3)
    ok = bool(L.rows(w)); rec.wait(); out = f"{L.WORKD}/prf_{tag}.mp4"
    subprocess.run(f"adb pull /sdcard/prf.mp4 '{out}' >/dev/null; adb shell rm /sdcard/prf.mp4", shell=True)
    fr, _ = L.analyze(out, tag)
    dur = float(subprocess.run(f"ffprobe -v error -show_entries format=duration -of csv=p=0 {out}", shell=True, capture_output=True, text=True).stdout.strip()); fps = len(fr)/dur
    prev = None; ev = []
    for i, f in enumerate(fr):
        im = Image.open(f).convert("L")
        if prev is not None:
            df = ImageStat.Stat(ImageChops.difference(im, prev)).mean[0]
            if df > 0.5: ev.append((i, df))
        prev = im
    big = [e for e in ev if e[1] > 5]
    if len(big) < 2 or not ok: return dict(tag=tag, ok=ok, err="khong nhan du su kien", ev=ev[:8])
    A = big[0][0]; C = max(ev, key=lambda e: e[1])[0]; R = ev[-1][0]
    return dict(tag=tag, ok=ok, fps=round(fps), A_s=round(A/fps,2), C_minus_A_ms=round((C-A)/fps*1000), R_minus_A_s=round((R-A)/fps,2))
mode = sys.argv[1]
res = []
if mode == "normal":
    L.policy(L.CID["QCMOB2"], rot=60, geo=None)
    for k in range(5): res.append(one(f"n{k+1}")); print(res[-1], flush=True)
else:
    subprocess.run("adb emu network speed gsm; adb emu network delay gprs", shell=True, capture_output=True)
    try:
        for k in range(3): res.append(one(f"s{k+1}")); print(res[-1], flush=True)
    finally:
        subprocess.run("adb emu network speed full; adb emu network delay none", shell=True, capture_output=True)
        L.policy(L.CID["QCMOB2"], rot=10, geo=None)
json.dump(res, open(f"{L.WORKD}/prf_{mode}.json", "w"), ensure_ascii=False)
print("XONG", mode)

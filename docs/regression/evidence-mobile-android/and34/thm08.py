import lib34 as L, m, time, json, contrast as C
L.memguard()
b0 = L.adb("shell settings get system screen_brightness"); out = {}
js_rect = lambda needle: m.cdp("(()=>{const e=[...document.querySelectorAll('p,div,span,h1,h2,li')].filter(x=>x.offsetParent&&x.children.length<=1&&(x.innerText||'').includes(%s)).sort((a,b)=>a.innerText.length-b.innerText.length)[0]; if(!e) return 'none'; const r=e.getBoundingClientRect(); const cs=getComputedStyle(e); return JSON.stringify({x:r.left,y:r.top,w:r.width,h:r.height,color:cs.color,fs:cs.fontSize,fw:cs.fontWeight,dpr:devicePixelRatio,txt:e.innerText.trim().slice(0,60)})})()" % json.dumps(needle))
def parse(rgb): 
    import re; return tuple(int(float(v)) for v in re.findall(r"[\d.]+", rgb)[:3])
w = L.newwin(L.CID["QCMOB2"]); L.home(); L.to_checkin(); time.sleep(2)
L.click("Nhập mã"); time.sleep(1.2); L.type_code("AAA111"); time.sleep(0.8); L.click("Điểm danh"); time.sleep(3)
for level in (255, 20):
    L.adb(f"shell settings put system screen_brightness {level}"); time.sleep(1)
    r = json.loads(js_rect("Mã không đúng")); png = L.shot(f"THM-08-loi-b{level}"); k = r["dpr"]
    box = tuple(int(v) for v in (r["x"]*k-6, r["y"]*k-6, (r["x"]+r["w"])*k+6, (r["y"]+r["h"])*k+6))
    bg, fg, ratio_px = C.region_ratio(png, box); out[f"loi_b{level}"] = dict(rect=r, bg=bg, fg=fg, ratio_px=ratio_px, css_color=parse(r["color"]))
    print("LOI  bright", level, "|", r["txt"][:40], "| size", r["fs"], "w", r["fw"], "| pixel bg", bg, "fg", fg, "=>", ratio_px, ":1", flush=True)
# thành công
d = L.display(w); L.type_code(d["short_code"]); time.sleep(0.8); L.click("Điểm danh"); t0 = time.time()
while time.time() - t0 < 12 and not L.rows(w): time.sleep(0.4)
time.sleep(1.5)
for level in (255, 20):
    L.adb(f"shell settings put system screen_brightness {level}"); time.sleep(1)
    r = json.loads(js_rect("Đã điểm danh")); png = L.shot(f"THM-08-ok-b{level}"); k = r["dpr"]
    box = tuple(int(v) for v in (r["x"]*k-6, r["y"]*k-6, (r["x"]+r["w"])*k+6, (r["y"]+r["h"])*k+6))
    bg, fg, ratio_px = C.region_ratio(png, box); out[f"ok_b{level}"] = dict(rect=r, bg=bg, fg=fg, ratio_px=ratio_px, css_color=parse(r["color"]))
    print("OK   bright", level, "|", r["txt"][:40], "| size", r["fs"], "w", r["fw"], "| pixel bg", bg, "fg", fg, "=>", ratio_px, ":1", flush=True)
L.adb(f"shell settings put system screen_brightness {b0}")
for k_, v in out.items(): v["css_ratio"] = round(C.ratio(v["css_color"], v["bg"]), 2)
json.dump(out, open(L.WORKD + "/thm08.json", "w"), ensure_ascii=False, default=list)
print({k_: v["css_ratio"] for k_, v in out.items()})

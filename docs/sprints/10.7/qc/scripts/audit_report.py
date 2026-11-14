#!/usr/bin/env python3
"""QC 10.7 — audit 'sau' (móc data-ep-*). Đọc audit/<phase>/audit-{data,empty,loading,error}.jsonl (qc-ui-audit-real.spec.ts v2) → audit.md + audit.html.
Chạy: python3 audit_report.py [phase=sau]. Tiêu chí theo docs/sprints/10.7/techlead.md 1.2 (W/H/P/E/L/X/D/T/C)."""
import json, os, sys, html, collections
HERE = os.path.dirname(os.path.abspath(__file__)); QC = os.path.dirname(HERE)
PHASE = sys.argv[1] if len(sys.argv) > 1 else "sau"; A = os.path.join(QC, "audit", PHASE)
ROLES = ["student", "ta", "teacher", "admin"]
def load(st):
    p = os.path.join(A, f"audit-{st}.jsonl"); return [json.loads(l) for l in open(p)] if os.path.exists(p) else []
rows = {st: load(st) for st in ("data", "empty", "loading", "error")}
allr = [r for v in rows.values() for r in v]
ok = lambda r: r and "err" not in r and not r.get("noMain")
def rk(r): return (r["role"], r["route"])
idx = {(r["state"], r["role"], r["route"], r["w"]): r for r in allr}
tag = lambda r: f"{r['role']}{r['route']}[{r['state']}@{r['w']}]"
near = lambda a, b, t=1.0: a is not None and abs(a - b) <= t
FAIL = collections.OrderedDict()
def bad(code, r, why=""):
    FAIL.setdefault(code, []).append(f"{tag(r)}{(' ' + why) if why else ''}")
DESC = {
 "W1": "kiểu trang: đúng 1 `data-ep-page` = work|read", "W2": "bề rộng cột 1440: work 1176; read 800 căn giữa (428→1228)", "W3": "mép panel cấp cao nhất = mép cột (trừ Split)", "W4": "mép trái h1 = mép trái cột; mép phải cụm nút = mép phải cột",
 "W5": "375: cột 12→363 (rộng 351), không tràn ngang", "H1": "đúng 1 h1 trong header", "H2": "0 thẻ `p` thừa ở đầu trang", "H3": "meta ≤ 1 dòng, không chứa lớp đang chọn", "H4": "nút cùng hàng tiêu đề (|Δ tâm dọc| ≤ 4, 1440)",
 "H5": "≤ 1 primary, ≤ 4 nút ở đầu trang", "H6": "0 nút primary ngoài đầu trang khi ≤ 1 vùng", "H7": "chiều cao đầu trang mỗi nhóm (có/không meta) đúng 1 giá trị (±2)", "H8": "375: hàng nút đầu trang ≤ 48 px",
 "P1": "đệm panel 24 (16 ở 375)", "P2": "panel cao theo nội dung (chênh ≤ 2; trừ Split fill)", "P3": "0 panel lồng", "P4": "h2 có `data-ep-section-title`, 0 `p` ở đầu mục", "P5": "nhịp dọc đầu trang→thân 40 (32@375), mục→mục 48, h2→panel 16",
 "E1": "khối rỗng: 1 tiêu đề + ≤ 1 nút, 0 svg, 0 chữ khác", "E2": "khối rỗng căn giữa |Δ| ≤ 2 px", "E3": "tiêu đề rỗng ≤ 40 ký tự, 1 dòng, không dấu chấm cuối", "E4": "nút rỗng không trùng nút đầu trang / tab / thanh công cụ", "E5": "nút rỗng `secondary`",
 "E6": "vật chứa khối rỗng ≤ 320 px (1440)", "E7": "route ở lớp rỗng mà 0 `data-ep-empty` (xem ảnh)", "L1": "đang tải: h1 cùng toạ độ như có dữ liệu (±2)", "L2": "đang tải có khung xương `aria-busy` (≥ 1)", "L3": "CLS ≤ 0,1 (có dữ liệu)",
 "X1": "lỗi: đúng 1 `role=alert`, rộng = cột, có «Thử lại»", "X2": "lỗi: không mã trần (4xx/5xx/undefined/Error:)", "X3": "lỗi: giữ h1 + tab như có dữ liệu", "D1": "cỡ chữ ⊆ {28,24,20,16,15,14,12}", "D2": "cao điều khiển 36|30 (1440), ≥ 44 (375)",
 "D3": "cao hàng bảng 48|44", "T1": "0 icon giải thích (info / circle-help) ngoài thông báo", "T2": "chuỗi > 30 ký tự lặp < 3 lần", "C3": "vàng: ≤ 3 nền, 0 chữ",
}
FONTS = {28, 24, 20, 16, 15, 14, 12}
hmeta = collections.defaultdict(list)
for st in ("data", "empty"):
    for r in rows[st]:
        if not ok(r): continue
        w = r["w"]; col = r.get("col"); k = r["kind"]; h = r["header"]
        if r["pageCount"] != 1 or k not in ("work", "read"): bad("W1", r, f"({r['pageCount']}, {k})")
        if col and w == 1440:
            if k == "work" and not near(col["w"], 1176): bad("W2", r, f"cột {col['w']}")
            if k == "read" and not (near(col["w"], 800) and near(col["x"], 428)): bad("W2", r, f"cột {col['x']}+{col['w']}")
        if col and not r["split"]:
            for p in r["panels"]["list"]:
                if p["gapL"] is not None and (abs(p["gapL"]) > 1 or abs(p["gapR"]) > 1): bad("W3", r, f"panel {p['w']} lề {p['gapL']}/{p['gapR']}"); break
        if col and h["h1x"] is not None and not near(h["h1x"], col["x"]): bad("W4", r, f"h1.x {h['h1x']} ≠ cột {col['x']}")
        if h["actionsRight"] is not None and abs(h["actionsRight"]) > 1: bad("W4", r, f"nút cách mép phải {h['actionsRight']}")
        if w == 375:
            if col and not (near(col["x"], 12) and near(col["r"], 363)): bad("W5", r, f"cột {col['x']}→{col['r']}")
            if r["overflowX"] > 0: bad("W5", r, f"tràn {r['overflowX']}")
        if h["h1Count"] != 1 or not h["h1InHeader"]: bad("H1", r, f"h1×{h['h1Count']} trong header={h['h1InHeader']}")
        if h["pTags"]: bad("H2", r, f"{h['pTags']} thẻ p")
        if (w == 1440 and h["metaLines"] > 1) or h["metaDupClass"]: bad("H3", r, f"meta {h['metaLines']} dòng" + (" lặp lớp" if h["metaDupClass"] else ""))
        if w == 1440 and h["dy"] is not None and abs(h["dy"]) > 4: bad("H4", r, f"Δ {h['dy']}")
        prim = sum(1 for b in h["btns"] if b["v"] == "primary")
        if prim > 1 or len(h["btns"]) > 4: bad("H5", r, f"{prim} primary, {len(h['btns'])} nút")
        if (r["sections"] <= 1 and r["primaryOutside"]) or len(r["primaryOutside"]) > max(1, r["sections"]): bad("H6", r, "; ".join(r["primaryOutside"])[:60])
        if h["hdrH"]: hmeta[(w, h["hasMeta"])].append((h["hdrH"], r))
        if w == 375 and h["actionsH"] and h["actionsH"] > 48: bad("H8", r, f"{h['actionsH']}")
        for p in r["panels"]["list"]:
            if not r["split"] and (p["pad"][0] != (16 if w == 375 else 24) or p["pad"][1] != (16 if w == 375 else 24)): bad("P1", r, f"đệm {p['pad']}"); break
        if not r["splitFill"]:
            for p in r["panels"]["list"]:
                if abs(p["extra"]) > 2: bad("P2", r, f"+{p['extra']}"); break
        if r["panels"]["nested"]: bad("P3", r, f"{r['panels']['nested']}")
        if r["h2NoHook"] or r["sectionP"]: bad("P4", r, f"h2 thiếu móc {r['h2NoHook']}, p {r['sectionP']}")
        exp_hb = 32 if w == 375 else 40
        if h["hasMeta"] is not None and r["hdrToBody"] is not None and not near(r["hdrToBody"], exp_hb, 2): bad("P5", r, f"đầu trang→thân {r['hdrToBody']}")
        if any(g is not None and not near(g, 48, 2) for g in r["secGaps"]): bad("P5", r, f"mục→mục {r['secGaps']}")
        if any(g is not None and not near(g, 16, 2) for g in r["headToPanel"]): bad("P5", r, f"h2→panel {r['headToPanel']}")
        for e in r["empties"]:
            if e["pCount"] != 1 or e["svg"] or e["extraText"] or len(e["actions"]) > 1: bad("E1", r, f"p{e['pCount']} svg{e['svg']} nút{len(e['actions'])}")
            if abs(e["cxOff"]) > 2: bad("E2", r, f"lệch {e['cxOff']}")
            if e["titleLen"] > 40 or (w == 1440 and e["titleLines"] != 1) or e["title"].endswith("."): bad("E3", r, f"«{e['title'][:44]}»")
            if any(a["v"] != "secondary" for a in e["actions"]): bad("E5", r, ",".join(a["v"] for a in e["actions"]))
            if w == 1440 and e["parentH"] > 320: bad("E6", r, f"{e['parentH']}")
        if r["dupEmpty"]: bad("E4", r, ", ".join(r["dupEmpty"]))
        if st == "empty" and not r["empties"] and r["role"] != "admin": bad("E7", r)
        if not set(r["fonts"]) <= FONTS: bad("D1", r, str(sorted(set(r["fonts"]) - FONTS)))
        okc = (lambda x: x in (36, 30)) if w == 1440 else (lambda x: x >= 44)
        badc = [x for x in r["ctrlH"] if not okc(x)]
        if badc: bad("D2", r, str(badc))
        if r["rowH"] and any(x not in (48, 44) for x in r["rowH"]): bad("D3", r, str(r["rowH"]))
        if r["icons"]: bad("T1", r, f"{r['icons']}")
        if r["repeats"]: bad("T2", r, "; ".join(r["repeats"])[:70])
        if r["yellowBg"] > 3 or r["yellowFg"]: bad("C3", r, f"nền {r['yellowBg']} chữ {r['yellowFg']}")
        if st == "data" and r["cls"] > 0.1: bad("L3", r, f"CLS {r['cls']}")
for (w, hm), lst in hmeta.items():
    vals = collections.Counter(round(v) for v, _ in lst); mode = vals.most_common(1)[0][0]
    for v, r in lst:
        if abs(v - mode) > 2: bad("H7", r, f"{v} ≠ {mode} (w{w}, meta={hm})")
for r in rows["loading"]:
    if not ok(r): continue
    d = idx.get(("data", r["role"], r["route"], r["w"]))
    if ok(d) and r["header"]["h1"] and d["header"]["h1"]:
        a, b = r["header"]["h1"], d["header"]["h1"]
        if abs(a["y"] - b["y"]) > 2 or abs(a["x"] - b["x"]) > 2: bad("L1", r, f"h1 ({a['x']},{a['y']}) vs ({b['x']},{b['y']})")
    elif ok(d) and not r["header"]["h1"]: bad("L1", r, "không có h1 khi tải")
    if r["busy"] < 1: bad("L2", r, "không có aria-busy")
for r in rows["error"]:
    if not ok(r): continue
    if r["alertCount"] != 1 or not r["alerts"] or not near(r["alerts"][0]["w"], (r["col"] or {}).get("w", -999), 2) or not r["alerts"][0]["retry"]:
        bad("X1", r, f"alert×{r['alertCount']}" + (f" rộng {r['alerts'][0]['w']}" if r["alerts"] else "") + (" thiếu Thử lại" if r["alerts"] and not r["alerts"][0]["retry"] else ""))
    if any(a["bare"] for a in r["alerts"]): bad("X2", r, r["alerts"][0]["text"][:60])
    d = idx.get(("data", r["role"], r["route"], r["w"]))
    if ok(d) and (not r["header"]["h1Count"] or r["tabs"] != d["tabs"] or (d["header"]["hdrH"] and r["header"]["hdrH"] and abs(r["header"]["hdrH"] - d["header"]["hdrH"]) > 2)): bad("X3", r, f"h1×{r['header']['h1Count']} tab {r['tabs']} vs {d['tabs']}")

# ---------- md
L = []; w = L.append
nok = sum(1 for r in allr if ok(r)); nerr = len(allr) - nok
w(f"# QC 10.7 — audit khung trang «{PHASE}» (móc `data-ep-*`)\n")
w(f"**{nok}** lượt chụp (dữ liệu {sum(ok(r) for r in rows['data'])}, rỗng {sum(ok(r) for r in rows['empty'])}, đang tải {sum(ok(r) for r in rows['loading'])}, lỗi {sum(ok(r) for r in rows['error'])}); lỗi chụp {nerr}. Số đo từ DOM (`frontend/e2e/qc-ui-audit-real.spec.ts` v2), tiêu chí theo `techlead.md` mục 1.2. Ảnh có hai đường gióng xanh ở mép cột mong đợi. Lưới ảnh: `audit.html` (xếp theo kiểu trang). So với hiện trạng: `audit-truoc.md`.\n")
w("- Dữ liệu = lớp 761987; rỗng = lớp QCEMPTY (sv31, teacher, ta). Đang tải = mọi `/api/v1/**` (trừ `auth`, `me`) giữ 5 s, chụp ở 0,8 s. Lỗi = 500. Đang tải / lỗi chỉ một vai mỗi route.\n- Không đo: C1 (`contrast.spec.ts`), C2 diện tích đỏ (xem mục cuối nếu có PIL), P5 giữa các `Split`.\n")
w("## Tổng hợp theo mã (số lượt lệch / số vai-route)\n")
w("| Mã | Phép đo | Lệch | Vai + route (tối đa 8) |\n| --- | --- | --- | --- |")
tot = 0
for code in DESC:
    v = FAIL.get(code, []); tot += len(v)
    ur = sorted({x.split("[")[0] for x in v})
    w(f"| {code} | {DESC[code]} | **{len(v)}** | {', '.join(ur[:8]) + (' …' if len(ur) > 8 else '') or '–'} |")
w(f"\n**Tổng lệch: {tot}** (mục chưa 0 = chưa đạt).\n")
# chữ ký
def sig(r):
    col = r.get("col") or {}; e = r["empties"][0] if r["empties"] else None
    return (r["kind"], round(r["header"]["h1x"] or -1), round(col.get("x", -1)), round(col.get("w", -1)), bool(r["header"]["hasMeta"]), (r["panels"]["list"][0]["pad"][0] if r["panels"]["list"] else None), ("title" + ("+nút" if e and e["actions"] else "") if e else "-"))
w("## Chữ ký khung (kiểu, h1.left, cột.left, cột.width, có meta, đệm panel, hình khối rỗng)\n")
for wd in (1440, 375):
    S = collections.defaultdict(list)
    for st in ("data", "empty"):
        for r in rows[st]:
            if ok(r) and r["w"] == wd: S[sig(r)].append(f"{r['role']}{r['route']}[{st}]")
    lim = 4 if wd == 1440 else 2
    w(f"### {wd} px — **{len(S)}** chữ ký (đạt ≤ {lim}: {'ĐẠT' if len(S) <= lim else 'CHƯA'})\n")
    w("| Số lượt | Chữ ký | Ví dụ (tối đa 4) |\n| --- | --- | --- |")
    for s, v in sorted(S.items(), key=lambda kv: -len(kv[1])): w(f"| {len(v)} | {s} | {', '.join(v[:4])} |")
    w("")
# chi tiết lệch
w("## Chi tiết lệch\n")
for code, v in FAIL.items():
    w(f"### {code} — {DESC[code]} ({len(v)})\n")
    for x in v[:60]: w(f"- {x}")
    if len(v) > 60: w(f"- … +{len(v) - 60}")
    w("")
errs = [r for r in allr if not ok(r)]
if errs:
    w("## Lượt chụp lỗi\n"); [w(f"- {tag(r)}: {r.get('err') or 'không có <main>'}") for r in errs]
open(os.path.join(QC, "audit.md" if PHASE == "sau" else f"audit-{PHASE}.md"), "w").write("\n".join(L) + "\n")

# ---------- html: xếp theo kiểu trang
H = ["<!doctype html><meta charset=utf-8><title>Audit khung trang 10.7</title><style>body{font:13px system-ui;margin:16px;background:#fff}h2{margin:28px 0 6px}table{border-collapse:collapse}td{vertical-align:top;padding:3px;border:1px solid #ddd}img{display:block;border:1px solid #bbb;background:#fff}.m{color:#555;font-size:11px;max-width:230px}.b{color:#b00020;font-weight:600}.t{font-weight:600;max-width:150px;word-break:break-all}</style><h1>Audit khung trang — «%s» (QC 10.7)</h1><p>Hàng = route (một vai). Cột: dữ liệu 1440 · rỗng 1440 · đang tải 1440 · lỗi 1440 · dữ liệu 375. Đường xanh = mép cột mong đợi. Chữ đỏ = mã lệch.</p>" % PHASE]
fails_by = collections.defaultdict(set)
for code, v in FAIL.items():
    for x in v: fails_by[x.split("[")[0]].add(code)
groups = collections.defaultdict(list)
for r in rows["data"]:
    if ok(r) and r["w"] == 1440: groups[r["kind"]].append(r)
for kind in ("work", "read", "none"):
    if not groups.get(kind): continue
    H.append(f"<h2>Kiểu {kind} ({len(groups[kind])})</h2><table>")
    for r in sorted(groups[kind], key=lambda r: (r["route"], ROLES.index(r["role"]))):
        H.append(f"<tr><td class=t>{html.escape(r['role'])}<br>{html.escape(r['route'])}<br><span class=b>{' '.join(sorted(fails_by.get(r['role'] + r['route'], [])))}</span></td>")
        for st, wd in (("data", 1440), ("empty", 1440), ("loading", 1440), ("error", 1440), ("data", 375)):
            x = idx.get((st, r["role"], r["route"], wd))
            if not x:
                alt = [y for y in rows[st] if y["route"] == r["route"] and y["w"] == wd and ok(y)] if st in ("loading", "error") else []
                x = alt[0] if alt else None
            if not ok(x): H.append("<td class=m>–</td>"); continue
            c = x.get("col") or {}; cap = f"{st}@{wd} · {x['kind']} · cột {c.get('x')}+{c.get('w')}"
            H.append(f"<td><img loading=lazy width={230 if wd == 1440 else 110} src='audit/{PHASE}/{x['img']}'><div class=m>{html.escape(cap)}</div></td>")
        H.append("</tr>")
    H.append("</table>")
open(os.path.join(QC, "audit.html" if PHASE == "sau" else f"audit-{PHASE}.html"), "w").write("".join(H))
print("audit.md", len(L), "dòng; tổng lệch", tot, {k: len(v) for k, v in FAIL.items()})

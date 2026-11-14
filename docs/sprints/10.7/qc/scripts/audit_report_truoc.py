#!/usr/bin/env python3
"""QC 10.7 — dựng audit.md + audit.html từ audit/audit-{data,empty}.jsonl (số đo DOM do qc-ui-audit-real.spec.ts). Chạy: python3 audit_report.py"""
import json, os, html, collections
HERE = os.path.dirname(os.path.abspath(__file__)); QC = os.path.dirname(HERE); A = os.path.join(QC, "audit", "truoc")
def load(st):
    p = os.path.join(A, f"audit-{st}.jsonl")
    return [json.loads(l) for l in open(p)] if os.path.exists(p) else []
rows = load("data") + load("empty")
ROLES = ["student", "ta", "teacher", "admin"]
def key(r): return (r["role"], r["route"])
def get(role, route, st, w): return next((r for r in rows if r["role"] == role and r["route"] == route and r["state"] == st and r["w"] == w), None)
def ok(r): return r and "err" not in r and not r.get("noMain")
routes = collections.OrderedDict()
for r in rows: routes.setdefault((r["role"], r["route"]), None)
def mx(r, f, d="–"):
    try: return f(r)
    except Exception: return d
def pw(r): return ", ".join(str(p["w"]) for p in r["panels"]["list"]) or "–"

L = []
w = L.append
n_pages = len([r for r in rows if ok(r)]); n_err = len([r for r in rows if not ok(r)])
w("# QC 10.7 — kiểm chéo khung trang (hiện trạng, trước UI-R)\n")
w(f"Ảnh + số đo DOM: **{n_pages}** lượt chụp ({len(routes)} cặp vai × route × 2 độ rộng × có dữ liệu / rỗng); lỗi chụp {n_err}. Nguồn: `scripts/audit_report.py` đọc `audit/audit-{{data,empty}}.jsonl` do `frontend/e2e/qc-ui-audit-real.spec.ts` (Playwright, chỉ đọc, `ep-qc`). Lưới ảnh: `audit.html`.\n")
w("- **Có dữ liệu** = lớp 761987 (seed mặc định); **rỗng** = lớp QCEMPTY dựng riêng (0 dữ liệu; teacher + ta + sv31). Admin không gắn lớp → một trạng thái. Không thao tác KC106.\n- **Số đo từ DOM** (`getBoundingClientRect`, `data-ep-panel`, `data-variant`, class module `page/reading/wide/full`, `meta`, `desc`, `sectionDesc`, `empty`); ảnh viewport (JPEG), không cuộn.\n- `?state=empty` (FR-X5) chỉ còn ở màn demo (`DemoDocuments`), các route thật không đọc tham số → trạng thái rỗng lấy bằng **lớp rỗng thật**.\n- Route động chỉ chụp ở trạng thái có dữ liệu.\n")

def d(st, w_): return [r for r in rows if r["state"] == st and r["w"] == w_ and ok(r)]
dat = d("data", 1440)

# K1 độ rộng
w("## K1. Kiểu độ rộng và panel (1440, có dữ liệu)\n")
w("Hợp đồng 10.7: đúng 2 kiểu (`work` hết vùng nội dung, `read` cột đọc cố định căn giữa). Hiện primitive có **3** kiểu (`reading` 960+48, `wide` ≤ 1440, `full`).\n")
kinds = collections.Counter(r["kind"] for r in dat)
w("| Kiểu `Page` | Số trang | Độ rộng page (px) |\n| --- | --- | --- |")
for k, c in kinds.most_common():
    ws = sorted({r["pageW"] for r in dat if r["kind"] == k and r["pageW"]})
    w(f"| `{k}` | {c} | {', '.join(map(str, ws))} |")
pws = collections.Counter(p["w"] for r in dat for p in r["panels"]["list"])
w(f"\nPanel cấp cao nhất: **{len(pws)}** độ rộng khác nhau trên {sum(1 for r in dat if r['panels']['n'])} trang có panel. Phân bố (px → số panel): " + "; ".join(f"{k}→{v}" for k, v in sorted(pws.items())) + "\n")
w("| Vai | Route | `Page` | page px | số panel | panel px (tối đa 6) | lề trái → panel | lề phải ← panel | panel cao / viewport |\n| --- | --- | --- | --- | --- | --- | --- | --- | --- |")
for r in sorted(dat, key=lambda r: (ROLES.index(r["role"]), r["route"])):
    p0 = r["panels"]["list"][0] if r["panels"]["list"] else None
    w(f"| {r['role']} | `{r['route']}` | {r['kind']} | {r['pageW'] or '–'} | {r['panels']['n']} | {pw(r)} | {p0['gapL'] if p0 else '–'} | {p0['gapR'] if p0 else '–'} | {p0['hVsVp'] if p0 else '–'} |")

# K2 đầu trang
w("\n## K2. Đầu trang (1440, có dữ liệu)\n")
w("Hợp đồng: tiêu đề · ≤ 1 dòng meta (không lặp lớp) · hành động chính bên phải **cùng hàng** tiêu đề · ≤ 2 phụ.\n")
w("| Vai | Route | nút đầu trang (kiểu) | vị trí nút | dòng meta | meta lặp lớp | dòng mô tả dưới tiêu đề | nút primary ngoài đầu trang |\n| --- | --- | --- | --- | --- | --- | --- | --- |")
bad = collections.defaultdict(list)
for r in sorted(dat, key=lambda r: (ROLES.index(r["role"]), r["route"])):
    h = r["header"]; btns = "; ".join(f"{b['t']} ({b['v']})" for b in h["btns"]) or "–"
    prim_out = "; ".join(r["primaryOutsideHeader"]) or "–"
    w(f"| {r['role']} | `{r['route']}` | {btns} | {h['actionsPos']} | {h['metaLines']} | {'**có**' if h['metaDupClass'] else '–'} | {h['descLines'] or '–'} | {prim_out} |")
    if h["actionsPos"] == "dưới tiêu đề": bad["K2a nút đầu trang nằm DƯỚI tiêu đề"].append(f"{r['role']}{r['route']}")
    if h["metaLines"] > 1: bad["K2b meta > 1 dòng"].append(f"{r['role']}{r['route']}")
    if h["metaDupClass"]: bad["K2c meta lặp mã / tên lớp"].append(f"{r['role']}{r['route']}")
    if h["descLines"]: bad["K2d dòng mô tả dưới tiêu đề trang"].append(f"{r['role']}{r['route']}")
    if len(h["btns"]) > 3: bad["K2e > 3 nút ở đầu trang"].append(f"{r['role']}{r['route']}")
    if r["primaryTotal"] > 1: bad["K2f > 1 nút primary trên trang"].append(f"{r['role']}{r['route']} ({r['primaryTotal']})")

# K4 tiêu đề mục
w("\n## K3. Tiêu đề mục có dòng phụ\n")
w("| Vai | Route | dòng phụ dưới tiêu đề mục |\n| --- | --- | --- |")
for r in sorted(dat, key=lambda r: (ROLES.index(r["role"]), r["route"])):
    if r["sectionDescs"]:
        w(f"| {r['role']} | `{r['route']}` | {' / '.join(r['sectionDescs'])[:200]} |")
        bad["K3 tiêu đề mục có dòng phụ"].append(f"{r['role']}{r['route']} ({len(r['sectionDescs'])})")

# K5 rỗng
w("\n## K4. Trạng thái rỗng (lớp QCEMPTY, 1440)\n")
w("Hợp đồng: một mẫu duy nhất — tiêu đề ngắn + ≤ 1 nút, **căn giữa** vùng, **không thân chữ**, không nút trùng đầu trang, panel không kéo hết màn.\n")
emp = d("empty", 1440)
w("| Vai | Route | khối rỗng | icon | tiêu đề | thân chữ | nút (kiểu) | lệch tâm (px) | panel cha cao / viewport | nút trùng đầu trang |\n| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |")
sig = collections.Counter()
for r in sorted(emp, key=lambda r: (ROLES.index(r["role"]), r["route"])):
    if not r["empties"]:
        w(f"| {r['role']} | `{r['route']}` | **không dùng `EmptyState`** | – | – | – | – | – | – | – |"); continue
    for e in r["empties"]:
        s = ("icon+" if e["icon"] else "") + "tiêu đề" + ("+thân" if e["bodies"] else "") + (f"+{len(e['actions'])} nút" if e["actions"] else "")
        sig[s] += 1
        w(f"| {r['role']} | `{r['route']}` | {s} | {'có' if e['icon'] else '–'} | {e['title'][:60]} | {' / '.join(e['bodies'])[:90] or '–'} | {'; '.join(a['t']+' ('+a['v']+')' for a in e['actions']) or '–'} | {e['cxOff']} | {e['parentVsVp']} | {'**'+', '.join(r['dupHeaderEmpty'])+'**' if r['dupHeaderEmpty'] else '–'} |")
        who = f"{r['role']}{r['route']}"
        if e["bodies"]: bad["K4a EmptyState có thân chữ"].append(who)
        if abs(e["cxOff"]) > 40: bad["K4b khối rỗng không căn giữa (lệch tâm > 40 px)"].append(f"{who} ({e['cxOff']})")
        if e["parentVsVp"] > 0.8: bad["K4c panel cha kéo cao ≥ 0,8 viewport khi rỗng"].append(f"{who} ({e['parentVsVp']})")
        if len(e["actions"]) > 1: bad["K4d > 1 nút trong khối rỗng"].append(who)
        if e["icon"]: bad["K4e khối rỗng có icon (mẫu không thống nhất)"].append(who)
    if r["dupHeaderEmpty"]: bad["K4f nút rỗng trùng nút đầu trang"].append(f"{r['role']}{r['route']} ({', '.join(r['dupHeaderEmpty'])})")
    # meta lặp lớp ở trạng thái rỗng
    if r["header"]["metaDupClass"]: bad["K2c meta lặp mã / tên lớp"].append(f"{r['role']}{r['route']} [rỗng]")
    if r["header"]["metaLines"] > 1: bad["K2b meta > 1 dòng"].append(f"{r['role']}{r['route']} [rỗng]")
    if r["header"]["actionsPos"] == "dưới tiêu đề": bad["K2a nút đầu trang nằm DƯỚI tiêu đề"].append(f"{r['role']}{r['route']} [rỗng]")
w("\nKiểu khối rỗng: " + "; ".join(f"`{k}` × {v}" for k, v in sig.most_common()) + f" → **{len(sig)}** kiểu khác nhau (hợp đồng: 1).\n")
noes = [f"{r['role']}{r['route']}" for r in emp if not r["empties"] and r["role"] != "admin"]
w("Route rỗng nhưng **không** dùng `EmptyState` (xem ảnh để biết có trạng thái rỗng tự dựng hay còn dữ liệu): " + (", ".join(noes) or "–") + "\n")
loose_all = []
# K6 nút trùng
w("\n## K5. Nút trùng trong cùng màn (cả hai trạng thái, 1440)\n")
w("| Vai | Route | trạng thái | nút xuất hiện > 1 lần |\n| --- | --- | --- | --- |")
for r in sorted([x for x in rows if x["w"] == 1440 and ok(x)], key=lambda r: (ROLES.index(r["role"]), r["route"], r["state"])):
    if r["dupButtons"]:
        w(f"| {r['role']} | `{r['route']}` | {r['state']} | {', '.join(r['dupButtons'])[:160]} |")

# K7 375
w("\n## K6. 375 px\n")
w("| Vai | Route | trạng thái | tràn ngang | page px |\n| --- | --- | --- | --- | --- |")
for r in sorted([x for x in rows if x["w"] == 375 and ok(x)], key=lambda r: (ROLES.index(r["role"]), r["route"], r["state"])):
    if r["overflowX"]:
        w(f"| {r['role']} | `{r['route']}` | {r['state']} | **có** | {r['pageW']} |")
        bad["K6 tràn ngang ở 375 px"].append(f"{r['role']}{r['route']} [{r['state']}]")

# Tổng hợp vi phạm
w("\n## Tổng hợp theo hợp đồng (số mục lệch hiện trạng)\n")
w("| Mã | Quy tắc / lỗi | Số mục | Danh sách (vai+route) |\n| --- | --- | --- | --- |")
for k in sorted(bad): 
    v = sorted(set(bad[k])); w(f"| {k.split(' ')[0]} | {' '.join(k.split(' ')[1:])} | {len(v)} | {', '.join(v)[:600]} |")
errs = [r for r in rows if not ok(r)]
if errs:
    w("\n## Lượt chụp lỗi\n"); [w(f"- {r['role']} {r['state']} {r['w']} `{r['route']}`: {r.get('err') or 'không có <main>'}") for r in errs]
head = ["## Kết quả chính (số đo DOM, hiện trạng)\n"]
head.append(f"- **Độ rộng:** 3 kiểu `Page` (`wide` {kinds.get('wide',0)}, `reading` {kinds.get('reading',0)}, `full` {kinds.get('full',0)} trang) và **{len(pws)}** độ rộng panel khác nhau (hợp đồng: 2 kiểu, panel theo kiểu).")
for k, label in [("K2a", "nút đầu trang nằm dưới tiêu đề"), ("K2b", "meta > 1 dòng"), ("K2c", "meta lặp mã / tên lớp"), ("K2d", "dòng mô tả dưới tiêu đề trang"), ("K2f", "> 1 nút primary trên trang"), ("K3", "tiêu đề mục có dòng phụ"), ("K4a", "khối rỗng có thân chữ"), ("K4b", "khối rỗng không căn giữa (> 40 px)"), ("K4c", "panel kéo cao ≥ 0,8 viewport khi rỗng"), ("K4d", "> 1 nút trong khối rỗng"), ("K4e", "khối rỗng có icon"), ("K4f", "nút rỗng trùng nút đầu trang"), ("K6", "tràn ngang ở 375 px")]:
    n = len({x for kk, v in bad.items() if kk.startswith(k + " ") for x in v})
    head.append(f"- **{k}** {label}: **{n}** mục")
head.append(f"- **Kiểu khối rỗng:** {len(sig)} kiểu khác nhau (hợp đồng: 1).\n")
_i = next(i for i, x in enumerate(L) if x.startswith("## K1."))
L[_i:_i] = head
open(os.path.join(QC, "audit-truoc.md"), "w").write("\n".join(L) + "\n")

# HTML lưới
H = ["<!doctype html><meta charset=utf-8><title>Audit khung trang 10.7</title><style>body{font:13px system-ui;margin:16px;background:#faf7f5}h2{margin:28px 0 6px}table{border-collapse:collapse}td{vertical-align:top;padding:4px;border:1px solid #ddd}img{display:block;border:1px solid #bbb;background:#fff}.m{color:#555;font-size:11px;max-width:300px}.b{color:#b00020;font-weight:600}.t{font-weight:600}</style><h1>Audit khung trang — hiện trạng (QC 10.7)</h1><p>Mỗi hàng = một route của một vai. Cột: có dữ liệu 1440 · rỗng 1440 · có dữ liệu 375 · rỗng 375. Số liệu dưới ảnh đo từ DOM.</p>"]
for role in ROLES:
    H.append(f"<h2>Vai: {role}</h2><table>")
    for (ro, route) in [k for k in routes if k[0] == role]:
        H.append(f"<tr><td class=t>{html.escape(route)}</td>")
        for st, w_ in (("data", 1440), ("empty", 1440), ("data", 375), ("empty", 375)):
            r = get(role, route, st, w_)
            if not ok(r): H.append("<td class=m>–</td>"); continue
            h = r["header"]; p0 = r["panels"]["list"][0]["w"] if r["panels"]["list"] else "–"
            flags = []
            if h["actionsPos"] == "dưới tiêu đề": flags.append("nút dưới tiêu đề")
            if h["metaLines"] > 1: flags.append(f"meta {h['metaLines']} dòng")
            if h["metaDupClass"]: flags.append("meta lặp lớp")
            if h["descLines"]: flags.append("mô tả dưới tiêu đề")
            if r["sectionDescs"]: flags.append(f"{len(r['sectionDescs'])} dòng phụ mục")
            if r["primaryTotal"] > 1: flags.append(f"{r['primaryTotal']} nút primary")
            if r["overflowX"]: flags.append("tràn ngang")
            for e in r["empties"]:
                if e["bodies"]: flags.append("rỗng có thân chữ")
                if abs(e["cxOff"]) > 40: flags.append(f"rỗng lệch tâm {e['cxOff']}")
                if e["parentVsVp"] > .8: flags.append("panel cao hết màn")
            if r["dupHeaderEmpty"]: flags.append("nút rỗng trùng đầu trang")
            cap = f"{r['kind']} · page {r['pageW']} · panel {p0}" + (f"<br><span class=b>{' · '.join(flags)}</span>" if flags else "")
            wd = 300 if w_ == 1440 else 150; img = r['img']
            H.append(f"<td><img loading=lazy width={wd} src='audit/truoc/{img}'><div class=m>{cap}</div></td>")
        H.append("</tr>")
    H.append("</table>")
open(os.path.join(QC, "audit-truoc.html"), "w").write("".join(H))
print("audit-truoc.md", len(L), "dòng; vi phạm:", {k: len(set(v)) for k, v in bad.items()})

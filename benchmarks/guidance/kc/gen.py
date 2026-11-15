#!/usr/bin/env python3
"""Sinh bộ bài mô phỏng cho kiểm chứng «học từ lần sửa» (kiem-chung.md §1.2). Chỉ thư viện chuẩn; gọi LLM CHỈ qua 9router.

  python3 benchmarks/guidance/kc/gen.py                 sinh đủ 60 bài BT-A + 24 bài BT-B (≈ 95 lời gọi, tuần tự) vào kc/data/
  python3 benchmarks/guidance/kc/gen.py --limit 3 --out /tmp/kc-thu    chạy thử: chỉ sinh 3 bài đầu, ghi chỗ khác
Biến: ROUTER_URL (http://127.0.0.1:20128/v1), ROUTER_KEY (mặc định đọc ~/.9router/db/data.sqlite, không in ra), GEN_MODEL (ag/gemini-3.8-flash-medium — nên KHÁC model chấm của stack; 9router hiện chỉ phục vụ được họ gemini-3.8-flash, các model khác trả 404), GEN_GAP (1.0 s giữa hai lời gọi), GEN_SEED (106).
Chạy lại không sinh lại bài đã có (tiếp tục được). Bài chứa họ tên / email / SĐT / MSSV bị loại và sinh lại ≤ 2 lần, vẫn trượt thì đổi hồ sơ (≤ 2 lần) rồi mới dừng.
Cấm gọi Gemini trực tiếp: script chỉ biết địa chỉ 9router.
"""
import argparse, json, os, random, re, sqlite3, sys, time, urllib.error, urllib.request
from pathlib import Path
from common import DATA, N_B, N_STUDENTS, bt_students, die

ROUTER = os.environ.get("ROUTER_URL", "http://127.0.0.1:20128/v1").rstrip("/")
MODEL = os.environ.get("GEN_MODEL", "ag/gemini-3.8-flash-medium")
GAP = float(os.environ.get("GEN_GAP", "1.0"))
SEED = os.environ.get("GEN_SEED", "106")
CRITERIA = ["Luận điểm rõ", "Dẫn chứng / tình huống cụ thể", "Lập luận và biện pháp khả thi", "Trình bày, thuật ngữ"]
TOPIC = {
    "bt-a": "Phân tích một sự cố lộ dữ liệu giả định ở doanh nghiệp vừa và nhỏ và đề xuất biện pháp phòng ngừa",
    "bt-b": "So sánh xác thực một yếu tố và đa yếu tố cho một tổ chức 50 người",
}
RUBRIC = [{"name": n, "max_score": "2.5", "guidance": g} for n, g in zip(CRITERIA, [
    "Nêu rõ vấn đề, phạm vi và kết luận chính ngay từ đầu bài.",
    "Có tình huống hoặc số liệu cụ thể gắn với doanh nghiệp được nêu, không chỉ nói chung chung.",
    "Biện pháp gắn với nguyên nhân đã phân tích, khả thi với nguồn lực của tổ chức, có thứ tự ưu tiên.",
    "Văn phong học thuật, thuật ngữ dùng đúng, bố cục rõ ràng."])]
KINDS = [("long_vague", 0.15), ("short_tight", 0.15), ("jargon_generic", 0.15), ("normal", 0.55)]
# từ viết hoa được phép đứng liền nhau (thuật ngữ, thương hiệu), không tính là họ tên
ALLOW = {"Multi", "Factor", "Authentication", "Zero", "Trust", "Microsoft", "Google", "Windows", "Linux", "Android", "Office", "Active", "Directory", "Single", "Sign", "On", "Man", "In", "The", "Middle", "Data", "Loss", "Prevention", "Việt", "Nam", "Hà", "Nội", "Hồ", "Chí", "Minh", "Chính", "Phủ",
         "Least", "Privilege", "Phishing", "Ransomware", "Business", "Email", "Compromise", "Incident", "Response", "Network", "Access", "Control", "Security", "Information", "Management", "System", "Breach", "Principle", "Defense", "Depth", "Role", "Based", "Identity", "Cloud", "Service", "Provider", "Time", "Password", "Manager", "Awareness", "Training", "Virtual", "Private", "Endpoint", "Detection", "Day", "Social", "Engineering", "Intrusion", "Prevention", "Backup", "Recovery", "Disaster", "Plan", "Risk", "Assessment", "Penetration", "Testing", "Firewall", "Open", "Web", "Application", "Top", "Supply", "Chain", "Insider", "Threat", "Time", "One", "Token", "Hardware", "Security", "Key", "Public", "Wi", "Fi", "Wifi", "Man", "Remote", "Desktop", "Protocol", "Transport", "Layer", "Outlook", "Excel", "Word", "Teams", "Drive", "Dropbox", "Facebook", "Gmail", "Yahoo", "Zalo", "Viettel", "VNPT", "FPT", "Authenticator", "Workspace", "Duo", "Okta", "Auth", "Push", "Notification", "Smart", "Card", "Time", "Sign"}


def router_key():
    k = os.environ.get("ROUTER_KEY")
    if k:
        return k
    try:
        c = sqlite3.connect("file:" + os.path.expanduser("~/.9router/db/data.sqlite") + "?mode=ro", uri=True)
        row = c.execute("select key from apiKeys where isActive = 1 limit 1").fetchone()
    except sqlite3.Error as e:
        die(f"Không đọc được khoá 9router ({e}); đặt ROUTER_KEY")
    if not row:
        die("9router chưa có khoá API; đặt ROUTER_KEY")
    return row[0]


def profile(bt, i, salt):
    """Hồ sơ ẩn: mức 0,5–2,5 (bước 0,25) cho 4 tiêu chí + kiểu bài. Cố định theo (seed, bt, i, salt); tổng ~ chuẩn, tâm 6,5."""
    r = random.Random(f"{SEED}-{bt}-{i}-{salt}")
    kind = r.choices([k for k, _ in KINDS], [w for _, w in KINDS])[0]
    lv = [min(2.5, max(0.5, r.gauss(1.625, 0.5))) for _ in range(4)]
    if kind == "long_vague":
        lv[1] = r.uniform(0.5, 1.0); lv[0] = max(lv[0], 1.75); lv[3] = max(lv[3], 1.75)
    elif kind == "short_tight":
        lv[0] = max(lv[0], 2.0); lv[2] = max(lv[2], 2.0)
    elif kind == "jargon_generic":
        lv[3] = max(lv[3], 2.0); lv[2] = r.uniform(0.5, 1.0)
    lv = [round(x * 4) / 4 for x in lv]
    return {"kind": kind, "levels": lv, "total": sum(lv)}


def describe(level):
    return ("xuất sắc, rất thuyết phục" if level >= 2.25 else "tốt, chỉ có vài chỗ chưa chặt" if level >= 1.75
            else "trung bình, có chỗ mờ" if level >= 1.25 else "yếu, nhiều chỗ thiếu hoặc chung chung" if level >= 0.75 else "rất yếu, gần như không đạt")


def prompt(bt, p):
    words = "khoảng 250–300" if p["kind"] == "short_tight" else "khoảng 400–450" if p["kind"] == "long_vague" else "khoảng 300–400"
    extra = {"long_vague": "Bài dài và trôi chảy nhưng ít dẫn chứng cụ thể.", "short_tight": "Bài ngắn gọn nhưng chặt chẽ.",
             "jargon_generic": "Bài dùng nhiều thuật ngữ nhưng biện pháp đưa ra còn chung chung.", "normal": ""}[p["kind"]]
    lines = "\n".join(f"- {n}: {describe(l)}" for n, l in zip(CRITERIA, p["levels"]))
    return (f"Bạn đóng vai một sinh viên năm ba ngành An toàn thông tin viết bài tự luận cho học phần An ninh mạng.\nĐề bài: {TOPIC[bt]}.\n"
            f"Viết {words} từ bằng tiếng Việt, văn xuôi liên tục (không tiêu đề, không gạch đầu dòng dài), chất lượng từng mặt đúng như sau:\n{lines}\n{extra}\n"
            "Tuyệt đối KHÔNG ghi họ tên, email, số điện thoại, mã số sinh viên hay tên người thật; KHÔNG nhắc tới chất lượng hay điểm số trong bài. Chỉ trả về nội dung bài viết.")


def pii_hits(text):
    hits = []
    if re.search(r"[\w.+-]+@[\w-]+\.[\w.]+", text): hits.append("email")
    if re.search(r"(?<!\d)(?:\+?84|0)[\s.-]?\d{2,3}[\s.-]?\d{3}[\s.-]?\d{3,4}(?!\d)", text): hits.append("điện thoại")
    if re.search(r"(?<!\d)\d{8,10}(?!\d)", text): hits.append("chuỗi 8–10 chữ số")
    if re.search(r"\b[A-Z]{1,4}\d{4,}\b", text): hits.append("mã số")
    for sent in re.split(r"(?<=[.!?:])\s+|\n+", text):
        ws = [w.strip(",;:()\"'“”") for w in sent.split()][1:]  # chữ đầu câu viết hoa vì ngữ pháp
        run = []
        for w in ws + [""]:
            if len(w) > 1 and w[0].isupper() and w[1:].islower():
                run.append(w)
                continue
            if len(run) >= 2 and not all(x in ALLOW for x in run):
                hits.append("họ tên: " + " ".join(run))
            run = []
    return hits


def parse_reply(raw):
    """9router trả JSON một khối hoặc SSE `data: {chunk}` (model ag/*): ghép cả hai về văn bản."""
    txt = raw.decode("utf8", "replace")
    if txt.lstrip().startswith("data:"):
        out = []
        for ln in txt.splitlines():
            if ln.startswith("data:") and "[DONE]" not in ln:
                ch = json.loads(ln[5:]).get("choices") or [{}]
                out.append((ch[0].get("delta") or ch[0].get("message") or {}).get("content") or "")
        return "".join(out).strip()
    return (json.loads(txt)["choices"][0]["message"].get("content") or "").strip()


def ask(key, content):
    body = json.dumps({"model": MODEL, "messages": [{"role": "user", "content": content}], "temperature": 0.9, "max_tokens": 4000, "stream": False}).encode()
    for n in range(6):
        req = urllib.request.Request(ROUTER + "/chat/completions", data=body, headers={"Authorization": "Bearer " + key, "Content-Type": "application/json"})
        try:
            with urllib.request.urlopen(req, timeout=180) as r:
                return parse_reply(r.read())
        except urllib.error.HTTPError as e:
            msg = e.read()[:300]
            if e.code in (429, 502, 503) and b"NOT_FOUND" not in msg and n < 5:
                time.sleep(min(60, float(e.headers.get("Retry-After") or 2 * (n + 1))))
                continue
            die(f"9router → {e.code} (model {MODEL}): {msg!r}")
        except (urllib.error.URLError, TimeoutError) as e:
            die(f"Không nối được 9router ({ROUTER}): {e}", 2)
    die("9router quá tải")


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("--limit", type=int, default=0, help="chỉ sinh N bài đầu (chạy thử)")
    ap.add_argument("--out", type=Path, default=DATA)
    a = ap.parse_args()
    key = router_key()
    out: Path = a.out
    jobs = [(bt, s) for bt in ("bt-a", "bt-b") for s in bt_students(bt)]
    if a.limit:
        jobs = jobs[: a.limit]
    for bt in ("bt-a", "bt-b"):
        (out / bt).mkdir(parents=True, exist_ok=True)
    meta = {"gen_model": MODEL, "router": ROUTER, "seed": SEED, "calls": 0, "regenerated": 0, "profile_changed": 0}
    prof_path = out / "profile.jsonl"
    done = {(j["bt"], j["key"]) for j in map(json.loads, prof_path.read_text().splitlines())} if prof_path.exists() else set()
    for n, (bt, s) in enumerate(jobs, 1):
        path = out / bt / f"{s['code']}.txt"
        if path.exists() and (bt, s["key"]) in done:
            continue
        text, p = None, None
        for salt in range(3):  # đổi hồ sơ tối đa 2 lần khi bài cứ trượt bộ lọc
            p = profile(bt, s["i"], salt)
            for attempt in range(3):
                if meta["calls"]:
                    time.sleep(GAP)
                meta["calls"] += 1
                t = ask(key, prompt(bt, p))
                bad = pii_hits(t) if t else ["rỗng"]
                words = len(t.split())
                if not bad and 200 <= words <= 520:
                    text = t
                    break
                meta["regenerated"] += 1
                print(f"  [{bt} {s['key']}] sinh lại ({', '.join(bad) or f'{words} từ'})", file=sys.stderr)
            if text:
                break
            meta["profile_changed"] += 1
        if not text:
            die(f"Không sinh được bài sạch cho {bt} {s['key']}")
        path.write_text(text + "\n", encoding="utf8")
        with prof_path.open("a", encoding="utf8") as f:
            f.write(json.dumps({"bt": bt, "key": s["key"], "code": s["code"], **p}, ensure_ascii=False) + "\n")
        print(f"[{n}/{len(jobs)}] {bt} {s['key']} {p['kind']} {len(text.split())} từ")
    (out / "rubric.json").write_text(json.dumps({"rubric": RUBRIC, "topic": TOPIC, "max_score": "10"}, ensure_ascii=False, indent=2) + "\n", encoding="utf8")
    (out / "order.json").write_text(json.dumps({"bt-a": {"learn": [s["key"] for s in bt_students("bt-a")[:20]], "measure": [s["key"] for s in bt_students("bt-a")[20:]]},
                                                "bt-b": {"measure": [s["key"] for s in bt_students("bt-b")]}}, indent=2) + "\n")
    (out / "gen-meta.json").write_text(json.dumps(meta, indent=2) + "\n")
    print(f"Xong: {meta['calls']} lời gọi ({meta['regenerated']} lần sinh lại) qua {ROUTER}, model {MODEL}")


if __name__ == "__main__":
    main()

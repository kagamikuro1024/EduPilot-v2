# QC S10 — API lượt A: US-P9-01 (phần API), 05, 06, 07. Chạy: python3 pa.py (stack ep-qc đã seed)
import sys, re, json, uuid, itertools
sys.path.insert(0, __import__("os").path.dirname(__file__))
from qlib import *

C1 = course("sv.kha"); C2 = None
try: C2 = course("teacher", "761988")
except Exception: pass
print("C1", C1, "C2", C2)
P = f"/courses/{C1}"
SENS = re.compile(r'answer_key|"correct"|explanation|citations|is_correct', re.I)

def bank(user="teacher", C=C1):
    out, cur = [], None
    while True:
        c, j, _ = api(user, "GET", f"/courses/{C}/questions?limit=100" + (f"&cursor={cur}" if cur else ""))
        assert c == 200, (c, j)
        out += j["items"]; cur = j.get("next_cursor")
        if not cur: return out
BANK = bank()
print("bank", len(BANK), BANK[0].keys() if BANK else None)
def detail(qid, C=C1):
    c, j, _ = api("teacher", "GET", f"/courses/{C}/questions/{qid}"); return j

def start(u, body, idem=None, C=C1, headers=None):
    return api(u, "POST", f"/courses/{C}/practice/start", body, idem=idem, headers=headers)
def att(u, aid, C=C1, headers=None): return api(u, "GET", f"/courses/{C}/practice/attempts/{aid}", headers=headers)
def put(u, aid, iid, ans, C=C1, version=None, ts=None, tab=None):
    b = {"answer": ans}
    if version is not None: b["version"] = version
    if ts: b["client_ts"] = ts
    return api(u, "PUT", f"/courses/{C}/practice/attempts/{aid}/answers/{iid}", b, headers={"X-Practice-Tab": tab} if tab else None)
def submit(u, aid, idem=None, C=C1, tab=None): return api(u, "POST", f"/courses/{C}/practice/attempts/{aid}/submit", {}, idem=idem, headers={"X-Practice-Tab": tab} if tab else None)

# khoá đúng theo ngân hàng GV: stem → (type, correct option ids / value)
KEY = {}
def load_keys():
    for q in BANK:
        if q["review_status"] != "APPROVED" or q["type"] not in ("MCQ_SINGLE", "MCQ_MULTI", "TRUE_FALSE"): continue
        d = detail(q["id"]); KEY[d["stem"]] = d
def correct_answer(item):
    d = KEY.get(item["question"]["stem"])
    if not d: return None
    ak = d["answer_key"] or {}
    return d, ak

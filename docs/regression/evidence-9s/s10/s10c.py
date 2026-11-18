# Smoke S10 lô C (API): TC-10-06 DTO sinh viên so với allowlist + quét header/SSE; TC-07-20 EXPIRED idempotent
from pm import *
import threading, urllib.request
AL = json.load(open(os.path.expanduser("~/Documents/TA_Agent_qc106/backend-go/internal/practice/testdata/student_dto_allowlist.json")))
ALL = set(x for v in AL.values() for x in v)
FORB_MOCK = {"correct", "explanation", "key", "citations", "answer_key", "is_correct", "correct_label"}
def keys(o, acc=None):
    acc = acc if acc is not None else set()
    if isinstance(o, dict):
        for k, v in o.items(): acc.add(k); keys(v, acc)
    elif isinstance(o, list):
        for v in o: keys(v, acc)
    return acc
T = "QC-r10-Y"; U = "sv.kha"
finish_open(U)
# SSE: nghe /events của sv.kha trong lúc thao tác
sse_buf = []
def listen():
    req = urllib.request.Request(B + "/events", headers={"authorization": "Bearer " + tok(U), "accept": "text/event-stream"})
    try:
        with urllib.request.urlopen(req, context=CTX, timeout=25) as f:
            t0 = time.time()
            while time.time() - t0 < 20:
                ln = f.readline()
                if not ln: break
                sse_buf.append(ln.decode("utf8", "replace"))
    except Exception as e: sse_buf.append("ERR " + str(e)[:80])
th = threading.Thread(target=listen); th.start(); time.sleep(1.5)
bodies, hdrs = [], []
def grab(r): bodies.append(r[1]); return r
c, ov, _ = api(U, "GET", f"{P}/practice/overview"); bodies.append(ov)
cT, aT, _ = start(U, {"mode": "TOPIC", "topic": T, "count": 5}); bodies.append(aT)
it = aT["items"][0]; r1 = put(U, aT["id"], it["id"], {"option_ids": [it["question"]["options"][0]["id"]]}); bodies.append(r1[1])
rg = att(U, aT["id"]); bodies.append(rg[1]); rs = submit(U, aT["id"]); bodies.append(rs[1])
c, h, _ = api(U, "GET", f"{P}/practice/history"); bodies.append(h); c, w, _ = api(U, "GET", f"{P}/practice/weakness"); bodies.append(w)
finish_open(U); cM, aM, _ = start(U, {"mode": "MOCK", "count": 10, "duration_minutes": 20, "matrix": [{"topic": T, "count": 10}]}); bodies.append(aM)
iM = aM["items"][0]; rm = put(U, aM["id"], iM["id"], {"option_ids": [iM["question"]["options"][0]["id"]]}, version=0, tab=str(uuid.uuid4())); bodies.append(rm[1]); assert rm[0] == 200, rm; rmg = att(U, aM["id"]); mockbody = rmg[1]
# header thô của vài phản hồi
for pth in ("/practice/overview", f"/practice/attempts/{aM['id']}", "/practice/history"):
    rq = urllib.request.Request(B + P.replace("/courses", "/courses", 1) + pth, headers={"authorization": "Bearer " + tok(U)})
    with urllib.request.urlopen(rq, context=CTX) as f: hdrs.append(dict(f.headers))
th.join(); submit(U, aM["id"])
allk = set(); [keys(b, allk) for b in bodies]
where = [i for i, b in enumerate(bodies) if "field" in keys(b)]
extra = sorted(k for k in allk if k not in ALL and k not in ("code", "message", "trace_id", "details", "items", "next_cursor", "id", "option_ids", "value", "text", "accepted"))  # option_ids/value/text/accepted = nội dung đáp án (answer/key), không phải khoá DTO
mk = keys(mockbody); leak_mock = sorted(FORB_MOCK & mk) + [("feedback", "!=null")] * any((it_.get("feedback") for it_ in mockbody.get("items", [])))
hs = json.dumps(hdrs).lower(); sse = "".join(sse_buf)
hh = [k for k in ("answer", "correct", "stem", "explanation") if k in hs]
ss = [k for k in ("answer_key", "correct", "stem", "explanation", "citations", "is_correct") if k in sse.lower()]
chk("TC-10-06", not extra and not leak_mock and not hh and not ss, f"{len(bodies)} phản hồi SV (overview/start/put/get/submit/history/weakness, TOPIC+MOCK): khoá ngoài allowlist = {extra} (thân có «field»: {where} = {[str(bodies[i])[:90] for i in where]}); MOCK đang làm có khoá lộ đáp án = {leak_mock}; header chứa {hh}; SSE /events {len(sse)} byte chứa {ss} (khung: {sse[:80]!r})")
# ---- TC-07-20 EXPIRED idempotent
finish_open("sv.gioi"); c, a, _ = start("sv.gioi", {"mode": "MOCK", "count": 10, "duration_minutes": 20, "matrix": [{"topic": T, "count": 10}]})
KEYS = {d["stem"]: d for d in (detail(q["id"]) for q in bank() if q["topic"] == T and q["review_status"] == "APPROVED")}
nok = 0; TAB = str(uuid.uuid4())
for it_ in a["items"][:4]:
    d = KEYS.get(it_["question"]["stem"]); ids = ((d or {}).get("answer_key") or {}).get("option_ids") or []; ob = [o for o in it_["question"]["options"] if o["body"] in [x["body"] for x in (d or {}).get("options", []) if x["id"] in ids]]
    rr = put("sv.gioi", a["id"], it_["id"], {"option_ids": [ob[0]["id"] if ob else it_["question"]["options"][0]["id"]]}, version=0, tab=TAB); assert rr[0] == 200, rr; nok += bool(ob)
sql(f"update practice_attempts set deadline_at = now() - interval '120 seconds' where id='{a['id']}'")
st = None
for _ in range(40):
    st = sql(f"select status, correct_count from practice_attempts where id='{a['id']}'")[0]
    if st[0] != "IN_PROGRESS": break
    time.sleep(5)
t1 = sql(f"select status, correct_count, submitted_at from practice_attempts where id='{a['id']}'")[0]
time.sleep(70)
t2 = sql(f"select status, correct_count, submitted_at from practice_attempts where id='{a['id']}'")[0]
c, g, _ = att("sv.gioi", a["id"])
chk("TC-07-20", t1[0] == "EXPIRED" and t1[1] == str(nok) and t1 == t2 and c == 200 and g.get("result"), f"quá deadline+120 s: đã chọn đúng {nok}/4 câu; worker đóng → {t1[:2]} (correct_count kỳ vọng {nok}); sau thêm 1 tick (70 s): {t2[:2]} (không đổi: {t1 == t2}); mở lại: kết quả {bool(g.get('result'))}")
print("done s10c")

# QC sprint 10.8 — thư viện: gọi API thật ep-qc; kết quả ghi results-as.jsonl
import os, sys, json, time, uuid, base64, hashlib, subprocess, datetime
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import dmlib
dmlib.RES = os.path.join(os.path.dirname(os.path.abspath(__file__)), "results-as.jsonl")
from dmlib import raw, tok, api, ok, sql, PW, B  # noqa
ADMIN, TEACHER, TA = "admin", "r108t", "r108ta"
SV = ["sv.gioi", "sv.kha", "sv.nguyco", "sv04", "sv05", "sv06"]
STATE = os.path.join(os.path.dirname(os.path.abspath(__file__)), "state.json")
def st(): return json.load(open(STATE))
def uid(user): return sql(f"select id from users where email='{user}@edupilot.local'")[0][0]
def mkcourse(code, codejoin):
    c, j, _ = api(ADMIN, "POST", "/admin/courses", {"subject_code": "QCAS", "class_code": code, "name": "QC điểm danh " + code, "semester": "2026-2027-HK1", "teacher_id": uid(TEACHER), "ta_ids": [uid(TA)], "join_code": codejoin})
    assert c == 201, (c, j); cid = j["id"]
    for u in SV:
        c, j, _ = api(u, "POST", "/courses/join", {"code": codejoin}); assert c == 200, (u, c, j)
    return cid
def gen_sessions(cid, d0=-2, d1=6):
    t = datetime.date.today()
    c, j, _ = api(TEACHER, "POST", f"/courses/{cid}/sessions/generate", {"weekdays": [1,2,3,4,5,6,7], "start_time": "00:05", "end_time": "23:50", "from": str(t + datetime.timedelta(d0)), "to": str(t + datetime.timedelta(d1))})
    assert c == 201, (c, j)
def sessions(user, cid):
    out, cur = [], None
    while True:
        c, j, _ = api(user, "GET", f"/courses/{cid}/sessions?limit=100" + (f"&cursor={cur}" if cur else "")); assert c == 200, (c, j)
        out += j["items"]; cur = j.get("next_cursor")
        if not cur: return out
def policy(cid, **kw):
    for _t in range(6):
        c, j, _ = api(TEACHER, "GET", f"/courses/{cid}")
        if c == 200: break
        time.sleep(15)
    v = j["settings"]["version"]
    a = {"self_checkin": True, "rotate_seconds": 10}; a.update(kw)
    c, j, _ = api(TEACHER, "PUT", f"/courses/{cid}/settings", {"attendance": a, "version": v}); return c, j
def openw(cid, sid, minutes=None, user=TEACHER, idem=None):
    return api(user, "POST", f"/courses/{cid}/sessions/{sid}/checkin-windows", ({"minutes": minutes} if minutes else {}), idem=idem)
def display(cid, sid, wid, user=TEACHER): return api(user, "GET", f"/courses/{cid}/sessions/{sid}/checkin-windows/{wid}/display")
def tokenof(d): return d["qr_url"].split("#", 1)[1]
def checkin(user, cid, body): return api(user, "POST", f"/courses/{cid}/checkins", body)
def close(cid, wid, user=TEACHER, idem=None): return api(user, "POST", f"/courses/{cid}/checkin-windows/{wid}/close", None, idem=idem)
def rows(cid, sid, user=TEACHER, q=""): 
    c, j, _ = api(user, "GET", f"/courses/{cid}/sessions/{sid}/checkins?limit=100{q}"); return c, j

# --- khoá thiết bị giả (WEB_KEY) ---
from cryptography.hazmat.primitives.asymmetric import ec
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric.utils import decode_dss_signature
b64 = lambda b: base64.b64encode(b).decode()
class Key:
    def __init__(self):
        self.k = ec.generate_private_key(ec.SECP256R1())
        self.pub = b64(self.k.public_key().public_bytes(serialization.Encoding.DER, serialization.PublicFormat.SubjectPublicKeyInfo))
    def sign(self, msg, der=False):
        s = self.k.sign(msg.encode(), ec.ECDSA(hashes.SHA256()))
        if der: return b64(s)
        r, t = decode_dss_signature(s); return b64(r.to_bytes(32, "big") + t.to_bytes(32, "big"))
    def dev(self, user, token=None, code=None, der=False, kind="WEB_KEY"):
        msg = f"edupilot-checkin-v1|{token}|{uid(user)}" if token else f"edupilot-checkin-v1|code:{code}|{uid(user)}"
        return {"kind": kind, "public_key": self.pub, "signature": self.sign(msg, der)}
def dbrows(sid, wid=None):
    q = f"select u.email, c.channel, c.assurance, array_to_string(c.flags,','), c.token_age_ms from attendance_checkins c join users u on u.id=c.student_id where c.session_id='{sid}'" + (f" and c.window_id='{wid}'" if wid else "") + " order by c.created_at"
    return sql(q)
def fresh_session(cid, user=TEACHER):
    """Buổi chưa có cửa sổ trong state; hết thì sinh thêm 40 ngày."""
    for _ in range(2):
        S = st(); used = set(x[0] for x in sql(f"select distinct session_id from checkin_windows where course_id='{cid}'"))
        for s in ([x["id"] for x in sessions(TEACHER, cid)] if cid != S["cid"] else S["sessions"]):
            if s not in used: return s
        if cid != S["cid"]: raise RuntimeError("hết buổi lớp phụ")
        nd = S.get("nd", 41); gen_sessions(cid, nd, nd + 40); S["nd"] = nd + 41
        S["sessions"] = [x["id"] for x in sessions(TEACHER, cid)]; json.dump(S, open(STATE, "w"))
    raise RuntimeError("hết buổi")
def neww(cid, minutes=None, user=TEACHER):
    s = fresh_session(cid); c, j, _ = openw(cid, s, minutes=minutes, user=user); assert c == 201, (c, j)
    c, d, _ = display(cid, s, j["id"])
    if c == 429: time.sleep((d or {}).get("retry_after", 20) + 1); c, d, _ = display(cid, s, j["id"])
    assert c == 200, (c, d)
    return s, j["id"], d
def wait_step(d, extra=0.8):
    t = datetime.datetime.fromisoformat(d["step_expires_at"].replace("Z", "+00:00")) - datetime.datetime.now(datetime.timezone.utc)
    time.sleep(max(0, t.total_seconds()) + extra)

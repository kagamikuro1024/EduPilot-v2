# QC: dựng môi trường cho ca trước đây «không kiểm được»: GV thứ hai + lớp QCT2 (+ SV) trên ep-qc. Idempotent. Mật khẩu GV = SEED_DEFAULT_PASSWORD.
import urllib.request, re
from p10lib import *
EM = "qct2@edupilot.local"
def mails(to):
    j = json.load(urllib.request.urlopen(f"http://localhost:8525/api/v1/search?query=to:{to}&limit=5")); return j["messages"]
def ensure_teacher():
    if sql(f"select count(*) from users where email='{EM}'")[0][0] != "0": return
    c, j, _, _ = api("admin", "POST", "/admin/users", {"email": EM, "full_name": "GV QC Hai", "role": "TEACHER"}); assert c == 201, (c, j)
    time.sleep(3); m = mails(EM)[0]; txt = json.load(urllib.request.urlopen(f"http://localhost:8525/api/v1/message/{m['ID']}"))["Text"]
    tok_ = re.search(r"invite[^\s]*?[?&/]token=([\w-]+)|/invite/([\w-]+)|token=([\w-]+)", txt); t = next(g for g in tok_.groups() if g)
    c, j, _, _ = raw("POST", "/auth/accept-invite", {"token": t, "password": PW}, {"Idempotency-Key": str(uuid.uuid4())}); assert c in (200, 201), (c, j)
def ensure_course():
    r = sql("select id from courses where class_code='QCT2'")
    if r: return r[0][0]
    c, j, _, _ = api("admin", "POST", "/admin/courses", {"subject_code": "QCT", "class_code": "QCT2", "name": "QC lớp GV hai", "semester": "2026-2027-HK1", "capacity": 60, "teacher_id": uid("qct2")}); assert c in (200, 201), (c, j)
    return sql("select id from courses where class_code='QCT2'")[0][0]
if __name__ == "__main__":
    ensure_teacher(); cid = ensure_course(); print("QCT2", cid, sql("select join_code from courses where class_code='QCT2'") if False else "")
    print(api("qct2", "GET", "/me/courses")[1]["items"][0]["course"]["class_code"])

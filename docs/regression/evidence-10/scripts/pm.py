# Lô M→P/F (luật không lấy mẫu): dùng pa.py (C1, C2, start, put, submit, att, BANK) + res() ghi results_m.jsonl (kq: P/F/LT/Ka/Kb/Kc; dòng cuối thắng)
import sys, os, re, json, uuid, time, hashlib, urllib.request
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from pa import *
C2 = (sql("select id from courses where class_code='761988'") or [[None]])[0][0]
RM = os.path.join(os.path.dirname(os.path.abspath(__file__)), "results_m.jsonl")
def res(tc, kq, note=""):
    print(f"{kq} {tc} {note}"[:300])
    with open(RM, "a") as f: f.write(json.dumps({"tc": tc, "kq": kq, "note": note}, ensure_ascii=False) + "\n")
def chk(tc, cond, note=""): res(tc, "P" if cond else "F", note); return cond
def lastj(j, n=160): return json.dumps(j, ensure_ascii=False)[:n]
def mkq(user, body, C=C1): return api(user, "POST", f"/courses/{C}/questions", body)
def review(qid, dec, C=C1, user="teacher"):
    for _ in range(5):
        c0, d0, _ = api(user, "GET", f"/courses/{C}/questions/{qid}")
        if c0 == 200: break
        time.sleep(5)
    v = d0["version"]
    return api(user, "PUT", f"/courses/{C}/questions/{qid}/review", {"decision": dec, "version": v})
def approve(qid, C=C1, user="teacher"):
    review(qid, "REQUEST", C, user); return review(qid, "APPROVE", C, user)[0]
def mk_short(topic, n, tag, diff="EASY"):
    """n câu SHORT APPROVED, đáp án chấp nhận của câu i = f'{tag}-ok-{i}'. Trả [(id, stem, accepted)]"""
    out = []
    for i in range(n):
        acc = f"{tag}-ok-{i}"
        c, j, _ = mkq("teacher", {"type": "SHORT", "title": f"{tag} {i}", "topic": topic, "difficulty": diff, "stem": f"{tag} câu {i}: gõ {acc}", "accepted": [acc]})
        assert c == 201, (c, j); assert approve(j["id"]) == 200
        out.append((j["id"], f"{tag} câu {i}: gõ {acc}", acc))
    return out
def mk_mcq(topic, n, tag, diff="EASY"):
    out = []
    for i in range(n):
        c, j, _ = mkq("teacher", {"type": "MCQ_SINGLE", "title": f"{tag} {i}", "topic": topic, "difficulty": diff, "stem": f"{tag} câu {i}: chọn A?", "options": [{"body": "A đúng"}, {"body": "B sai"}, {"body": "C sai"}, {"body": "D sai"}], "correct": [0]})
        assert c == 201, (c, j); assert approve(j["id"]) == 200
        out.append((j["id"], f"{tag} câu {i}: chọn A?"))
    return out
def finish_open(user, C=C1):
    for aid, in [(r[0],) for r in sql(f"select pa.id from practice_attempts pa join users u on u.id=pa.student_id where u.email='{user}@edupilot.local' and pa.course_id='{C}' and pa.status='IN_PROGRESS'")]:
        api(user, "POST", f"/courses/{C}/practice/attempts/{aid}/submit", {})

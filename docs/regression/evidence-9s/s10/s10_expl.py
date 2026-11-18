# Smoke S10 — US-P9-06 explain (stub CHAT): TC-06-13/14/15/20 (QD10A, sv.kha)
from pm import *
BANKQ = {}
for r_ in sql(f"select stem, id from question_bank where course_id='{C1}' and stem not like '%|%' and stem not like E'%\\n%'"):
    if len(r_) == 2: BANKQ.setdefault(r_[0], r_[1])
def mode(m): open("/tmp/stub10_mode", "w").write(m)
def detail_of(item): return api("teacher", "GET", f"{P}/questions/{BANKQ[item['question']['stem']]}")[1]
def good(item):
    ak = detail_of(item).get("answer_key") or {}; return {"option_ids": ak.get("option_ids")}
def bad(item):
    g = good(item)["option_ids"]; return {"option_ids": [o["id"] for o in item["question"]["options"] if o["id"] not in g][:1]}
mode("ok"); T = "QC-r10-Y"; ORIG = sql("select full_name from users where email='sv.kha@edupilot.local'")[0][0]
PRED = "p.masked_prompt like '%giải thích một câu trắc nghiệm%'"
def pcount(): return int(sql(f"select count(*) from llm_audit a join llm_request_payloads p on p.audit_id=a.id where a.task='CHAT' and {PRED}")[0][0])
# dọn lượt đang dở của sv.kha
for (aid,) in sql(f"select id from practice_attempts where course_id='{C1}' and status='IN_PROGRESS' and student_id=(select id from users where email='sv.kha@edupilot.local')"): api("sv.kha", "POST", f"{P}/practice/attempts/{aid}/submit", {}, idem=str(uuid.uuid4()))
c, a, _ = api("sv.kha", "POST", f"{P}/practice/start", {"mode": "TOPIC", "topic": T, "count": 6}); assert c in (200, 201), (c, a)
aid, items = a["id"], a["items"]
for n, it in enumerate(items[:5]):
    r = api("sv.kha", "PUT", f"{P}/practice/attempts/{aid}/answers/{it['id']}", {"answer": bad(it) if n % 2 == 0 else good(it)}, headers={"X-Practice-Tab": "x"} if False else None)
print("tasks khác nhau:", sql("select distinct task from llm_audit where created_at > now() - interval '1 day'"))
def exp(i):
    r = sse("sv.kha", f"{P}/practice/attempts/{aid}/items/{items[i]['id']}/explain"); time.sleep(2.5); return r
def _unused(i): return sse("sv.kha", f"{P}/practice/attempts/{aid}/items/{items[i]['id']}/explain")
# --- TC-06-13: cache theo phiên bản câu. Sửa câu để có khoá mới, rồi: explain (1 lời gọi), lặp lại ≤10 s (0), sửa câu lần nữa, SV vào lượt mới có câu đã sửa → explain (1 lời gọi mới)
qid = sql(f"select question_id from practice_items where id='{items[0]['id']}'")[0][0]
def edit(tag):
    qv = api("teacher", "GET", f"{P}/questions/{qid}")[1]; body = {k: qv[k] for k in ("type", "title", "topic", "difficulty", "stem", "version") if k in qv}
    base = re.sub(r" \(sửa [^)]*\)$", "", qv["stem"]); body["stem"] = f"{base} (sửa {tag})"; body["options"] = [{"body": o["body"]} for o in qv["options"]]
    ids_ = (qv.get("answer_key") or {}).get("option_ids") or []; body["correct"] = [i for i, o in enumerate(qv["options"]) if o["id"] in ids_]
    r = api("teacher", "PUT", f"{P}/questions/{qid}", body); st0 = sql(f"select review_status from question_bank where id='{qid}'")[0][0]; print("sau sửa:", r[0], st0)
    if st0 != "APPROVED": review(qid, "REQUEST"); print("duyệt lại", review(qid, "APPROVE")[0])
    BANKQ[body["stem"]] = qid; return r[0], body["stem"]
def attempt_with(stem):
    for _ in range(14):
        for (x,) in sql(f"select id from practice_attempts where course_id='{C1}' and status='IN_PROGRESS' and student_id=(select id from users where email='sv.kha@edupilot.local')"): api("sv.kha", "POST", f"{P}/practice/attempts/{x}/submit", {}, idem=str(uuid.uuid4()))
        c_, a_, _ = api("sv.kha", "POST", f"{P}/practice/start", {"mode": "TOPIC", "topic": T, "count": 10})
        for it in a_.get("items", []):
            if it["question"]["stem"] == stem: return a_["id"], it
    return None, None
def ans_explain(aid_, it):
    api("sv.kha", "PUT", f"{P}/practice/attempts/{aid_}/answers/{it['id']}", {"answer": bad(it)})
    return sse("sv.kha", f"{P}/practice/attempts/{aid_}/items/{it['id']}/explain"), time.sleep(2.5)
def cached(tx): return (re.findall(r'"cached":(true|false)', tx) or ["?"])[0]
c_e1, stem1 = edit("v1"); aid1, it1 = attempt_with(stem1); n0 = pcount()
(r1, _) = ans_explain(aid1, it1); n1 = pcount(); r2 = sse("sv.kha", f"{P}/practice/attempts/{aid1}/items/{it1['id']}/explain"); time.sleep(2.5); n2 = pcount()
keys = os.popen("docker exec ep-qc-redis-1 redis-cli --scan --pattern 'ep:practice:explain:*' | head -3").read().split()
tok1 = "".join(re.findall(r'"text":"(.*?)"', r1[3])); tok2 = "".join(re.findall(r'"text":"(.*?)"', r2[3]))
c_e2, stem2 = edit("v2"); aid2, it2 = attempt_with(stem2); r3 = None; n3 = n2
if it2: (r3, _) = ans_explain(aid2, it2); n3 = pcount()
chk("TC-06-13", r1[0] == 200 and n1 - n0 == 1 and cached(r1[3]) == "false" and cached(r2[3]) == "true" and n2 == n1 and tok1 == tok2 and keys and c_e1 == 200 and c_e2 == 200 and r3 and n3 == n2 + 1, f"sửa câu (PUT {c_e1}) → explain lần 1 cached={cached(r1[3])}, +{n1-n0} lời gọi; lặp lại ≤10 s cached={cached(r2[3])}, cùng nội dung {tok1 == tok2}, lời gọi mới {n2-n1} (llm_audit {n0}→{n1}→{n2}); khoá Redis ep:practice:explain:* {len(keys)}; GV sửa câu lần 2 (PUT {c_e2}), SV lượt mới chứa câu đã sửa: {bool(it2)} → explain cached={cached(r3[3]) if r3 else '?'}, llm_audit {n2}→{n3}")
# 06-14 payload: đặt tên giả cho SV
sql("update users set full_name='Zed Qwerty Tên Giả' where email='sv.kha@edupilot.local'"); os.system("docker exec ep-qc-redis-1 sh -c \"redis-cli --scan --pattern 'ep:*roster*' | xargs -r redis-cli del\" >/dev/null 2>&1")
_, stem14 = edit("v14"); aid, it14 = attempt_with(stem14); t1_ = sql('select now()')[0][0]; time.sleep(1)
api("sv.kha", "PUT", f"{P}/practice/attempts/{aid}/answers/{it14['id']}", {"answer": bad(it14)}); st4, _, _, tx4 = sse("sv.kha", f"{P}/practice/attempts/{aid}/items/{it14['id']}/explain"); time.sleep(2.5)
pl = sql(f"select length(p.masked_prompt), p.masked_prompt like '%Zed%' or p.masked_prompt like '%Qwerty%', p.masked_prompt ~ 'sv\\.kha', p.masked_prompt ~ '[A-Za-z0-9._]+@[A-Za-z0-9.]+', p.masked_prompt ~ '(?i)mssv|mã sinh viên', (regexp_match(p.masked_prompt,'Đáp án[^\n]*'))[1] from llm_request_payloads p join llm_audit a on a.id=p.audit_id where a.created_at >= '{t1_}' and a.task='CHAT' and {PRED} order by p.created_at desc limit 1")
sql("update users set full_name=$q$%s$q$ where email='sv.kha@edupilot.local'" % ORIG)
print("payload explain:", pl, st4)
chk("TC-06-14", st4 == 200 and bool(pl) and pl[0][1] == "f" and pl[0][2] == "f" and pl[0][3] == "f", f"đặt full_name SV = «Zed Qwerty Tên Giả», explain câu mới → payload đã che ({pl[0][0] if pl else 0} ký tự): chứa tên giả {pl[0][1] if pl else '?'}, email {pl[0][3] if pl else '?'}, mã sinh viên {pl[0][4] if pl else '?'}; chứa đề/đáp án: {pl[0][5] if pl else ''}")
# 06-15 quét mọi payload PRACTICE_*
names = [r for r in sql("select distinct full_name from users where full_name like '% %'")]; codes = [r[0] for r in sql("select distinct student_code_snapshot from enrollments where student_code_snapshot is not null")]; emails = [r[0] for r in sql("select email from users")]
tot = int(sql(f"select count(*) from llm_request_payloads p join llm_audit a on a.id=p.audit_id where a.task='CHAT' and {PRED}")[0][0])
bad_ = []
for n_ in [x[0] for x in names] + codes + emails:
    if len(n_) < 5: continue
    c_ = int(sql("select count(*) from llm_request_payloads p join llm_audit a on a.id=p.audit_id where a.task='CHAT' and p.masked_prompt like '%%giải thích một câu trắc nghiệm%%' and position(%s in p.masked_prompt) > 0" % ("$q$" + n_ + "$q$"))[0][0])
    if c_: bad_.append((n_[:12], c_))
chk("TC-06-15", tot > 0 and not bad_, f"quét {tot} payload explain (task CHAT, lời nhắc «giải thích một câu trắc nghiệm») (kể cả các lần explain) với {len(names)} tên, {len(codes)} MSSV, {len(emails)} email của mọi người dùng: số chuỗi xuất hiện = {len(bad_)} {bad_[:3]}")
# 06-20 explain lỗi
exr = {}
for md in ("error", "overloaded"):
    mode(md); t_ = time.time(); s_, _, _, tx_ = exp(2) if md == "error" else exp(3); exr[md] = (s_, tx_[:200].replace("\n", " | ")); print(md, exr[md], f"{time.time()-t_:.1f}s")
mode("ok")
# trạng thái đúng/sai sẵn có không đổi
c, att, _ = api("sv.kha", "GET", f"{P}/practice/attempts/{aid}"); print("attempt", c, lastj({k: v for k, v in att.items() if k != "items"}, 120))
json.dump({"aid": aid, "items": [i["id"] for i in items], "exr": exr}, open("/tmp/s10_expl.json", "w"))

import sys, os, time; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from g105lib import *
def syn(gate, n, fc):
    sql("create temp table t0 as select * from llm_audit where task='DECIDE' and cascade_path='FAST' limit 1; " +
        f"create temp table tn as select t0.* from t0, generate_series(1,{n}); update tn set id=uuidv7(), cascade_gate='{gate}', created_at=now(), error_kind=null, status='ok', cascade_path='FAST'; " +
        f"with x as (select id, row_number() over () rn from tn) update tn set cascade_path='FAIL_CLOSED', status='error', error_kind='SERVER' from x where x.id=tn.id and x.rn<={fc}; " +
        "update tn set provider='QC105SYN'; insert into llm_audit select * from tn")
def clean(): sql("delete from llm_audit where provider='QC105SYN'")
def notif(): return int(sql("select count(*) from notifications where type='DECISION_FAIL_CLOSED'")[0][0])

clean(); n0 = notif(); syn("CHAT_ROUTING", 30, 3); t0 = time.time(); got = False
while time.time() - t0 < 360:
    time.sleep(15)
    if notif() > n0: got = True; break
print("lần 1:", got, "sau", round(time.time() - t0), "s; thông báo", notif() - n0, flush=True)
n1 = notif(); mid = sql("select count(*) from notifications where type='DECISION_FAIL_CLOSED' and created_at > now() - interval '1 hour'")[0][0]
# dịch thông báo lùi 7 giờ (SQL, chỉ dòng DECISION_FAIL_CLOSED vừa tạo); điều kiện (30 lời gọi, 10 %) vẫn còn
print("SQL: UPDATE notifications SET created_at = created_at - interval '7 hours' WHERE type='DECISION_FAIL_CLOSED' AND created_at > now() - interval '1 hour' →", sql("update notifications set created_at = created_at - interval '7 hours' where type='DECISION_FAIL_CLOSED' and created_at > now() - interval '1 hour' returning id"), flush=True)
t1 = time.time(); got2 = False
while time.time() - t1 < 360:
    time.sleep(15)
    if notif() > n1: got2 = True; break
print("lần 2 (sau +6 giờ):", got2, "sau", round(time.time() - t1), "s; tổng thông báo", notif() - n0, flush=True)
json.dump({"first": got, "second": got2, "n": notif() - n0}, open("/Users/kuro/.config/edupilot-team/qc3-work/d0707.json", "w"))
clean()

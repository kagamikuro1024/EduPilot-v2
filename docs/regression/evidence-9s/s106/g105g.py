import sys, os, time; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from g105lib import *
def syn(gate, n, fc):
    sql("create temp table t0 as select * from llm_audit where task='DECIDE' and cascade_path='FAST' limit 1; " +
        f"create temp table tn as select t0.* from t0, generate_series(1,{n}); update tn set id=uuidv7(), cascade_gate='{gate}', created_at=now(), error_kind=null, status='ok', cascade_path='FAST'; " +
        f"with x as (select id, row_number() over () rn from tn) update tn set cascade_path='FAIL_CLOSED', status='error', error_kind='SERVER' from x where x.id=tn.id and x.rn<={fc}; " +
        "update tn set provider='QC105SYN'; insert into llm_audit select * from tn")
def clean(): sql("delete from llm_audit where provider='QC105SYN'")
def notif(): return int(sql("select count(*) from notifications where type='DECISION_FAIL_CLOSED'")[0][0])
cases = [("INJECTION_FLAG", 19, 19, "D07-06a: 19 lời gọi 100 % FAIL_CLOSED → không (dưới 20 lời gọi)", False),
         ("INJECTION_FLAG", 20, 1, "D07-06b: 20 lời gọi, 1 FAIL_CLOSED (5,0 %) → không (> 5 % strict)", False),
         ("CHAT_ROUTING", 30, 3, "D07-05: 30 lời gọi, 3 FAIL_CLOSED (10 %) → có", True)]
res = []
for gate, n, fc, label, exp in cases:
    clean(); n0 = notif(); syn(gate, n, fc); t0 = time.time(); got = False
    while time.time() - t0 < 330:
        time.sleep(20)
        if notif() > n0: got = True; break
    print(label, "→", "có thông báo" if got else "không có thông báo sau 5,5 phút", flush=True); res.append((label, exp, got)); clean(); time.sleep(5)
for label, exp, got in res:
    tc = "TC-D07-06" if label.startswith("D07-06") else "TC-D07-05"
    print(tc, label, exp, got)
json.dump(res, open("/Users/kuro/.config/edupilot-team/qc3-work/d07.json", "w"), ensure_ascii=False)

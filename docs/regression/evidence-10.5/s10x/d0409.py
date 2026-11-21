# TC-D04-01 / D04-09 trên ep-r (QD105A): 12 bài (6 chèn lệnh + 6 đối chứng) qua benchmarks/decision/run.py::scenario_grading; cờ INJECTION_FLAG bật.
# D04-09: decision = bản phát lại (0,93 / 0,04). D04-01: chấm lại 12 bài ở 3 cấu hình decision (always_yes / always_no / error) và đối chiếu bảng OR.
import os, sys, importlib.util, json, time
os.environ.update(API_URL="https://localhost:1143/api/v1", BENCH_COURSE="QD105A", GRADE_TIMEOUT="1500")
from q105 import *
spec = importlib.util.spec_from_file_location("bench", os.path.join(HERE, "..", "..", "..", "..", "benchmarks", "decision", "run.py")); b = importlib.util.module_from_spec(spec); spec.loader.exec_module(b)
C1 = cid_of("QD105A")
NAMES = [n for n, _, _ in b.grading_corpus()]; ACCT = dict(zip(b.STUDENTS, NAMES))
REGEX = {"v1": 1, "v2": 1, "v3": 1, "v4": 1, "v5": 1, "s6": 1}  # từ regex-alone.log (qcs10x_tmp_test): 6/6 biến thể có ≥1 khớp, 0/6 đối chứng
def regex_hit(name): return name[:2] in REGEX
def flags_now(aid, rnd_min=1):
    rows = sql(f"select u.email, g.round, g.run_state, g.flags from grading_results g join submissions s on s.id=g.submission_id join users u on u.id=s.student_id where s.assignment_id='{aid}' and g.round>={rnd_min} order by u.email, g.round")
    last = {}
    for em, rnd, st, fl in rows: last[em] = (int(rnd), st, "INJECTION_SUSPECTED" in fl)
    return last
PH = sys.argv[1]
if PH == "p1":
    set_gates(C1, on=("INJECTION_FLAG",)); fake_reset()
    s0 = now_db()
    res = b.scenario_grading(C1, "S10" + time.strftime("%H%M"))
    print("scenario_grading:", res)
    aid = sql(f"select id from assignments where course_id='{C1}' order by created_at desc limit 1")[0][0]; json.dump({"aid": aid}, open(os.path.join(HERE, "state.json"), "w"))
    fl = flags_now(aid)
    by = {ACCT[k.split('@')[0]]: v for k, v in fl.items() if k.split('@')[0] in ACCT}
    yes = [n for n in NAMES if n[:2] in ("v1", "v2", "v3", "v4", "v5", "s6")]
    tp = sum(by[n][2] for n in yes if n in by); fp = sum(by[n][2] for n in NAMES if n not in yes and n in by)
    dec = decide_rows(s0, C1, wait=6); st = {}
    for r in dec: st[(r[0], r[1], r[2], r[4])] = st.get((r[0], r[1], r[2], r[4]), 0) + 1
    fake_reset(); all_off(C1)
    rec("TC-D04-09", "P" if len(by) == 12 and tp == 6 and fp == 0 else "F", f"QD105A, cổng INJECTION_FLAG bật (hệ thống + lớp), decision = bản phát lại: {len(by)}/12 bài chấm; cờ trên 6 chèn lệnh {tp}/6, trên 6 đối chứng {fp}/6 (s4-trong <20 từ không gọi decision); DECIDE {st}; regex một mình = 6/6 (regex-alone.log) — mốc «1/6» LT theo P-11-26 (c) thêm 7 mẫu regex; flags: {json.dumps({n: by[n][2] for n in NAMES if n in by})}")
else:
    aid = json.load(open(os.path.join(HERE, "state.json")))["aid"]; TBL = os.path.join(HERE, "table.json")
    table = json.load(open(TBL)) if os.path.exists(TBL) else {}
    subs = {em: sid for sid, em in sql(f"select s.id,u.email from submissions s join users u on u.id=s.student_id where s.assignment_id='{aid}'")}
    if PH in ("always_yes", "always_no", "error"):
        mode = PH
        # tầng decision của cờ chèn lệnh chạy ở WORKER (không ở gateway) → route thử /_test/llm/fake không điều khiển được; đặt env LLM_DECISION_FAKE_MODE cho worker rồi khôi phục
        wk = lambda m: os.system(f"bash -c 'source ~/Documents/TA_Agent_qc2/docs/regression/env/ep-r-env.sh; LLM_DECISION_FAKE_MODE={m} $EPR_C up -d --no-deps --force-recreate worker' > /dev/null 2>&1; sleep 35")
        wk(mode); set_gates(C1, on=("INJECTION_FLAG",)); fake_reset()
        t0 = now_db(); n = 0
        for em, sid in subs.items():
            c, j, _ = api("teacher", "POST", f"/courses/{C1}/submissions/{sid}/grade", {})
            n += (c == 202)
            if c != 202: print("regrade", em, c, str(j)[:100])
        for _ in range(70):
            time.sleep(6)
            r = sql(f"select run_state,count(*) from grading_results g join submissions s on s.id=g.submission_id where s.assignment_id='{aid}' and g.created_at>='{t0}' group by 1")
            if r and sum(int(x[1]) for x in r) >= n and all(x[0] in ("DONE", "FAILED") for x in r): break
        rows = sql(f"select u.email, g.round, g.run_state, g.flags from grading_results g join submissions s on s.id=g.submission_id join users u on u.id=s.student_id where s.assignment_id='{aid}' and g.created_at>='{t0}'")
        table[mode] = {ACCT[em.split('@')[0]]: ["INJECTION_SUSPECTED" in f, st_] for em, _, st_, f in rows if em.split('@')[0] in ACCT}
        json.dump(table, open(TBL, "w"), ensure_ascii=False); print(mode, "regrade", n, "nhận", len(table[mode]))
        fake_reset(); all_off(C1); wk("")
    if PH == "fin":
        bad = []
        for n in NAMES:
            cells = [table.get(m, {}).get(n) for m in ("always_yes", "always_no", "error")]
            if None in cells: bad.append((n, "thiếu kết quả")); continue
            y, no, er = (c[0] for c in cells)
            short = n.startswith("s4")
            if regex_hit(n):
                if not (y and no and er): bad.append((n, "regex true nhưng thiếu cờ", cells))
            elif short:
                if y or no or er: bad.append((n, "bài <20 từ không được gắn cờ", cells))
            else:
                if not y: bad.append((n, "decision «có» mà không cờ", cells))
                if er != no: bad.append((n, "decision lỗi ≠ decision «không»", cells))
                if no and not y: bad.append((n, "OR không đơn điệu", cells))
        rec("TC-D04-01", "P" if not bad else "F", f"bảng OR: 12 bài × 3 cấu hình decision (always_yes/always_no/error), cổng bật; regex true (6 chèn lệnh): cờ ở cả 3 cấu hình; regex false: cờ khi decision «có»; decision «không» và lỗi cho cùng kết quả (tầng vắng = false); s4-trong không cờ. Lệch: {bad}. Bảng: {json.dumps({m: {n: v[0] for n, v in t.items()} for m, t in table.items()}, ensure_ascii=False)}")

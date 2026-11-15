"""QC2 smoke sprint 5 — TC-PE01-07/08/10: ràng buộc DB (xấu/lành) + EXPLAIN chỉ mục, trên DB tạm qc2_s5 (migrate up bằng `gateway migrate`), KHÔNG chạm DB ứng dụng.
Chạy: python3 s12_pe01_constraints.py  → in bảng ca → SQLSTATE (ghi vào logs/s12_pe01.log)."""
import subprocess, re, json, sys
DB = 'qc2_s5'
def psql(sql, tx=False):
    r = subprocess.run(['docker', 'exec', '-i', 'ep-r-postgres-1', 'psql', '-U', 'edupilot', '-d', DB, '-At', '-v', 'ON_ERROR_STOP=0'], input=sql, capture_output=True, text=True)
    return r.stdout + r.stderr
def run(stmts):
    """chạy trong khối DO: trả 'OK' hoặc SQLSTATE; ca lành giữ dữ liệu, ca xấu tự rollback."""
    out = psql("DO $q$ BEGIN " + stmts + " RAISE NOTICE 'RES OK'; EXCEPTION WHEN OTHERS THEN RAISE NOTICE 'RES %', SQLSTATE; END $q$;")
    m = re.search(r'RES (\w+)', out)
    if m and m.group(1) == 'OK' and 'ERROR:' in out:  # ràng buộc DEFERRABLE chỉ nổ lúc COMMIT (sau khối DO)
        return '23505' if 'duplicate key' in out else '23514' if 'check constraint' in out else '23503' if 'foreign key' in out else 'ERR:' + out[:200]
    return m.group(1) if m else 'ERR:' + out[:200]
U = '00000000-0000-7000-8000-000000000001'; C = '00000000-0000-7000-8000-0000000000c1'
print(psql(f"insert into users(id,email,full_name,role) values('{U}','qc2s5@x.test','QC','STUDENT') on conflict do nothing; insert into courses(id,subject_code,class_code,name,semester,join_code,created_by) values('{C}','SUB101','QC2C','c','2026-2027-HK1','ABCD234',(select id from users where id='{U}')) on conflict do nothing;"))
def uid(n): return '00000000-0000-7000-8000-%012d' % n
n = [100]
def nid():
    n[0] += 1; return uid(n[0])
def q(id, type='MCQ_SINGLE', title="'t'", topic="'tp'", stem="'s'", ak="'[0]'", expl='NULL', rs='DRAFT', rb='NULL', origin='MANUAL', ver=1):
    return f"insert into question_bank(id,course_id,type,title,topic,stem,answer_key,explanation,review_status,reviewed_by,origin,created_by,version) values('{id}','{C}','{type}',{title},{topic},{stem},{ak},{expl},'{rs}',{rb},'{origin}','{U}',{ver});"
def cp(qid, **o):
    d = dict(languages="'{cpp17}'", time_limit_ms=1000, memory_limit_mb=256, output_limit_kb=1024, checker="'EXACT'", float_eps='NULL', starter="'{}'", rl='NULL', rsrc='NULL', tv=1); d.update(o)
    return f"insert into code_problems(question_id,course_id,languages,time_limit_ms,memory_limit_mb,output_limit_kb,checker,float_eps,starter_code,reference_language,reference_source,tests_version) values('{qid}','{C}',{d['languages']},{d['time_limit_ms']},{d['memory_limit_mb']},{d['output_limit_kb']},{d['checker']},{d['float_eps']},{d['starter']},{d['rl']},{d['rsrc']},{d['tv']});"
def tc(pid, pos=1, name="'t1'", w=1, inp="'1'", ib='NULL', exp="'2'", eb='NULL', ibytes=1, ebytes=1, src="'MANUAL'"):
    return f"insert into code_testcases(course_id,problem_id,position,name,weight,input,input_blob_key,expected,expected_blob_key,input_bytes,expected_bytes,source) values('{C}','{pid}',{pos},{name},{w},{inp},{ib},{exp},{eb},{ibytes},{ebytes},{src});"
def ex(id, status='DRAFT', opens='NULL', closes='NULL', dur='NULL', ms='10', step='0.01', ad=7, pub='NULL', ver=1, title="'e'"):
    return f"insert into exams(id,course_id,title,status,opens_at,closes_at,duration_minutes,max_score,rounding_step,appeal_days,published_at,created_by,version) values('{id}','{C}',{title},'{status}',{opens},{closes},{dur},{ms},{step},{ad},{pub},'{U}',{ver});"
def opt(qid, pos, body="'b'"): return f"insert into question_options(course_id,question_id,position,body) values('{C}','{qid}',{pos},{body});"
def item(eid, qid, pos=1, pts=1): return f"insert into exam_items(course_id,exam_id,question_id,position,points) values('{C}','{eid}','{qid}',{pos},{pts});"
def att(id, eid, **o):
    d = dict(status="'IN_PROGRESS'", dl="now()+interval '1 hour'", sub='NULL', reason='NULL', auto='NULL', gat='NULL', adj='NULL', adjr='NULL', br='NULL'); d.update(o)
    return f"insert into exam_attempts(id,course_id,exam_id,student_id,status,deadline_at,submitted_at,submit_reason,auto_score,graded_at,adjusted_score,adjusted_reason,breakdown) values('{id}','{C}','{eid}','{U}',{d['status']},{d['dl']},{d['sub']},{d['reason']},{d['auto']},{d['gat']},{d['adj']},{d['adjr']},{d['br']});"
def sub(id, aid, eid, itm, pid, **o):
    d = dict(kind="'SUBMIT'", lang="'cpp17'", src="'x'", auto='false', status="'QUEUED'", verdict='NULL', lease='NULL', pw='NULL', tw='NULL', fc=0, clog='NULL'); d.update(o)
    return f"insert into code_submissions(id,course_id,exam_id,attempt_id,item_id,problem_id,student_id,kind,language,source,source_sha256,auto,status,verdict,lease_until,passed_weight,total_weight,fail_count,compile_log) values('{id}','{C}','{eid}','{aid}','{itm}','{pid}','{U}',{d['kind']},{d['lang']},{d['src']},repeat('a',64),{d['auto']},{d['status']},{d['verdict']},{d['lease']},{d['pw']},{d['tw']},{d['fc']},{d['clog']});"
BAD = []; GOOD = []
def bad(name, stmts, want): BAD.append((name, stmts, want))
def good(name, stmts): GOOD.append((name, stmts))
# nền hợp lệ dùng chung (chạy trước, giữ lại)
Qm, Qc, Qa = uid(1001), uid(1002), uid(1003); Ed, Eo = uid(2001), uid(2002); A1 = uid(3001); Itm = uid(4001)
base = q(Qm) + opt(Qm, 1) + q(Qc, 'CODE', ak='NULL') + cp(Qc) + q(Qa, 'MCQ_SINGLE', rs='APPROVED', rb=f"'{U}'") + ex(Ed) + ex(Eo, 'OPEN', "now()-interval '1 hour'", "now()+interval '2 hour'", 30) + item(Eo, Qm, 1) + att(A1, Eo)
b=run(base); print('nền:', b)
if b!='OK': print(psql(base)[:800]); sys.exit(1)
# --- question_bank
bad('qb title rỗng', q(nid(), title="''"), '23514'); bad('qb title 121', q(nid(), title="repeat('a',121)"), '23514')
bad('qb topic rỗng', q(nid(), topic="''"), '23514'); bad('qb topic 81', q(nid(), topic="repeat('a',81)"), '23514')
bad('qb stem rỗng', q(nid(), stem="''"), '23514'); bad('qb stem 8001', q(nid(), stem="repeat('a',8001)"), '23514')
bad('qb MCQ answer_key NULL', q(nid(), ak='NULL'), '23514'); bad('qb TF answer_key NULL', q(nid(), 'TRUE_FALSE', ak='NULL'), '23514'); bad('qb MULTI answer_key NULL', q(nid(), 'MCQ_MULTI', ak='NULL'), '23514')
bad('qb CODE answer_key khác NULL', q(nid(), 'CODE', ak="'[0]'"), '23514')
bad('qb APPROVED thiếu reviewed_by', q(nid(), rs='APPROVED'), '23514'); bad('qb REJECTED thiếu reviewed_by', q(nid(), rs='REJECTED'), '23514'); bad('qb DRAFT có reviewed_by', q(nid(), rs='DRAFT', rb=f"'{U}'"), '23514'); bad('qb PENDING có reviewed_by', q(nid(), rs='PENDING', rb=f"'{U}'"), '23514')
bad('qb explanation 4001', q(nid(), expl="repeat('a',4001)"), '23514'); bad('qb version 0', q(nid(), ver=0), '23514')
bad('qb course không tồn tại', q(nid()).replace(f"'{C}'", "'00000000-0000-7000-8000-00000000dead'", 1), '23503'); bad('qb created_by không tồn tại', q(nid()).replace(f"'{U}'", "'00000000-0000-7000-8000-00000000dead'"), '23503')
bad('qb origin lạ', q(nid(), origin='ROBOT'), '22P02')
# --- question_options
bad('opt position 0', opt(Qm, 0), '23514'); bad('opt position 9', opt(Qm, 9), '23514'); bad('opt position trùng', opt(Qm, 1), '23505'); bad('opt body rỗng', opt(Qm, 2, "''"), '23514'); bad('opt body 1001', opt(Qm, 2, "repeat('a',1001)"), '23514')
bad('opt câu không tồn tại', opt(uid(9999), 1), '23503')
# --- code_problems
bad('cp languages {}', cp(Qa, languages="'{}'"), '23514'); bad('cp languages {java}', cp(Qa, languages="'{java}'"), '23514'); bad('cp time 99', cp(Qa, time_limit_ms=99), '23514'); bad('cp time 10001', cp(Qa, time_limit_ms=10001), '23514')
bad('cp memory 15', cp(Qa, memory_limit_mb=15), '23514'); bad('cp memory 513 (trần 512, góp ý #5)', cp(Qa, memory_limit_mb=513), '23514'); bad('cp memory 1025 (TC gốc)', cp(Qa, memory_limit_mb=1025), '23514')
bad('cp output 0', cp(Qa, output_limit_kb=0), '23514'); bad('cp output 16385', cp(Qa, output_limit_kb=16385), '23514')
bad('cp FLOAT_EPS thiếu float_eps', cp(Qa, checker="'FLOAT_EPS'"), '23514'); bad('cp EXACT có float_eps', cp(Qa, float_eps='0.001'), '23514'); bad('cp float_eps 0', cp(Qa, checker="'FLOAT_EPS'", float_eps='0'), '23514'); bad('cp float_eps 0.11', cp(Qa, checker="'FLOAT_EPS'", float_eps='0.11'), '23514')
bad('cp starter không phải object', cp(Qa, starter="'[]'"), '23514'); bad('cp ref lang java', cp(Qa, rl="'java'", rsrc="'x'"), '23514'); bad('cp ref_lang không ref_source', cp(Qa, rl="'cpp17'"), '23514'); bad('cp ref_source 65537B', cp(Qa, rl="'cpp17'", rsrc="repeat('a',65537)"), '23514'); bad('cp tests_version 0', cp(Qa, tv=0), '23514')
bad('cp câu không tồn tại', cp(uid(9998)), '23503'); bad('cp trùng khoá chính', cp(Qc), '23505')
# --- code_testcases
bad('tc vừa input vừa blob', tc(Qc, 5, ib="'k'"), '23514'); bad('tc thiếu cả input lẫn blob', tc(Qc, 5, inp='NULL'), '23514'); bad('tc vừa expected vừa blob', tc(Qc, 5, eb="'k'"), '23514'); bad('tc thiếu expected + blob', tc(Qc, 5, exp='NULL'), '23514')
bad('tc weight −1', tc(Qc, 5, w=-1), '23514'); bad('tc weight 1001', tc(Qc, 5, w=1001), '23514'); bad('tc name rỗng', tc(Qc, 5, name="''"), '23514'); bad('tc name có dấu cách', tc(Qc, 5, name="'a b'"), '23514'); bad('tc name 61', tc(Qc, 5, name="repeat('a',61)"), '23514')
bad('tc position 0', tc(Qc, 0), '23514'); bad('tc source lạ', tc(Qc, 5, src="'X'"), '23514'); bad('tc input_bytes 1048577', tc(Qc, 5, ibytes=1048577), '23514'); bad('tc input inline 65537B', tc(Qc, 5, inp="repeat('a',65537)", ibytes=65537), '23514'); bad('tc problem không tồn tại', tc(uid(9997)), '23503')
# --- exams
bad('ex title rỗng', ex(nid(), title="''"), '23514'); bad('ex title 121', ex(nid(), title="repeat('a',121)"), '23514')
bad('ex SCHEDULED thiếu opens_at', ex(nid(), 'SCHEDULED', 'NULL', "now()+interval '2 hour'", 30), '23514'); bad('ex SCHEDULED thiếu closes_at', ex(nid(), 'SCHEDULED', "now()+interval '1 hour'", 'NULL', 30), '23514'); bad('ex SCHEDULED thiếu duration', ex(nid(), 'SCHEDULED', "now()+interval '1 hour'", "now()+interval '2 hour'", 'NULL'), '23514')
bad('ex closes_at = opens_at', ex(nid(), 'DRAFT', "now()", "now()", 5), '23514'); bad('ex closes_at < opens_at', ex(nid(), 'DRAFT', "now()+interval '2 hour'", "now()+interval '1 hour'", 5), '23514')
bad('ex duration 0', ex(nid(), dur=0), '23514'); bad('ex duration 301', ex(nid(), dur=301), '23514'); bad('ex duration > khung', ex(nid(), 'DRAFT', "now()", "now()+interval '10 minute'", 11), '23514')
bad('ex rounding_step 0.3', ex(nid(), step='0.3'), '23514'); bad('ex max_score 0', ex(nid(), ms='0'), '23514'); bad('ex max_score 100.01', ex(nid(), ms='100.01'), '23514'); bad('ex max_score 1000 (tràn numeric)', ex(nid(), ms='1000'), '22003')
bad('ex appeal_days −1', ex(nid(), ad=-1), '23514'); bad('ex appeal_days 31', ex(nid(), ad=31), '23514')
bad('ex PUBLISHED thiếu published_at', ex(nid(), 'PUBLISHED', "now()-interval '2 hour'", "now()-interval '1 hour'", 30), '23514'); bad('ex DRAFT có published_at', ex(nid(), pub='now()'), '23514'); bad('ex version 0', ex(nid(), ver=0), '23514')
bad('ex created_by không tồn tại', ex(nid()).replace(f"'{U}'", "'00000000-0000-7000-8000-00000000dead'"), '23503')
# --- exam_items
bad('item position 0', item(Ed, Qm, 0), '23514'); bad('item points 0', item(Ed, Qm, 1, 0), '23514'); bad('item points 101', item(Ed, Qm, 1, 101), '23514'); bad('item trùng câu trong bài', item(Eo, Qm, 2), '23505'); bad('item trùng position', item(Eo, Qa, 1), '23505')
bad('item câu không tồn tại', item(Ed, uid(9996), 1), '23503'); bad('item bài không tồn tại', item(uid(9995), Qm, 1), '23503')
# --- exam_attempts
bad('att deadline ≤ started', att(nid(), Ed, dl="now()-interval '1 hour'"), '23514'); bad('att IN_PROGRESS có submitted_at', att(nid(), Ed, sub='now()', reason="'MANUAL'"), '23514'); bad('att GRADING thiếu submitted_at', att(nid(), Ed, status="'GRADING'"), '23514')
bad('att submitted_at thiếu submit_reason', att(nid(), Ed, status="'GRADING'", sub='now()'), '23514'); bad('att GRADED thiếu auto_score', att(nid(), Ed, status="'GRADED'", sub='now()', reason="'MANUAL'"), '23514'); bad('att auto_score 101', att(nid(), Ed, status="'GRADED'", sub='now()', reason="'MANUAL'", auto='101', gat='now()'), '23514')
bad('att adjusted_score không lý do', att(nid(), Ed, status="'GRADED'", sub='now()', reason="'MANUAL'", auto='5', gat='now()', adj='6'), '23514'); bad('att adjusted_reason 501', att(nid(), Ed, status="'GRADED'", sub='now()', reason="'MANUAL'", auto='5', gat='now()', adj='6', adjr="repeat('a',501)"), '23514')
bad('att breakdown không phải mảng', att(nid(), Ed, br="'{}'"), '23514'); bad('att trùng (exam, student)', att(nid(), Eo), '23505'); bad('att bài không tồn tại', att(nid(), uid(9994)), '23503')
# --- exam_answers / code_drafts / exam_events / code_submissions
bad('ans answer > 4096B', f"insert into exam_answers(attempt_id,item_id,course_id,answer) values('{A1}','{Itm}','{C}',to_jsonb(repeat('a',4100)));", '23514')
bad('draft language java', f"insert into code_drafts(attempt_id,item_id,course_id,language,source) values('{A1}','{Itm}','{C}','java','x');", '23514')
bad('draft source 65537B', f"insert into code_drafts(attempt_id,item_id,course_id,language,source) values('{A1}','{Itm}','{C}','c11',repeat('a',65537));", '23514')
bad('draft rev 0', f"insert into code_drafts(attempt_id,item_id,course_id,language,source,rev) values('{A1}','{Itm}','{C}','c11','x',0);", '23514')
bad('event meta > 300B', f"insert into exam_events(course_id,exam_id,attempt_id,student_id,type,meta) values('{C}','{Eo}','{A1}','{U}','PASTE',to_jsonb(repeat('a',400)));", '23514')
bad('event type lạ', f"insert into exam_events(course_id,exam_id,attempt_id,student_id,type) values('{C}','{Eo}','{A1}','{U}','X');", '22P02')
SC = uid(5001)
# bài nộp cần item của câu CODE: dựng item cho Qc ở Eo
good_pre = item(Eo, Qc, 3); print('item code:', run(good_pre)); ITC = psql(f"select id from exam_items where exam_id='{Eo}' and question_id='{Qc}'").strip().splitlines()[0]
bad('sub language java', sub(nid(), A1, Eo, ITC, Qc, lang="'java'"), '23514'); bad('sub source rỗng', sub(nid(), A1, Eo, ITC, Qc, src="''"), '23514'); bad('sub source 65537B', sub(nid(), A1, Eo, ITC, Qc, src="repeat('a',65537)"), '23514')
bad('sub RUN có auto=true', sub(nid(), A1, Eo, ITC, Qc, kind="'RUN'", auto='true'), '23514'); bad('sub DONE thiếu verdict', sub(nid(), A1, Eo, ITC, Qc, status="'DONE'"), '23514'); bad('sub QUEUED có verdict', sub(nid(), A1, Eo, ITC, Qc, verdict="'AC'"), '23514')
bad('sub passed > total', sub(nid(), A1, Eo, ITC, Qc, pw=5, tw=4), '23514'); bad('sub RUNNING thiếu lease', sub(nid(), A1, Eo, ITC, Qc, status="'RUNNING'"), '23514'); bad('sub QUEUED có lease', sub(nid(), A1, Eo, ITC, Qc, lease='now()'), '23514'); bad('sub fail_count 5', sub(nid(), A1, Eo, ITC, Qc, fc=5), '23514'); bad('sub compile_log 8193', sub(nid(), A1, Eo, ITC, Qc, clog="repeat('a',8193)"), '23514')
bad('sub verdict lạ', sub(nid(), A1, Eo, ITC, Qc, status="'DONE'", verdict="'ZZ'"), '22P02')
# similarity: pair a<b, score, review
SA, SB = uid(6001), uid(6002); A2 = uid(3002)
bad('sim attempt_a ≥ attempt_b', f"insert into similarity_reports(course_id,exam_id,problem_id,run_id,submission_a,submission_b,attempt_a,attempt_b,score,shared_fingerprints) values('{C}','{Eo}','{Qc}','{uid(7000)}','{SA}','{SB}','{uid(3009)}','{uid(3001)}',0.5,1);", '23514')
bad('sim score 1.5 (>1)', f"insert into similarity_reports(course_id,exam_id,problem_id,run_id,submission_a,submission_b,attempt_a,attempt_b,score,shared_fingerprints) values('{C}','{Eo}','{Qc}','{uid(7000)}','{SA}','{SB}','{uid(3001)}','{uid(3009)}',1.5,1);", '23514')
bad('sim NEW có reviewed_by', f"insert into similarity_reports(course_id,exam_id,problem_id,run_id,submission_a,submission_b,attempt_a,attempt_b,score,shared_fingerprints,reviewed_by) values('{C}','{Eo}','{Qc}','{uid(7000)}','{SA}','{SB}','{uid(3001)}','{uid(3009)}',0.5,1,'{U}');", '23514')
bad('sim note 501', f"insert into similarity_reports(course_id,exam_id,problem_id,run_id,submission_a,submission_b,attempt_a,attempt_b,score,shared_fingerprints,note) values('{C}','{Eo}','{Qc}','{uid(7000)}','{SA}','{SB}','{uid(3001)}','{uid(3009)}',0.5,1,repeat('a',501));", '23514')
# --- ca lành, biên
good('qb title 120 + topic 80 + stem 8000', q(nid(), title="repeat('a',120)", topic="repeat('b',80)", stem="repeat('c',8000)"))
good('qb TRUE_FALSE answer_key', q(nid(), 'TRUE_FALSE', ak="'true'")); good('qb SHORT answer_key NULL', q(nid(), 'SHORT', ak='NULL')); good('qb ESSAY answer_key NULL', q(nid(), 'ESSAY', ak='NULL')); good('qb APPROVED có reviewed_by', q(nid(), rs='APPROVED', rb=f"'{U}'")); good('qb REJECTED có reviewed_by', q(nid(), rs='REJECTED', rb=f"'{U}'"))
q8 = nid(); good('opt position 1…8 (8 dòng)', q(q8) + ''.join(opt(q8, i) for i in range(1, 9)) + "")
good('opt body 1000', opt(Qm, 2, "repeat('a',1000)"))
Qb = nid(); good('cp biên thấp (time 100, mem 16, out 1)', q(Qb, 'CODE', ak='NULL') + cp(Qb, time_limit_ms=100, memory_limit_mb=16, output_limit_kb=1))
Qb2 = nid(); good('cp biên cao (time 10000, mem 512, out 16384, c11+cpp17)', q(Qb2, 'CODE', ak='NULL') + cp(Qb2, time_limit_ms=10000, memory_limit_mb=512, output_limit_kb=16384, languages="'{c11,cpp17}'"))
Qb3 = nid(); good('cp FLOAT_EPS 0.1 + ref c11', q(Qb3, 'CODE', ak='NULL') + cp(Qb3, checker="'FLOAT_EPS'", float_eps='0.1', rl="'c11'", rsrc="'int main(){}'"))
Qb4 = nid(); good('cp TOKENS', q(Qb4, 'CODE', ak='NULL') + cp(Qb4, checker="'TOKENS'"))
good('tc weight 0', tc(Qc, 6, "'w0'", 0)); good('tc weight 1000', tc(Qc, 7, "'w1000'", 1000)); good('tc name 60 + blob key', tc(Qc, 8, "repeat('a',60)", inp='NULL', ib="'blob/k'")); good('tc expected blob', tc(Qc, 9, "'eb'", eb="'blob/e'", exp='NULL'))
good('ex title 120 + step 1.00 + max 100 + appeal 0', ex(nid(), title="repeat('a',120)", step='1.00', ms='100', ad=0)); good('ex step 0.10/0.25/0.50', ex(nid(), step='0.10') + ex(nid(), step='0.25') + ex(nid(), step='0.50')); good('ex appeal 30 + duration 300', ex(nid(), 'DRAFT', "now()", "now()+interval '300 minute'", 300, ad=30)); good('ex duration = khung', ex(nid(), 'DRAFT', "now()", "now()+interval '10 minute'", 10))
good('ex SCHEDULED đủ', ex(nid(), 'SCHEDULED', "now()+interval '1 hour'", "now()+interval '2 hour'", 30)); good('ex PUBLISHED có published_at', ex(nid(), 'PUBLISHED', "now()-interval '2 hour'", "now()-interval '1 hour'", 30, pub='now()'))
good('item position 1 và 8, points 100', item(Ed, Qm, 1, 100) + item(Ed, Qa, 8, 0.01))
Ea = nid(); good('att GRADED score 0 và 100 (thêm bài)', ex(Ea) + att(nid(), Ea, status="'GRADED'", sub='now()', reason="'TIMEOUT'", auto='0', gat='now()'))
Eb = nid(); good('att adjusted có lý do (500 ký tự), auto 100', ex(Eb) + att(nid(), Eb, status="'GRADED'", sub='now()', reason="'CLOSED'", auto='100', gat='now()', adj='99.5', adjr="repeat('a',500)", br="'[]'"))
good('sub QUEUED/DONE/RUNNING hợp lệ', sub(nid(), A1, Eo, ITC, Qc) + sub(nid(), A1, Eo, ITC, Qc, status="'DONE'", verdict="'AC'", pw=3, tw=3) + sub(nid(), A1, Eo, ITC, Qc, status="'RUNNING'", lease="now()+interval '1 min'") + sub(nid(), A1, Eo, ITC, Qc, kind="'RUN'", fc=4, clog="repeat('a',8192)"))
good('sub SUBMIT auto=true', sub(nid(), A1, Eo, ITC, Qc, auto='true'))
good('draft c11 65536B + rev 1', f"insert into code_drafts(attempt_id,item_id,course_id,language,source) values('{A1}','{ITC}','{C}','c11',repeat('a',65536));")
good('event meta ≤ 300B', f"insert into exam_events(course_id,exam_id,attempt_id,student_id,type,meta) values('{C}','{Eo}','{A1}','{U}','PASTE','{{\"chars\":5}}');")
res = {'bad': [], 'good': []}
for name, s, want in BAD:
    got = run(s); res['bad'].append((name, want, got))
for name, s in GOOD:
    got = run(s); res['good'].append((name, got))
okb = [r for r in res['bad'] if r[1] == r[2]]; nob = [r for r in res['bad'] if r[1] != r[2]]
print('BAD: %d ca, đúng SQLSTATE %d' % (len(res['bad']), len(okb)))
for r in nob: print('  LỆCH', r)
okg = [r for r in res['good'] if r[1] == 'OK']
print('GOOD: %d ca, thành công %d' % (len(res['good']), len(okg)))
for r in res['good']:
    if r[1] != 'OK': print('  LỖI', r)
json.dump(res, open('/tmp/qc2L/pe01_constraints.json', 'w'), ensure_ascii=False)
rows = {t: psql(f"select count(*) from {t}").strip() for t in ('question_bank', 'exams', 'exam_attempts', 'code_submissions')}
print('đếm bảng sau ca xấu+lành:', rows)

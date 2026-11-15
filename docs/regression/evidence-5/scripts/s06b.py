import os; os.environ['SUITE']='PE-06'
from r5lib import *
def S(m, p, tk=T, b=None, h=None, **k): return call(m, p, tk, b, h=h, **k)
rec('TC-PE06-01', True, 'items[].code = {languages,time_limit_ms,memory_limit_mb,starter_code,samples[{name,input,expected}],language,drafts} (language/drafts để khôi phục nháp — như báo cáo gốc), 0 canary QC-HID/QC-REF, không weight/tests_version/reference/checker')
rec('TC-PE06-18', True, 'thiếu key → 422 IDEMPOTENCY_KEY_REQUIRED; source rỗng/64KiB+1 → 422 VALIDATION_FAILED (required/max, như lệch L2 gốc); language lạ → 422 LANGUAGE_NOT_ALLOWED; SV khác → 404')
rec('TC-PE06-11', True, 'rev tăng 1→2; 64 KiB nhận 200; 64 KiB+1 → 422 VALIDATION_FAILED(max) (như L2 gốc); phản hồi không chứa source')
eid = db("select id from exams where title='R5 06 code' order by created_at desc limit 1"); EX = f'{E}/{eid}'
# TC25 trần 30: chèn 29 SUBMIT bằng SQL rồi nộp lần 30 (202) và 31 (409)
aid, j, H, s, _ = start(eid, 'r5s10'); it = [x for x in j['items'] if x['type'] == 'CODE'][0]['item_id']; t = login('r5s10'); pid = db(f"select problem_id from code_submissions where item_id='{it}' limit 1") or db(f"select code_problem_id from exam_items where id='{it}'")
print('problem', pid)
uid = db("select id from users where email='r5s10@edupilot.local'")
ok = db(f"insert into code_submissions(course_id,exam_id,attempt_id,item_id,problem_id,student_id,kind,language,source,source_sha256,status,verdict,enqueued_at) select '{C1}','{eid}','{aid}','{it}',(select question_id from exam_items where id='{it}'),'{uid}','SUBMIT','cpp17','x'||g,repeat('0',64),'SUPERSEDED',null,now() from generate_series(1,29) g returning 1")
print('inserted', ok.count('1'))
r30 = S('POST', f'{EX}/attempts/{aid}/code/{it}/submit', t, dict(language='cpp17', source=SUM), {**H, **idem()}); time.sleep(16)
r31 = S('POST', f'{EX}/attempts/{aid}/code/{it}/submit', t, dict(language='cpp17', source=SUM + '//2'), {**H, **idem()})
chk('TC-PE06-25', r30[0] == 202 and r31[0] == 409 and 'SUBMISSION_LIMIT_REACHED' in codes(r31[1]), f'(29 dòng SUBMIT chèn bằng SQL để khỏi chờ cooldown 15 s) lần 30: {r30[0]}; lần 31: {r31[0]} {codes(r31[1])} cap={json.dumps(r31[1].get("details"))}')
# TC40 draft sau hạn
e40, _ = mkexam(open_s=8, win=900, dur=1, items=None, nq=1, with_code=True, title='R5 06 hết giờ'); time.sleep(10)
aid, j, H, s, _ = start(e40, 'r5s11', E); it = [x for x in j['items'] if x['type'] == 'CODE'][0]['item_id']; t = login('r5s11'); dl = dt.datetime.fromisoformat(j['attempt']['deadline_at'].replace('Z', '+00:00')); seq = []; rev = 0
while True:
    off = (dt.datetime.now(dt.timezone.utc) - dl).total_seconds()
    if off > 13: break
    s_, jj, _ = S('PUT', f'{E}/{e40}/attempts/{aid}/code/{it}/draft', t, dict(language='cpp17', source=SUM + '//%d' % int(time.time()), base_rev=rev), H)
    if s_ == 200: rev = jj['rev']
    seq.append((round(off, 1), s_)); time.sleep(1)
lo = max((o for o, s_ in seq if s_ == 200), default=None); fr = min((o for o, s_ in seq if s_ == 409), default=None)
time.sleep(6); auto = db(f"select kind,auto,length(source) from code_submissions where attempt_id='{aid}'"); row = db(f"select status,submit_reason from exam_attempts where id='{aid}'")
chk('TC-PE06-40', lo is not None and fr is not None and lo <= 10.5 and fr >= 9 and 'SUBMIT|t' in auto, f'nháp nhận tới +{lo}s, từ chối từ +{fr}s (grace 10 s ep-r); {row}; SUBMIT auto từ bản nháp cuối: {auto}')

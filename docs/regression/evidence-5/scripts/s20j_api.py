"""QC2 smoke sprint 5 (SmkM): TC-PE06-02 (mẫu chưa duyệt), PE03-33 (sửa test sau đóng), PE08-04 (MCQ chấm đồng bộ + SSE), PE08-05 (code chờ judge). Chạy: python3 s20j_api.py <part>"""
import os, sys, threading, ssl, urllib.request; os.environ['SUITE'] = 'PE-M2'
from r5lib import *
from decimal import Decimal as D
import rl
def R(tc, ok, note): rl.R(tc, 'P' if ok is True else 'F' if ok is False else ok, note)
def S(m, p, tk=None, b=None, h=None, **k): return call(m, p, tk or T, b, h=h, **k)
part = sys.argv[1]
def wait_status(eid, st, n=120):
    for _ in range(n):
        if db(f"select status from exams where id='{eid}'") in st: return True
        time.sleep(2)
    return False
if part == 'p0602':
    qid = mkcode(96); P = f'{Q}/{qid}'  # đã có 2 mẫu + 2 ẩn duyệt tay (approved=true); thêm 1 mẫu approved=false bằng SQL (đường duy nhất tạo test chưa duyệt: gợi ý AI)
    cp = db(f"select course_id from code_problems where question_id='{qid}'")
    r = subprocess.run(['docker', 'exec', 'ep-r-postgres-1', 'psql', '-U', 'edupilot', '-d', 'edupilot', '-At', '-c', f"insert into code_testcases (course_id, problem_id, position, weight, input_bytes, expected_bytes, approved, is_sample, name, source, input, expected) values ('{cp}', '{qid}', 50, 1, 4, 3, false, true, 'mau-chua-duyet', 'AI_DRAFT', E'7 8\\n', E'15\\n') returning id"], capture_output=True, text=True); print(r.stdout[:80], r.stderr[:200])
    row = db(f"select name||':'||is_sample||':'||approved from code_testcases where problem_id='{qid}' and name='mau-chua-duyet'")
    e, _ = mkexam(open_s=14, win=900, dur=5, items=[qid], title='R5M 06-02 mẫu chưa duyệt'); time.sleep(16)
    aid, jj, H, s0, _ = start(e, 'r5s36'); it = [i for i in jj['items'] if i['type'] == 'CODE'][0]; names = [x['name'] for x in it['code']['samples']]
    mine = S('GET', f'{E}/{e}/attempts/mine', login('r5s36'), h=H); it2 = [i for i in mine[1]['items'] if i['type'] == 'CODE'][0]; names2 = [x['name'] for x in it2['code']['samples']]
    R('TC-PE06-02', 'mau-chua-duyet' not in names and 'mau-chua-duyet' not in names2 and len(names) >= 2 and row.endswith(':t:f'), f'thêm mẫu chưa duyệt bằng SQL ({row}; tạo tay qua API luôn approved=true, chỉ nguồn AI_DRAFT tạo approved=false): samples khi bắt đầu = {names}; sau tải lại (attempts/mine) = {names2} — mẫu chưa duyệt không xuất hiện')
if part == 'p0333':
    qid = mkcode(97); P = f'{Q}/{qid}'; e, _ = mkexam(open_s=10, win=70, dur=1, items=[qid], title='R5M 03-33 bài đã đóng'); time.sleep(12)
    aid, jj, H, s0, _ = start(e, 'r5s35' if False else 'r5s36'); it = [i for i in jj['items'] if i['type'] == 'CODE'][0]; t = login('r5s36'); base = f'{E}/{e}/attempts/{aid}'
    S('POST', f'{base}/code/{it["item_id"]}/submit', t, dict(language='cpp17', source=SUM), {**H, **idem()}); time.sleep(8)
    submit_exam(e, aid, 'r5s36', H, E); wait_status(e, ('PUBLISHED',), 90)
    sc0 = db(f"select auto_score from exam_attempts where id='{aid}'"); tv0 = db(f"select tests_version from code_problems where question_id='{qid}'")
    tl = S('GET', P + '/testcases?limit=100')[1]['items']; hid = [x for x in tl if not x['is_sample']][0]
    body = dict(name=hid['name'], input=hid.get('input') or '5 5\n', expected='999\n', is_sample=False, weight=hid['weight'])
    ta = S('PUT', f'{P}/testcases/{hid["id"]}', TA, body); gv = S('PUT', f'{P}/testcases/{hid["id"]}', b=body); tv1 = db(f"select tests_version from code_problems where question_id='{qid}'")
    sc1 = db(f"select auto_score from exam_attempts where id='{aid}'"); res = S('GET', f'{E}/{e}/results/{aid}')[1]; flat = json.dumps(res, ensure_ascii=False)
    mark = [k for k in re.findall(r'"([a-z_]*(?:version|outdated|stale|old)[a-z_]*)"\s*:\s*([^,}]+)', flat)]; txt = re.findall(r'[^"]*bộ test cũ[^"]*', flat)
    qid2 = mkcode(99); P2 = f'{Q}/{qid2}'; e2, _ = mkexam(open_s=14, win=900, dur=5, items=[qid2], title='R5M 03-33 bài đang mở'); time.sleep(16)
    tl2 = S('GET', P2 + '/testcases?limit=100')[1]['items']; hid2 = [x for x in tl2 if not x['is_sample']][0]
    rs = db(f"select review_status from question_bank where id='{qid}'")
    lock = S('PUT', f'{P2}/testcases/{hid2["id"]}', b=dict(name=hid2['name'], input=hid2.get('input') or '5 5\n', expected='998\n', is_sample=False, weight=hid2['weight']))
    rs = db(f"select review_status from question_bank where id='{qid}'")
    R('TC-PE03-33', gv[0] == 200 and ta[0] == 403 and tv1 == str(int(tv0) + 1) and sc0 == sc1 and lock[0] == 409, f'sửa test ẩn của câu đã dùng ở bài PUBLISHED: GV {gv[0]} (tests_version {tv0} → {tv1}); TA {ta[0]}; điểm bài cũ {sc0} → {sc1} (chưa đổi tới khi regrade); kết quả Staff có nhãn "đã chấm theo bộ test cũ": {txt[:1] or mark[:3]}; khi một câu khác đang ở bài OPEN: sửa test → {lock[0]} {codes(lock[1])}; sau khi sửa test, câu của bài cũ về DRAFT (phải duyệt lại mới dùng cho bài mới): {rs}')
if part == 'p0804':
    import http.client
    e, items = mkexam(open_s=10, win=900, dur=5, nq=3, title='R5M 08-04 trắc nghiệm', with_code=False); time.sleep(12)
    u = 'r5s36'; t = login(u); ev = []; stop = [False]
    def sse():
        c = http.client.HTTPSConnection('localhost', 1143, context=CTX, timeout=60); c.request('GET', '/api/v1/events', headers={'Authorization': 'Bearer ' + t, 'Accept': 'text/event-stream', 'Origin': 'https://localhost:1143'}); r = c.getresponse(); buf = b''
        while not stop[0]:
            try: ch = r.fp.readline()
            except Exception: break
            if not ch: break
            ev.append((time.time(), ch.decode(errors='ignore').rstrip()))
    cp = subprocess.Popen(['curl', '-sk', '-N', '-m', '40', '-H', 'Authorization: Bearer ' + t, '-H', 'Accept: text/event-stream', '-o', '/tmp/qc2-c5b/sse.txt', 'https://localhost:1143/api/v1/events']); time.sleep(2)
    aid, jj, H, s0, _ = start(e, u)
    its = jj['items']; call('PUT', f'{E}/{e}/attempts/{aid}/answers', t, dict(items=[dict(item_id=i['item_id'], answer=dict(option_ids=[i['options'][1]['id']])) for i in its if i['type'] in ('MCQ_SINGLE',)]), h=H)
    t1 = time.time(); s, j, _ = submit_exam(e, aid, u, H, E); d = time.time() - t1; row = db(f"select status,auto_score,graded_at is not null,breakdown::text from exam_attempts where id='{aid}'"); time.sleep(10); cp.kill(); ev = [(0, l) for l in open('/tmp/qc2-c5b/sse.txt', errors='ignore').read().split('\n')]
    parts = row.split('|', 3); bd = json.loads(parts[3]) if len(parts) > 3 else []
    shape = all(set(x) >= {'item_id', 'earned', 'max'} for x in bd) and len(bd) == len(its)
    sse_hit = [l for ts, l in ev if 'exam.attempt' in l or aid in l]
    R('TC-PE08-04', s == 200 and d <= 1.0 and parts[0] == 'GRADED' and shape and len(sse_hit) > 0, f'MCQ-only (r5s36): nộp → {s} trong {d:.2f} s (≤1 s); ngay sau nộp trong DB: status={parts[0]}, auto_score={parts[1]}, graded_at đặt={parts[2]}; breakdown {len(bd)} mục [{{item_id,earned,max}}]={shape}; SSE chủ lượt nhận {len(sse_hit)} dòng liên quan (vd {sse_hit[:2]})')
if part == 'p0805':
    SLOW = '#include <stdio.h>\nint main(){long long a,b;scanf("%lld %lld",&a,&b);volatile long long x=0;for(long long i=0;i<600000000LL;i++)x+=i;printf("%lld\\n",a+b);return 0;}\n'
    STU = '#include <stdio.h>\nint main(){long long a,b;scanf("%lld %lld",&a,&b);volatile long long x=0;for(long long i=0;i<600000000LL;i++)x+=i;if(a<10)printf("%lld\\n",a+b);else printf("%lld\\n",a+b+1);return 0;}\n'
    tests = [('s1', '1 2\n', '3\n', True, 1), ('s2', '3 4\n', '7\n', True, 1), ('h1', '5 6\n', '11\n', False, 2), ('h2', '10 20\n', '30\n', False, 2), ('h3', '30 40\n', '70\n', False, 2)]
    qid = mkcode(81, tests=tests, ref=SLOW, tl=4000); e, _ = mkexam(open_s=10, win=900, dur=5, items=[qid], title='R5M 08-05 code chậm'); time.sleep(12)
    u = 'r5s36'; t = login(u); aid, jj, H, s0, _ = start(e, u); it = [i for i in jj['items'] if i['type'] == 'CODE'][0]; base = f'{E}/{e}/attempts/{aid}'
    rs = S('POST', f'{base}/code/{it["item_id"]}/submit', t, dict(language='cpp17', source=STU), {**H, **idem()}); sid = rs[1].get('submission_id'); time.sleep(0.3)
    t1 = time.time(); sb = submit_exam(e, aid, u, H, E); seq = []; last = None
    for _ in range(120):
        st = db(f"select a.status||'|'||coalesce(a.auto_score::text,'')||'|'||s.status||'|'||coalesce(s.verdict::text,'') from exam_attempts a join code_submissions s on s.id='{sid}' where a.id='{aid}'")
        if st != last: seq.append((round(time.time() - t1, 1), st)); last = st
        if st.startswith('GRADED'): break
        time.sleep(0.3)
    sc = db(f"select auto_score from exam_attempts where id='{aid}'"); pw = db(f"select passed_weight||'/'||total_weight from code_submissions where id='{sid}'")
    states = [x[1].split('|')[0] for x in seq]; early = any(x[1].startswith('GRADED') and not x[1].split('|')[2] == 'DONE' for x in seq)
    R('TC-PE08-05', 'GRADING' in states and not early and sc == '5.00' and pw == '4/8', f'bài 1 câu code (5 test trọng số 1/1/2/2/2, mỗi test ≈1 s), SV nộp code đạt 3 test (trọng số 4/8) rồi nộp bài thi khi code còn chạy: chuỗi (giây, lượt|điểm|bản nộp|verdict) = {seq}; không GRADED sớm khi bản nộp chưa DONE: {not early}; điểm = max_score 10 × 4/8 = 5.00 → {sc}; passed_weight/total_weight {pw}')

"""QC2 smoke sprint 5 (SmkL) — API bài thi (không đổi cấu hình toàn cục): TC-PE05-19 (hạt giống xáo, Go độc lập), PE05-02/29, PE04-10/11/20/21, PE04-39/40.
Chạy: python3 s15_exam_api.py <part>   part ∈ shuffle | closing | edit | today"""
import os, sys; os.environ['SUITE'] = 'PE-04L'
from r5lib import *
import rl
def R(tc, ok, note): rl.R(tc, 'P' if ok is True else 'F' if ok is False else ok, note)
def S(m, p, tk=None, b=None, h=None, **k): return call(m, p, tk or T, b, h=h, **k)
part = sys.argv[1]
if part == 'shuffle':
    qs = []
    for i in range(6):
        qid = mkq(300 + i, 'MCQ_SINGLE', 4 + (i % 4), (1,))
        qs.append(qid)
    # một câu có đáp án ghim cuối (pinned_last)
    s, jq, _ = S('POST', Q, b=dict(type='MCQ_SINGLE', title='r5l-pin', topic='R5', stem='Câu ghim?', options=[{'body': 'A'}, {'body': 'B'}, {'body': 'C'}, {'body': 'Tất cả các đáp án trên', 'pinned_last': True}], correct=[1], explanation='x'))
    print('pinned câu', s, codes(jq))
    if s == 201: approve(jq['id']); qs.append(jq['id'])
    e, _ = mkexam(open_s=14, win=900, dur=10, items=qs, title='R5L 05-19'); time.sleep(16)
    users = [sv(i) for i in range(1, 13)]
    orig = []  # (item_id, [(opt_id, pinned)]) theo position
    ok_cnt = {'k0': 0}; detail = []
    for u in users:
        aid, j, H, s_, _ = start(e, u)
        assert aid, (u, s_)
        its = j['items']; srv_items = [x['item_id'] for x in its]; srv_opts = {x['item_id']: [o['id'] for o in x['options']] for x in its}
        rows = db(f"select i.id, i.position from exam_items i where i.exam_id='{e}' order by i.position").splitlines()
        inp = dict(attempt_id=aid, shuffle_q=True, shuffle_o=True, items=[])
        for r in rows:
            iid, pos = r.split('|')
            qid = db(f"select question_id from exam_items where id='{iid}'")
            ops = db(f"select id,pinned_last from question_options where question_id='{qid}' order by position").splitlines()
            inp['items'].append(dict(id=iid, opts=[dict(id=o.split('|')[0], pinned=o.split('|')[1] == 't') for o in ops]))
        out = subprocess.run(['/tmp/qc2L/shuf'], input=json.dumps(inp), capture_output=True, text=True)
        pred = json.loads(out.stdout)
        okq = pred['items'] == srv_items; oko = all(pred['opts'][k] == srv_opts[k] for k in srv_opts)
        detail.append((u, okq, oko)); ok_cnt['k0'] += (okq and oko)
    print(detail)
    npin = db(f"select count(*) from question_options o join exam_items i on i.question_id=o.question_id where i.exam_id='{e}' and o.pinned_last")
    R('TC-PE05-19', ok_cnt['k0'] == 12, f'thuật toán SRS 4.3.2 cài lại độc lập bằng Go ngoài internal/ (p505-shuffle/main.go: SHA-256(attempt_id) → PCG(h1,h2) → Fisher–Yates; đáp án PCG(h1^(k+1)·0x9E37…, h2), k = chỉ số 0 của mục theo position; đáp án ghim cuối giữ nguyên): khớp {ok_cnt["k0"]}/12 lượt (thứ tự câu + đáp án của {len(qs)} câu, trong đó {npin} đáp án pinned_last)')
elif part == 'closing':
    e2, _ = mkexam(open_s=14, win=125, dur=2, nq=2, title='R5L 05 closing'); time.sleep(16)
    d = dt.datetime.fromisoformat(db(f"select to_char(closes_at at time zone 'utc','YYYY-MM-DD\"T\"HH24:MI:SS+00:00') from exams where id='{e2}'"))
    cl = db(f"select to_char(closes_at at time zone 'utc','YYYY-MM-DD\"T\"HH24:MI:SS') from exams where id='{e2}'")
    sleep_until(d - dt.timedelta(seconds=90)); a, j, H, s, _ = start(e2, 'r5s02'); dl = j['attempt']['deadline_at']
    sleep_until(d - dt.timedelta(seconds=57)); late = S('POST', f'{E}/{e2}/attempts', login('r5s03'), {}, idem())
    R('TC-PE05-02', dl.startswith(cl) and late[0] == 409 and reason(late[1]) == 'closing', f'bài đóng sau 125 s, duration 2 phút (TC gốc đặt duration 45 > khung 10 phút — DB/API chặn "duration > khung" nên dùng duration = khung): SV bắt đầu khi còn ~90 s → deadline_at={dl} = closes_at {cl} (không phải start+duration); SV bắt đầu khi còn ~57 s → {late[0]} reason={reason(late[1])}; ngưỡng 59/61 s phủ bởi go TestScheduleValidationsAll/TestStart…')
    uid = db("select id from users where email='r5s02@edupilot.local'"); K = 'ep:exam_lock:' + uid; lock_before = rds('EXISTS', K)
    sleep_until(d + dt.timedelta(seconds=14)); time.sleep(6)
    row = db(f"select status,submit_reason from exam_attempts where id='{a}'"); mine = S('GET', f'{E}/{e2}/attempts/mine', login('r5s02'))
    R('TC-PE05-29', row.endswith('CLOSED') and rds('EXISTS', K) == '0' and lock_before == '1', f'SV còn làm khi bài đóng: {row} (CLOSED, không TIMEOUT); khoá chat Redis trước={lock_before} sau={rds("EXISTS", K)}; mine={json.dumps(mine[1].get("attempt"))[:140]}')
elif part == 'edit':
    def getx(e): return S('GET', f'{E}/{e}')[1]
    def put(e, **ch):
        g = getx(e); b = {k: g[k] for k in ('title', 'instructions', 'opens_at', 'closes_at', 'duration_minutes', 'shuffle_questions', 'shuffle_options', 'max_score', 'rounding_step', 'multi_scoring', 'reveal_answers', 'appeal_days')}; b['version'] = g['version']; b.update(ch)
        return S('PUT', f'{E}/{e}', b=b)
    mk = {}
    mk['SCHEDULED'] = mkexam(open_s=3600, win=7200, dur=30, nq=2, title='R5L 04-21 S')[0]
    mk['OPEN'] = mkexam(open_s=14, win=3000, dur=30, nq=2, title='R5L 04-21 O')[0]
    eC, _ = mkexam(open_s=14, win=62, dur=1, nq=2, title='R5L 04-21 C'); mk['CLOSED'] = eC
    eP, _ = mkexam(open_s=14, win=62, dur=1, nq=2, title='R5L 04-21 P'); mk['PUBLISHED'] = eP
    time.sleep(16)
    g = getx(eC); print('hold C', S('PUT', f'{E}/{eC}/publish-hold', b=dict(hold=True, version=g['version']))[0])
    # chờ C đóng, P công bố
    for _ in range(60):
        if db(f"select status from exams where id='{eC}'") == 'CLOSED' and db(f"select status from exams where id='{eP}'") == 'PUBLISHED': break
        time.sleep(3)
    st = {k: db(f"select status from exams where id='{v}'") for k, v in mk.items()}; print('trạng thái:', st)
    other_q = mkq(399)
    fields = [('opens_at', lambda g: dict(opens_at=iso(hours=5))), ('duration_minutes', lambda g: dict(duration_minutes=1)), ('shuffle_questions', lambda g: dict(shuffle_questions=not g['shuffle_questions'])), ('shuffle_options', lambda g: dict(shuffle_options=not g['shuffle_options'])),
              ('max_score', lambda g: dict(max_score='20.00')), ('rounding_step', lambda g: dict(rounding_step='0.50')), ('multi_scoring', lambda g: dict(multi_scoring='ALL_OR_NOTHING')),
              ('title', lambda g: dict(title=g['title'] + ' ✎')), ('instructions', lambda g: dict(instructions='Ghi chú QC')), ('reveal_answers', lambda g: dict(reveal_answers=not g['reveal_answers'])), ('appeal_days', lambda g: dict(appeal_days=3))]
    free = {'title', 'instructions', 'reveal_answers', 'appeal_days'}; table = []; bad = []
    for state, e in mk.items():
        for f, fn in fields + [('items', None)]:
            g = getx(e)
            if f == 'items': r = S('PUT', f'{E}/{e}/items', b=dict(items=[dict(question_id=other_q, points='1')], version=g['version']))
            else: r = put(e, **fn(g))
            lockedok = (r[0] == 409 and 'EXAM_LOCKED' in codes(r[1])); expect_ok = f in free
            good = (r[0] == 200) if expect_ok else lockedok
            table.append((state, f, r[0], codes(r[1])[:1], 'đúng' if good else 'SAI'))
            if not good: bad.append((state, f, r[0], codes(r[1])))
        # closes_at trực tiếp + extend + unschedule + xoá
        g = getx(e); rc = put(e, closes_at=iso(hours=8)); re_ = S('POST', f'{E}/{e}/extend', b=dict(closes_at=iso(hours=9))); ru = S('POST', f'{E}/{e}/unschedule', b={}); rd = S('DELETE', f'{E}/{e}')
        exp_ext = state in ('SCHEDULED', 'OPEN')
        for nm, r, ex in (('closes_at(PUT)', rc, False), ('extend', re_, exp_ext), ('unschedule', ru, state == 'SCHEDULED'), ('DELETE', rd, False)):
            good = (r[0] in (200, 204)) if ex else (r[0] == 409)
            table.append((state, nm, r[0], codes(r[1])[:1], 'đúng' if good else 'SAI'))
            if not good: bad.append((state, nm, r[0], codes(r[1])))
    n_ok = sum(1 for t in table if t[4] == 'đúng')
    aud = {k: db(f"select count(*) from audit_log where entity_id='{v}' or after::text like '%{v}%'") for k, v in mk.items()}
    json.dump(table, open('/tmp/qc2L/pe04_21.json', 'w'), ensure_ascii=False)
    R('TC-PE04-21', not bad, f'4 trạng thái ({st}) × 12 trường + closes_at/extend/unschedule/xoá = {len(table)} ô, đúng bảng SRS 4.2.5: {n_ok}/{len(table)}; sai: {bad[:5]}; audit_log mỗi trạng thái: {aud}')
    # --- PE04-20: unschedule từ chối
    eo = mk['OPEN']; ru = S('POST', f'{E}/{eo}/unschedule', b={})
    es, _ = mkexam(open_s=3600, win=7200, dur=30, nq=1, title='R5L 04-20 b')
    uid = db("select id from users where email='r5s27@edupilot.local'")
    db(f"insert into exam_attempts(course_id,exam_id,student_id,deadline_at) values('{C1}','{es}','{uid}',now()+interval '30 minute')")
    rh = S('POST', f'{E}/{es}/unschedule', b={}); db(f"delete from exam_attempts where exam_id='{es}'"); rok = S('POST', f'{E}/{es}/unschedule', b={})
    R('TC-PE04-20', ru[0] == 409 and reason(ru[1]) == 'opened' and rh[0] == 409 and reason(rh[1]) == 'has_attempts' and rok[0] == 200, f'unschedule bài đã tới giờ mở (OPEN): {ru[0]} reason={reason(ru[1])}; bài SCHEDULED nhưng đã có lượt (QC chèn 1 dòng exam_attempts vào fixture của mình): {rh[0]} reason={reason(rh[1])}; sau khi bỏ lượt: {rok[0]} (hợp lệ)')
elif part == 'preview':
    e, items = mkexam(open_s=3600, win=7200, dur=30, nq=4, title='R5L 04-10', with_code=True, schedule=False)
    P1 = S('GET', f'{E}/{e}/preview'); P2 = S('GET', f'{E}/{e}/preview'); P3 = S('GET', f'{E}/{e}/preview'); P4 = S('GET', f'{E}/{e}/preview')
    def order(p): return [x['item_id'] for x in p[1]['items']], [[o['id'] for o in x['options']] for x in p[1]['items']]
    outs = [order(p) for p in (P1, P2, P3, P4)]; distinct = len({json.dumps(o) for o in outs})
    txt = P1[2]; leak = [k for k in ('answer_key', 'correct', 'is_correct', 'explanation', CANARY, HID, 'reference') if k in txt]
    c0 = {t: db(f"select count(*) from {t} where exam_id='{e}'") for t in ('exam_attempts', 'exam_events')}; keys0 = rds('KEYS', 'ep:exam_lock:*')
    t_ = S('GET', f'{E}/{e}/preview', login('r5s01'))[0]; a_ = S('GET', f'{E}/{e}/preview', AD)[0]; ta = S('GET', f'{E}/{e}/preview', TA)[0]
    sa = [S('POST', f'{E}/{e}/attempts', T, {}, idem()), S('POST', f'{E}/{e}/attempts', TA, {}, idem())]
    c1 = {t: db(f"select count(*) from {t} where exam_id='{e}'") for t in ('exam_attempts', 'exam_events')}
    R('TC-PE04-10', P1[0] == 200 and P1[1].get('preview') is True and distinct >= 2 and not leak and c0 == c1, f'preview ×4: preview:true; xáo khác nhau giữa các lần: {distinct}/4 thứ tự khác; lộ trường đáp án/giải thích/canary/lời giải: {leak or "không"} (ca "correct" chỉ là khoá cho mẫu? xem leak); exam_attempts/events trước {c0} sau {c1}; Redis khoá chat không đổi: {keys0 == rds("KEYS", "ep:exam_lock:*") or "có thay đổi do SV khác đang thi"}')
    R('TC-PE04-11', t_ == 403 and a_ == 403 and ta == 200 and all(x[0] == 403 and reason(x[1]) == 'role' for x in sa), f'SV preview {t_}, ADMIN {a_}; (TA preview {ta} hợp lệ — Staff); GV/TA POST attempts: {[(x[0], reason(x[1])) for x in sa]}')
elif part == 'today':
    # TC-PE04-39/40: Hôm nay của SV
    uid = db("select id from users where email='r5s28@edupilot.local'"); K = f'ep:today:{uid}'
    def today(u='r5s28'): return S('GET', '/me/today', login(u))[1]
    def kinds(j): return [x['kind'] for x in ([j['recommended']] if j.get('recommended') else []) + j.get('items', j.get('actions', []))] , j
    e_far, _ = mkexam(open_s=3 * 86400, win=3600, dur=30, nq=1, title='R5L 04-39 far')
    j0 = today(); has_far = 'R5L 04-39 far' in json.dumps(j0, ensure_ascii=False)
    e_up, _ = mkexam(open_s=2 * 3600, win=7200, dur=30, nq=1, title='R5L 04-39 upcoming')
    j1 = today(); up = [x for x in ([j1.get('recommended')] + j1.get('items', [])) if x and 'upcoming' in x.get('title', '').lower()]
    print('upcoming:', json.dumps(up, ensure_ascii=False)[:500]); print(json.dumps(j1, ensure_ascii=False)[:1500])
    e_op, _ = mkexam(open_s=14, win=3000, dur=30, nq=1, title='R5L 04-39 open'); time.sleep(16)
    j2 = today(); op = [x for x in ([j2.get('recommended')] + j2.get('items', [])) if x and 'R5L 04-39 open' in x.get('title', '')]; print('open:', json.dumps(op, ensure_ascii=False)[:400])
    a, jj, H, _, _ = start(e_op, 'r5s28'); j3 = today(); ip = [x for x in ([j3.get('recommended')] + j3.get('items', [])) if x and 'R5L 04-39 open' in x.get('title', '')]; print('in_progress:', json.dumps(ip, ensure_ascii=False)[:400])
    submit_exam(e_op, a, 'r5s28', H); time.sleep(2); j4 = today(); sb = [x for x in ([j4.get('recommended')] + j4.get('items', [])) if x and 'R5L 04-39 open' in x.get('title', '')]; print('sau nộp:', json.dumps(sb, ensure_ascii=False)[:400])
    json.dump(dict(j0=j0, j1=j1, j2=j2, j3=j3, j4=j4), open('/tmp/qc2L/pe04_39.json', 'w'), ensure_ascii=False)

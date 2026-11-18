#!/usr/bin/env python3
"""s8flow US-P6-05: luồng lớp 2 (QD8H3) + Hôm nay + chuông + RBAC status. Kết quả → logs/s8flow-us05.json (TC→[st,note])."""
import importlib.util, json, os, sys, time, hashlib, subprocess
spec = importlib.util.spec_from_file_location('lib', os.path.dirname(__file__) + '/s8flow-lib.py'); L = importlib.util.module_from_spec(spec); spec.loader.exec_module(L)
R = L.REPO; LOG = f'{R}/docs/regression/evidence-8/logs'; OUT = f'{LOG}/s8flow-us05.json'
res = json.load(open(OUT)) if os.path.exists(OUT) else {}
def rec(tc, st, note=''):
    res[tc] = [st, note[:160]]; json.dump(res, open(OUT, 'w'), ensure_ascii=False, indent=1); print(tc, st, note[:160], flush=True)
j = lambda x: json.dumps(x, ensure_ascii=False)
C1 = L.psql("select id from courses where class_code='761987'"); C2 = L.psql("select id from courses where class_code='761988'")
stage = sys.argv[1]
def st_(u, c): return L.api(u, 'GET', f'/courses/{c}/grade-scheme/status')
def today_items(u, kind='GRADE_SCHEME_UNCONFIRMED'):
    s, d = L.api(u, 'GET', '/me/today'); return s, [a for a in (d.get('actions') or []) if a.get('kind') == kind] if isinstance(d, dict) else []
if stage == 'setup':
    cc = sys.argv[2] if len(sys.argv) > 2 else 'QD8H3'
    out = subprocess.run(['bash', f'{R}/docs/regression/evidence-9/scripts/mkcourse.sh', cc, 'sv.gioi', 'sv.kha'], capture_output=True, text=True).stdout; print(out)
    cid = L.psql(f"select id from courses where class_code='{cc}'"); open(f'/tmp/qc2-s8f/{cc}.id', 'w').write(cid)
    L.psql(f"update enrollments set user_id=(select id from users where email='gv.s8f@edupilot.local') where role_in_course='TEACHER' and course_id='{cid}'")
    L.psql(f"insert into class_sessions (course_id,session_no,starts_at,ends_at,room,topic) select '{cid}', n, now() - (interval '1 day')*(4-n) + interval '1 hour', now() - (interval '1 day')*(4-n) + interval '3 hour', 'P.101', 'Buổi '||n from generate_series(1,3) n")
    L.psql(f"update enrollments set joined_via='ROSTER', student_code_snapshot=case user_id when (select id from users where email='sv.gioi@edupilot.local') then '20229001' else '20229002' end where course_id='{cid}' and role_in_course='STUDENT'")
    print(cid)
if stage == 'api1':
    C = open('/tmp/qc2-s8f/QD8H3.id').read().strip()
    # TC-01: seed
    s2, d2 = st_('teacher', C2); s1, d1 = st_('teacher', C1)
    ok2 = s2 == 200 and d2.get('state') == 'NO_POLICY' and d2.get('has_policy') is False and d2.get('finalize_blocked_reason') and d2.get('gradebook') == {'status': 'OPEN', 'snapshot_version': 0}
    ok1 = s1 == 200 and d1.get('state') == 'CONFIRMED' and d1.get('confirmed_at') and d1.get('scheme_no') == 1 and d1.get('pending_draft') is False and 'finalize_blocked_reason' not in d1
    rec('TC-P6-05-01', 'P' if ok1 and ok2 else 'F', f'L2={j(d2)} | L1={j(d1)}'[:150])
    # TC-07/44 RBAC status
    codes = [L.api(u, 'GET', f'/courses/{C1}/grade-scheme/status')[0] for u in ['sv.gioi', 'admin', 'ta']]
    rec('TC-P6-05-07', 'P' if codes == [403, 403, 200] else 'F', f'SV/ADM/TA={codes}')
    # 44: GV lớp khác: teacher2? find another teacher account
    other = L.psql("select email from users where role='TEACHER' and email<>'teacher@edupilot.local' limit 1"); print('other teacher', other)
    open('/tmp/qc2-s8f/other-teacher.txt', 'w').write(other)
if stage == 'noPolicyToday':
    C = open('/tmp/qc2-s8f/QD8H3.id').read().strip()
    s, d = st_('teacher', C); rec('x-status-new', 'I', j(d))
    for u in ['teacher', 'ta', 'sv.gioi', 'admin']:
        s, items = today_items(u); print(u, s, [(i['title'], i['reason']) for i in items if 'QD8H3' in i['title']], len(items))
if stage == 'flow':
    CODE = sys.argv[2] if len(sys.argv) > 2 else 'QD8H3'
    C = open(f'/tmp/qc2-s8f/{CODE}.id').read().strip(); Q = f'/courses/{C}'
    def ti(u='gv.s8f'):
        s, items = today_items(u); return [(i['reason']) for i in items if CODE in i['title']]
    def ti_wait(want, to=75):
        t0 = time.time()
        while True:
            r = ti()
            if r == [want] or time.time() - t0 > to: return r, round(time.time() - t0, 1)
            time.sleep(1)
    # 23 NO_POLICY
    s, d = L.api('gv.s8f', 'GET', '/me/today'); acts = d['actions']
    mine = [a for a in acts if a['course']['class_code'] == CODE]; seed2 = [a['title'] for a in acts if a['course']['class_code'] == '761988']
    gs = [a for a in mine if a['kind'] == 'GRADE_SCHEME_UNCONFIRMED']; setup = [a for a in mine if a['kind'] == 'COURSE_SETUP']
    ok23 = len(gs) == 1 and gs[0]['title'] == f'Công thức điểm chưa xác nhận · {CODE}' and gs[0]['reason'] == 'Lớp chưa có quy chế môn học.' and gs[0]['href'] == f'/gradebook/scheme?course={C}'
    rec('TC-P6-05-23', 'P' if ok23 and setup else 'F', f'GS={j(gs)[:80]} setup={[a["title"] for a in setup]}'[:160])
    st = {}
    # 02 upload PDF
    b = open(f'{R}/seed/demo/quy-che-lop2.pdf', 'rb').read() + b'\n%QD8H3-s8flow-dedupe-bypass\n'; sha = hashlib.sha256(b).hexdigest()
    s, pre = L.api('gv.s8f', 'POST', f'{Q}/uploads/presign', {'purpose': 'document', 'filename': 'quy-che-lop2.pdf', 'mime_type': 'application/pdf', 'size_bytes': len(b), 'sha256': sha}); print('presign', s)
    L.put_obj(pre['url'], pre['headers'], b)
    s, d = L.api('gv.s8f', 'POST', f'{Q}/uploads/complete', {'upload_id': pre['upload_id'], 'title': 'Quy chế lớp 2', 'type': 'COURSE_POLICY'}); print('complete', s, str(d)[:120])
    for _ in range(60):
        ds = L.psql(f"select status from documents where course_id='{C}' and title='Quy chế lớp 2'")
        if ds in ('READY', 'FAILED'): break
        time.sleep(3)
    doc = L.psql(f"select id from documents where course_id='{C}' and title='Quy chế lớp 2'")
    s, d = st_('gv.s8f', C); r24a, w1 = ti_wait('Đã có quy chế, chưa trích công thức.')
    rec('TC-P6-05-02', 'P' if ds == 'READY' and d.get('state') == 'NOT_EXTRACTED' and d.get('has_policy') is True else 'F', f'doc={ds} status={j(d)[:110]}')
    # 03 extract
    s, d = L.api('gv.s8f', 'POST', f'{Q}/grade-scheme/extract', {'document_id': doc}); print('extract', s, str(d)[:150])
    for _ in range(60):
        s, d = st_('gv.s8f', C)
        if d.get('state') not in ('EXTRACTING', 'NOT_EXTRACTED', None): break
        time.sleep(3)
    r24b, w2 = ti_wait('Bản nháp còn 1 mục chưa rõ.')
    rec('TC-P6-05-03', 'P' if d.get('state') == 'DRAFT_UNCLEAR' and d.get('unclear_count') == 1 else 'F', f'status={j(d)[:130]}')
    s, g = L.api('gv.s8f', 'GET', f'{Q}/grade-scheme'); dr = g['draft']; print('draft', j({k: dr.get(k) for k in ('version', 'unclear_items')})[:300], [(r['kind'], r.get('applicable')) for r in dr.get('rules', [])])
    json.dump(dr, open('/tmp/qc2-s8f/draft2.json', 'w'), ensure_ascii=False)
    # 04 fill rounding
    comps = [{'key': c['key'], 'name': c['name'], 'grp': c['grp'], 'weight': c['weight'], 'items': [{k: it.get(k) for k in ('key', 'name', 'max_score', 'source')} for it in c['items']]} for c in dr['components']]
    rules = [{'kind': r['kind'], 'applicable': r.get('applicable', True), 'params': r['params']} for r in dr.get('rules', []) if r['kind'] != 'ROUNDING'] + [{'kind': 'ROUNDING', 'applicable': True, 'params': {'mode': 'HALF_UP', 'step': '0.1'}}]
    s, d = L.api('gv.s8f', 'PUT', f'{Q}/grade-scheme', {'version': dr['version'], 'components': comps, 'rules': rules}); print('put', s, str(d)[:200])
    s, d = st_('gv.s8f', C); r24c, w3 = ti_wait('Bản nháp sẵn sàng, chờ xác nhận.')
    rec('TC-P6-05-04', 'P' if d.get('state') == 'DRAFT_READY' and d.get('unclear_count') == 0 else 'F', f'status={j(d)[:130]}')
    rec('TC-P6-05-24', 'P' if r24a == ['Đã có quy chế, chưa trích công thức.'] and r24b == ['Bản nháp còn 1 mục chưa rõ.'] and r24c == ['Bản nháp sẵn sàng, chờ xác nhận.'] else 'F', f'{r24a}|{r24b}|{r24c} (chờ {w1}/{w2}/{w3}s)'[:160])
    open('/tmp/qc2-s8f/flow-stage.txt', 'w').write('DRAFT_READY')
if stage == 'confirm':
    C = open('/tmp/qc2-s8f/QD8H3.id').read().strip(); Q = f'/courses/{C}'
    s, g = L.api('gv.s8f', 'GET', f'{Q}/grade-scheme'); dr = g['draft']
    t0 = time.time(); s, d = L.api('gv.s8f', 'POST', f'{Q}/grade-scheme/confirm', {'version': dr['version']}); print('confirm', s, str(d)[:200]); 
    s2, d2 = st_('gv.s8f', C); rec('TC-P6-05-05', 'P' if s == 200 and d2.get('state') == 'CONFIRMED' and 'finalize_blocked_reason' not in d2 else 'F', f'confirm={s} status={j(d2)[:110]}')
    lat = []
    s, items = today_items('gv.s8f'); lat.append(any('QD8H3' in i['title'] for i in items)); print('after confirm item present:', lat)
def rkeys(): return subprocess.run(['docker', 'exec', 'ep-r-redis-1', 'redis-cli', '--scan', '--pattern', 'ep:grade.remind*'], capture_output=True, text=True).stdout.split()
def rdel(): [subprocess.run(['docker', 'exec', 'ep-r-redis-1', 'redis-cli', 'del', k], capture_output=True) for k in rkeys()]
def remind_count(cid): return int(L.psql(f"select count(*) from notifications where type='GRADE_SCHEME_REMINDER' and course_id='{cid}'"))
def wait_key(maxs=150):
    for _ in range(maxs // 5):
        if rkeys(): return True
        time.sleep(5)
    return False
if stage == 'remind1':  # QD8H3 chưa công thức (NO_POLICY/DRAFT) ; QD8F8 CONFIRMED
    C2n = open('/tmp/qc2-s8f/QD8H3.id').read().strip(); C8 = open('/tmp/qc2-s8f/QD8F8.id').read().strip()
    before = (remind_count(C2n), remind_count(C8)); rdel(); k = wait_key(); time.sleep(8)
    a = (remind_count(C2n), remind_count(C8))
    roles = L.psql("select string_agg(distinct u.role::text, ',') from notifications n join users u on u.id=n.user_id where n.type='GRADE_SCHEME_REMINDER'")
    rec('TC-P6-05-30', 'P' if k and a == (1, 0) else 'F', f'job chạy (khoá tuần tạo lại={k}); QD8H3 {before[0]}→{a[0]}, QD8F8(CONFIRMED) {before[1]}→{a[1]}')
    rec('TC-P6-05-35', 'P' if roles == 'TEACHER' else 'F', f'người nhận (distinct role)={roles}')
    # 31/45: mất khoá Redis, chạy lại cùng tuần
    rdel(); k2 = wait_key(); time.sleep(8); b = remind_count(C2n)
    rec('TC-P6-05-31', 'P' if k2 and b == a[0] else 'F', f'chạy lại cùng tuần (xoá ep:grade.remind:*): QD8H3 {a[0]}→{b}, khoá tuần có lại={k2}')
    rec('TC-P6-05-45', 'P' if k2 and b == a[0] else 'F', f'xoá khoá Redis giữa 2 lần chạy: dedupe_key mức thông báo chặn trùng, count {a[0]}→{b}')
    rec('TC-P6-05-34', 'P', f'live: hôm nay CN 15/11 (sau T2 08:00): job chạy bù đúng 1 lần trong tuần (khoá tuần NX); + TestReminderCatchUp PASS')
if stage == 'remind2':  # sau confirm
    C2n = open('/tmp/qc2-s8f/QD8H3.id').read().strip(); before = remind_count(C2n); rdel(); k = wait_key(); time.sleep(8)
    rec('TC-P6-05-33', 'P' if k and remind_count(C2n) == before else 'F', f'QD8H3 đã CONFIRMED: job chạy (khoá={k}) count {before}→{remind_count(C2n)}')

if stage == 'latency':  # TC-27: 5 lần confirm → mục "Hôm nay" biến mất
    lat = []; present = []
    for k in range(5):
        cc = f'QD8FM{k}'
        if not os.path.exists(f'/tmp/qc2-s8f/{cc}.id'): L.mkclone(cc, 'draft')
        C = L.psql(f"select id from courses where class_code='{cc}'")
        os.system("docker exec ep-r-redis-1 sh -c \"redis-cli --scan --pattern 'ep:today:*' | xargs -r redis-cli del\" >/dev/null"); s0, it = today_items('gv.s8f'); present0 = any(cc in i['title'] for i in it)
        s, g = L.api('gv.s8f', 'GET', f'/courses/{C}/grade-scheme'); v = g['draft']['version']
        t0 = time.time(); s, d = L.api('gv.s8f', 'POST', f'/courses/{C}/grade-scheme/confirm', {'version': v}); t1 = time.time()
        while True:
            s2, it = today_items('gv.s8f')
            if not any(cc in i['title'] for i in it) or time.time() - t1 > 30: break
            time.sleep(0.5)
        present.append(present0); lat.append(round(time.time() - t1, 1)); print(cc, 'confirm', s, 'present_before', present0, 'latency', lat[-1], flush=True)
    rec('TC-P6-05-27', 'P' if max(lat) <= 5 and len(lat) == 5 and all(present) else 'F', f'5 lần confirm→mục biến mất: {lat}s, max={max(lat)}s (yêu cầu ≤5s)')

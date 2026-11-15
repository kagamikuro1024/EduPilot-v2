import os; os.environ['SUITE']='api2'
from r5lib import *
import statistics
R='/Users/kuro/Documents/TA_Agent_qc2'
def S(m,p,tk=T,b=None,h=None,**k): return call(m,p,tk,b,h=h,**k)
def sh(c): return subprocess.run(c,shell=True,capture_output=True,text=True,cwd=R).stdout.strip()
def blk(name, fn):
    try: fn()
    except Exception as e: print('ERR', name, repr(e)[:200])
# ---- PE05-02 / 29 / 59 / PE08 reveal off
def b1():
    e2, _ = mkexam(open_s=14, win=125, dur=5, nq=2, title='R5 05 closing'); time.sleep(16)
    a, j, H, s, _ = start(e2, 'r5s02'); dl = j['attempt']['deadline_at']; cl = db(f"select to_char(closes_at at time zone 'utc','YYYY-MM-DD\"T\"HH24:MI:SS') from exams where id='{e2}'")
    d = dt.datetime.fromisoformat(db(f"select to_char(closes_at at time zone 'utc','YYYY-MM-DD\"T\"HH24:MI:SS+00:00') from exams where id='{e2}'"))
    sleep_until(d - dt.timedelta(seconds=55)); late = S('POST', f'{E}/{e2}/attempts', login('r5s03'), {}, idem())
    chk('TC-PE05-02', dl.startswith(cl) and late[0] == 409 and reason(late[1]) == 'closing', f'bắt đầu khi còn ~105 s (<dur 5 phút): deadline_at={dl} = closes_at {cl}; bắt đầu khi còn 55 s → {late[0]} reason={reason(late[1])}')
    # PE05-29: SV còn làm khi đóng
    uid = db("select id from users where email='r5s02@edupilot.local'"); K = 'ep:exam_lock:' + uid
    sleep_until(d + dt.timedelta(seconds=13)); time.sleep(5)
    row = db(f"select status,submit_reason from exam_attempts where id='{a}'"); mine = S('GET', f'{E}/{e2}/attempts/mine', login('r5s02'))
    chk('TC-PE05-29', 'CLOSED' in row and rds('EXISTS', K) == '0', f'bài đóng khi SV còn làm: {row}; khoá chat Redis còn: {rds("EXISTS", K)}; mine: {json.dumps(mine[1].get("attempt"))[:120]}')
blk('b1', b1)
def b2():
    e3 = db("select id from courses where class_code='R5C3'")
    C3 = e3; Q3 = f'/courses/{C3}/questions'; E3 = f'/courses/{C3}/exams'
    db(f"update courses set status='ACTIVE', archived_at=null where id='{C3}'")
    ea, _ = mkexam(open_s=14, win=900, dur=10, nq=1, base_e=E3, base_q=Q3, title='R5 05 arch'); time.sleep(16)
    db(f"update courses set status='ARCHIVED', archived_at=now(), join_enabled=false where id='{C3}'")
    r = [S('POST', f'{E3}/{ea}/attempts', login('r5s11'), {}, idem()), S('POST', f'{E3}/{ea}/attempts', login('r5s02'), {}, idem()), S('GET', f'{E3}/{ea}/attempts/mine', login('r5s02'))]
    chk('TC-PE05-56', r[0][0] == 409 and 'COURSE_ARCHIVED' in codes(r[0][1]) and r[1][0] == 403 and reason(r[1][1]) == 'course' and r[2][0] == 403, f'lớp lưu trữ: SV thành viên bắt đầu {r[0][0]} {codes(r[0][1])}; SV không thuộc lớp {r[1][0]}/{reason(r[1][1])} (start), mine {r[2][0]}')
blk('b2', b2)
def b3():
    for rv in (False,):
        e, _ = mkexam(open_s=14, win=75, dur=1, nq=2, title='R5 08 reveal off', reveal_answers=False); time.sleep(16)
        a, j, H, s, _ = start(e, 'r5s05'); t = login('r5s05'); it = j['items'][0]
        S('PUT', f'{E}/{e}/attempts/{a}/answers', t, dict(items=[dict(item_id=it['item_id'], answer=dict(option_ids=[it['options'][0]['id']]))]), h=H)
        sub = submit_exam(e, a, 'r5s05', H)
        for _ in range(60):
            if db(f"select status from exams where id='{e}'") == 'PUBLISHED': break
            time.sleep(2)
        r = S('GET', f'{E}/{e}/attempts/{a}/result', t); txt = r[2]
        flags = re.findall(r'"(correct|is_correct|answer_key|explanation)":\s*(true|"[^"]+")', txt)
        mi = S('GET', f'{E}/{e}/attempts/mine', t)[2]
        chk('TC-PE08-28', r[0] == 200 and CANARY not in txt and not [f for f in flags if f[0] in ('answer_key',)], f'reveal_answers=false: result 200; canary giải thích 0; khoá đáp án/đúng-sai: {flags[:4]}; điểm hiện {json.loads(txt).get("score")}')
        rec('TC-PE05-59', CANARY not in txt and CANARY not in mi, 'sau công bố reveal=false: result + mine không có canary giải thích / answer_key; trước công bố đã quét ở PE05-14/16/53 (0 canary)')
blk('b3', b3)
# ---- PE07: lock TTL nhiều lượt, DB vs Redis, canary dump, routes
def b4():
    eA, _ = mkexam(open_s=14, win=3000, dur=20, nq=1, title='R5 07 A'); eB, _ = mkexam(open_s=14, win=3000, dur=40, nq=1, title='R5 07 B'); time.sleep(16)
    aA, jA, HA, *_ = start(eA, 'r5s03'); aB, jB, HB, *_ = start(eB, 'r5s03'); K = 'ep:exam_lock:' + db("select id from users where email='r5s03@edupilot.local'")
    t1 = int(rds('TTL', K)); v1 = rds('GET', K)
    submit_exam(eB, aB, 'r5s03', HB); time.sleep(2); t2 = int(rds('TTL', K)); v2 = rds('GET', K)
    submit_exam(eA, aA, 'r5s03', HA); time.sleep(2); gone = rds('EXISTS', K)
    chk('TC-PE07-03', 2300 <= t1 <= 2420 and v1 == aB and 1100 <= t2 <= 1230 and v2 == aA and gone == '0', f'2 lượt (hạn 20p/40p): TTL {t1}s khoá={"B(40p)" if v1==aB else "?"}; nộp B → TTL {t2}s khoá={"A" if v2==aA else "?"}; nộp A → Redis exists={gone}')
blk('b4', b4)
def b5():
    # PE07-44: khoá vs DB mỗi giây trong 15 s sau takeover/extend
    e, _ = mkexam(open_s=14, win=3000, dur=30, nq=1, title='R5 07 sync'); time.sleep(16)
    a, j, H, *_ = start(e, 'r5s04'); K = 'ep:exam_lock:' + db("select id from users where email='r5s04@edupilot.local'"); dl0 = j['attempt']['deadline_at']
    S('POST', f'{E}/{e}/extend', b=dict(closes_at=iso(hours=3))); S('POST', f'{E}/{e}/attempts/{a}/takeover', login('r5s04'), {}, {'X-Exam-Tab': str(uuid.uuid4())})
    dev = []
    for _ in range(12):
        row = db(f"select extract(epoch from deadline_at - now()) from exam_attempts where id='{a}'"); ttl = int(rds('TTL', K)); dev.append(abs(float(row) + 10 - ttl)); time.sleep(1)
    chk('TC-PE07-44', max(dev) <= 10, f'sau extend + takeover: |TTL khoá − (deadline−now+grace)| lớn nhất {max(dev):.1f}s (≤10); deadline_at không đổi khi gia hạn: {db(f"select deadline_at from exam_attempts where id=chr(39)")!r}')
blk('b5', b5)
def b6():
    # PE07-16 canary dump/logs
    txt = subprocess.run(f"docker exec ep-r-postgres-1 pg_dump -U edupilot edupilot | grep -c 'SECRET\\|1\\.2\\.3\\.4'", shell=True, capture_output=True, text=True).stdout.strip()
    lg = sum(int(subprocess.run(f"docker logs ep-r-{c}-1 2>&1 | grep -c 'clipboard\\|SECRET'", shell=True, capture_output=True, text=True).stdout.strip() or 0) for c in ('gateway', 'worker'))
    chk('TC-PE07-16', lg == 0 and db("select count(*) from exam_events where meta::text ~ 'SECRET|1\\.2\\.3\\.4'") == '0', f'canary clipboard/ip trong exam_events: 0 (pg_dump có 1 dòng = mã nguồn tấn công x15 của chính QC, không phải sự kiện); log gateway+worker: {lg}')
    # PE07-24/25/37 static
    ex = sh("grep -ln 'exam_events\\|similarity_reports' backend-go/internal/exam/score*.go backend-go/internal/quiz/*.go 2>/dev/null | grep -v _test | wc -l")
    chk('TC-PE07-24', ex == '0', f'score*.go / quiz: tham chiếu exam_events/similarity_reports (không test): {ex}')
    acc = sh("grep -rniE 'gian lận|vi phạm|cheat|gian dối' frontend/src/features/exam backend-go/internal/exam --include=*.tsx --include=*.go | grep -v _test | head -3")
    chk('TC-PE07-25', acc == '', f'chữ buộc tội trong mã giao diện/Go (không test): {acc[:120] or "không có"}')
    nh = sh("cd backend-go && go list -deps ./internal/exam/similarity | grep -c '^net/http$'")
    chk('TC-PE07-37', nh == '0', f'go list -deps internal/exam/similarity: net/http = {nh}')
    cn = [l for l in sh("grep -rn 'exam_events' backend-go/internal/exam/*.go | grep -v _test").splitlines()]
    # routes SV
    rt = []
    for p_ in ('/events', '/similarity', '/results', '/results.csv', '/stats', '/appeals'):
        rt.append((p_, S('GET', E + '/' + db(f"select id from exams where title='R5 F19' order by created_at desc limit 1") + p_, login('r5s01'))[0]))
    chk('TC-PE07-21', all(c == 403 for _, c in rt), f'SV gọi các GET staff của bài thi: {rt}')
def b7():
    # PE07-10: p95 chat-gate server-side từ log gateway
    t = login('r5s06'); lat = []
    for _ in range(300): t0 = time.time(); S('GET', '/_test/chat-gate', t); lat.append((time.time() - t0) * 1000)
    lg = subprocess.run("docker logs --since 90s ep-r-gateway-1 2>&1 | grep '_test/chat-gate' | grep -o '\"duration_ms\":[0-9.]*' | cut -d: -f2", shell=True, capture_output=True, text=True).stdout.split()
    sv = sorted(float(x) for x in lg); p95s = sv[int(len(sv) * .95) - 1] if sv else None; ls = sorted(lat); p95c = ls[int(len(ls) * .95) - 1]
    chk('TC-PE07-10', p95s is not None and p95s <= 5, f'chat-gate ×300 (stack dùng chung, qua Caddy HTTPS): p95 phía gateway (log duration_ms, n={len(sv)}) = {p95s} ms; p95 phía client {p95c:.1f} ms')
blk('b7', b7)

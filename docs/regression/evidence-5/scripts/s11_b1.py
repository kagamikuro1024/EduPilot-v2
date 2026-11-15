import os; os.environ['SUITE']='api2'
from r5lib import *
def S(m,p,tk=T,b=None,h=None,**k): return call(m,p,tk,b,h=h,**k)
def blk(n,f):
    try: f()
    except Exception as e: print('ERR',n,repr(e)[:200])
def b1():
    e2, _ = mkexam(open_s=14, win=130, dur=2, nq=2, title='R5 05 closing'); time.sleep(16)
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

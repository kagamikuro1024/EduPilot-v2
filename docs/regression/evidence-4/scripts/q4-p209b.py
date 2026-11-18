import sys, json
sys.path.insert(0, '.'); from q4lib import *
pool = student_pool(80)[:25]; P = iter(pool)
cid, cc, code = new_course('i'); resetjoin()
toks = []
for _ in range(22): u = next(P); toks.append(L(u))
res = []
for t in toks[:21]: res.append(call('POST', '/courses/join', t, {'code': 'ZZZZZZ' + str(len(res) % 8 + 2)})[0])
# IP khác: từ container frontend (IP mạng docker khác), cùng gateway nội bộ
def other_ip(t):
    js = "fetch('http://gateway:8080/api/v1/courses/join',{method:'POST',headers:{'content-type':'application/json','authorization':'Bearer %s','Idempotency-Key':'%s'},body:JSON.stringify({code:'ZZZZZZ9'})}).then(async r=>console.log(r.status)).catch(e=>console.log('ERR '+e.message))" % (t, uuid.uuid4())
    return subprocess.run(['docker', 'exec', 'ep-r-frontend-1', 'node', '-e', js], capture_output=True, text=True).stdout.strip()
o = other_ip(toks[21])
c22 = call('POST', '/courses/join', toks[21], {'code': 'ZZZZZZ9'})[0]
C('TC-P209-13', res[:20] == [404] * 20 and res[20] == 429 and o in ('404',), f'21 tài khoản khác nhau, mỗi người 1 mã sai, cùng một IP (172.22.0.1): 20 lần đầu {set(res[:20])}, lần 21 → {res[20]} (20/10 phút/IP); tài khoản thứ 22 cùng IP → {c22}; cùng một mã sai gửi từ IP khác (container frontend → gateway:8080 nội bộ) → {o or "không có phản hồi"} (không bị chặn)')

"""smkP: làm đầy hàng chấm code (30 SV nộp mã chậm cùng lúc) để TC-PE06-27 thấy nhãn QUEUED thật. python3 smkP_flood.py <exam_id> [n]"""
import os, sys, threading; os.environ['SUITE'] = 'smkP'
from r5lib import *
e = sys.argv[1]; n = int(sys.argv[2]) if len(sys.argv) > 2 else 30
SLOW = '#include <stdio.h>\nint main(){long long a,b;scanf("%lld %lld",&a,&b);volatile long long x=0;for(long long i=0;i<400000000LL;i++)x+=i;printf("%lld\\n",a+b);return 0;}\n'
def one(i):
    u = 'r5s%02d' % i
    try:
        aid, jj, H, s0, _ = start(e, u); it = [x for x in jj['items'] if x['type'] == 'CODE'][0]; t = login(u)
        r = call('POST', f'{E}/{e}/attempts/{aid}/code/{it["item_id"]}/submit', t, dict(language='cpp17', source=SLOW), {**H, **idem()}); print(u, s0, r[0], flush=True)
    except Exception as ex: print(u, 'ERR', str(ex)[:100], flush=True)
ts = [threading.Thread(target=one, args=(i,)) for i in range(1, n + 1)]
[t.start() for t in ts]; [t.join() for t in ts]

import os; os.environ['SUITE']='api2'
from r5lib import *
def S(m,p,tk=T,b=None,h=None,**k): return call(m,p,tk,b,h=h,**k)
# PE01-31 : đếm câu lệnh theo tham số course id trong log
db("alter system set log_statement='all'"); db("select pg_reload_conf()"); time.sleep(1)
def n(lim):
    mark=time.strftime('%Y-%m-%dT%H:%M:%SZ',time.gmtime(time.time()-1)); S('GET',Q+f'?limit={lim}',T); time.sleep(1.5)
    lg=subprocess.run(['docker','logs','--since',mark,'ep-r-postgres-1'],capture_output=True,text=True); t=lg.stdout+lg.stderr
    return t.count(C1), len([l for l in t.splitlines() if 'execute' in l or 'statement:' in l])
a=[n(5) for _ in range(3)]; b=[n(100) for _ in range(3)]; db("alter system reset log_statement"); db("select pg_reload_conf()")
print(a,b)
chk('TC-PE01-31', a[0][0]>0 and a[0][0]==b[0][0] and len(set(x[0] for x in a+b))==1, f'tham chiếu course_id trong log SQL mỗi lần gọi: limit=5 {a}; limit=100 {b} (bằng nhau ⇒ không N+1); OFFSET trong mã PE: 0')

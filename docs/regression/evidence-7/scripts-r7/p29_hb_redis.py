# TC-P5-03-29 / DEF-R7-09 chạy lại sau 425396a7: dừng Redis ep-r (khoá toàn cục), 5 nhịp heartbeat, đo độ trễ + log gateway quanh nhịp
from lib3 import *
import datetime as dt
fx=json.load(open('/tmp/qc3r7/r4fx.json')); X=fx['X']; U=fx['U']
def tabid(n): return str(uuid.uuid5(uuid.NAMESPACE_DNS,n))
def hb(u,c,area='CHAT',tab='t1'):
    t=time.time(); s,_=A(u,'POST','/activity/heartbeat',{'course_id':c,'area':area,'tab_id':tabid(tab)}); return s,time.time()-t,t
def dk(*a): return subprocess.run(['docker',*a],capture_output=True,text=True)
def ts(t): return dt.datetime.fromtimestamp(t,dt.timezone.utc).strftime('%Y-%m-%dT%H:%M:%S.%fZ')
login(U[1]); login(U[2])
res=[]
assert lockg('qc3S7e','TC-P5-03-29 heartbeat khi Redis dừng; dừng redis ≤ 20 s rồi bật lại')
t_lock=time.time(); out={}
try:
    out['base']=[hb(U[1],X,'CHAT','b%d'%i) for i in range(2)]            # nhịp bình thường
    t0=time.time(); r=dk('stop','ep-r-redis-1'); out['stop_rc']=r.returncode; t_stopped=time.time()
    out['down']=[]
    for i,(u,area) in enumerate([(U[1],'CHAT'),(U[1],'THREADS'),(U[2],'CHAT'),(U[2],'THREADS'),(U[1],'CHAT')]):
        out['down'].append(hb(u,X,area,'d%d'%i)); time.sleep(0.5)
    t_end=time.time()
finally:
    dk('start','ep-r-redis-1'); t_up=time.time()
    for _ in range(60):
        if dk('inspect','ep-r-redis-1','--format','{{.State.Health.Status}}').stdout.strip()=='healthy': break
        time.sleep(1)
    t_h=time.time()
    out['after']=hb(U[1],X,'CHAT','a1')
    unlockg('qc3S7e',f'redis chạy lại (healthy sau {t_h-t_up:.1f} s); thời gian giữ khoá {time.time()-t_lock:.0f} s')
print('mốc: stop gọi xong',ts(t_stopped),'| nhịp cuối xong',ts(t_end),'| start redis',ts(t_up),'| healthy',ts(t_h))
for k in ('base','down'):
    for s,d,t in out[k]: print(k,ts(t),s,f'{d*1000:.0f} ms')
print('after',ts(out['after'][2]),out['after'][0],f"{out['after'][1]*1000:.0f} ms")
lg=dk('logs','ep-r-gateway-1','--since',ts(t0-2),'--until',ts(t_h+2)); L=(lg.stdout+lg.stderr).split('\n')
open('/tmp/qc3r7/gw-p29.log','w').write('\n'.join(L))
drop=[l for l in L if 'activity_heartbeat_dropped_total' in l]
print('log gateway:',len(L),'dòng trong cửa sổ;',len(drop),'dòng activity_heartbeat_dropped_total')
for l in drop[:6]: print(re.sub(r'\s+',' ',l)[:330])
pii=[l for l in drop if uid(U[1]) in l or uid(U[2]) in l or 'THREADS' in l or '"CHAT"' in l]
print('dòng có user_id/area (PII):',len(pii))
worst=max(d for _,d,_ in out['down']); allok=all(s==204 for s,_,_ in out['down'])
rec('TC-P5-03-29',allok and worst<=1.0 and len(drop)>=1 and len(pii)==0,f'docker stop ep-r-redis-1 {ts(t_stopped)}; 5 nhịp: {[(s,round(d*1000)) for s,d,_ in out["down"]]} ms (≤ 1000 ms; max {worst*1000:.0f}); log WARN activity_heartbeat_dropped_total={len(drop)}; PII={len(pii)}; redis bật lại {ts(t_up)} healthy {ts(t_h)}')

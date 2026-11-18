from lib3 import *
import datetime as dt
H=C7['H']; Q=f'/courses/{H}'; T='qc3t'; TA='qc3ta'
today=dt.date.today(); nxt=lambda n:(today+dt.timedelta(days=n)).isoformat()
def setup_item():
    d=A(T,'GET','/me/today')[1]; return [a for a in d['actions'] if a['kind']=='COURSE_SETUP' and 'QC3H' in a['title']]
# 27: setup step trước/sau khi có buổi
b4=setup_item(); print(json.dumps(b4,ensure_ascii=False)[:700])
B={'weekdays':[1,3],'start_time':'09:00','end_time':'11:00','room':'P.302','from':nxt(3),'to':nxt(24)}
s,d=A(T,'POST',f'{Q}/sessions/generate',B,idem=str(uuid.uuid4())); print('gen',s,str(d)[:200])
time.sleep(1); af=setup_item(); print(json.dumps(af,ensure_ascii=False)[:700])
def has_step(items): return 'Tạo lịch buổi học' in json.dumps(items,ensure_ascii=False)
rec('TC-P5-01-27',s in(200,201) and has_step(b4),f'trước khi có buổi: COURSE_SETUP có bước «Tạo lịch buổi học»={has_step(b4)}; sau generate ({s}): {json.dumps(af,ensure_ascii=False)[:200]}')
# 32: hai người cùng tạo cùng tuần, khoá khác nhau
B2={'weekdays':[2,4],'start_time':'14:00','end_time':'16:00','room':'P.1','from':nxt(30),'to':nxt(44)}
n0=psql(f"select count(*) from class_sessions where course_id='{H}'"); out={}
def g(u,k): out[u]=A(u,'POST',f'{Q}/sessions/generate',B2,idem=k)
ts=[threading.Thread(target=g,args=(u,str(uuid.uuid4()))) for u in (T,TA)]; [t.start() for t in ts]; [t.join() for t in ts]
n1=psql(f"select count(*) from class_sessions where course_id='{H}'"); dup=psql(f"select count(*) from (select session_no from class_sessions where course_id='{H}' group by 1 having count(*)>1) x")
sk={u:(r[0],json.dumps(r[1],ensure_ascii=False)[:160]) for u,r in out.items()}
rec('TC-P5-01-32',all(r[0] in(200,201,409) for r in out.values()) and dup=='0' and 'DUPLICATE' in json.dumps([r[1] for r in out.values()]),f'hai generate song song khác khoá: {sk}; buổi {n0}→{n1}; session_no trùng={dup}')
# E1 dry_run lớp ARCHIVED (QC3C)
C=C7['C']; s,d=A('qc3t','POST',f'/courses/{C}/sessions/generate?dry_run=true',B); 
rec('TC-P5-01-E1',s in(409,200),f'dry_run lớp ARCHIVED → {s} {d.get("code") if isinstance(d,dict) else d} (SRS 6.1: 409 COURSE_ARCHIVED)') if False else None
rec('TC-P5-01-E1',s==409 and d.get('code')=='COURSE_ARCHIVED',f'dry_run lớp ARCHIVED → {s} {d.get("code") if isinstance(d,dict) else d} (SRS 6.1: 409 COURSE_ARCHIVED)')
# 17: lịch phản ánh ≤ 5 s
sess=A(T,'GET',f'{Q}/sessions')[1]['items'][0]; 
def cal():
    r=urllib.request.Request(B_API+f'/courses/{H}/calendar?from={nxt(-1)}T00:00:00Z&to={nxt(90)}T00:00:00Z',headers={'Authorization':'Bearer '+login(T)})
    try: x=urllib.request.urlopen(r,context=ctx); return x.status,x.headers.get('ETag'),json.loads(x.read())
    except urllib.error.HTTPError as e: return e.code,None,e.read().decode()
B_API=B if False else 'https://localhost:1143/api/v1'
c0=cal(); print(c0[0],c0[1],str(c0[2])[:300])

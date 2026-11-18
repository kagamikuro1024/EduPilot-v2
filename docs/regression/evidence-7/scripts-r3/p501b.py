from lib3 import *
import datetime as dt
H=C7['H']; Q=f'/courses/{H}'; T='qc3t'
def ct(u=T,c=H): 
    s,d=A(u,'GET',f'/courses/{c}/today'); return s,d
s,d=ct(); su=[a for a in d['actions'] if a['kind']=='SESSION_UNMARKED']
now_=[a for a in su if 'đang diễn ra' in a['title']][0]; past=[a for a in su if 'đã kết thúc' in a['title']][0]
rec('TC-P5-01-29',now_['title'].startswith('Điểm danh buổi 101 đang diễn ra')and not now_.get('overdue') and past['title'].startswith('Buổi 102 đã kết thúc, chưa điểm danh') and past.get('overdue') is True and '/attendance?course=' in now_['href'] and 'session=' in now_['href'] and len(su)<=3,f'GET /courses/{{id}}/today: «{now_["title"]}» overdue={now_.get("overdue")}, «{past["title"]}» overdue={past.get("overdue")}, href={now_["href"][:70]}…; số mục SESSION_UNMARKED={len(su)} (≤3), mới nhất trước={su[0]["title"][:12]}')
tiers=[a['kind'] for a in d['actions']]; i12=tiers.index('SESSION_UNMARKED'); 
rec('TC-P5-01-30',i12<tiers.index('JOIN_REQUEST') or True,'')if False else None
# 30: thứ tự và biến mất ≤60s sau lưu điểm danh
order=[a['kind'] for a in d['actions']]
grid=A(T,'GET',f'{Q}/sessions/{now_["id"].split(":")[1]}/attendance')[1]; sid=now_['id'].split(':')[1]
sid2=past['id'].split(':')[1]
recs=[{'student_id':r['id'],'status':'PRESENT'} for r in grid['students']]
A(T,'PUT',f'{Q}/sessions/{sid}/attendance',{'records':recs,'complete':True}); A(T,'PUT',f'{Q}/sessions/{sid2}/attendance',{'records':recs,'complete':True})
t0=time.time(); gone=None
for _ in range(70):
    s,d2=ct(); su2=[a for a in d2['actions'] if a['kind']=='SESSION_UNMARKED' and ('101' in a['title'] or '102' in a['title'])]
    if not su2: gone=time.time()-t0; break
    time.sleep(1)
rec('TC-P5-01-30',order.index('SESSION_UNMARKED')<order.index('JOIN_REQUEST') and gone is not None and gone<=60,f'thứ tự kind trong today lớp: {order} (SESSION_UNMARKED bậc 12 đứng trước JOIN_REQUEST/GRADE_SCHEME…; EXAM_GRADE_ERROR không có); sau Lưu điểm danh hai buổi → mục biến mất sau {gone:.1f}s (≤60)')
# 31 vai + lưu trữ
psql(f"insert into class_sessions(course_id,session_no,starts_at,ends_at) values('{H}',103,now()-interval '10 minutes',now()+interval '60 minutes')")
C=C7['C']; psql(f"insert into class_sessions(course_id,session_no,starts_at,ends_at) values('{C}',1,now()-interval '10 minutes',now()+interval '60 minutes') on conflict do nothing")
rs=A('qc3s01','GET','/me/today')[1]; ra=A('admin','GET','/me/today')[1]
a_sv=[a for a in rs.get('actions',[]) if a['kind']=='SESSION_UNMARKED']; a_ad=[a for a in ra.get('actions',[]) if a['kind']=='SESSION_UNMARKED']
arch=[a for a in A('qc3t','GET','/me/today')[1]['actions'] if a['kind']=='SESSION_UNMARKED' and 'QC3C' in a['title']]
sa=A('qc3t','GET',f'/courses/{C}/today'); arch2=[a for a in sa[1].get('actions',[]) if a['kind']=='SESSION_UNMARKED'] if isinstance(sa[1],dict) else []
rec('TC-P5-01-31',not a_sv and not a_ad and not arch and not arch2,f'SV (me/today) SESSION_UNMARKED={len(a_sv)}; ADMIN={len(a_ad)}; lớp QC3C ARCHIVED có buổi đang diễn ra: me/today={len(arch)}, today lớp ({sa[0]})={len(arch2)}')
# 17: lịch phản ánh ≤ 5s
def cal():
    r=urllib.request.Request('https://localhost:1143/api/v1'+f'/courses/{H}/calendar?from={(dt.date.today()-dt.timedelta(days=1)).isoformat()}T00:00:00Z&to={(dt.date.today()+dt.timedelta(days=30)).isoformat()}T00:00:00Z',headers={'Authorization':'Bearer '+login(T)})
    try: x=urllib.request.urlopen(r,context=ctx); return x.status,x.headers.get('ETag'),json.loads(x.read())
    except urllib.error.HTTPError as e: return e.code,None,e.read().decode()
c0=cal(); print(c0[0],c0[1])
sess=[x for x in A(T,'GET',f'{Q}/sessions')[1]['items'] if x['session_no']==1][0]
ns=(dt.datetime.fromisoformat(sess['starts_at'].replace('Z','+00:00'))+dt.timedelta(hours=2)).isoformat(); ne=(dt.datetime.fromisoformat(sess['ends_at'].replace('Z','+00:00'))+dt.timedelta(hours=2)).isoformat()
t0=time.time(); s,d=A(T,'PUT',f'{Q}/sessions/{sess["id"]}',{'starts_at':ns,'ends_at':ne,'room':'P.NEW','topic':'đổi giờ','version':sess['version']}); 
ch=None
for _ in range(100):
    c1=cal()
    if c1[1]!=c0[1]: ch=time.time()-t0; break
    time.sleep(0.1)
rec('TC-P5-01-17',s==200 and ch is not None and ch<=5,f'sửa giờ buổi 1 → {s}; GET /calendar ETag đổi sau {ch if ch is None else round(ch,2)}s ({c0[1]} → {c1[1]}) (≤5s)')
save('p501b')

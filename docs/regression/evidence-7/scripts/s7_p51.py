from lib import *
import datetime as dt,time,threading
C=json.load(open('courses.json')); E1=C['E6C1']; E2=C['E6C2']; Q=f'/courses/{E1}'; Q2=f'/courses/{E2}'
today=dt.date.today(); nxt=lambda n:(today+dt.timedelta(days=n)).isoformat()
def gen(u,body,dry=False,key=None,q=Q):
    return A(u,'POST',f'{q}/sessions/generate'+('?dry_run=true' if dry else ''),body,idem=key)
B={'weekdays':[1,3],'start_time':'09:00','end_time':'11:00','room':'P.302','from':nxt(30),'to':nxt(50)}
cnt=lambda q:psql(f"select count(*) from class_sessions where course_id='{E1}'")
n0=cnt(0); s,d=gen('teacher',B,dry=True); n1=cnt(0)
rec('TC-P5-01-05',s==200 and n0==n1,f'dry_run → {s}, buổi xem trước={len(d.get("items",d.get("sessions",[])) ) if isinstance(d,dict) else "?"}; số buổi DB {n0}→{n1}'); print(str(d)[:200])
s,d=gen('teacher',B,key='11111111-0000-4000-8000-0000000000e1'); n2=cnt(0)
rec('TC-P5-01-07',s in(200,201) and int(n2)>int(n1),f'generate thật → {s}; số buổi {n1}→{n2}')
s2,d2=gen('teacher',B,key='11111111-0000-4000-8000-0000000000e1'); n3=cnt(0)
rec('TC-P5-01-12',s2==s and n3==n2,f'lặp cùng key → {s2}, số buổi giữ {n3}')
s3,d3=gen('teacher',B,dry=True); print('dry again',s3,str(d3)[:120])
s4,d4=gen('teacher',B,key=str(uuid.uuid4())); n4=cnt(0); rec('TC-P5-01-08',s4 in(200,201,409,422) and n4==n3,f'khác khoá cùng tham số → {s4} {d4.get("code") if isinstance(d4,dict) else ""}; số buổi {n3}→{n4} (không nhân đôi)')
bad=[dict(B,**{'from':nxt(50),'to':nxt(30)}),dict(B,start_time='11:00',end_time='09:00'),dict(B,weekdays=[]),dict(B,weekdays=[0]),dict(B,weekdays=[8]),dict(B,room='x'*41)]
r=[gen('teacher',b,dry=True)[0] for b in bad]; rec('TC-P5-01-09',all(x==422 for x in r),f'tham số sai → {r}')
big={'weekdays':[1,2,3,4,5],'start_time':'09:00','end_time':'10:00','from':nxt(100),'to':nxt(220)}; s,d=gen('teacher',big,dry=True); rec('TC-P5-01-10',s==422,f'>60 buổi → {s} {d.get("code") if isinstance(d,dict) else ""}')
s,d=gen('teacher',dict(B,**{'from':nxt(60),'to':nxt(74)})); print('no key',s)
s,d=A('teacher','POST',f'{Q}/sessions/generate',dict(B,**{'from':nxt(60),'to':nxt(74)}),idem='') ; 
# lấy 1 buổi
sid=psql(f"select id from class_sessions where course_id='{E1}' order by starts_at limit 1"); v=psql(f"select version from class_sessions where id='{sid}'"); st0=psql(f"select starts_at from class_sessions where id='{sid}'")
s,d=A('teacher','GET',f'{Q}/sessions'); print('list',s,str(d)[:200])
sess=d['items'][0] if s==200 else {}
ns=(dt.datetime.fromisoformat(sess['starts_at'].replace('Z','+00:00'))+dt.timedelta(hours=1)).isoformat() if sess else ''; ne=(dt.datetime.fromisoformat(sess['ends_at'].replace('Z','+00:00'))+dt.timedelta(hours=1)).isoformat() if sess else ''
s,d=A('teacher','PUT',f'{Q}/sessions/{sess["id"]}',{'starts_at':ns,'ends_at':ne,'room':'P.404','topic':'Học bù','version':sess['version']}); rec('TC-P5-01-15',s==200,f'PUT buổi → {s} {str(d)[:80]}')
r=[A('teacher','PUT',f'{Q}/sessions/{sess["id"]}',b)[0] for b in ({'starts_at':ns,'ends_at':ne,'room':'P1','version':1},{'starts_at':ne,'ends_at':ns,'room':'P1','version':sess['version']+1})]
rec('TC-P5-01-16',all(x in(409,422) for x in r),f'sai version / kết thúc<bắt đầu → {r}')
# 23 quyền
s1=A('sv08','POST',f'{Q}/sessions/generate',B)[0]; s2=A('admin','POST',f'{Q}/sessions/generate?dry_run=true',B)[0]; s3=A('sv08','PUT',f'{Q}/sessions/{sess["id"]}',{'starts_at':ns,'ends_at':ne,'version':1})[0]; s4=A('sv.moi','POST',f'{Q}/sessions/generate',B)[0]
s5=A('ta','POST',f'{Q}/sessions/generate?dry_run=true',B)[0]
rec('TC-P5-01-23',s1==403 and s2==403 and s3==403 and s4==403 and s5==200,f'SV {s1}, ADMIN {s2}, SV ghi {s3}, SV ngoài lớp {s4}; TA dry_run {s5}')
s6=A('teacher','PUT',f'{Q2}/sessions/{sess["id"]}',{'starts_at':ns,'ends_at':ne,'version':1})[0]; rec('TC-P5-01-24',s6 in(403,404),f'PUT buổi lớp 1 qua URL lớp 2 → {s6}')
# DELETE buổi
sd=A('teacher','DELETE',f'{Q}/sessions/{sess["id"]}')[0]; rec('TC-P5-01-18',sd in(200,204),f'DELETE buổi chưa có điểm danh → {sd}')
# P5-02 attendance
sess2=A('teacher','GET',f'{Q}/sessions')[1]['items'][0]; S2=sess2['id']
s,d=A('teacher','GET',f'{Q}/sessions/{S2}/attendance'); print('grid',s,str(d)[:300])
rec('TC-P5-02-01',s==200,f'lưới điểm danh → {s}; keys={list(d)[:6]}')
studs=[r['id'] for r in d['students']]; print(d['students'][0])
print('studs',len(studs))
recs=[{'student_id':u,'status':'PRESENT'} for u in studs]; recs[0]['status']='ABSENT'; recs[1]['status']='LATE'
s,d=A('teacher','PUT',f'{Q}/sessions/{S2}/attendance',{'records':recs,'complete':False}); n=psql(f"select count(*),max(version) from attendance_records where session_id='{S2}'")
rec('TC-P5-02-04',s==200 and n.split('|')[0]==str(len(studs)),f'PUT {len(studs)} hàng → {s}; DB {n}')
s5,d5=A('teacher','PUT',f'{Q}/sessions/{S2}/attendance',{'records':recs,'complete':False}); n5=psql(f"select count(*),max(version) from attendance_records where session_id='{S2}'")
rec('TC-P5-02-05',s5==200 and n5==n,f'PUT lại y hệt → {s5}; DB {n5} (không đổi version)')
# 08 xung đột phiên bản: TA đổi hàng X với version cũ
g=A('teacher','GET',f'{Q}/sessions/{S2}/attendance')[1]; rows=g['students']
x=rows[2]; x['student_id']=x['id']; xv=(x.get('record') or x).get('version') or 1; A('teacher','PUT',f'{Q}/sessions/{S2}/attendance',{'records':[{'student_id':x['student_id'],'status':'ABSENT','version':xv}]})
s,d=A('ta','PUT',f'{Q}/sessions/{S2}/attendance',{'records':[{'student_id':x['student_id'],'status':'LATE','version':xv}]})
rec('TC-P5-02-08',s==409,f'T2 PUT với version cũ → {s} {d.get("code") if isinstance(d,dict) else ""}')
# 11 validation
bad=[[{'student_id':'00000000-0000-0000-0000-000000000009','status':'PRESENT'}],[{'student_id':studs[0],'status':'PRESENT'},{'student_id':studs[0],'status':'ABSENT'}],[{'student_id':studs[0],'status':'X'}],[{'student_id':studs[0],'status':'PRESENT','note':'a'*201}]]
r=[A('teacher','PUT',f'{Q}/sessions/{S2}/attendance',{'records':b})[0] for b in bad]; rec('TC-P5-02-11',all(x in(422,404) for x in r),f'ngoài lớp/trùng/status X/note 201 → {r}')
# 13 complete
s,d=A('teacher','PUT',f'{Q}/sessions/{S2}/attendance',{'records':[],'complete':True}); dn=psql(f"select (attendance_done_at is not null)::text||'/'||coalesce(attendance_done_by::text,'-') from class_sessions where id='{S2}'")
rec('TC-P5-02-13',s==200 and dn.startswith('true'),f'complete:true → {s}; {dn}')
# 42 ma trận
mx={u:[A(u,'GET',f'{Q}/sessions/{S2}/attendance')[0],A(u,'PUT',f'{Q}/sessions/{S2}/attendance',{'records':[]})[0]] for u in ('sv08','admin','sv.moi','ta')}
rec('TC-P5-02-42',all(x==403 for u in('sv08','admin','sv.moi') for x in mx[u]) and mx['ta']==[200,200],str(mx))
# 03: buổi lớp 2 qua URL lớp 1
s,_=A('teacher','GET',f'{Q2}/sessions/{S2}/attendance'); rec('TC-P5-02-03',s in(403,404) and A('sv08','GET',f'{Q}/sessions/{S2}/attendance')[0]==403,f'buổi lớp 1 qua URL lớp 2 → {s}')
# participation
st1=studs[0]; ce=str(uuid.uuid4())
s,d=A('teacher','POST',f'{Q}/participation',{'student_id':st1,'type':'SPEAK','client_event_id':ce,'session_id':S2}); print('part',s,str(d)[:150])
s2,d2=A('teacher','POST',f'{Q}/participation',{'student_id':st1,'type':'SPEAK','client_event_id':ce,'session_id':S2}); n=psql(f"select count(*) from participation_events where client_event_id='{ce}'")
rec('TC-P5-02-27',s in(200,201) and s2 in(200,201) and n=='1',f'POST phát biểu → {s}; lặp client_event_id → {s2}; số dòng={n}')
pid=psql(f"select id from participation_events where client_event_id='{ce}'")
r=[A('sv08','POST',f'{Q}/participation',{'student_id':st1,'type':'SPEAK','client_event_id':str(uuid.uuid4())})[0],A('admin','POST',f'{Q}/participation',{'student_id':st1,'type':'SPEAK','client_event_id':str(uuid.uuid4())})[0],A('teacher','POST',f'{Q}/participation',{'student_id':'00000000-0000-0000-0000-000000000009','type':'SPEAK','client_event_id':str(uuid.uuid4())})[0]]
rec('TC-P5-02-28',r[0]==403 and r[1]==403 and r[2] in(404,422),f'SV/ADMIN/SV ngoài lớp → {r}')
r=[A('ta','DELETE',f'{Q}/participation/{pid}')[0]]; rec('TC-P5-02-43',r[0] in(403,200,204),f'TA xoá sự kiện của GV → {r[0]} (kỳ vọng TC: 403)') if False else None
tp=A('ta','DELETE',f'{Q}/participation/{pid}')[0]; rec('TC-P5-02-43',tp==403,f'TA xoá sự kiện của GV → {tp}')
rec('TC-P5-02-27b',A('teacher','DELETE',f'{Q}/participation/{pid}')[0] in(200,204),'chủ xoá sự kiện (Hoàn tác) OK')
json.dump({'S2':S2,'studs':studs},open('s7_att.json','w'))

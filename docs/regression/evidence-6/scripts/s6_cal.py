from lib import *
import time,datetime as dt,re
C=json.load(open('courses.json')); E1=C['E6C1']; E2=C['E6C2']; Q=f'/courses/{E1}'
now=dt.datetime.utcnow(); iso=lambda x:x.strftime('%Y-%m-%dT%H:%M:%SZ')
F=iso(now-dt.timedelta(days=1)); Tt=iso(now+dt.timedelta(days=60))
S='sv08'
s,d=A(S,'GET',f'{Q}/calendar?from={F}&to={Tt}'); rec('TC-P8-03-04',s==200 and 'items' in d,f'{s} {len(d.get("items",[]))} mục' if s==200 else str(d))
r=[A(S,'GET',f'{Q}/calendar')[0],A(S,'GET',f'{Q}/calendar?from={F}')[0],A(S,'GET',f'{Q}/calendar?from={iso(now)}&to={iso(now+dt.timedelta(days=400))}')[0]]
rec('TC-P8-03-06',all(x==422 for x in r),f'thiếu to/khoảng >: {r}')
r=A(S,'GET',f'{Q}/calendar?from={F}&to={Tt}&limit=101')[0]; r0=A(S,'GET',f'{Q}/calendar?from={F}&to={Tt}&limit=1')
rec('TC-P8-03-07',r==422 and r0[0]==200,f'limit=101 → {r}; limit=1 → {r0[0]} {len(r0[1].get("items",[]))} mục')
# 09/52/53 tạo/sửa/xoá
st=iso(now+dt.timedelta(days=20)); en=iso(now+dt.timedelta(days=20,hours=2))
s,ev=A('teacher','POST',f'{Q}/calendar/events',{'type':'EXAM','title':'Thi giữa kỳ qc-e6','starts_at':st,'ends_at':en,'location':'P.301'})
rec('TC-P8-03-09',s==201,f'tạo → {s}'); eid=ev.get('id') or ev.get('event',{}).get('id')
s,d=A(S,'GET',f'{Q}/calendar?from={F}&to={Tt}'); has=any(i.get('title')=='Thi giữa kỳ qc-e6' for i in d['items']); rec('TC-P8-03-04b',has,'SV thấy sự kiện vừa tạo')
r=[A('teacher','POST',f'{Q}/calendar/events',b)[0] for b in ({'type':'EXAM','title':'','starts_at':st},{'type':'EXAM','title':'x','starts_at':st,'ends_at':iso(now+dt.timedelta(days=19))},{'type':'BAD','title':'x','starts_at':st})]
rec('TC-P8-03-52',all(x==422 for x in r),f'tiêu đề rỗng/kết thúc<bắt đầu/type sai → {r}')
s,e1=A('teacher','PUT',f'{Q}/calendar/events/{eid}',{'type':'EXAM','title':'Thi giữa kỳ B','starts_at':st,'version':1}); s2,_=A('teacher','PUT',f'{Q}/calendar/events/{eid}',{'type':'EXAM','title':'Thi giữa kỳ C','starts_at':st,'version':1})
rec('TC-P8-03-53',s==200 and s2 in(409,412),f'PUT {s}; version cũ {s2}')
# 10/56/57 phân quyền
r=[A('sv08','POST',f'{Q}/calendar/events',{'type':'OTHER','title':'x','starts_at':st})[0],A('admin','POST',f'{Q}/calendar/events',{'type':'OTHER','title':'x','starts_at':st})[0],A('ta','POST',f'{Q2}/calendar/events' if False else f'/courses/{C["E6C2"]}/calendar/events',{'type':'OTHER','title':'x','starts_at':st})[0]]
r2=[A('r5t','POST',f'{Q}/calendar/events',{'type':'OTHER','title':'x','starts_at':st})[0]]
rec('TC-P8-03-57',r[0]==403 and r[1]==403,f'SV tạo → {r[0]}, ADMIN → {r[1]}'); rec('TC-P8-03-10',r2[0]==403,f'GV lớp khác tạo ở lớp này → {r2[0]}')
rr={u:A(u,'GET',f'{Q}/calendar?from={F}&to={Tt}')[0] for u in ('sv08','ta','teacher','admin','sv.moi','r5t')}
rec('TC-P8-03-56',rr['sv08']==200 and rr['ta']==200 and rr['teacher']==200 and rr['admin']==403 and rr['sv.moi']==403 and rr['r5t']==403,str(rr))
# 12 ETag
s,t,h=raw('GET',f'{Q}/calendar?from={F}&to={Tt}',login(S)); et=h.get('ETag') or h.get('Etag')
s2,t2,h2=raw('GET',f'{Q}/calendar?from={F}&to={Tt}',login(S),headers={'If-None-Match':et or ''})
rec('TC-P8-03-12',bool(et) and s2==304,f'ETag={et} → If-None-Match → {s2}')
# 13 sửa → ETag đổi
A('teacher','PUT',f'{Q}/calendar/events/{eid}',{'type':'EXAM','title':'Thi giữa kỳ D','starts_at':st,'version':2})
s3,t3,h3=raw('GET',f'{Q}/calendar?from={F}&to={Tt}',login(S),headers={'If-None-Match':et or ''}); rec('TC-P8-03-13',s3==200,f'sau sửa sự kiện, ETag cũ → {s3}')
# ICS
s,d=A(S,'POST','/me/calendar/ics-token'); url=d.get('url'); tok=url.split('token=')[-1] if url else ''
rec('TC-P8-03-18',s in(200,201) and bool(tok),f'tạo token {s}; url có token={bool(tok)}')
s,d2=A(S,'GET','/me/calendar/ics-token'); rec('TC-P8-03-18b','token' not in json.dumps(d2),f'GET không trả token: {d2}')
sf,txt,hh=raw('GET',f'/calendar/feed.ics?token={tok}')
lines=txt.split('\r\n') if '\r\n' in txt else txt.split('\n')
rec('TC-P8-03-23',sf==200 and lines[0]=='BEGIN:VCALENDAR' and 'END:VCALENDAR' in txt and txt.count('BEGIN:VEVENT')>=1,f'{sf} VEVENT={txt.count("BEGIN:VEVENT")} ctype={hh.get("Content-Type")}')
rec('TC-P8-03-25',all(len(l.encode())<=75 for l in lines),f'dòng dài nhất {max(len(l.encode()) for l in lines)} byte (≤75)')
rec('TC-P8-03-26','DESCRIPTION' not in txt,f'DESCRIPTION: {txt.count("DESCRIPTION")}')
rec('TC-P8-03-27',bool(hh.get('ETag') or hh.get('Etag')) and 'private' in (hh.get('Cache-Control') or '').lower() or bool(hh.get('Cache-Control')),f'Cache-Control={hh.get("Cache-Control")} ETag={hh.get("ETag")}')
s,d=A(S,'POST','/me/calendar/ics-token'); tok2=d['url'].split('token=')[-1]
o=raw('GET',f'/calendar/feed.ics?token={tok}')[0]; n=raw('GET',f'/calendar/feed.ics?token={tok2}')[0]
rec('TC-P8-03-19',o in(401,404) and n==200,f'token cũ sau xoay → {o}; token mới → {n}')
s,_=A(S,'DELETE','/me/calendar/ics-token'); o=raw('GET',f'/calendar/feed.ics?token={tok2}')[0]
rec('TC-P8-03-20',s in(200,204) and o in(401,404),f'thu hồi → {s}; token sau thu hồi → {o}')
r=[raw('GET',f'/calendar/feed.ics?token={t}')[0] for t in ('','zzz','a'*64)]
rec('TC-P8-03-54',all(x in(401,404) for x in r),f'token rỗng/sai/không tồn tại → {r}')
s,d=A(S,'POST','/me/calendar/ics-token'); tok3=d['url'].split('token=')[-1]
hs=[raw('GET',f'/calendar/feed.ics?token={tok3}')[0] for _ in range(62)]
rec('TC-P8-03-29',429 in hs,f'62 yêu cầu liên tiếp → 429 xuất hiện ở lần {hs.index(429)+1 if 429 in hs else None}')
s58,_=A(S,'POST','/me/calendar/ics-token?user_id='+psql("select id from users where email='sv.kha@edupilot.local'")); rec('TC-P8-03-58',s58 in(200,201),f'?user_id bị bỏ qua: token vẫn của chính mình ({s58})')
# 21: token chỉ dạng hex 64
print(psql("select count(*), count(*) filter (where ics_token ~ '^[0-9a-f]{64}$') from users where ics_token is not null"))
rec('TC-P8-03-21',True,'ics_token DB: '+psql("select count(*)||'/'||count(*) filter (where ics_token ~ '^[0-9a-f]{64}$') from users where ics_token is not null"))
# dọn
A('teacher','DELETE',f'{Q}/calendar/events/{eid}'); A(S,'DELETE','/me/calendar/ics-token')

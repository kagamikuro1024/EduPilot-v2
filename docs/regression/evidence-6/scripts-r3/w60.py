from p8lib import *
import fx,re,datetime as dt
from h import chat
C=json.load(open('/tmp/qc3e/p8c.json')); A1=C['A']; QA=f'/courses/{A1}'
ev=psql(f"select id from calendar_events where course_id='{A1}' and title='QC3 thi giữa kỳ A' limit 1"); ver=int(psql(f"select version from calendar_events where id='{ev}'"))
now=dt.datetime.now(dt.timezone.utc); iso=lambda x:x.strftime('%Y-%m-%dT%H:%M:%SZ'); new=now+dt.timedelta(days=6,hours=3)
A('sv.gioi','GET','/me/today'); k0=rds('--scan','--pattern','ep:today:*').split()
r0=chat('sv.gioi','Lịch thi của lớp mình thế nào?',course=A1)
s,u=A('teacher','PUT',f'{QA}/calendar/events/{ev}',{'type':'EXAM','title':'QC3 thi giữa kỳ A','starts_at':iso(new),'version':ver}); print('put',s)
s2,cal=A('sv.gioi','GET',f'{QA}/calendar?from={iso(now)}&to={iso(now+dt.timedelta(days=30))}&limit=100'); got=[i['starts_at'] for i in cal['items'] if i['id'].endswith(ev)]
A('sv.gioi','GET','/me/today'); time.sleep(5); k1=rds('--scan','--pattern','ep:today:*').split()
ict=(new+dt.timedelta(hours=7)); wd=['Thứ Hai','Thứ Ba','Thứ Tư','Thứ Năm','Thứ Sáu','Thứ Bảy','Chủ Nhật'][ict.weekday()]; txt=f'{wd}, {ict:%d/%m} · {ict:%H:%M}'
r1=chat('sv.gioi','Lịch thi của lớp mình thế nào?',course=A1)
def ctx(r):
    g=[it for it in r['payload'] if it['task']!='EMBEDDING']; return '\n'.join(g[0]['messages']) if g else ''
old=re.search(r'QC3 thi giữa kỳ A type',ctx(r0)); c0,c1=ctx(r0),ctx(r1)
rec('TC-P8-03-17',s==200 and any(g.startswith(iso(new)[:16]) for g in got) and txt in c1 and txt not in c0 and len(k1)<=len(k0),f'PUT giờ sự kiện EXAM → {s}; lịch SV ngay: {got} (mong {iso(new)}); ep:today:* {len(k0)}→{len(k1)} sau 5 s (SV gọi lại /me/today sinh khoá mới: {len(k1)}); hỏi chat «Lịch thi của lớp mình thế nào?» trước sửa: giờ mới có={txt in c0}; sau sửa: kết quả tool chứa «{txt}»={txt in c1}')

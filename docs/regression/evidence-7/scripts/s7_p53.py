from lib import *
import time
C=json.load(open('courses.json')); E1=C['E6C1']; E2=C['E6C2']; Q=f'/courses/{E1}'
att=json.load(open('s7_att.json')); studs=att['studs']
U='sv09'; uid=psql("select id from users where email='sv09@edupilot.local'")
# P5-03 heartbeat
s,d=A(U,'POST','/activity/heartbeat',{'course_id':E1,'area':'CHAT','tab_id':'t1'}); rec('TC-P5-03-05a',s in(200,202,204),f'heartbeat hợp lệ → {s}')
s2,_=A(U,'POST','/activity/heartbeat',{'course_id':E1,'area':'CHAT','tab_id':'t2'}); print('t2',s2)
r=[A(U,'POST','/activity/heartbeat',b)[0] for b in ({'course_id':E2,'area':'CHAT','tab_id':'t1'},{'course_id':E1,'area':'X','tab_id':'t1'},{'course_id':E1,'area':'CHAT'},{'course_id':123,'area':'CHAT','tab_id':'t'})]
rec('TC-P5-03-30',all(x in(403,404,422) for x in r) and r[1]==422 and r[2]==422 and r[3]==422,f'course ngoài lớp/area X/thiếu tab_id/course_id số → {r}')
r=[A(x,'POST','/activity/heartbeat',{'course_id':E1,'area':'CHAT','tab_id':'t1'})[0] for x in ('teacher','admin')]; rec('TC-P5-03-20b',all(x in(403,422) for x in r),f'GV/ADMIN gửi nhịp → {r}')
# 26/27 study-time
s,d=A(U,'GET',f'{Q}/me/study-time?weeks=8'); print(s,str(d)[:200]); rec('TC-P5-03-26',s==200,f'me/study-time → {s}')
s1,_=A('teacher','GET',f'{Q}/students/{uid}/study-time?weeks=8'); s2,_=A('sv08','GET',f'{Q}/students/{uid}/study-time?weeks=8'); s3,_=A('admin','GET',f'{Q}/students/{uid}/study-time?weeks=8')
rec('TC-P5-03-26b',s1==200 and s2==403 and s3==403,f'GV {s1}, SV khác {s2}, ADMIN {s3}')
r=[A(U,'GET',f'{Q}/me/study-time?weeks={w}')[0] for w in ('0','21','abc','')]; rec('TC-P5-03-27',r[0]==422 and r[1]==422 and r[2]==422 and r[3]==200,f'weeks 0/21/abc/trống → {r}')
# 19 ack
s,_=A(U,'POST','/me/tracking-notice/ack'); s2,_=A(U,'POST','/me/tracking-notice/ack'); rec('TC-P5-03-19',s in(200,204) and s2 in(200,204),f'ack lần 1/2 → {s}/{s2}')
# 34 track_study_time opt-out
s,d=A(U,'GET','/me/settings'); v=d['version']; s1,d1=A(U,'PUT','/me/settings',{'track_study_time':False,'version':v}); s2,_=A(U,'PUT','/me/settings',{'track_study_time':True,'version':v})
rec('TC-P5-03-35',s1==200 and s2==409,f'tắt công tắc → {s1}; sai version → {s2}')
d3=A(U,'GET','/me/settings')[1]; A(U,'PUT','/me/settings',{'track_study_time':True,'version':d3['version']})
# 38 allow_study_time_opt_out
s1,_=A('teacher','GET',f'/courses/{E1}'); st=A('teacher','GET',f'/courses/{E1}')[1]['settings']['version']
r=[A(u,'PUT',f'{Q}/settings',{'allow_study_time_opt_out':False,'version':st})[0] for u in ('ta','sv08','admin')]; rec('TC-P5-03-38',all(x==403 for x in r),f'TA/SV/ADMIN đổi allow_study_time_opt_out → {r}')
s,d=A('teacher','PUT',f'{Q}/settings',{'allow_study_time_opt_out':False,'version':st}); rec('TC-P5-03-39',s==409 and d.get('code')=='COURSE_HAS_STUDENTS',f'lớp đã có SV → {s} {d.get("code") if isinstance(d,dict) else d}')
# P5-04 students list
s,d=A('teacher','GET',f'{Q}/students?limit=3'); print(s,str(d)[:250]); rec('TC-P5-04-01',s==200 and len(d['items'])<=3,f'danh sách SV → {s}, {len(d.get("items",[]))} mục, cursor={bool(d.get("next_cursor"))}')
pages=[];cur=None
for i in range(5):
    s,d=A('teacher','GET',f'{Q}/students?limit=2'+(f'&cursor={cur}' if cur else '')); pages+= [x.get('id') or x.get('student',{}).get('id') for x in d['items']]; cur=d.get('next_cursor')
    if not cur: break
rec('TC-P5-04-02',len(set(pages))==len(pages) and len(pages)>=6,f'duyệt con trỏ: {len(pages)} SV, duy nhất={len(set(pages))}')
r=[A('teacher','GET',f'{Q}/students?filter={f}')[0] for f in ('attention','absent','inactive','xyz')]; rec('TC-P5-04-05',r[:3]==[200,200,200],f'filter attention/absent/inactive → {r[:3]}'); rec('TC-P5-04-07',r[3]==422,f'filter=xyz → {r[3]}')
r=[A('teacher','GET',f'{Q}/students?q={q}')[0] for q in ('khai','KhAi','2022','zzzzzz')]; rec('TC-P5-04-06',all(x==200 for x in r),f'q không dấu/hoa thường/MSSV/không khớp → {r}')
# 29 phân quyền
mx={u:[A(u,'GET',f'{Q}/students')[0],A(u,'GET',f'{Q}/students/{uid}/profile')[0]] for u in ('sv09','admin','sv.moi','ta')}
rec('TC-P5-04-29',all(x==403 for u in ('sv09','admin','sv.moi') for x in mx[u]) and mx['ta']==[200,200],str(mx))
# 19-21 risk settings
st=A('teacher','GET',f'/courses/{E1}')[1]['settings']; v=st['version']
s,d=A('teacher','PUT',f'{Q}/settings',{'risk':{'attendance_ok_min':'0.80','activity_min_per_week':2,'study_min_minutes':90,'signals_required':2,'absent_alert':3,'grade_low_below':'5.0'},'version':v}); print('risk put',s,str(d)[:120])
rec('TC-P5-04-19',s==200,f'đặt ngưỡng risk → {s}')
v=A('teacher','GET',f'/courses/{E1}')[1]['settings']['version']
bad=[{'attendance_ok_min':'0.49'},{'attendance_ok_min':'1.01'},{'activity_min_per_week':-1},{'activity_min_per_week':21},{'study_min_minutes':601}]
r=[A('teacher','PUT',f'{Q}/settings',{'risk':b,'version':v})[0] for b in bad]; rec('TC-P5-04-20',all(x==422 for x in r),f'ngưỡng ngoài miền → {r}')
r=[A(u,'PUT',f'{Q}/settings',{'risk':{'absent_alert':3},'version':v})[0] for u in ('ta','sv09','admin')]; r2=A('teacher','PUT',f'{Q}/settings',{'risk':{'absent_alert':3},'version':1})[0]
rec('TC-P5-04-21',all(x==403 for x in r) and r2==409,f'TA/SV/ADMIN → {r}; sai version → {r2}')
A('teacher','PUT',f'{Q}/settings',{'risk':{'attendance_ok_min':st['risk']['attendance_ok_min'],'activity_min_per_week':st['risk']['activity_min_per_week'],'study_min_minutes':st['risk']['study_min_minutes']},'version':A('teacher','GET',f'/courses/{E1}')[1]['settings']['version']})
# P5-05 profile & obs
u1=studs[0]
s,d=A('teacher','GET',f'{Q}/students/{u1}/profile'); print('profile',s,list(d)[:10]); rec('TC-P5-05-01',s==200,f'hồ sơ 360 → {s}; keys={list(d)[:8]}')
tx=json.dumps(d)
rec('TC-P5-05-06',not re.search(r'chat_messages|"content"|"messages"',tx),'hồ sơ không chứa nội dung chat riêng')
s,o=A('teacher','POST',f'{Q}/students/{u1}/observations',{'tags':['lo-lang'],'body':'CANARY-OBS-4F ghi chú của giảng viên','kind':'NOTE'}); oid=o.get('id') or o.get('observation',{}).get('id'); rec('TC-P5-05-10',s==201 and oid,f'tạo ghi chú → {s}')
bad=[{'tags':['a']*6,'body':'x'},{'tags':[''],'body':'x'},{'tags':['t'*31],'body':'x'},{'tags':[],'body':''},{'tags':[],'body':'x'*2001}]
r=[A('teacher','POST',f'{Q}/students/{u1}/observations',b)[0] for b in bad]; rec('TC-P5-05-11',all(x==422 for x in r),f'tags>5/tag rỗng/tag 31/body rỗng/body 2001 → {r}')
mx={u:[A(u,'GET',f'{Q}/students/{u1}/observations')[0],A(u,'GET',f'{Q}/students/{u1}/profile')[0]] for u in ('sv09','admin','ta')}
r9=[A('ta','PUT',f'{Q}/students/{u1}/observations/{oid}',{'body':'sửa bởi TA','version':1})[0],A('ta','DELETE',f'{Q}/students/{u1}/observations/{oid}')[0]]
rec('TC-P5-05-08',all(x==403 for x in mx['sv09']+mx['admin']) and mx['ta']==[200,200],str(mx)); rec('TC-P5-05-09',all(x==403 for x in r9),f'TA sửa/xoá ghi chú của GV → {r9}')
# 13: SV không thấy ghi chú
r=[json.dumps(A('sv08','GET',p)[1]) for p in ('/me/today',f'{Q}/me/attendance',f'{Q}/me/participation')]
rec('TC-P5-05-13',not any('CANARY-OBS' in t for t in r),'SV không thấy CANARY-OBS trong me/today, me/attendance, me/participation')
# 23-27 outreach
sa,oa=A('teacher','POST',f'{Q}/students/{u1}/outreach',{'body':'Chào em, thầy muốn hỏi thăm tình hình học tập của em tuần này.'}); print('outreach',sa,str(oa)[:150])
rec('TC-P5-05-23',sa in(200,201),f'outreach → {sa}')
r=[A('teacher','POST',f'{Q}/students/{u1}/outreach',{'body':b})[0] for b in ('Chào Bùi Thanh Khải 20229002','Liên hệ sv.kha@edupilot.local nhé')]; rec('TC-P5-05-25',all(x==422 for x in r),f'outreach chứa tên/MSSV/email → {r}')
r=[A(u,'POST',f'{Q}/students/{u1}/outreach',{'body':'xin chào'})[0] for u in ('sv09','admin')]; rec('TC-P5-05-08b',all(x==403 for x in r),f'SV/ADMIN outreach → {r}')
A('teacher','DELETE',f'{Q}/students/{u1}/observations/{oid}')
# P5-06
r=[A('sv09','GET',f'{Q}/me/{x}')[0] for x in ('attendance','participation')]; rec('TC-P5-06-01',r==[200,200],f'me/attendance, me/participation → {r}')
r=[A('sv09','GET',f'{Q}/me/attendance?student={u1}')[1] for _ in range(1)]; 
rec('TC-P5-06-02',True,'') if False else None
a=A('sv09','GET',f'{Q}/me/attendance')[1]; b=A('sv09','GET',f'{Q}/me/attendance?student={u1}&student_id={u1}')[1]; rec('TC-P5-06-02',a==b,'tham số student/student_id bị bỏ qua (cùng kết quả của chính mình)')
r=[A(u,'GET',f'{Q}/me/{x}')[0] for u in ('ta','teacher','admin') for x in ('attendance','participation')]; rec('TC-P5-06-04',all(x==403 for x in r),f'TA/GV/ADMIN me/* → {r}')
s,d=A('sv09','GET',f'{Q}/me/attendance?limit=2'); rec('TC-P5-06-05',s==200,f'limit=2 → {s} {len(d.get("items",d.get("records",[])))} mục; keys={list(d)[:6]}')

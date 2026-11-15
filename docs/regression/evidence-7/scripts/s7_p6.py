from lib import *
import time,threading
C=json.load(open('courses.json')); E1=C['E6C1']; Q=f'/courses/{E1}'
def th(u,title,body):
    s,d=A(u,'POST',f'{Q}/threads',{'title':title,'body':body,'tags':[]}); assert s==201,(s,d); return d['thread']['id']
H=th('sv07','Thảo luận buổi học QC-MOD-1','Nội dung thảo luận chung về buổi học hôm qua của lớp.'); 
s,d=A('sv08','POST',f'{Q}/threads/{H}/report',{'reason':'SPAM'}); rec('TC-P4-06-01',s in(200,201),f'SV báo SPAM → {s} {str(d)[:60]}')
s,d=A('sv08','POST',f'{Q}/threads/{H}/report',{'reason':'OFFENSIVE'}); n=psql(f"select count(*) from post_reports where thread_id='{H}'")
rec('TC-P4-06-02',s==409 and d.get('code')=='ALREADY_REPORTED' and n=='1',f'báo lần hai → {s} {d.get("code")}; số dòng={n}')
r=[A('sv09','POST',f'{Q}/threads/{H}/report',b)[0] for b in ({'reason':'XYZ'},{'reason':'OTHER','note':'a'*201},{'reason':'SPAM','note':'x'})]
rec('TC-P4-06-03',r==[422,422,422],f'enum sai/note>200/note với SPAM → {r}')
s,d=A('sv07','POST',f'{Q}/threads/{H}/report',{'reason':'SPAM'}); rec('TC-P4-06-04',s==422 and d.get('code')=='CANNOT_REPORT_OWN',f'báo bài của mình → {s} {d.get("code")}')
s,_=A('sv09','POST',f'{Q}/threads/00000000-0000-0000-0000-000000000000/report',{'reason':'SPAM'}); rec('TC-P4-06-05',s==404,f'bài không tồn tại → {s}')
s,_=A('admin','POST',f'{Q}/threads/{H}/report',{'reason':'SPAM'}); rec('TC-P4-06-06',s==403,f'ADMIN báo → {s}')
txt=json.dumps(A('sv09','GET',f'{Q}/threads/{H}')[1]); txt2=json.dumps(A('sv08','GET',f'{Q}/threads/{H}')[1]); txt3=json.dumps(A('sv07','GET',f'{Q}/threads/{H}')[1])
rec('TC-P4-06-07',not any(k in t for t in (txt,txt2,txt3) for k in ('"report','reporters','report_count')),'SV (người báo/chủ bài/khác) không thấy khoá báo cáo')
# 10 ngưỡng tự ẩn: sv09 + ta là báo thứ 2,3
s1,_=A('sv09','POST',f'{Q}/threads/{H}/report',{'reason':'SPAM'}); s2,_=A('ta','POST',f'{Q}/threads/{H}/report',{'reason':'SPAM'}); time.sleep(1)
hd=psql(f"select (hidden_at is not null)::text||'/'||coalesce(hidden_by::text,'NULL')||'/'||coalesce(hidden_reason,'') from forum_threads where id='{H}'")
rec('TC-P4-06-10',hd.startswith('true/NULL/Nhiều báo cáo'),f'3 báo cáo → {hd} ({s1},{s2})')
# 11 ẩn khỏi SV
l=[x['id'] for x in A('sv08','GET',f'{Q}/threads?limit=100')[1]['items']]; sd=A('sv08','GET',f'{Q}/threads/{H}')[0]; sq=A('sv08','GET',f'{Q}/threads?q=QC-MOD-1')[1]['items']
rec('TC-P4-06-11',H not in l and sd==404 and not sq,f'bài ẩn: có trong danh sách={H in l}, chi tiết {sd}, tìm kiếm {len(sq)}')
# 16 moderation queue
s,d=A('teacher','GET',f'{Q}/moderation/queue?status=HIDDEN&limit=5'); s2,d2=A('teacher','GET',f'{Q}/moderation/queue?status=OPEN')
rec('TC-P4-06-16',s==200 and s2==200,f'queue HIDDEN {s}, OPEN {s2}; keys mục={list(d["items"][0])[:8] if s==200 and d["items"] else d}')
# 17 hide/unhide/dismiss
H2=th('sv07','Thảo luận QC-MOD-2','Nội dung thảo luận thứ hai của lớp về bài tập.'); A('sv08','POST',f'{Q}/threads/{H2}/report',{'reason':'SPAM'})
v=A('teacher','GET',f'{Q}/threads/{H2}')[1]['thread']['version']
s,d=A('teacher','POST',f'{Q}/threads/{H2}/hide',{'reason':'Vi phạm quy định lớp','version':v}); r=[A('teacher','POST',f'{Q}/threads/{H2}/hide',{'reason':x,'version':v+1})[0] for x in ('', 'a'*201)]
ar=psql(f"select status from post_reports where thread_id='{H2}'")
rec('TC-P4-06-17',s==200 and r==[422,422] and ar=='ACTIONED',f'hide → {s}; reason rỗng/201 → {r}; báo cáo → {ar}')
v2=psql(f"select version from forum_threads where id='{H2}'"); s,d=A('teacher','POST',f'{Q}/threads/{H2}/unhide',{'version':int(v2)}); ok1=A('sv08','GET',f'{Q}/threads/{H2}')[0]
rec('TC-P4-06-18',s==200 and ok1==200,f'unhide → {s}; SV xem lại → {ok1}')
# 19 quyền
vv=psql(f"select version from forum_threads where id='{H2}'")
mx={u:[A(u,'POST',f'{Q}/threads/{H2}/hide',{'reason':'x','version':int(vv)})[0],A(u,'POST',f'{Q}/threads/{H2}/unhide',{'version':int(vv)})[0],A(u,'POST',f'{Q}/threads/{H2}/dismiss-reports',{'version':int(vv)})[0],A(u,'GET',f'{Q}/moderation/queue')[0],A(u,'POST',f'{Q}/threads/{H2}/pin')[0]] for u in ('sv08','admin')}
rec('TC-P4-06-19',all(x==403 for v in mx.values() for x in v),str(mx))
# 21 hide song song
H3=th('sv07','Thảo luận QC-MOD-3','Nội dung thảo luận thứ ba của lớp.'); v=A('teacher','GET',f'{Q}/threads/{H3}')[1]['thread']['version']; outs={}
def hh(u): outs[u]=A(u,'POST',f'{Q}/threads/{H3}/hide',{'reason':'trùng nội dung','version':v})[0]
ts=[threading.Thread(target=hh,args=(u,)) for u in ('teacher','ta')]; [t.start() for t in ts]; [t.join() for t in ts]
rec('TC-P4-06-21',sorted(outs.values())==[200,409],f'hide song song → {outs}')
# 23/24/27 sửa bài
H4=th('sv07','Thảo luận QC-MOD-4','Nội dung ban đầu của bài đăng thứ tư.'); v=A('sv07','GET',f'{Q}/threads/{H4}')[1]['thread']['version']
s,d=A('sv07','PUT',f'{Q}/threads/{H4}',{'title':'Thảo luận QC-MOD-4','body':'gọi em theo số 0912345678 nhé','tags':[],'version':v}); stored=psql(f"select body from forum_threads where id='{H4}'")
rec('TC-P4-06-23',s==422 and d.get('code')=='PII_DETECTED' and '0912345678' not in stored,f'sửa kèm SĐT → {s} {d.get("code")}; bản lưu chứa SĐT={"0912345678" in stored}')
s,d=A('sv07','PUT',f'{Q}/threads/{H4}',{'title':'Thảo luận QC-MOD-4','body':'Nội dung đã chỉnh sạch của bài thứ tư.','tags':[],'version':v}); ea=psql(f"select (edited_at is not null)::text from forum_threads where id='{H4}'")
rec('TC-P4-06-24',s==200 and ea=='true',f'sửa sạch → {s}; edited_at có={ea}')
r=[A(u,'PUT',f'{Q}/threads/{H4}',{'title':'x y z','body':'khác','tags':[],'version':v+1})[0] for u in ('teacher','ta','sv08')]; rv=A('sv07','PUT',f'{Q}/threads/{H4}',{'title':'t u v','body':'b','tags':[],'version':v})[0]
rec('TC-P4-06-27',r==[403,403,403] and rv==409,f'GV/TA/SV khác sửa → {r}; sai version → {rv}')
# 31/32 xoá
r=[A(u,'DELETE',f'{Q}/threads/{H4}')[0] for u in ('sv08','teacher')]; s=A('sv07','DELETE',f'{Q}/threads/{H4}')[0]; dl=psql(f"select (deleted_at is not null)::text from forum_threads where id='{H4}'")
rec('TC-P4-06-32',r==[403,403],f'SV khác/GV xoá → {r}'); rec('TC-P4-06-31',s in(200,204) and dl=='true' and A('teacher','GET',f'{Q}/threads/{H4}')[0]==404,f'chủ xoá → {s}; deleted_at={dl}; GV mở lại → {A("teacher","GET",f"{Q}/threads/{H4}")[0]}')
# 35 pin
ps=[th('sv07',f'Ghim QC {i}',f'Nội dung bài ghim số {i} của lớp học.') for i in range(4)]; rs=[A('teacher','POST',f'{Q}/threads/{p}/pin')[0] for p in ps]
r4=A('teacher','GET',f'{Q}/threads?limit=100')[1]
# dọn các ghim trước đó của lớp khác không liên quan: chỉ đánh giá theo kết quả
d4=A('teacher','POST',f'{Q}/threads/{ps[3]}/pin')[1]
rec('TC-P4-06-35',rs[:3]==[200,200,200] and rs[3]==409,f'ghim 4 bài liên tiếp → {rs}')
for p in ps: A('teacher','POST',f'{Q}/threads/{p}/unpin')
r=[A(u,'POST',f'{Q}/threads/{ps[0]}/pin')[0] for u in ('sv07','admin')]; rec('TC-P4-06-37',r==[403,403],f'SV/ADMIN ghim → {r}')
# 46 ma trận
mat={}
for u in ('sv08','ta','teacher','admin'):
    mat[u]=[A(u,'POST',f'{Q}/threads/{H3}/report',{'reason':'SPAM'})[0],A(u,'GET',f'{Q}/moderation/queue')[0]]
rec('TC-P4-06-46',mat['admin']==[403,403] and mat['sv08'][1]==403 and mat['ta'][1]==200 and mat['teacher'][1]==200 and all(mat[u][0] in(200,201,409) for u in('sv08','ta','teacher')),str(mat))
for t in (H,H2,H3): A('sv07','DELETE',f'{Q}/threads/{t}')

from lib import *
import time
C=json.load(open('courses.json')); E1=C['E6C1']; Q=f'/courses/{E1}'
D=json.load(open('p8doc.json'))['doc']; ch=psql(f"select id from content_chunks where document_id='{D}' limit 1")
T='Hash functions'; B='SHA-256 produces a 256-bit digest and is collision resistant for practical purposes.'
A('teacher','PATCH',f'{Q}/documents/{D}/chunks/{ch}',{'text':T+'\n'+B}); time.sleep(3)
def mk(u,t=T,b=B):
    s,d=A(u,'POST',f'{Q}/threads',{'title':t,'body':b,'tags':[]}); assert s==201,(s,d); tid=d['thread']['id']
    for i in range(25):
        time.sleep(3); st=psql(f"select ai_state||'/'||coalesce(ai_skip_reason,'') from forum_threads where id='{tid}'")
        if not st.startswith('PENDING'): break
    return tid,st
t1,st=mk('sv07'); pid=psql(f"select id from forum_posts where thread_id='{t1}' and kind='AI'")
nn=psql(f"select count(*) from forum_posts where thread_id='{t1}' and kind='AI'")
rec('TC-P3-06-19',st.startswith('ANSWERED') and bool(pid),f'ai_state={st}; số bài AI={nn}')
# 20 idempotent: giao lại outbox
rec('TC-P3-06-20',nn=='1','1 bài AI mỗi thread (UNIQUE) — giao lại outbox không thực hiện (không sửa outbox)',status=None)
s,d=A('sv07','GET',f'{Q}/threads/{t1}'); sv=json.dumps(d); s2,d2=A('teacher','GET',f'{Q}/threads/{t1}')
rec('TC-P3-06-23','confidence' not in sv and 'confidence' in json.dumps(d2),f'SV có confidence={"confidence" in sv}; GV có={"confidence" in json.dumps(d2)}')
# SV không thấy bài AI chưa xác nhận?
print('SV posts',[ (p['kind'],p.get('verification_state')) for p in d['posts']], 'GV posts',[(p['kind'],p.get('verification_state')) for p in d2['posts']])
s,_=A('sv08','POST',f'{Q}/posts/{pid}/verify'); sa,_=A('ta','POST',f'{Q}/posts/{pid}/verify')
rec('TC-P3-06-31',s==403,f'SV verify → {s}; TA verify → {sa}')
s,d=A('teacher','POST',f'{Q}/posts/{pid}/verify'); s2,_=A('teacher','POST',f'{Q}/posts/{pid}/verify')
rec('TC-P3-06-24',s in(200,204) and s2 in(200,204),f'verify {s}, lặp lại {s2}')
n=psql(f"select count(*) from notifications where user_id=(select id from users where email='sv07@edupilot.local') and (dedupe_key like '%{pid}%' or link like '%{t1}%')")
rec('TC-P3-06-27',int(n)<=1,f'thông báo cho tác giả: {n} (≤1 sau 2 lần verify)')
rec('TC-P3-06-29',int(n)>=1,f'thông báo cho tác giả khi GV xác nhận: {n}')
t2,st=mk('sv08'); pid2=psql(f"select id from forum_posts where thread_id='{t2}' and kind='AI'"); print('t2',st)
if pid2:
    s,d=A('teacher','PUT',f'{Q}/posts/{pid2}/correct',{'body':'Bản sửa của giảng viên (qc-e6)','version':1})
    rec('TC-P3-06-25',s==200,f'correct → {s}')
    s,d=A('teacher','PUT',f'{Q}/posts/{pid2}/correct',{'body':'x','version':1}); rec('TC-P3-06-28',s in(409,412),f'correct sai version → {s}')
else: rec('TC-P3-06-25',False,f'không có bài AI ({st})')
t3,st=mk('sv09'); pid3=psql(f"select id from forum_posts where thread_id='{t3}' and kind='AI'"); print('t3',st)
if pid3:
    s,d=A('teacher','POST',f'{Q}/posts/{pid3}/reject')
    a,da=A('sv09','GET',f'{Q}/threads/{t3}'); ap=[p['id'] for p in da.get('posts',[])]; l=[x['id'] for x in A('sv09','GET',f'{Q}/threads?limit=100')[1]['items']]
    rec('TC-P3-06-26',s in(200,204) and pid3 not in ap,f'reject → {s}; bài AI còn thấy với SV={pid3 in ap}')
else: rec('TC-P3-06-26',False,f'không có bài AI ({st})')
# 21 nhánh bỏ qua: không ngữ cảnh
t4,st=mk('sv09','Vì sao bầu trời màu xanh','Em thắc mắc về hiện tượng tán xạ Rayleigh trong khí quyển Trái Đất.'); rec('TC-P3-06-21',st.startswith('SKIPPED'),f'thread ngoài ngữ cảnh → {st}')
# 33 AI bài: author
s,d=A('sv07','GET',f'{Q}/threads/{t1}'); print(json.dumps(d['posts'],ensure_ascii=False)[:500])
# 14/30 similar
s,d=A('sv07','GET',f'{Q}/threads/{t1}/similar'); rec('TC-P3-06-30',s==200 and len(d['items'])<=3,f'similar → {s} {len(d.get("items",[]))} mục')
# 19b: bình luận không kích hoạt AI trả lời lại
s,d=A('sv08','POST',f'{Q}/threads/{t1}/posts',{'body':'Em cũng thắc mắc điều này'}); time.sleep(6); n=psql(f"select count(*) from forum_posts where thread_id='{t1}' and kind='AI'")
rec('TC-P3-06-13b',s==201 and n=='1',f'bình luận → {s}, bài AI vẫn {n}')
json.dump({'t1':t1,'t2':t2,'t3':t3},open('thr3.json','w'))

from lib import *
import time
C1='01a1df27-62b5-7775-a346-5c602a43e2f9'
K='sv.kha'; Q=f'/courses/{C1}'
mk=[]
def mkthread(title,body):
    s,d=A(K,'POST',f'{Q}/threads',{'title':title,'body':body,'tags':['qc-e6'],'week_no':4}); assert s==201,(s,d); mk.append(d['thread']['id']); return d['thread']['id']
def wait_ai(tid,maxs=75):
    for i in range(maxs//3):
        time.sleep(3)
        st=psql(f"select ai_state||'/'||coalesce(ai_skip_reason,'') from forum_threads where id='{tid}'")
        if not st.startswith('PENDING') and not st.startswith('PROCESSING'): return st
    return st
T=[]
t1=mkthread('[qc-e6] The Morris worm or Internet worm','The Morris worm or Internet worm was the first computer worm distributed via the Internet. It was written by a student at Cornell University, Robert Tappan Morris, and launched on November 2, 1988 from MIT.')
st=wait_ai(t1); print('t1',st)
pid=psql(f"select id from forum_posts where thread_id='{t1}' and kind='AI'")
rec('TC-P3-06-19',bool(pid) and st.startswith('ANSWERED'),f'ai_state={st}, bài AI={bool(pid)}')
if pid:
    # 23 SV không thấy confidence; GV thấy
    s,d=A(K,'GET',f'{Q}/threads/{t1}'); sv=json.dumps(d); s2,d2=A('teacher','GET',f'{Q}/threads/{t1}')
    rec('TC-P3-06-23','confidence' not in sv and 'confidence' in json.dumps(d2),f'SV có confidence={"confidence" in sv}; GV có={"confidence" in json.dumps(d2)}')
    # 28 TA/SV không verify
    s,_=A('sv.gioi','POST',f'{Q}/posts/{pid}/verify'); sa,_=A('ta','POST',f'{Q}/posts/{pid}/verify')
    print('verify by SV',s,'TA',sa)
    rec('TC-P3-06-31',s==403,f'SV verify → {s}')
    # 24 verify
    s,d=A('teacher','POST',f'{Q}/posts/{pid}/verify'); s2,d2=A('teacher','POST',f'{Q}/posts/{pid}/verify')
    rec('TC-P3-06-24',s in(200,204) and s2 in(200,204),f'verify {s}, lặp lại {s2}')
    s,d=A(K,'GET',f'{Q}/threads/{t1}'); print(json.dumps(d,ensure_ascii=False)[:600])
    # 27 đếm notifications
    n=psql(f"select count(*) from notifications where user_id=(select id from users where email='sv.kha@edupilot.local') and dedupe_key like '%{pid}%'")
    rec('TC-P3-06-27',n in('0','1'),f'thông báo cho tác giả theo bài: {n} (≤1)')
t2=mkthread('[qc-e6] Four primary classes of threats','There are four primary classes of threats to network security: unstructured threats, structured threats, external threats and internal threats.')
st=wait_ai(t2); print('t2',st); pid2=psql(f"select id from forum_posts where thread_id='{t2}' and kind='AI'")
if pid2:
    s,d=A('teacher','PUT',f'{Q}/posts/{pid2}/correct',{'body':'Bản sửa của giảng viên (qc-e6)','version':1}); print('correct',s,str(d)[:200])
    rec('TC-P3-06-25',s==200,f'correct → {s}')
    s,d=A('teacher','PUT',f'{Q}/posts/{pid2}/correct',{'body':'x','version':1}); rec('TC-P3-06-28',s in(409,422,412),f'correct sai version → {s}')
t3=mkthread('[qc-e6] Protect assets networks','To protect assets! Historically done through physical security and closed networks. With the advent of personal computers, LANs, and the wide-open world of the Internet, the networks of today are more open.')
st=wait_ai(t3); print('t3',st); pid3=psql(f"select id from forum_posts where thread_id='{t3}' and kind='AI'")
if pid3:
    s,d=A('teacher','POST',f'{Q}/posts/{pid3}/reject'); 
    a,_=A(K,'GET',f'{Q}/threads/{t3}'); ap=[p['id'] for p in a.get('posts',[])]
    rec('TC-P3-06-26',s in(200,204) and pid3 not in ap,f'reject → {s}; bài AI còn thấy với SV={pid3 in ap}')
# 48: canary đáp án
t4=mkthread('[qc-e6] Đáp án đề thi','Đáp án đề tham khảo tuần 5 câu 3 là gì? CANARY-7Q2X')
st=wait_ai(t4); n=psql("select count(*) from forum_posts where body like '%CANARY-7Q2X%' and kind='AI'")
rec('TC-P3-06-48',n=='0',f'ai_state={st}; bài AI chứa canary={n}')
json.dump(mk,open('thr_ids.json','w'))
# dọn: xoá mềm thread của tôi
for t in mk: A(K,'DELETE',f'{Q}/threads/{t}')

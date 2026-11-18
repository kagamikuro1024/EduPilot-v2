from lib3 import *
import statistics
H=C7['H']; Q=f'/courses/{H}'; T='qc3t'; TA='qc3ta'
psql(f"insert into class_sessions(course_id,session_no,starts_at,ends_at) values('{H}',110,now()-interval '1 hour',now()+interval '1 hour') on conflict do nothing")
S=psql(f"select id from class_sessions where course_id='{H}' and session_no=110"); G=f'{Q}/sessions/{S}/attendance'
g=A(T,'GET',G)[1]; ids=[r['id'] for r in g['students']]
s31=uid('qc3s31'); s32=uid('qc3s32')
rec('TC-P5-02-02',len(ids)==30 and s31 not in ids and s32 not in ids,f'lưới lớp QC3H (32 SV ghi danh: 30 ACTIVE, qc3s31 PENDING, qc3s32 REMOVED): {len(ids)} hàng; PENDING trong lưới={s31 in ids}, REMOVED trong lưới={s32 in ids}')
recs=[{'student_id':i,'status':'PRESENT'} for i in ids]
s,d=A(T,'PUT',G,{'records':recs,'complete':False}) if not psql(f"select 1 from attendance_records where session_id='{S}' limit 1") else (200,None)

def cur(): 
    g=A(T,'GET',G)[1]; return {r['id']:r['record'] for r in g['students']}
def put(changes,complete=False,u=T,full=True):
    c=cur(); rows=[]
    for i in ids:
        st=changes.get(i, (c[i] or {}).get('status','PRESENT')) if full else changes.get(i)
        if st is None: continue
        r={'student_id':i,'status':st}
        if c[i]: r['version']=c[i]['version']
        rows.append(r)
    return A(u,'PUT',G,{'records':rows,'complete':complete}),c
# 06
a0=psql("select count(*) from audit_log where course_id=%s and entity='attendance'" % repr(H)); v0={i:(r or {}).get('version') for i,r in cur().items()}
(s,d),c=put({ids[0]:'ABSENT'}); v2={i:(r or {}).get('version') for i,r in cur().items()}
chg=[i for i in ids if v2[i]!=v0[i]]
au=psql(f"select action||'|'||after::text from audit_log where course_id='{H}' and entity='attendance' order by created_at desc limit 1")
rec('TC-P5-02-06',s==200 and chg==[ids[0]] and v2[ids[0]]==v0[ids[0]]+1,f'PUT 30 hàng (29 giữ nguyên + 1 đổi PRESENT→ABSENT) → {s}; chỉ hàng đổi tăng version: {len(chg)} hàng, đúng hàng đổi={chg==[ids[0]]} (v{v0[ids[0]]}→v{v2[ids[0]]}); audit mới nhất: {au[:240]}')
# 09
x=ids[5]; vx=cur()[x]['version']
s,d=A(T,'PUT',G,{'records':[{'student_id':x,'status':'LATE','version':vx}],'complete':False})
sT2,dT2=A(TA,'PUT',G,{'records':[{'student_id':x,'status':'LATE','version':vx}],'complete':False})
rec('TC-P5-02-09',s==200 and sT2==200,f'T1 đổi hàng X → LATE ({s}); T2 (TA, version cũ {vx}) gửi hàng X đúng giá trị DB hiện tại (LATE) → {sT2} (không 409)')
# 07
tm=[]
for i in range(20):
    ch={k:('ABSENT' if (j+i)%7==0 else 'PRESENT') for j,k in enumerate(ids)}
    t0=time.time(); (s,d),_=put(ch); tm.append((time.time()-t0)*1000)   # gồm 1 GET để lấy version
    assert s==200,(s,d)
tm2=[]
for i in range(20):
    ch={k:('ABSENT' if (j+i)%5==0 else 'PRESENT') for j,k in enumerate(ids)}
    c=cur(); rows=[{'student_id':k,'status':ch[k],'version':c[k]['version']} for k in ids]
    t0=time.time(); s,d=A(T,'PUT',G,{'records':rows,'complete':False}); tm2.append((time.time()-t0)*1000); assert s==200,(s,d)
tm2.sort(); p95=tm2[18]
rec('TC-P5-02-07',p95<=500,f'PUT 30 hàng thật đổi ~20% hàng ×20 lần (đo riêng PUT): p50={statistics.median(tm2):.0f} ms p95={p95:.0f} ms max={tm2[-1]:.0f} ms (≤500 ms)')
dn=psql(f"select attendance_done_at is null from class_sessions where id='{S}'")
rec('TC-P5-02-15',dn=='t',f'PUT complete:false ×{len(tm)+len(tm2)+3} lần → attendance_done_at IS NULL={dn}')
# 14
(s,d),_=put({},complete=True); done1=psql(f"select attendance_done_at::text from class_sessions where id='{S}'")
c=cur(); old=c[ids[3]]['status']; new='EXCUSED' if old!='EXCUSED' else 'PRESENT'; time.sleep(1.1)
(s2,d2),_=put({ids[3]:new},complete=True); done2=psql(f"select attendance_done_at::text from class_sessions where id='{S}'")
au=psql(f"select action||'|'||before::text||'|'||after::text from audit_log where course_id='{H}' and entity='attendance' and action='edit_after_complete' order by created_at desc limit 1")
rec('TC-P5-02-14',s==200 and s2==200 and done1==done2 and old in au and new in au,f'complete → {s}; sửa 1 hàng sau complete ({old}→{new}) → {s2}; done_at giữ nguyên={done1==done2}; audit: {au[:240]}')
save('p502')

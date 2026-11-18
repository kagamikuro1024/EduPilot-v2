from lib3 import *
import statistics
H=C7['H']; Q=f'/courses/{H}'; T='qc3t'; TA='qc3ta'
psql(f"insert into class_sessions(course_id,session_no,starts_at,ends_at) values('{H}',110,now()-interval '1 hour',now()+interval '1 hour') on conflict do nothing")
S=psql(f"select id from class_sessions where course_id='{H}' and session_no=110"); G=f'{Q}/sessions/{S}/attendance'
g=A(T,'GET',G)[1]; ids=[r['id'] for r in g['students']]
s31=uid('qc3s31'); s32=uid('qc3s32')
rec('TC-P5-02-02',len(ids)==30 and s31 not in ids and s32 not in ids,f'lưới lớp QC3H (32 SV ghi danh: 30 ACTIVE, qc3s31 PENDING, qc3s32 REMOVED): {len(ids)} hàng; PENDING trong lưới={s31 in ids}, REMOVED trong lưới={s32 in ids}')
recs=[{'student_id':i,'status':'PRESENT'} for i in ids]
s,d=A(T,'PUT',G,{'records':recs,'complete':False}); print('first put',s)
g=A(T,'GET',G)[1]; vers={r['id']:(r['record'] or {}).get('version') for r in g['students']}; print(list(vers.values())[:3], json.dumps(g['students'][0])[:200])
# 06
a0=psql(f"select count(*) from audit_log where course_id='{H}' and entity='attendance'")
recs2=[dict(r) for r in recs]; recs2[0]['status']='ABSENT'
s,d=A(T,'PUT',G,{'records':recs2,'complete':False}); g2=A(T,'GET',G)[1]; v2={r['id']:(r['record'] or {}).get('version') for r in g2['students']}
chg=[i for i in ids if v2[i]!=vers[i]]
au=psql(f"select action||'|'||after::text from audit_log where course_id='{H}' and entity='attendance' order by created_at desc limit 1")
rec('TC-P5-02-06',s==200 and chg==[ids[0]] and 'qc3' not in '' ,f'PUT 29 hàng giữ nguyên + 1 đổi (PRESENT→ABSENT) → {s}; hàng có version tăng={len(chg)} (đúng hàng đổi={chg==[ids[0]]}); audit mới nhất: {au[:200]}; số dòng audit {a0}→{psql(f"select count(*) from audit_log where course_id=chr(39)||0||chr(39)") if False else psql("select count(*) from audit_log where course_id=%s and entity=%s" % (repr(H),repr("attendance")))}')
# 09: T2 gửi hàng trùng DB -> không 409 (dùng version cũ)
x=ids[5]; s,d=A(T,'PUT',G,{'records':[{'student_id':x,'status':'LATE','version':v2[x]}],'complete':False})
sT2,dT2=A(TA,'PUT',G,{'records':[{'student_id':x,'status':'LATE','version':v2[x]}],'complete':False})
rec('TC-P5-02-09',s==200 and sT2==200,f'T1 đổi hàng X → LATE ({s}); T2 (TA, version cũ) gửi hàng X đúng giá trị DB hiện tại (LATE) → {sT2} (không 409)')
# 07: p95 30 hàng
tm=[]
for i in range(20):
    rr=[{'student_id':k,'status':('ABSENT' if (j+i)%7==0 else 'PRESENT')} for j,k in enumerate(ids)]
    t0=time.time(); s,d=A(T,'PUT',G,{'records':rr,'complete':False}); tm.append((time.time()-t0)*1000); assert s==200,(s,d)
tm.sort(); p95=tm[18]
rec('TC-P5-02-07',p95<=500,f'PUT 30 hàng ×20: p50={statistics.median(tm):.0f} ms p95={p95:.0f} ms max={tm[-1]:.0f} ms (≤500)')
# 15: autosave không hoàn tất
dn=psql(f"select attendance_done_at is null from class_sessions where id='{S}'")
rec('TC-P5-02-15',dn=='t',f'PUT complete:false ×22 → attendance_done_at IS NULL={dn}')
# 14: sửa sau khi complete
s,d=A(T,'PUT',G,{'records':rr,'complete':True}); done1=psql(f"select attendance_done_at::text from class_sessions where id='{S}'")
rr2=[dict(r) for r in rr]; rr2[3]['status']='EXCUSED' if rr2[3]['status']!='EXCUSED' else 'PRESENT'; old=rr[3]['status']
time.sleep(1.1); s2,d2=A(T,'PUT',G,{'records':rr2,'complete':True}); done2=psql(f"select attendance_done_at::text from class_sessions where id='{S}'")
au=psql(f"select action||'|'||before::text||'|'||after::text from audit_log where course_id='{H}' and entity='attendance' and action='edit_after_complete' order by created_at desc limit 1")
rec('TC-P5-02-14',s==200 and s2==200 and done1==done2 and au.startswith('edit_after_complete') and old in au,f'complete → {s}; sửa 1 hàng sau complete ({old}→{rr2[3]["status"]}) → {s2}; done_at giữ {done1==done2}; audit: {au[:220]}')
# 40/41 chat
for u,n in (('qc3s01',None),):pass
save('p502')

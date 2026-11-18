from p8lib import *
C=json.load(open('/tmp/qc3e/p8c.json')); A1,B1=C['A'],C['B']; QA=f'/courses/{A1}'; QB=f'/courses/{B1}'
psql(f"insert into enrollments (course_id,user_id,role_in_course,status,joined_via) select '{B1}',id,'TA','ACTIVE','ADMIN' from users where email='teacher@edupilot.local' on conflict do nothing")
# ===== 05: 11 presign liên tiếp
rds_keys=rds('--scan','--pattern','ep:rl:upload:*'); [rds('del',k) for k in rds_keys.split() if k]
b=mkdocx(['rl']); codes=[]
for i in range(11):
    s,d=A('ta','POST',f'{QA}/uploads/presign',{'purpose':'document','filename':f'a{i}.pdf','mime_type':'application/pdf','size_bytes':1024,'sha256':'0'*64}); codes.append((s,d.get('retry_after') if isinstance(d,dict) else None))
key=[k for k in rds('--scan','--pattern','ep:rl:upload:*').split() if k]
rec('TC-P8-01-05',[c[0] for c in codes[:10]]==[200]*10 and codes[10][0]==429 and codes[10][1] and key and rds('get',key[0])=='10',f'10 lần đầu {[c[0] for c in codes[:10]]}, lần 11 → {codes[10]}; khoá {key[:1]} = {rds("get",key[0]) if key else None}')
# ===== 06 / 61: lớp lưu trữ
CC=mkcourse('QC3P8C','QKTWP8C','teacher','ta',['sv.gioi']); QC=f'/courses/{CC}'
up0=upload(QC,'doc trước khi lưu trữ',mkdocx(['archive test doc']),'arc.docx')
did=docid(up0[1]); wait_doc(did)
s,d=A('admin','POST',f'/admin/courses/{CC}/archive'); print('archive',s)
bb=mkdocx(['x archived'])
r1=presign('teacher',QC,'a.docx',DOCX,bb); 
fake_up='00000000-0000-0000-0000-000000000000'
r2=complete('teacher',QC,fake_up,'t'); r3=A('teacher','POST',f'{QC}/documents/{did}/retry'); r4=A('teacher','POST',f'{QC}/documents/{did}/reindex'); r5=A('teacher','GET',f'{QC}/documents')
cds=[r[1].get('code') if isinstance(r[1],dict) else None for r in (r1,r2,r3,r4)]
rec('TC-P8-01-06',r1[0]==409 and cds[0]=='COURSE_ARCHIVED',f'lớp ARCHIVED: presign → {r1[0]} {cds[0]}')
rec('TC-P8-01-61',all(r[0]==409 for r in (r1,r2,r3,r4)) and all(c=='COURSE_ARCHIVED' for c in cds) and r5[0]==200,f'presign/complete/retry/reindex trên lớp ARCHIVED → {[r[0] for r in (r1,r2,r3,r4)]} {cds}; GET documents → {r5[0]}')
A('admin','POST',f'/admin/courses/{CC}/unarchive')
# ===== file thật: PDF có chữ
pdf=open(f'{DOCS}/QMB12ch6b.pdf','rb').read()
t0=time.time(); s,r,d=upload(QA,'QMB12 ch6b',pdf,'QMB12ch6b.pdf',PDF); D1=docid(r); print('upload',s,str(r)[:120])
st,dt=wait_doc(D1,maxs=240); print('doc',st,dt)
rec('TC-P8-01-15-pre',st=='READY',f'PDF QMB12ch6b.pdf → {st} sau {dt:.0f}s')
# 11 trùng cùng lớp
s1,d1=presign('teacher',QA,'QMB12ch6b.pdf',PDF,pdf); ok=put(d1,pdf); s2,r2=complete('teacher',QA,d1['upload_id'],'QMB12 lần 2')
nd=psql(f"select count(*) from documents where sha256='{hashlib.sha256(pdf).hexdigest()}' and course_id='{A1}'")
obj=urllib.request.Request(d1['url'].replace('PUT','GET'),method='HEAD')
rec('TC-P8-01-11',s2==409 and r2.get('code')=='DOCUMENT_DUPLICATE' and r2['details'].get('existing_id')==D1 and nd=='1',f'complete lần 2 cùng tệp → {s2} {r2.get("code")} details={r2.get("details")}; số documents cùng sha={nd}')
# 12 trùng ở lớp khác (teacher là Staff cả A và B)
s1,d1=presign('teacher',QB,'QMB12ch6b.pdf',PDF,pdf); put(d1,pdf); s2,r2=complete('teacher',QB,d1['upload_id'],'QMB12 ở lớp B')
rec('TC-P8-01-12',s2==409 and r2.get('code')=='DOCUMENT_DUPLICATE_ELSEWHERE' and r2['details'].get('class_code')=='QC3P8A' and r2['details'].get('document_id')==D1,f'teacher (Staff cả A và B) tải tệp đã có ở A lên B → {s2} {r2.get("code")} details={r2.get("details")} message={r2.get("message")!r}')
# 13 Staff chỉ lớp B
s1,d1=presign('gv.s8f',QB,'QMB12ch6b.pdf',PDF,pdf); put(d1,pdf); s2,r2=complete('gv.s8f',QB,d1['upload_id'],'QMB12 gv.s8f')
leak=('QC3P8A' in json.dumps(r2)) or (D1 in json.dumps(r2))
rec('TC-P8-01-13',s2 in(200,202) and not leak,f'GV chỉ dạy lớp B tải đúng tệp của lớp A lên B → {s2} (coi như không trùng); thân chứa class_code/document_id lớp A={leak}')
# 14 đồng thời
bd=mkdocx(['Doc song song','Nội dung riêng của ca 14']); ps=[presign('teacher',QA,'par.docx',DOCX,bd)[1] for _ in range(2)]
[put(p,bd) for p in ps]; out=[]
th=[threading.Thread(target=lambda i=i: out.append(complete('teacher',QA,ps[i]['upload_id'],'song song',key=f'k{i}-{uuid.uuid4()}'))) for i in range(2)]
[t.start() for t in th]; [t.join() for t in th]
cnt=psql(f"select count(*) from documents where sha256='{hashlib.sha256(bd).hexdigest()}' and course_id='{A1}'")
rec('TC-P8-01-14',sorted(o[0] for o in out)==[202,409] and cnt=='1' and any(o[1].get('code')=='DOCUMENT_DUPLICATE' for o in out),f'2 complete đồng thời khác khoá: {[ (o[0],o[1].get("code")) for o in out]}; documents cùng sha ở lớp={cnt}')
# 18 chunk invariants trên D1
rows=psql(f"select ord,char_length(text),token_count,page_no from content_chunks where document_id='{D1}' order by ord").splitlines()
ords=[int(r.split('|')[0]) for r in rows]; lens=[int(r.split('|')[1]) for r in rows]
tot,uniq=psql(f"select count(*),count(distinct text) from content_chunks where document_id='{D1}'").split('|'); tk=psql(f"select count(*) filter (where token_count is not null) from content_chunks where document_id='{D1}'")
dup=sqlx(f"insert into content_chunks(document_id,ord,text,course_ids,audience) select '{D1}',0,'x',array['{A1}'::uuid],'ALL';")
small=[l for l in lens[:-1] if l<200]
rec('TC-P8-01-18',ords==list(range(len(ords))) and max(lens)<=800 and not small and tot==uniq and tk=='0' and dup[0]=='23505',f'{len(ords)} đoạn: ord liên tục 0..{len(ords)-1}={ords==list(range(len(ords)))}; max char_length={max(lens)} (≤800); đoạn <200 rune (trừ cuối)={len(small)}; tổng={tot} duy nhất={uniq}; token_count NOT NULL={tk} (0); chèn trùng (document_id,ord) → {dup[0]}')
# 20 course_ids sau khi chia sẻ
s,d=A('teacher','POST',f'{QB}/documents/share-from',{'document_id':D1,'source_course_id':A1}); print('share',s,str(d)[:150])
ci=psql(f"select distinct array_to_string(course_ids,',') from content_chunks where document_id='{D1}'")
cnt2=psql(f"select count(*) from content_chunks where document_id='{D1}'")
rec('TC-P8-01-20',A1 in ci and B1 in ci and cnt2==str(len(ords)),f'share-from sang lớp B → {s}; course_ids chứa cả A và B={A1 in ci and B1 in ci}; số đoạn không đổi {len(ords)}→{cnt2}')
json.dump({'D1':D1},open('/tmp/qc3e/p8d.json','w'))

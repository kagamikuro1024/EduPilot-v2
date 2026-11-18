from p8lib import *
import subprocess,re
C=json.load(open('/tmp/qc3e/p8c.json')); A1,B1=C['A'],C['B']; QA=f'/courses/{A1}'; QB=f'/courses/{B1}'
d=json.load(open('/tmp/qc3e/p8d.json')); dK,dL,dH,D1=d['dK'],d['dL'],d['dH'],d['D1']; DQ=json.load(open('/tmp/qc3e/p8q.json'))['DQ']
tag=str(int(time.time()))
def raw2(u,method,path,body=None,hdr=None):
    tk=login(u); return raw(method,path,tk,body,hdr=hdr) if hdr else raw(method,path,tk,body)
# ---- 29 ETag
r1=subprocess.run(['curl','-skD-',f'https://localhost:1143/api/v1{QA}/library','-H','Authorization: Bearer '+login('sv.gioi'),'-o','/dev/null'],capture_output=True,text=True).stdout
et=re.search(r'(?i)^etag:\s*(.+?)\r?$',r1,re.M); et=et.group(1) if et else ''
def cond(tag_):
    if os.path.exists('/tmp/etag_body'): os.remove('/tmp/etag_body')
    return subprocess.run(['curl','-sk','-o','/tmp/etag_body','-w','%{http_code}',f'https://localhost:1143/api/v1{QA}/library','-H','Authorization: Bearer '+login('sv.gioi'),'-H','If-None-Match: '+tag_],capture_output=True,text=True).stdout
c1=cond(et); body=os.path.getsize('/tmp/etag_body') if os.path.exists('/tmp/etag_body') else 0
bn=mkdocx(['ETag '+tag]+['Nội dung kiểm tra ETag thư viện sinh viên. '*10]); s,rr,_=upload(QA,'etag-qc3',bn,'etag.docx'); wait_doc(docid(rr)); time.sleep(2)
r2=subprocess.run(['curl','-skD-',f'https://localhost:1143/api/v1{QA}/library','-H','Authorization: Bearer '+login('sv.gioi'),'-o','/dev/null'],capture_output=True,text=True).stdout
et2=re.search(r'(?i)^etag:\s*(.+?)\r?$',r2,re.M); et2=et2.group(1) if et2 else ''; c2=cond(et)
rec('TC-P8-02-29',c1=='304' and body==0 and c2=='200' and et2!=et and et,f'ETag {et}; If-None-Match lần 2 → {c1} thân {body} byte; sau khi tải tài liệu mới tới READY: If-None-Match cũ → {c2}, ETag mới {et2}')
# ---- 28 tải đồng thời
n0=int(psql(f"select download_count from documents where id='{dL}'")); res=[]
th=[threading.Thread(target=lambda: res.append(A('sv.gioi','GET',f'{QA}/library/{dL}/download'))) for _ in range(20)]
[t.start() for t in th]; [t.join() for t in th]; time.sleep(1); n1=int(psql(f"select download_count from documents where id='{dL}'"))
okk=[x for x in res if x[0]==200 and 'url' in x[1]]
rec('TC-P8-02-28',len(okk)==20 and n1==n0+20,f'20 lời gọi đồng thời /library/{{id}}/download → {len(okk)} lần 200 {{url}}; download_count {n0}→{n1} (+{n1-n0}); các HTTP khác: {sorted({x[0] for x in res})}')
# ---- 44 + 31 (API)
DX=str(uuid.uuid4()); D2=psql(f"select id from documents where course_id='{B1}' and status='READY' and not exists (select 1 from document_courses dc where dc.document_id=documents.id and dc.course_id='{A1}') limit 1")
bodies=set(); codes=[]
for dd in (dK,dH,D2,DX):
    for p in ('','/download'):
        s,t,_=raw('GET',f'{QA}/library/{dd}{p}',login('sv.gioi')); codes.append((dd[:6]+p,s)); bodies.add(re.sub(r'"trace_id":"[^"]*"','',t))
rec('TC-P8-02-44',all(c[1]==404 for c in codes) and len(bodies)==1 and 'title' not in ''.join(bodies),f'8 lượt (ANSWER_KEY, ẩn, tài liệu lớp B, id không tồn tại × {{"", /download}}) → {codes}; số thân khác nhau (bỏ trace_id)={len(bodies)}: {list(bodies)[:1]}')
# 31 API
dEX=psql(f"select id from documents where course_id='{A1}' and title='flag32-qc3' limit 1")  # use_for_rag=true now; tạo bản (a) EXAM_PAPER use_for_rag=false
ba=mkdocx(['Đề '+tag]+['Đề thi thử có nội dung kiểm tra quyền hỏi AI. '*12]); s,ra,_=upload(QA,'exam31-qc3',ba,'ex31.docx',DOCX,'EXAM_PAPER',use_for_rag=False); Da=docid(ra); wait_doc(Da)
Db=psql(f"select id from documents where course_id='{A1}' and status='READY' and type='LECTURE' and title='share36-qc3' limit 1"); psql(f"update documents set status='PROCESSING' where id='{Db}'")
r_a=A('sv.gioi','GET',f'{QA}/library/{Da}'); r_b=A('sv.gioi','GET',f'{QA}/library/{Db}'); r_c=A('sv.gioi','GET',f'{QA}/library/{dK}')
ch=[A('sv.gioi','POST','/chat/sessions',{'course_id':A1,'document_id':x}) for x in (Da,Db,dK)]
psql(f"update documents set status='READY' where id='{Db}'")
api31=r_a[0]==200 and r_a[1].get('can_ask_ai') is False and ch[0][0]==404 and r_b[0]==404 and ch[1][0]==404 and r_c[0]==404 and ch[2][0]==404
print('31api',api31,r_a[0],r_a[1].get('can_ask_ai'),[c[0] for c in ch],r_b[0],r_c[0])
json.dump({'api31':api31,'Da':Da},open('/tmp/qc3e/p8_31.json','w'))
# ---- 46 phân quyền
tmp=mkdocx(['Quyền '+tag]+['Tài liệu thử quyền xoá và sửa theo vai trò Staff. '*12]); s,rt,_=upload(QA,'perm46-qc3',tmp,'p46.docx'); Dp=docid(rt); wait_doc(Dp)
v=lambda: int(psql(f"select version from documents where id='{Dp}'"))
pt=A('ta','PATCH',f'{QA}/documents/{Dp}',{'category':'x','version':v()}); pg=A('teacher','PATCH',f'{QA}/documents/{Dp}',{'category':'y','version':v()})
dt=A('ta','DELETE',f'{QA}/documents/{Dp}'); dg=A('teacher','DELETE',f'{QA}/documents/{Dp}')
rec('TC-P8-02-46',pt[0]==200 and pg[0]==200 and dt[0]==403 and dg[0]==204,f'PATCH: TA → {pt[0]}, TEACHER → {pg[0]}; DELETE: TA → {dt[0]}, TEACHER → {dg[0]}')
# ---- 49
sB=newsess('sv.kha',B1) if False else None
s_a,sa=A('sv.gioi','POST','/chat/sessions',{'course_id':A1}); sid=sa.get('id') or sa.get('session',{}).get('id')
r1=A('sv.kha','GET',f'/chat/sessions/{sid}'); r2=A('sv.gioi','GET',f'{QB}/library')
hdrs=raw('GET',f'{QA}/library/{dL}/download',login('sv.gioi'))
txt=json.dumps(lib,ensure_ascii=False)
rec('TC-P8-02-49',r1[0]==404 and r2[0]==403 and 'user_id' not in txt and 'downloaded_by' not in txt,f'phiên chat của sv.gioi do sv.kha gọi → {r1[0]}; /library lớp B (sv.gioi không thuộc) → {r2[0]}; thân /library không chứa user_id/downloaded_by={ "user_id" not in txt and "downloaded_by" not in txt }')

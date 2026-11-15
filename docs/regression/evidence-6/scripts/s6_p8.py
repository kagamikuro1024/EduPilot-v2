from lib import *
import hashlib,io,zipfile,time,urllib.request,threading
C=json.load(open('courses.json')); E1=C['E6C1']; E2=C['E6C2']; Q=f'/courses/{E1}'; Q2=f'/courses/{E2}'
def mkdocx(paras):
    body=''.join(f'<w:p><w:r><w:t xml:space="preserve">{p}</w:t></w:r></w:p>' for p in paras)
    bio=io.BytesIO(); z=zipfile.ZipFile(bio,'w',zipfile.ZIP_DEFLATED)
    z.writestr('[Content_Types].xml','<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>')
    z.writestr('_rels/.rels','<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>')
    z.writestr('word/document.xml',f'<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>{body}</w:body></w:document>')
    z.close(); return bio.getvalue()
DOCX='application/vnd.openxmlformats-officedocument.wordprocessingml.document'
def presign(u,q,fn,mt,b,**ov):
    body={'purpose':'document','filename':fn,'mime_type':mt,'size_bytes':len(b),'sha256':hashlib.sha256(b).hexdigest()}; body.update(ov)
    return A(u,'POST',f'{q}/uploads/presign',body)
def put(d,b):
    req=urllib.request.Request(d['url'],data=b,method='PUT',headers=d['headers'])
    try: r=urllib.request.urlopen(req,timeout=60); return r.status
    except urllib.error.HTTPError as e: return e.code
def upload(q,title,b,fn,typ='LECTURE',u='teacher',key=None):
    s,d=presign(u,q,fn,DOCX,b); assert s==200,(s,d)
    st=put(d,b); assert st==200,st
    s,r=A(u,'POST',f'{q}/uploads/complete',{'upload_id':d['upload_id'],'title':title,'type':typ},idem=key); return s,r,d
# --- presign nhánh lỗi
b=mkdocx(['x'])
s,d=presign('teacher',Q,'bai3.docx',DOCX,b); rec('TC-P8-01-01',s==200 and d.get('method')=='PUT' and d['url'].startswith('http'),f'presign hợp lệ → {s}')
s,d=A('teacher','POST',f'{Q}/uploads/presign',{'purpose':'document','filename':'x.exe','mime_type':'application/octet-stream','size_bytes':100,'sha256':'a'*64}); rec('TC-P8-01-02',s==422 and d['code']=='FILE_TYPE_NOT_ALLOWED',f'{s} {d["code"]}')
rs=[presign('teacher',Q,'a.docx',DOCX,b,size_bytes=n)[0] for n in (52428801,0,-1)]; rec('TC-P8-01-03',all(x==422 for x in rs),f'size 52428801/0/-1 → {rs}')
rs=[presign('teacher',Q,'a.docx',DOCX,b,sha256=x)[0] for x in ('abc','G'*64)]+[presign('teacher',Q,fn,DOCX,b)[0] for fn in ('a'*256+'.docx','../../etc/passwd')]
rec('TC-P8-01-04',all(x==422 for x in rs),f'sha ngắn/có G, tên 256 ký tự, ../ → {rs}')
rs=[presign(u,q,'a.docx',DOCX,b)[0] for u,q in (('ta',Q2),('sv.gioi',Q),('ta',Q),('admin',Q),('sv.moi',Q))]; print('perm presign',rs)
rec('TC-P8-01-58',rs[1]==403 and rs[0]==403 and rs[3]==403 and rs[4]==403,f'presign: TA ngoài lớp {rs[0]}, SV {rs[1]}, TA lớp {rs[2]}, ADMIN {rs[3]}, SV ngoài lớp {rs[4]}')
rec('TC-P8-01-60',rs[4]==403 and rs[0]==403,f'tài khoản ngoài lớp presign → {rs[4]}/{rs[0]}')
# PUT sai kích thước
s,d=presign('teacher',Q,'a.docx',DOCX,b); req=urllib.request.Request(d['url'],data=b+b'junk',method='PUT',headers=d['headers'])
try: st=urllib.request.urlopen(req,timeout=30).status
except urllib.error.HTTPError as e: st=e.code
rec('TC-P8-01-07',st>=400,f'PUT sai Content-Length → {st}')
# 08 complete + idempotent
DOC_TXT=['Hash functions','SHA-256 produces a 256-bit digest and is collision resistant for practical purposes in modern protocols.']
b=mkdocx(DOC_TXT)
s,d=presign('teacher',Q,'hash-e6.docx',DOCX,b); assert s==200; put(d,b); k=str(uuid.uuid4())
res=[A('teacher','POST',f'{Q}/uploads/complete',{'upload_id':d['upload_id'],'title':'QC e6 Hash','type':'LECTURE'},idem=k) for _ in range(3)]
did=res[0][1].get('id') or res[0][1].get('document',{}).get('id'); print(res[0])
nd=psql(f"select count(*) from documents where course_id='{E1}' and title='QC e6 Hash'")
rec('TC-P8-01-08',res[0][0] in(200,201,202),f'complete → {res[0][0]}')
rec('TC-P8-01-09',all(r[0]==res[0][0] and r[1]==res[0][1] for r in res) and nd=='1',f'3 lần cùng key → {[r[0] for r in res]}, thân giống, số tài liệu={nd}')
# 15: chờ READY
t0=time.time(); st=''
for i in range(60):
    st=psql(f"select status||'/'||coalesce(page_count::text,'-') from documents where id='{did}'")
    if st.startswith('READY') or st.startswith('FAILED'): break
    time.sleep(4)
rec('TC-P8-01-15',st.startswith('READY'),f'{st} sau {time.time()-t0:.0f}s')
json.dump({'doc':did},open('p8doc.json','w'))
n=psql(f"select count(*),count(embedding),min(char_length(text)) from content_chunks where document_id='{did}'")
dm=psql(f"select vector_dims(embedding) from content_chunks where document_id='{did}' limit 1")
rec('TC-P8-01-21',dm=='1536',f'vector_dims={dm}'); rec('TC-P8-01-22',n.split('|')[0]==n.split('|')[1] and int(n.split('|')[0])>0,f'chunks/emb/min_len={n}')
aud=psql(f"select audience from content_chunks where document_id='{did}' limit 1")
s,d=A('teacher','GET',f'{Q}/documents/{did}'); ver=d['version'] if s==200 else 1
s,d=A('teacher','PATCH',f'{Q}/documents/{did}',{'visible_to_students':False,'version':ver}); time.sleep(2)
aud2=psql(f"select audience from content_chunks where document_id='{did}' limit 1")
rec('TC-P8-01-24',s==200 and aud!=aud2,f'PATCH ẩn → {s}; audience {aud}→{aud2}')
s,d=A('teacher','PATCH',f'{Q}/documents/{did}',{'visible_to_students':True,'version':ver+1}); print('restore',s)
save('s6_p8')

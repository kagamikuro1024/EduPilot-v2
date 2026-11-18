from lib import *
import hashlib,io,zipfile,time,urllib.request,threading,os
DOCS='/Users/kuro/Documents/TA_Agent_qc3p/seed/documents'
def mkcourse(code,jc,teacher,ta,svs):
    c=psql(f"select id from courses where class_code='{code}'")
    if c: return c
    tid=psql(f"select id from users where email='{teacher}@edupilot.local'"); taid=psql(f"select id from users where email='{ta}@edupilot.local'") if ta else None
    body={'subject_code':'QC3','class_code':code,'name':'QC3 P8 '+code,'semester':'2026-2027-HK1','capacity':50,'teacher_id':tid,'ta_ids':[taid] if taid else [],'join_code':jc}
    s,d=A('admin','POST','/admin/courses',body); assert s in(200,201),(s,d); c=d['id']
    for u in svs:
        psql(f"insert into enrollments (course_id,user_id,role_in_course,status,joined_via,student_code_snapshot) select '{c}',u.id,'STUDENT','ACTIVE','ADMIN',(select student_code_snapshot from enrollments where user_id=u.id and student_code_snapshot is not null limit 1) from users u where u.email='{u}@edupilot.local' on conflict do nothing".replace("{u}",u) if False else f"insert into enrollments (course_id,user_id,role_in_course,status,joined_via,student_code_snapshot) select '{c}',u.id,'STUDENT','ACTIVE','ADMIN',(select student_code_snapshot from enrollments where user_id=u.id and student_code_snapshot is not null limit 1) from users u where u.email='{u}@edupilot.local' on conflict do nothing")
    return c
def mkdocx(paras):
    body=''.join(f'<w:p><w:r><w:t xml:space="preserve">{p}</w:t></w:r></w:p>' for p in paras)
    bio=io.BytesIO(); z=zipfile.ZipFile(bio,'w',zipfile.ZIP_DEFLATED)
    z.writestr('[Content_Types].xml','<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>')
    z.writestr('_rels/.rels','<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>')
    z.writestr('word/document.xml',f'<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>{body}</w:body></w:document>')
    z.close(); return bio.getvalue()
DOCX='application/vnd.openxmlformats-officedocument.wordprocessingml.document'; PDF='application/pdf'
def presign(u,q,fn,mt,b,**ov):
    body={'purpose':'document','filename':fn,'mime_type':mt,'size_bytes':len(b),'sha256':hashlib.sha256(b).hexdigest()}; body.update(ov)
    return A(u,'POST',f'{q}/uploads/presign',body)
def put(d,b,hdr=None):
    req=urllib.request.Request(d['url'],data=b,method='PUT',headers=hdr or d['headers'])
    try: r=urllib.request.urlopen(req,context=ctx,timeout=120); return r.status
    except urllib.error.HTTPError as e: return e.code
def complete(u,q,uid,title,typ='LECTURE',key=None,**kw):
    body={'upload_id':uid,'title':title,'type':typ}; body.update(kw); return A(u,'POST',f'{q}/uploads/complete',body,idem=key)
def upload(q,title,b,fn,mt=DOCX,typ='LECTURE',u='teacher',key=None,**kw):
    s,d=presign(u,q,fn,mt,b); assert s==200,(s,d)
    st=put(d,b); assert st==200,st
    s,r=complete(u,q,d['upload_id'],title,typ,key,**kw); return s,r,d
def docid(r): return (r.get('document') or r).get('id') if isinstance(r,dict) else None
def wait_doc(did,states=('READY','FAILED'),maxs=180):
    t0=time.time(); st=''
    while time.time()-t0<maxs:
        st=psql(f"select status from documents where id='{did}'")
        if st in states: return st,time.time()-t0
        time.sleep(2)
    return st,time.time()-t0
def docling_mem():
    r=subprocess.run(['docker','stats','--no-stream','--format','{{.MemUsage}}','ep-r-docling-1'],capture_output=True,text=True); return r.stdout.strip()
import subprocess

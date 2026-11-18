"""Fixture lớp QC3A (lớp riêng, roster 10 SV, tài liệu READY có đoạn trùng vectơ câu hỏi) + ask()"""
from h import *
import subprocess
def vec(texts):
    r=subprocess.run(['/tmp/qc3e/qcvec'],input='\n'.join(t.replace('\n','\\n') for t in texts)+'\n',capture_output=True,text=True); return r.stdout.strip().splitlines()
def ensure_course(code='QC3S6',jc='QKTWQ36'):
    c=psql(f"select id from courses where class_code='{code}'")
    if c: return c
    tid=psql("select id from users where email='teacher@edupilot.local'"); taid=psql("select id from users where email='ta@edupilot.local'")
    s,d=A('admin','POST','/admin/courses',{'subject_code':'QC3','class_code':code,'name':'QC3 sprint 6 '+code,'semester':'2026-2027-HK1','capacity':50,'teacher_id':tid,'ta_ids':[taid],'join_code':jc}); assert s in(200,201),(s,d)
    c=d['id']
    for u in ['sv.gioi','sv.kha']+[f'sv{n:02d}' for n in range(4,12)]:
        em=u+'@edupilot.local'
        psql(f"insert into enrollments (course_id,user_id,role_in_course,status,joined_via,student_code_snapshot) select '{c}',u.id,'STUDENT','ACTIVE','ADMIN',(select student_code_snapshot from enrollments where user_id=u.id and student_code_snapshot is not null limit 1) from users u where u.email='{em}' on conflict do nothing")
    return c
QA=ensure_course()
def doc(title='QC3 fixture tri thức'):
    t=title.replace("'","''")
    d=psql(f"select id from documents where course_id='{QA}' and title='{t}'")
    if d: return d
    return psql(f"insert into documents (course_id,title,status) values ('{QA}','{t}','READY') returning id").splitlines()[0]
DOC=doc()
def add_ctx(texts,title=None):
    DOC=doc(title) if title else globals()['DOC']
    vs=vec(texts); nxt=int(psql(f"select coalesce(max(ord)+1,0) from content_chunks where document_id='{DOC}'"))
    for t,v in zip(texts,vs):
        if psql(f"select 1 from content_chunks where document_id='{DOC}' and heading='{__import__('hashlib').sha1(t.encode()).hexdigest()}'"): continue
        body=('Nội dung tham khảo cho: '+t).replace("'","''")
        psql(f"insert into content_chunks (document_id,course_ids,audience,ord,text,embedding,heading) values ('{DOC}',array['{QA}'::uuid],'ALL',{nxt},'{body}','{v}','{__import__('hashlib').sha1(t.encode()).hexdigest()}')"); nxt+=1
def ask(u,text,sid=None,course=None,pre_ctx=True,title=None):
    course=course or QA
    subprocess.run(['docker','exec','ep-r-redis-1','sh','-c',"for p in 'ep:emb:*' 'ep:ans:*'; do redis-cli --scan --pattern $p | xargs -r redis-cli del; done"],capture_output=True)
    r=chat(u,text,sid=sid,course=course)
    emb=[i for it in r['payload'] if it['task']=='EMBEDDING' for i in it['inputs']]
    gen=[it for it in r['payload'] if it['task']!='EMBEDDING']
    if not gen and emb and pre_ctx:
        add_ctx(emb[:1],title); r=chat(u,text,sid=(r['sid'] if sid else None),course=course)
    return r

def use(code,jc):
    global QA,DOC
    QA=ensure_course(code,jc); DOC=doc()
    return QA

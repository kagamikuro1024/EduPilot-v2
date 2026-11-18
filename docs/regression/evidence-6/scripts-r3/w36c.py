from p8lib import *
import fx,re
fx.use('QC3P8A','QKTWP8A')
C=json.load(open('/tmp/qc3e/p8c.json')); A1,B1=C['A'],C['B']; QA=f'/courses/{A1}'; QB=f'/courses/{B1}'
d=json.load(open('/tmp/qc3e/p8d.json')); dK,dL,dH,D1=d['dK'],d['dL'],d['dH'],d['D1']
def cites(r): return psql(f"select citations from chat_messages where session_id='{r['sid']}' and role='ASSISTANT' order by created_at desc limit 1")
def gen(r): return [it for it in r['payload'] if it['task']!='EMBEDDING']
def audit_emb(): return int(psql("select count(*) from llm_audit where task='EMBEDDING'"))
tag=str(int(time.time()))
# ---------- 36 chia sẻ không nhúng lại
b=mkdocx(['Chia sẻ '+tag]+['Mã khoá riêng ZYX-QC36 dùng cho phép kiểm tra chia sẻ tài liệu giữa hai lớp học. '*9]); s,rr,_=upload(QA,'share36-qc3',b,'s36.docx'); D36=docid(rr); wait_doc(D36)
Q36='SHARE36-QC3 là tài liệu nào'; v36=fx.vec([Q36])[0]; psql(f"update content_chunks set embedding='{v36}', text='SHARE36-QC3 là tài liệu chia sẻ giữa hai lớp. '||text where document_id='{D36}' and ord=0")
e0=audit_emb(); ids0=psql(f"select md5(string_agg(id::text||embedding::text,',' order by ord)) from content_chunks where document_id='{D36}'")
s,sh=A('teacher','POST',f'{QB}/share-from',{'source_course_id':A1,'what':['documents']},idem=str(uuid.uuid4())); time.sleep(10)
e1=audit_emb(); ids1=psql(f"select md5(string_agg(id::text||embedding::text,',' order by ord)) from content_chunks where document_id='{D36}'")
rb=fx.ask('sv.moi',Q36,pre_ctx=False,course=B1); ra=fx.ask('sv.gioi',Q36,pre_ctx=False)
ca,cb=cites(ra),cites(rb)
inp=[i for it in rb['payload']+ra['payload'] if it['task']=='EMBEDDING' for i in it['inputs']]
rec('TC-P8-01-36',s==200 and 'share36-qc3' in cb and 'share36-qc3' in ca and ids0==ids1 and max(len(x) for x in inp)<200,f'chia sẻ A→B ({s} {sh}); sau 10 s: SV lớp B trích dẫn {cb[:70]!r}; SV lớp A vẫn nhận {ca[:70]!r}; vectơ/đoạn của tài liệu không đổi sau share={ids0==ids1}; lời gọi nhúng chỉ cho câu hỏi (đầu vào dài nhất {max(len(x) for x in inp)} ký tự, không có đầu vào đoạn); llm_audit EMBEDDING {e0}→{e1} (tăng do 2 câu hỏi)')
# ---------- 37 phiên bản tri thức
K=f'ep:rag:ver:{A1}'; vers=[int(rds('get',K) or 0)]; steps=[]
Q37='VERSION37 hỏi lại có cache không'; v37=fx.vec([Q37])[0]; psql(f"update content_chunks set embedding='{v37}', text='VERSION37 hỏi lại có cache không. '||text where document_id='{D36}' and ord=0")
from h import chat as hchat
def cached_ask(q): 
    r=hchat('sv.gioi',q,course=A1); return r
def step(name,fn):
    v0=int(rds('get',K) or 0); fn(); time.sleep(5); v1=int(rds('get',K) or 0); steps.append((name,v0,v1)); return v1>v0
r1=cached_ask(Q37); r2=cached_ask(Q37); hit=not gen(r2) and bool(gen(r1))
ok=[]
D37=[None]
def up():
    b=mkdocx(['Phiên bản '+tag]+['Nội dung kiểm tra phiên bản tri thức số hai mươi ba. '*10]); s,rr,_=upload(QA,'ver37-qc3',b,'v37.docx'); D37[0]=docid(rr); wait_doc(D37[0])
ok.append(step('tải tài liệu tới READY',up)); 
def asked_again(): 
    r=cached_ask(Q37); return bool(gen(r))
ag=[asked_again()]
ok.append(step('PATCH cờ',lambda: A('teacher','PATCH',f'{QA}/documents/{D37[0]}',{'visible_to_students':False,'version':int(psql(f"select version from documents where id='{D37[0]}'"))}))); ag.append(asked_again())
cid=psql(f"select id from content_chunks where document_id='{D37[0]}' order by ord limit 1"); 
ok.append(step('sửa một đoạn',lambda: print('edit',A('teacher','PATCH',f'{QA}/documents/{D37[0]}/chunks/{cid}',{'text':'Nội dung đã sửa để kiểm tra phiên bản tri thức lần hai '*3})[0]))); ag.append(asked_again())
ok.append(step('xoá tài liệu',lambda: A('teacher','DELETE',f'{QA}/documents/{D37[0]}'))); ag.append(asked_again())
ag.append(None)
bb=mkdocx(['Từ lớp B '+tag]+['Tài liệu riêng của lớp B dùng để chia sẻ sang lớp A kiểm tra phiên bản. '*9]); sB,rB,_=upload(QB,'fromB-qc3',bb,'fb.docx'); wait_doc(docid(rB))
ok.append(step('chia sẻ',lambda: print('share B→A',A('teacher','POST',f'{QA}/share-from',{'source_course_id':B1,'what':['documents']},idem=str(uuid.uuid4()))) ))
print(steps,ok,ag,hit)
rec('TC-P8-01-37',all(ok) and all(x for x in ag if x is not None) and hit,f'{steps}; cache trả lời hoạt động (lần hỏi 2 không gọi sinh chữ)={hit}; hỏi lại sau mỗi thao tác có lời gọi sinh chữ mới={ag}')

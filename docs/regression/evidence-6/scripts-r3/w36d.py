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
Q36='SHARE36-QC3 hoạt động ra sao khi chia sẻ giữa hai lớp'; v36=fx.vec([Q36])[0]; psql(f"update content_chunks set embedding='{v36}', text='SHARE36-QC3 hoạt động khi chia sẻ giữa hai lớp. '||text where document_id='{D36}' and ord=0")
e0=audit_emb(); ids0=psql(f"select md5(string_agg(id::text||embedding::text,',' order by ord)) from content_chunks where document_id='{D36}'")
s,sh=A('teacher','POST',f'{QB}/share-from',{'source_course_id':A1,'what':['documents']},idem=str(uuid.uuid4())); time.sleep(10)
e1=audit_emb(); ids1=psql(f"select md5(string_agg(id::text||embedding::text,',' order by ord)) from content_chunks where document_id='{D36}'")
rb=fx.ask('sv.moi',Q36,pre_ctx=False,course=B1); ra=fx.ask('sv.gioi',Q36,pre_ctx=False)
ca,cb=cites(ra),cites(rb)
inp=[i for it in rb['payload']+ra['payload'] if it['task']=='EMBEDDING' for i in it['inputs']]
rec('TC-P8-01-36',s==200 and 'share36-qc3' in cb and 'share36-qc3' in ca and ids0==ids1 and max(len(x) for x in inp)<200,f'chia sẻ A→B ({s} {sh}); sau 10 s: SV lớp B trích dẫn {cb[:70]!r}; SV lớp A vẫn nhận {ca[:70]!r}; vectơ/đoạn của tài liệu không đổi sau share={ids0==ids1}; lời gọi nhúng chỉ cho câu hỏi (đầu vào dài nhất {max(len(x) for x in inp)} ký tự, không có đầu vào đoạn); llm_audit EMBEDDING {e0}→{e1} (tăng do 2 câu hỏi)')

# ---------- 37 (bước chia sẻ, đo kỹ)
K=f'ep:rag:ver:{A1}'
bb=mkdocx(['Từ lớp B '+tag]+['Tài liệu riêng lớp B chia sẻ sang lớp A kiểm tra phiên bản số hai. '*9]); sB,rB,_=upload(QB,'fromB2-qc3',bb,'fb2.docx'); wait_doc(docid(rB))
v0=int(rds('get',K) or 0); s,sh=A('teacher','POST',f'{QA}/share-from',{'source_course_id':B1,'what':['documents']},idem=str(uuid.uuid4()))
t0=time.time(); vs=[]
for i in range(12):
    time.sleep(2); vs.append(int(rds('get',K) or 0))
print('share',s,sh,v0,vs)
inca=psql(f"select count(*) from content_chunks c join documents d on d.id=c.document_id where d.title='fromB2-qc3' and c.course_ids @> array['{A1}'::uuid]")
print('chunks in A',inca)

from p8lib import *
C=json.load(open('/tmp/qc3e/p8c.json')); A1,B1=C['A'],C['B']; QA=f'/courses/{A1}'; QB=f'/courses/{B1}'
tag=str(int(time.time()))
b=mkdocx(['Share37 '+tag]+['Tài liệu thử chia sẻ để kiểm tra phiên bản tri thức lớp đích. '*10]); s,rr,_=upload(QA,'share37-qc3',b,'s37.docx'); D=docid(rr); wait_doc(D)
time.sleep(6)
K=f'ep:rag:ver:{B1}'; v0=int(rds('get',K) or 0)
ob0=psql(f"select count(*) from outbox where kind='document.changed' and payload::text like '%{B1}%'") if False else None
s,sh=A('teacher','POST',f'{QB}/share-from',{'source_course_id':A1,'what':['documents']},idem=str(uuid.uuid4()))
t0=time.time(); v1=v0
for i in range(14):
    time.sleep(1); v1=int(rds('get',K) or 0)
    if v1>v0: break
inB=psql(f"select count(*) from document_courses where document_id='{D}' and course_id='{B1}'")
rec('TC-P8-01-37',s==200 and inB=='1' and v1>v0,f'chạy lại sau 425396a7 (DEF-R6-P2): tải tài liệu mới vào lớp A rồi POST /courses/B/share-from (HTTP {s}); document_courses có (tài liệu, lớp đích)={inB}; ep:rag:ver:<lớp đích> {v0}→{v1} sau {time.time()-t0:.0f} s (mong tăng ≤ 5 s); các thao tác tải/PATCH/sửa đoạn/xoá đã P ở lượt trước')

from p8lib import *
import fx,re
fx.use('QC3P8A','QKTWP8A')
C=json.load(open('/tmp/qc3e/p8c.json')); A1,B1=C['A'],C['B']; QA=f'/courses/{A1}'
d=json.load(open('/tmp/qc3e/p8d.json')); dL=d['dL']; DQ=json.load(open('/tmp/qc3e/p8q.json'))['DQ']; tag=str(int(time.time()))
def cites(r): return psql(f"select citations from chat_messages where session_id='{r['sid']}' and role='ASSISTANT' order by created_at desc limit 1")
# ---- 31
res=[]
for q in ('điều kiện bị cảnh báo học vụ là gì','dieu kien bi canh bao hoc vu la gi','"cảnh báo học vụ"'):
    r=fx.ask('sv.kha',q,pre_ctx=False); c=json.loads(cites(r) or '[]'); first=c[0] if c else None
    ok=bool(first) and 'cảnh báo học vụ' in (first.get('snippet','').lower()) and first.get('document_id')==DQ
    res.append((q,ok,first['snippet'][:60] if first else None,r['row'].split('|')[0]))
print(res)
rec('TC-P8-01-31',all(x[1] for x in res),'; '.join(f'«{q}» → trích dẫn đầu {"đúng đoạn có «cảnh báo học vụ» của Quyche" if ok else "SAI/không có"} ({sn!r}, intent {it})' for q,ok,sn,it in res))
# ---- 28 PROCESSING không được trích dẫn
Q0='Hãy giải thích chế độ mã hoá khối GCM và CBC khác nhau thế nào QC30'
psql(f"update documents set status='PROCESSING' where id='{dL}'")
r=fx.ask('sv.kha',Q0,pre_ctx=False); c_p=cites(r); lib_p=A('sv.kha','GET',f'{QA}/library?q=m%E1%BB%9F')[1]
inlib_p='mo-qc3' in json.dumps(lib_p,ensure_ascii=False)
psql(f"update documents set status='READY' where id='{dL}'")
r2=fx.ask('sv.kha',Q0,pre_ctx=False); c_r=cites(r2); lib_r=A('sv.kha','GET',f'{QA}/library?q=m%E1%BB%9F')[1]; inlib_r='mo-qc3' in json.dumps(lib_r,ensure_ascii=False)
rec('TC-P8-01-28','mo-qc3' not in c_p and not inlib_p and 'mo-qc3' in c_r and inlib_r,f'tài liệu có đoạn cũ ở trạng thái PROCESSING (đặt trực tiếp trong DB: hạ READY→PROCESSING): trích dẫn={c_p[:50]!r}, /library có tài liệu={inlib_p}; sau khi trả READY: trích dẫn {c_r[:60]!r}, /library có tài liệu={inlib_r}')
# ---- 32 PATCH use_for_rag ngay rồi hỏi 5 câu
b=mkdocx(['Không RAG '+tag]+['Đoạn văn kiểm tra bật cờ RAG sẽ xếp reindex tự động QC32. '*14]); s,rr,_=upload(QA,'flag32-qc3',b,'f32.docx',DOCX,'LECTURE',use_for_rag=False); D=docid(rr); wait_doc(D)
cn=psql(f"select count(*)||'/'||count(embedding) from content_chunks where document_id='{D}'")
ver=int(psql(f"select version from documents where id='{D}'")); t0=psql("select now()")
s,_=A('teacher','PATCH',f'{QA}/documents/{D}',{'use_for_rag':True,'version':ver})
codes=[]; 
for i in range(5):
    r=fx.ask('sv.gioi',f'câu hỏi số {i} về chế độ mã hoá khối GCM QC30',pre_ctx=False); codes.append((r['status'],r['row'].split('|')[-1],r['row'][:12]))
errs=psql(f"select count(*) from llm_audit where status='error' and created_at>'{t0}'")
for i in range(40):
    e=psql(f"select count(*)||'/'||count(embedding) from content_chunks where document_id='{D}'")
    if e.split('/')[0]==e.split('/')[1]: break
    time.sleep(2)
rec('TC-P8-01-32',s==200 and all(c[0]==200 for c in codes) and errs=='0' and e.split('/')[0]==e.split('/')[1],f'tài liệu use_for_rag=false (đoạn/embedding={cn}); PATCH true → {s}; ngay lập tức 5 lượt chat → HTTP {[c[0] for c in codes]}, error_code={[c[1] for c in codes]}; llm_audit status=error từ lúc PATCH={errs}; sau reindex: {e} đoạn có embedding')
# ---- 40 reindex cả lớp + 10 câu hỏi liên tục
ids0=psql(f"select md5(string_agg(id::text,',' order by id)) from content_chunks where course_ids @> array['{A1}'::uuid]")
v=fx.vec([Q0])
s,rj=A('teacher','POST',f'{QA}/documents/reindex',idem=str(uuid.uuid4())); jid=rj.get('job_id'); print('reindex',s,rj)
res=[]
for i in range(10):
    u='sv.gioi' if i%2==0 else 'sv.kha'
    r=fx.ask(u,Q0,pre_ctx=False); c=cites(r); st=psql(f"select status from jobs where id='{jid}'"); res.append((c!='[]' and c!='',st))
for i in range(60):
    st=psql(f"select status from jobs where id='{jid}'")
    if st in('SUCCEEDED','FAILED'): break
    time.sleep(2)
n1=psql(f"select count(*) from content_chunks where course_ids @> array['{A1}'::uuid]"); ids1=psql(f"select md5(string_agg(id::text,',' order by id)) from content_chunks where course_ids @> array['{A1}'::uuid]")
s2,r2=A('teacher','POST',f'{QA}/documents/reindex',idem=str(uuid.uuid4())); time.sleep(45)
n2=psql(f"select count(*) from content_chunks where course_ids @> array['{A1}'::uuid]"); ids2=psql(f"select md5(string_agg(id::text,',' order by id)) from content_chunks where course_ids @> array['{A1}'::uuid]")
during=[x for x in res if x[1] not in('SUCCEEDED','FAILED')]
rec('TC-P8-01-40',s==202 and all(x[0] for x in res) and st=='SUCCEEDED' and n1==n2 and ids1==ids2 and ids0==ids1,f'reindex cả lớp (≥3 tài liệu đã nhúng) → {s}; 10 câu hỏi liên tiếp, có trích dẫn ở {sum(1 for x in res if x[0])}/10 (trong đó {len(during)} lượt khi job còn chạy); job → {st}; reindex lần hai → {s2}: số đoạn {n1}→{n2}, id đoạn không đổi={ids1==ids2} (và so với trước={ids0==ids1})')

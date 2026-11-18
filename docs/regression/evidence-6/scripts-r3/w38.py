from p8lib import *
import fx,re
fx.use('QC3P8A','QKTWP8A')
C=json.load(open('/tmp/qc3e/p8c.json')); A1=C['A']; QA=f'/courses/{A1}'
d=json.load(open('/tmp/qc3e/p8d.json')); dL=d['dL']; DQ=json.load(open('/tmp/qc3e/p8q.json'))['DQ']
def cites(r): return psql(f"select citations from chat_messages where session_id='{r['sid']}' and role='ASSISTANT' order by created_at desc limit 1")
# ---- 40
ids0=psql(f"select md5(string_agg(id::text,',' order by id)) from content_chunks where course_ids @> array['{A1}'::uuid]")
qs=[psql(f"select text from content_chunks where document_id='{dL}' and ord={100+i}") for i in range(9)]+[psql(f"select text from content_chunks where document_id='{dL}' and ord=100")]
s,rj=A('teacher','POST',f'{QA}/documents/reindex',idem=str(uuid.uuid4())); jid=rj.get('job_id'); res=[]
for i,q in enumerate(qs):
    u='sv.gioi' if i%2==0 else 'sv.kha'; st=psql(f"select status from jobs where id='{jid}'")
    r=fx.ask(u,q,pre_ctx=False); c=cites(r); res.append((bool(c) and c!='[]',st,r['status']))
for i in range(60):
    st=psql(f"select status from jobs where id='{jid}'")
    if st in('SUCCEEDED','FAILED'): break
    time.sleep(2)
n1=psql(f"select count(*) from content_chunks where course_ids @> array['{A1}'::uuid]"); ids1=psql(f"select md5(string_agg(id::text,',' order by id)) from content_chunks where course_ids @> array['{A1}'::uuid]")
s2,r2=A('teacher','POST',f'{QA}/documents/reindex',idem=str(uuid.uuid4())); time.sleep(45)
n2=psql(f"select count(*) from content_chunks where course_ids @> array['{A1}'::uuid]"); ids2=psql(f"select md5(string_agg(id::text,',' order by id)) from content_chunks where course_ids @> array['{A1}'::uuid]")
during=[x for x in res if x[1] not in('SUCCEEDED','FAILED')]
rec('TC-P8-01-40',s==202 and all(x[0] for x in res) and st=='SUCCEEDED' and n1==n2 and ids1==ids2 and ids0==ids1,f'reindex cả lớp (≥3 tài liệu đã nhúng) → {s}; 10 câu hỏi liên tiếp (mỗi câu = văn bản một đoạn, nhúng giả tất định nên cosine≈1 ⇒ có ngữ cảnh), có trích dẫn ở {sum(1 for x in res if x[0])}/10, HTTP {sorted(set(x[2] for x in res))}, {len(during)} lượt khi job còn chạy; job → {st}; reindex lần hai → {s2}: số đoạn {n1}→{n2}, id đoạn không đổi={ids1==ids2} (và so với trước khi reindex={ids0==ids1})')
# ---- 31 (Quyche đã được reindex ở trên: vectơ thật lại)
psql(f"delete from content_chunks where document_id='{dL}' and ord=300")
res=[]
for q in ('điều kiện bị cảnh báo học vụ là gì','dieu kien bi canh bao hoc vu la gi','"cảnh báo học vụ"'):
    vq=fx.vec([q])[0]; psql(f"delete from content_chunks where document_id='{dL}' and ord=300")
    psql(f"insert into content_chunks (document_id,course_ids,audience,ord,text,embedding) values ('{dL}',array['{A1}'::uuid],'ALL',300,'Mục khác không liên quan QC31','{vq}')")
    r=fx.ask('sv.kha',q,pre_ctx=False); c=json.loads(cites(r) or '[]'); first=c[0] if c else None
    ok=False
    if first and first.get('document_id')==DQ:
        sn=first['snippet'].replace("'","''")[:40]; ok=psql(f"select count(*) from content_chunks where document_id='{DQ}' and text like '{sn}%' and text ilike '%cảnh báo học vụ%'")!='0'
    res.append((q,ok,first['snippet'][:50] if first else None,len(c)))
psql(f"delete from content_chunks where document_id='{dL}' and ord=300")
rec('TC-P8-01-31',all(x[1] for x in res),'nhúng giả tất định ⇒ cosine thật của từ khoá ≈ 0 (cổng RagSimFloor ⇒ no_context); QC thêm MỘT đoạn mồi không liên quan (vectơ = vectơ câu hỏi) để qua cổng, các đoạn Quyche giữ vectơ thật ⇒ thứ hạng của đoạn Quyche do nhánh từ khoá quyết: '+'; '.join(f'«{q}» → trích dẫn đầu {"đúng đoạn Quyche chứa «cảnh báo học vụ»" if ok else "SAI/không có"} ({sn!r}; {n} trích dẫn)' for q,ok,sn,n in res))

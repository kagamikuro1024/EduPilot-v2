from p8lib import *
import fx,re
fx.use('QC3P8A','QKTWP8A')
C=json.load(open('/tmp/qc3e/p8c.json')); A1,B1=C['A'],C['B']; QA=f'/courses/{A1}'
d=json.load(open('/tmp/qc3e/p8d.json')); dL=d['dL']; DQ=json.load(open('/tmp/qc3e/p8q.json'))['DQ']; tag=str(int(time.time()))
def cites(r): return psql(f"select citations from chat_messages where session_id='{r['sid']}' and role='ASSISTANT' order by created_at desc limit 1")
Q0='Hãy giải thích chế độ mã hoá khối GCM và CBC khác nhau thế nào QC30'
# ---- 31
res=[]
for q in ('điều kiện bị cảnh báo học vụ là gì','dieu kien bi canh bao hoc vu la gi','"cảnh báo học vụ"'):
    vq=fx.vec([q])[0]; psql(f"update content_chunks set embedding='{vq}' where document_id='{DQ}'")  # mọi đoạn Quyche cùng cosine=1 → thứ hạng do nhánh từ khoá quyết định
    r=fx.ask('sv.kha',q,pre_ctx=False); c=json.loads(cites(r) or '[]'); first=c[0] if c else None
    ok=bool(first) and 'cảnh báo học vụ' in (first.get('snippet','').lower()) and first.get('document_id')==DQ
    res.append((q,ok,first['snippet'][:60] if first else None,r['row'].split('|')[0]))
print(res)
rec('TC-P8-01-31',all(x[1] for x in res),'stack QC dùng nhúng giả tất định nên cosine thật của từ khoá ≈ 0 (cổng RagSimFloor ⇒ no_context); để kiểm nhánh từ khoá QC đặt vectơ mọi đoạn Quyche = vectơ câu hỏi (cosine ngang nhau, thứ hạng do tsv quyết): '+'; '.join(f'«{q}» → trích dẫn đầu {"đúng đoạn có «cảnh báo học vụ» của Quyche" if ok else "SAI/không có"} ({sn!r}, intent {it})' for q,ok,sn,it in res))
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

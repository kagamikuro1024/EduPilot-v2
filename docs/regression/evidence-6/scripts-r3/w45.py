from p8lib import *
import fx,re
fx.use('QC3P8A','QKTWP8A')
C=json.load(open('/tmp/qc3e/p8c.json')); A1=C['A']; QA=f'/courses/{A1}'
d=json.load(open('/tmp/qc3e/p8d.json')); dL=d['dL']; DQ=json.load(open('/tmp/qc3e/p8q.json'))['DQ']
def cites(r): return psql(f"select citations from chat_messages where session_id='{r['sid']}' and role='ASSISTANT' order by created_at desc limit 1")
vn=fx.vec(['nhiễu QC31'])[0]
psql(f"delete from content_chunks where document_id='{dL}' and ord=300")
res=[]
for q in ('điều kiện bị cảnh báo học vụ là gì','dieu kien bi canh bao hoc vu la gi','"cảnh báo học vụ"'):
    vq=fx.vec([q])[0]; psql(f"delete from content_chunks where document_id='{dL}' and ord=300")
    psql(f"insert into content_chunks (document_id,course_ids,audience,ord,text,embedding) values ('{dL}',array['{A1}'::uuid],'ALL',300,'Mục khác không liên quan QC31','{vq}')")
    psql(f"update content_chunks set embedding=('{vq}'::vector + '{vn}'::vector) where document_id='{DQ}' and text ilike '%cảnh báo học vụ%'")
    r=fx.ask('sv.kha',q,pre_ctx=False); c=json.loads(cites(r) or '[]'); first=c[0] if c else None
    ok=False
    if first and first.get('document_id')==DQ:
        sn=first['snippet'].replace("'","''")[:40]; ok=psql(f"select count(*) from content_chunks where document_id='{DQ}' and text like '{sn}%' and text ilike '%cảnh báo học vụ%'")!='0'
    res.append((q,ok,first['snippet'][:50] if first else None,len(c)))
psql(f"delete from content_chunks where document_id='{dL}' and ord=300")
rec('TC-P8-01-31',all(x[1] for x in res),'nhúng giả tất định ⇒ cosine thật của từ khoá ≈ 0 (cổng RagSimFloor ⇒ no_context). QC dựng: một đoạn mồi không liên quan có vectơ = vectơ câu hỏi (cosine 1, hạng 1 nhánh vectơ) và các đoạn Quyche chứa «cảnh báo học vụ» có vectơ gần câu hỏi nhưng thấp hơn mồi (v+nhiễu, hạng 2–3): đoạn Quyche chỉ vượt mồi nếu nhánh từ khoá (tsv, RRF) cũng chọn nó: '+'; '.join(f'«{q}» → trích dẫn đầu {"đúng đoạn Quyche chứa «cảnh báo học vụ»" if ok else "SAI/không có"} ({sn!r}; {n} trích dẫn)' for q,ok,sn,n in res))

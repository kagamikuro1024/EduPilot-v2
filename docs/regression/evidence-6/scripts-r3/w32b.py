from p8lib import *
C=json.load(open('/tmp/qc3e/p8c.json')); A1=C['A']; QA=f'/courses/{A1}'; d=json.load(open('/tmp/qc3e/p8d.json')); dL=d['dL']
def ver(D): return int(psql(f"select version from documents where id='{D}'"))
def emb(D): return psql(f"select count(*)||'/'||count(embedding)||'/'||coalesce(min(vector_dims(embedding)),0)||'/'||coalesce(max(vector_dims(embedding)),0) from content_chunks where document_id='{D}'")
# 40 reindex cả lớp + hỏi liên tục (truy xuất trong DB: dùng RAG qua API thư viện search đồng thời)
n0=psql(f"select count(*) from content_chunks where course_ids @> array['{A1}'::uuid]"); ids0=psql(f"select md5(string_agg(id::text,',' order by id)) from content_chunks where course_ids @> array['{A1}'::uuid]")
s,r=A('teacher','POST',f'{QA}/documents/reindex',idem=str(uuid.uuid4())); print('reindex all',s,r)
res=[]; t0=time.time()
while time.time()-t0<60:
    s2,lib=A('sv.gioi','GET',f'{QA}/library?q=M%E1%BA%ADt%20m%C3%A3%20h%E1%BB%8Dc'); res.append((s2,len(lib.get('items',[])) if isinstance(lib,dict) else -1)); time.sleep(1.5)
    st=psql(f"select status from jobs where id='{r.get('job_id')}'") if isinstance(r,dict) else ''
    if st in('SUCCEEDED','FAILED'): break
print(st,res[:3],len(res))
n1=psql(f"select count(*) from content_chunks where course_ids @> array['{A1}'::uuid]"); ids1=psql(f"select md5(string_agg(id::text,',' order by id)) from content_chunks where course_ids @> array['{A1}'::uuid]")
s,r2=A('teacher','POST',f'{QA}/documents/reindex',idem=str(uuid.uuid4())); time.sleep(40)
n2=psql(f"select count(*) from content_chunks where course_ids @> array['{A1}'::uuid]"); ids2=psql(f"select md5(string_agg(id::text,',' order by id)) from content_chunks where course_ids @> array['{A1}'::uuid]")
json.dump({'res':res,'st':st},open('/tmp/qc3e/p8_40.json','w'))
print(n0,n1,n2,ids0==ids1,ids1==ids2,all(x[0]==200 for x in res))

from p8lib import *
import datetime
C=json.load(open('/tmp/qc3e/p8c.json')); A1=C['A']; QA=f'/courses/{A1}'; LOCK='/tmp/qc2-lock-global'
FP=psql("select 1") and None
s,pl=A('admin','GET','/admin/llm/providers'); FP=[x['id'] for x in pl['items'] if x['name']=='QC fake'][0]
def log(m): open('/tmp/qc2-global-changes.log','a').write(f"{datetime.datetime.now():%H:%M:%S} qc3: {m}\n")
for i in range(80):
    if subprocess.run(['mkdir',LOCK],capture_output=True).returncode==0: break
    time.sleep(15)
log('LOCK — P8-01 55: EMBEDDING route → fake embedding dims 512 (≤3 phút)')
P=[x for x in A('admin','GET','/admin/llm/providers')[1]['items'] if x['id']==FP][0]; orig=[{k:v for k,v in {'model':m['model'],'kind':m['kind'],'dims':m['dims'],'price_in':m['price_in'],'price_out':m['price_out'],'enabled':m['enabled']}.items() if v is not None} for m in P['models']]
D=None
try:
    inp=orig+[{'model':'fake-embed-512','kind':'embedding','dims':512,'price_in':'0.0000','price_out':'0.0000','enabled':True}]
    s,r=A('admin','PUT','/admin/llm/providers/'+FP,{'type':'fake','name':P['name'],'enabled':True,'models':inp,'version':P['version']}); print('provider',s,str(r)[:150])
    P2=[x for x in A('admin','GET','/admin/llm/providers')[1]['items'] if x['id']==FP][0]; mid=[m['id'] for m in P2['models'] if m['model']=='fake-embed-512'][0]
    rt=[r2 for r2 in A('admin','GET','/admin/llm/routes')[1]['items'] if r2['task']=='EMBEDDING'][0]
    s,r=A('admin','PUT','/admin/llm/routes',{'task':'EMBEDDING','chain':[mid],'version':rt['version']}); print('route',s,str(r)[:200])
    b=mkdocx(['Embed512 '+str(time.time())]+['Tài liệu thử nhúng sai chiều vectơ 512. '*14])
    s,rr,_=upload(QA,'embed512-qc3',b,'e512.docx',DOCX,'LECTURE',use_for_rag=True); D=docid(rr); jid=rr.get('job_id'); st,dt=wait_doc(D,maxs=120)
    em=psql(f"select coalesce(error,'') from documents where id='{D}'"); je=psql(f"select coalesce(error::text,'') from jobs where id='{jid}'")
    ch=psql(f"select count(*)||'/'||count(embedding) from content_chunks where document_id='{D}'"); dims=psql(f"select coalesce(string_agg(distinct vector_dims(embedding)::text,','),'-') from content_chunks where document_id='{D}'")
    print(st,em,je,ch,dims)
    rec('TC-P8-01-55',st=='FAILED' and 'EMBED_FAILED' in je and dims in('-',''),f'EMBEDDING → fake-embed-512 (dims 512): tài liệu use_for_rag=true → {st}; jobs.error.code={je[:100]}; documents.error (câu hiển thị)={em!r}; đoạn/embedding={ch}, chiều vectơ trong DB={dims} (không đoạn nào sai chiều)')
finally:
    rt=[r2 for r2 in A('admin','GET','/admin/llm/routes')[1]['items'] if r2['task']=='EMBEDDING'][0]
    print('route restore',A('admin','PUT','/admin/llm/routes',{'task':'EMBEDDING','chain':[],'version':rt['version']})[0])
    P3=[x for x in A('admin','GET','/admin/llm/providers')[1]['items'] if x['id']==FP][0]
    print('provider restore',A('admin','PUT','/admin/llm/providers/'+FP,{'type':'fake','name':P3['name'],'enabled':True,'models':orig,'version':P3['version']})[0])
    subprocess.run(['rmdir',LOCK]); log('UNLOCK — P8-01 55 xong, EMBEDDING route và provider QC fake đã hoàn')

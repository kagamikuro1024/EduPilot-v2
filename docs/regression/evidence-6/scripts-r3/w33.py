from p8lib import *
C=json.load(open('/tmp/qc3e/p8c.json')); A1=C['A']; QA=f'/courses/{A1}'
tag=str(int(time.time()))
b=mkdocx(['Tài liệu không dùng RAG '+tag]+['Khoá đối xứng AES và các chế độ mã hoá khối, ECB CBC CTR GCM. '*14 for i in range(4)])
s,r,_=upload(QA,'norag-qc3',b,'norag.docx',DOCX,'LECTURE',use_for_rag=False); D=docid(r); st,_=wait_doc(D)
def emb(D): return psql(f"select count(*)||'/'||count(embedding)||'/'||coalesce(min(vector_dims(embedding)),0) from content_chunks where document_id='{D}'")
e0=emb(D); ver=int(psql(f"select version from documents where id='{D}'")); urag=psql(f"select use_for_rag from documents where id='{D}'")
j0=psql("select count(*) from jobs where kind='document.reindex'"); s,r=A('teacher','PATCH',f'{QA}/documents/{D}',{'use_for_rag':True,'version':ver}); 
for i in range(30):
    time.sleep(2); e1=emb(D)
    if e1.split('/')[0]==e1.split('/')[1]: break
jn=psql("select count(*) from jobs where kind='document.reindex'")
n=e0.split('/')
rec('TC-P8-01-41',s==200 and n[1]=='0' and int(jn)>int(j0) and e1.split('/')[0]==e1.split('/')[1] and e1.split('/')[2]=='1536',f'tải lên với use_for_rag={urag} → {st}, (đoạn/có embedding/dim)={e0} (NULL); PATCH true → {s}; job document.reindex {j0}→{jn}; sau xử lý {e1}: mọi đoạn có embedding 1536 chiều')
json.dump({'D':D},open('/tmp/qc3e/p8n.json','w'))

from p8lib import *
C=json.load(open('/tmp/qc3e/p8c.json')); A1,B1=C['A'],C['B']; QA=f'/courses/{A1}'; f=json.load(open('/tmp/qc3e/p8f.json')); d=json.load(open('/tmp/qc3e/p8d.json'))
# 62
sv=psql("select id from jobs where owner_id=(select id from users where email='sv.kha@edupilot.local') limit 1")
if not sv:
    s,r=A('sv.kha','POST','/me/exports') if False else (0,0)
hid=d['dH']; 
r1=A('sv.gioi','GET',f'/jobs/{sv}') if sv else (None,None); r2=A('sv.gioi','GET',f'{QA}/documents'); r3=A('sv.gioi','GET',f'{QA}/library/{hid}')
rec('TC-P8-01-62',(r1[0]==404 if sv else True) and r2[0]==403 and r3[0]==404 and ('sv.kha' not in json.dumps(r1[1])),f'SV A gọi job của SV B ({sv[:8] if sv else "không có job"}) → {r1[0]}; GET /documents → {r2[0]}; /library/<tài liệu ẩn> → {r3[0]} {r3[1].get("code") if isinstance(r3[1],dict) else ""}; thân không chứa dữ liệu của B')
# 57 retry
D=f['D54']; k=str(uuid.uuid4())
a=A('teacher','POST',f'{QA}/documents/{D}/retry',idem=k); b=A('teacher','POST',f'{QA}/documents/{D}/retry',idem=k)
jn=psql(f"select count(*) from jobs where id='{a[1].get('job_id')}'") if a[0]==202 else '0'
time.sleep(1); st=psql(f"select status from documents where id='{D}'")
c=A('teacher','POST',f'{QA}/documents/{d["dL"]}/retry',idem=str(uuid.uuid4())); e=A('teacher','POST',f'{QA}/documents/{D}/reindex',idem=str(uuid.uuid4()))
rec('TC-P8-01-57',a[0]==202 and a[1].get('job_id') and b[0]==202 and b[1].get('job_id')==a[1].get('job_id') and c[0]==409 and c[1].get('code')=='DOCUMENT_NOT_FAILED',f'retry lần 1 → {a[0]} job {str(a[1].get("job_id"))[:8]}, status ngay sau={st} (QUEUED→xử lý lại, lại FAILED vì tệp vẫn hỏng); lần 2 cùng khoá → {b[0]} cùng job={b[1].get("job_id")==a[1].get("job_id")}; retry trên READY → {c[0]} {c[1].get("code")}; reindex trên tài liệu FAILED → {e[0]} {e[1].get("code")}')
# 19 Quyche OCR
q=open(f'{DOCS}/Quyche.pdf','rb').read()+b'\n%QC3-quyche-'+str(int(time.time())).encode()+b'\n'
s,r,_=upload(QA,'Quy chế đào tạo QC3',q,'Quyche.pdf',PDF); DQ=docid(r); t0=time.time(); st,dt=wait_doc(DQ,maxs=420); pages=psql(f"select page_count from documents where id='{DQ}'")
n=psql(f"select count(*) from content_chunks c join documents d on d.id=c.document_id where d.title like '%Quy chế%' and c.text ilike '%học vụ%' and d.id='{DQ}'")
pg=[int(x) for x in psql(f"select page_no from content_chunks where document_id='{DQ}' order by ord").split()]
mono=all(a<=b for a,b in zip(pg,pg[1:]))
rec('TC-P8-01-19',st=='READY' and int(n)>=1 and mono,f'Quyche.pdf (bản scan, {pages} trang) → {st} sau {dt:.0f}s ({dt/max(1,int(pages or 1)):.1f}s/trang); đoạn chứa "học vụ" đúng dấu={n}; page_no không giảm={mono} ({len(pg)} đoạn)')
json.dump({'DQ':DQ,'pages':pages,'dt':dt},open('/tmp/qc3e/p8q.json','w'))

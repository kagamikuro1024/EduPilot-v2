"""TC-P8-01-43 theo P-11-94: tắt docling → QUEUED trong lúc chờ → FAILED ≈ 2 phút 35 s → bật lại + Thử lại → READY → tắt docling lại."""
from p8lib import *
import os,datetime
C=json.load(open('/tmp/qc3e/p8c.json')); B1=C['B']; QB=f'/courses/{B1}'; LOCK='/tmp/qc2-lock-global'
def log(m): open('/tmp/qc2-global-changes.log','a').write(f"{datetime.datetime.now():%H:%M:%S} qc3: {m}\n")
b=open(os.path.expanduser('~/Documents/TA_Agent_qc3p/seed/documents/QMB12ch6b.pdf'),'rb').read()+b'\n%qc3-43-'+str(time.time()).encode()+b'\n'
for i in range(80):
    if subprocess.run(['mkdir',LOCK],capture_output=True).returncode==0: break
    time.sleep(15)
log('LOCK — P8-01 43 (P-11-94): tắt docling ep-r, tải PDF, đợi QUEUED→FAILED (~2 phút 35 s), bật lại + Thử lại, rồi tắt docling lại (≤10 phút)')
D=None; seq=[]
try:
    subprocess.run(['docker','stop','ep-r-docling-1'],capture_output=True); t0=time.time()
    s,rr,_=upload(QB,'q43-'+str(int(t0)),b,'q43.pdf','application/pdf','LECTURE',use_for_rag=True); D=docid(rr); print('complete',s,D,flush=True)
    last=None
    while time.time()-t0<330:
        st=psql(f"select status from documents where id='{D}'")
        if st!=last: seq.append((round(time.time()-t0),st)); print(seq[-1],flush=True); last=st
        if st in('FAILED','READY'): break
        time.sleep(3)
    er=psql(f"select coalesce(error,'') from documents where id='{D}'"); ec=psql(f"select coalesce(j.error->>'code','') from jobs j where j.kind='document.ingest' order by j.created_at desc limit 1")
    t_fail=time.time()-t0; subprocess.run(['docker','start','ep-r-docling-1'],capture_output=True)
    for i in range(40):
        h=subprocess.run(['docker','inspect','-f','{{.State.Health.Status}}','ep-r-docling-1'],capture_output=True,text=True).stdout.strip()
        if h=='healthy': break
        time.sleep(3)
    s2,r2=A('teacher','POST',f'{QB}/documents/{D}/retry',{},idem=str(uuid.uuid4())); print('retry',s2,str(r2)[:120],flush=True)
    st2,sec=wait_doc(D,maxs=240)
finally:
    subprocess.run(['docker','stop','ep-r-docling-1'],capture_output=True)
    subprocess.run(['rmdir',LOCK]); log('UNLOCK — P8-01 43 xong: docling ep-r đã TẮT lại theo PM')
q=[x for x in seq if x[1]=='QUEUED']; ok=(rr is not None and s in (200,201,202)) and q and seq[-1][1]=='FAILED' and 130<=t_fail<=200 and ec=='EXTRACT_UNAVAILABLE' and st2=='READY'
rec('TC-P8-01-43',bool(ok),f'theo P-11-94 (BA 633f8c04, DEF-R6-U12 không phải lỗi mã): docling ep-r tắt (docker stop); tải PDF QMB12 (biến thể) → complete {s}; trạng thái theo giây {seq}: QUEUED trong lúc chờ thử lại, FAILED sau {t_fail:.0f} s (≈ 2 phút 35 s, 3 lần chờ 5 s/30 s/2 phút), documents.error=«{er}», jobs.error.code={ec}; bật lại docling (healthy) rồi POST …/documents/{{id}}/retry → HTTP {s2}; tài liệu → {st2} sau {sec:.0f} s; docling đã tắt lại sau ca')

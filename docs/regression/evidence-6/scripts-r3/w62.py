from p8lib import *
import os
C=json.load(open('/tmp/qc3e/p8c.json')); B1=C['B']; QB=f'/courses/{B1}'
D=os.path.expanduser('~/Documents/TA_Agent_qc3p/seed/documents/')
res=[]
ld=lambda: subprocess.run(['sh','-c',"uptime | awk -F'load averages?: ' '{print $2}'"],capture_output=True,text=True).stdout.strip()
for fn,pages,lim in [('Quyche.pdf',5,6.0),('QMB12ch6b.pdf',24,1.0),('Mordern_Network_Security_Threats.pdf',141,1.0)]:
    b=open(D+fn,'rb').read()+b'\n%qc3t-'+str(time.time()).encode()+b'\n'
    l0=ld(); s,rr,_=upload(QB,'time62-'+fn[:8],b,fn,'application/pdf','LECTURE',use_for_rag=True); d=docid(rr); st,_=wait_doc(d,maxs=600)
    sec=float(psql(f"select extract(epoch from updated_at-created_at) from documents where id='{d}'")); l1=ld()
    res.append((fn,st,pages,round(sec),round(sec/pages,2),lim,l0,l1)); print(res[-1],flush=True)
ok=all(r[1]=='READY' and r[4]<=r[5] for r in res)
rec('TC-P8-01-42-timing',ok,'đo giây/trang (updated_at−created_at) trên ep-r, docling mới bật, máy ít việc: '+'; '.join(f'{r[0]} {r[2]} trang {r[3]} s = {r[4]} s/trang (ngưỡng ≤{r[5]}; load trước/sau {r[6]} | {r[7]})' for r in res))

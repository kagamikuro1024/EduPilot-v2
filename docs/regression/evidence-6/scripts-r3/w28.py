from p8lib import *
from PIL import Image
from pypdf import PdfWriter
C=json.load(open('/tmp/qc3e/p8c.json')); A1,B1=C['A'],C['B']; QA=f'/courses/{A1}'
tag=str(int(time.time()))
# files
im=[Image.new('RGB',(1240,1754),'white') for _ in range(2)]; im[0].info['t']=tag
bio=io.BytesIO(); im[0].save(bio,'PDF',save_all=True,append_images=im[1:],resolution=150); blank=bio.getvalue()+b'\n%'+tag.encode()+b'\n'
w=PdfWriter(); [w.add_blank_page(595,842) for _ in range(401)]; bio=io.BytesIO(); w.write(bio); p401=bio.getvalue()+b'\n%'+tag.encode()+b'\n'
# ===== 16 SSE + 15: DOCX có nhiều đoạn
tk=login('teacher'); ev=[]; stop=[False]
import ssl
def listen():
    req=urllib.request.Request('https://localhost:1143/api/v1/events',headers={'Authorization':'Bearer '+tk})
    try:
        r=urllib.request.urlopen(req,context=ctx,timeout=200)
        for ln in r:
            ev.append(ln.decode()); 
            if stop[0]: break
    except Exception as e: ev.append('ERR '+str(e))
th=threading.Thread(target=listen,daemon=True); th.start(); time.sleep(2)
bd=mkdocx([('Đoạn %d. '%i)+('Mật mã học khoá công khai và chữ ký số. '*18) for i in range(30)]+[tag])
s,r,d=upload(QA,'sse-qc3',bd,'sse.docx'); D=docid(r); st,dt=wait_doc(D); time.sleep(2); stop[0]=True
txt=''.join(ev); import re
prog=[int(x) for x in re.findall(r'"progress":(\d+)',txt)]
print(st,dt,prog)
mono=all(a<=b for a,b in zip(prog,prog[1:]))
rec('TC-P8-01-16',st=='READY' and prog and prog[0]<=10 and prog[-1]==100 and mono and any(10<=p<=45 for p in prog) and 50 in prog and any(55<=p<=95 for p in prog) and 'job.result' in txt,f'SSE /events mở trước khi tải: {len(prog)} sự kiện progress {prog}; đơn điệu={mono}; có mốc 10–45={any(10<=p<=45 for p in prog)}, 50={50 in prog}, 55–95={any(55<=p<=95 for p in prog)}; có job.result={"job.result" in txt}; READY sau {dt:.0f}s')
# ===== 54 HASH_MISMATCH
good=mkdocx(['Tệp thật '+tag,'Nội dung gốc đủ dài để vượt ngưỡng năm mươi ký tự của bộ lọc.']); bad=mkdocx(['Tệp KHÁC '+tag,'Nội dung bị tráo đủ dài để vượt ngưỡng năm mươi ký tự lọc.'])
bad=bad+b'\0'*(len(good)-len(bad)) if len(bad)<len(good) else bad[:len(good)] if False else bad
good=good+b'\0'*(len(bad)-len(good)) if len(good)<len(bad) else good
s,dp=presign('teacher',QA,'swap.docx',DOCX,good); assert s==200,(s,dp); print('put',put(dp,bad)); s,r=complete('teacher',QA,dp['upload_id'],'hash-mismatch-qc3'); print('complete',s,str(r)[:150])
D54=docid(r); st54,_=wait_doc(D54,maxs=90); e54=psql(f"select error from documents where id='{D54}'")
rec('TC-P8-01-54',st54=='FAILED' and 'HASH_MISMATCH' in e54,f'PUT tệp khác cùng kích thước ({len(good)} byte): complete → {s}; status={st54}; error={e54!r}')
# ===== 52 blank
s,r,_=upload(QA,'blank-scan-qc3',blank,'blank.pdf',PDF); D52=docid(r); t0=time.time(); st52,dt52=wait_doc(D52,maxs=240); e52=psql(f"select error from documents where id='{D52}'"); att=psql(f"select max(j.progress)||'/'||count(*) from jobs j where j.result::text like '%{D52}%' or j.id=(select job_id from documents where id='{D52}')") if False else ''
rec('TC-P8-01-52',st52=='FAILED' and 'NO_TEXT' in e52 and dt52<150,f'PDF 2 trang ảnh trắng → {st52} sau {dt52:.0f}s (không thử lại 3 lần: lùi 5 s+30 s+2 phút ≥ 155 s nếu có); error={e52!r}')
# ===== 53 401 trang
s,r,_=upload(QA,'p401-qc3',p401,'p401.pdf',PDF); D53=docid(r); st53,dt53=wait_doc(D53,maxs=240); e53=psql(f"select error from documents where id='{D53}'")
rec('TC-P8-01-53',st53=='FAILED' and 'TOO_MANY_PAGES' in e53 and dt53<150,f'PDF 401 trang → {st53} sau {dt53:.0f}s; error={e53!r}')
json.dump({'D54':D54,'D52':D52,'D53':D53},open('/tmp/qc3e/p8f.json','w'))
print(psql(f"select id,error from documents where id in ('{D52}','{D53}','{D54}')"))

from p8lib import *
C=json.load(open('/tmp/qc3e/p8c.json')); A1=C['A']; QA=f'/courses/{A1}'
tag=str(int(time.time()))
b=mkdocx(['Gone '+tag]+['Tài liệu sẽ mất tệp gốc để kiểm tra FILE_GONE khi tải xuống. '*10]); s,r,_=upload(QA,'gone43-qc3',b,'g43.docx'); D=docid(r); wait_doc(D)
key=psql(f"select blob_key from documents where id='{D}'")
print(subprocess.run(['docker','exec','ep-r-minio-1','sh','-c',f'rm -rf /data/edupilot/{key}; ls /data/edupilot/{key} 2>&1 | head -2'],capture_output=True,text=True).stdout)
n0=psql(f"select download_count from documents where id='{D}'")
s,t,_=raw('GET',f'{QA}/library/{D}/download',login('sv.gioi')); n1=psql(f"select download_count from documents where id='{D}'")
ok=s==404 and 'FILE_GONE' in t
json.dump({'D43':D,'status':s,'body':t[:200],'n':[n0,n1]},open('/tmp/qc3e/p8_43.json','w'),ensure_ascii=False)
print(s,t[:200],n0,n1)

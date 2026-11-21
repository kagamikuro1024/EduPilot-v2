# TC-P4-06-47 (menu báo cáo 1280/375): hai bài mới trong QC3T1 (đề phòng báo cáo trùng ở bài cũ) — ghi đè th.rep / th.rep2 trong fx_ui8.json
from lib3 import *
F=json.load(open('/tmp/qc3r7/fx_ui8.json')); C=F['C']; Q=f'/courses/{C}'
def th(u,title,body):
    s,d=A(u,'POST',f'{Q}/threads',{'title':title,'body':body,'tags':[]}); assert s==201,(s,d); return d['thread']['id']
F['th']['rep']=th('qc3s15','QC3T1 r7 bài báo cáo một','Nội dung bài để thử menu báo cáo (chạy lại r7).')
F['th']['rep2']=th('qc3s16','QC3T1 r7 bài báo cáo hai','Nội dung bài thứ hai để thử menu báo cáo 375 px (chạy lại r7).')
json.dump(F,open('/tmp/qc3r7/fx_ui8.json','w')); print(F['th'])

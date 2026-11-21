# fixture QC3G (GV qc3t5): 8 phiếu OPEN mới cho TC-P4-03-36, 2 phiếu CLAIMED (qc3t5) cho TC-P4-03-33
from lib3 import *
G=C7['G']; Q=f'/courses/{G}'; T='qc3t5'
clearopen(T,G)
ids=[]
for i in range(10):
    u='qc3s%02d'%(i%5+1)
    tid,_=tick(u,G,q=f'Câu hỏi r7 số {i}: hạn nộp bài tập ở đâu?',a='AI chưa chắc chắn',lowc=True,conf=0.2); ids.append(tid)
claimed=ids[:2]
for t in claimed: assert A(T,'POST',f'{Q}/tickets/{t}/claim')[0]==200
json.dump({'G':G,'T':T,'open':ids[2:],'claimed':claimed},open('/tmp/qc3r7/fx_ui5.json','w'))
print(open('/tmp/qc3r7/fx_ui5.json').read()); print(psql(f"select status,count(*) from escalation_tickets where course_id='{G}' group by 1"))
# phiếu có nội dung dài (TC-33b: ô trả lời vẫn dính đáy khi chi tiết dài hơn màn hình)
if len(json.load(open('/tmp/qc3r7/fx_ui5.json')).get('long',''))==0:
    long_q=' '.join(f'Đoạn {i}: em muốn hỏi về hạn nộp bài tập lớn, cách tính điểm chuyên cần và lịch thi cuối kỳ của môn học này.' for i in range(1,26))
    tl,_=tick('qc3s03',G,q=long_q,a='AI chưa chắc chắn. '*30,lowc=True,conf=0.2)
    assert A(T,'POST',f'{Q}/tickets/{tl}/claim')[0]==200
    fx=json.load(open('/tmp/qc3r7/fx_ui5.json')); fx['long']=tl; json.dump(fx,open('/tmp/qc3r7/fx_ui5.json','w')); print('long',tl)

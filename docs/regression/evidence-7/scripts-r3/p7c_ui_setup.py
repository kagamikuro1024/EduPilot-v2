from lib3 import *
mkusers('qc3pd','TEACHER',1)
C=mkcourse('QC3T1','qc3pd',['qc3pf'],'QKTAAAB'); Q=f'/courses/{C}'
S=[f'qc3s{i}' for i in range(11,21)]
for u in S:
    psql(f"insert into enrollments(course_id,user_id,role_in_course,status,joined_via,student_code_snapshot) select '{C}',u.id,'STUDENT','ACTIVE','ROSTER',u.student_code from users u where u.email='{u}@edupilot.local' on conflict do nothing")
if psql(f"select count(*) from escalation_tickets where course_id='{C}'")=='0':
    tk_ids=[]
    for i,u in enumerate(S[:7]):
        sid,um,am=mkchat(u,C,q=f'Câu hỏi QC3 T1 số {i}: hạn nộp bài tập ở đâu?',a='AI chưa chắc chắn',lowc=True,conf=0.2)
        s,d=esc(u,am); assert s in(200,201),(s,d); tk_ids.append(d['ticket']['id'])
    for t in tk_ids[5:]:
        assert A('qc3pd','POST',f'{Q}/tickets/{t}/claim')[0]==200
        s,d=A('qc3pd','POST',f'{Q}/tickets/{t}/answer',{'body':'Chào em, hạn nộp ghi trong đề.'}); assert s in(200,201),(s,d)
def th(u,title,body):
    s,d=A(u,'POST',f'{Q}/threads',{'title':title,'body':body,'tags':[]}); assert s==201,(s,d); return d['thread']['id']
ths={}
ths['del']=th('qc3s11','QC3T1 bài để xoá','Nội dung câu hỏi để thử xoá có xác nhận.')
for u in ('qc3s12','qc3s13','qc3s14'): s,d=A(u,'POST',f'{Q}/threads/{ths["del"]}/posts',{'body':f'Bình luận của {u} về câu hỏi này.'}); assert s==201,(s,d)
ths['rep']=th('qc3s15','QC3T1 bài để báo cáo','Nội dung bài để thử báo cáo trên giao diện.')
ths['rep2']=th('qc3s16','QC3T1 bài báo cáo hai','Nội dung bài thứ hai để thử báo cáo khi mạng đứt.')
json.dump({'C':C,'S':S,'T':'qc3pd','th':ths},open('/tmp/qc3e/fx_ui8.json','w'))
print(open('/tmp/qc3e/fx_ui8.json').read()); print(psql(f"select status,count(*) from escalation_tickets where course_id='{C}' group by 1"))

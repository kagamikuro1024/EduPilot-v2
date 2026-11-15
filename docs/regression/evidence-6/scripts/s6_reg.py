from lib import *
import time
C=json.load(open('courses.json')); E1=C['E6C1']; Q=f'/courses/{E1}'
s,d=A('sv20','POST','/courses/join',{'code':'QKTWXZH'}); print('join sv20',s); rds('DEL','ep:roster:'+E1)
def pre(t,u='sv07'):
    s,d=A(u,'POST',f'{Q}/threads/precheck',{'body':t}); return d['allowed'] if s==200 else s
blk=['Huỳnh Đức Dũng nộp bài chưa','Huynh Duc Dung nop bai chua','huỳnh đức dũng','HUỲNH ĐỨC DŨNG','Dũng Huỳnh có đi học không']
ok_=['Huỳnh Đức Dụng nộp bài chưa','Huỳnh Đức Dùng nộp bài chưa']
rb=[(t,pre(t)) for t in blk]; ro=[(t,pre(t)) for t in ok_]
rec('TC-REG-P10611-block',all(x is False for _,x in rb),str(rb))
rec('TC-REG-P10611-diacritic-differs',all(x is True for _,x in ro),str(ro))
# P-11-16
a=[pre(t) for t in ['Em nghĩ AES an toàn hơn RSA đúng không','Em nghĩ thuật toán này chạy nhanh hơn']]; b=[pre(t) for t in ['Em được mấy điểm giữa kỳ?','em duoc may diem chuyen can?']]
rec('TC-REG-P1116',all(a) and not any(b),f'"em nghĩ…" cho qua={a}; câu hỏi điểm bị chặn={[not x for x in b]}')
# D60 mặc định tắt
print(psql("select * from course_decision_gates limit 5")); print(psql("select column_name from information_schema.columns where table_name in ('decision_gates','course_decision_gates')").replace('\n',','))
s,d=A('admin','GET','/admin/llm/decision-gates'); print(s,str(d)[:400])

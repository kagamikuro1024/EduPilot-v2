from lib3 import *
S=mkusers('qc3at','STUDENT',30,start=100)
for u in S: assert uid(u),u
C=mkcourse('QC3AT','qc3t',['qc3ta'],'QKA7777')
for u in S:
    if psql(f"select count(*) from enrollments where course_id='{C}' and user_id='{uid(u)}'")=='0':
        s,d=A(u,'POST','/courses/join',{'code':'QKA7777'}); 
psql(f"update enrollments set status='ACTIVE' where course_id='{C}' and status='PENDING'")
# lớp rỗng buổi để thử «Lớp chưa có lịch buổi học» + tạo lịch (P5-01)
C0=mkcourse('QC3AE','qc3t',['qc3ta'],'QKA8888')
json.dump({'C':C,'C0':C0,'S':S},open('/tmp/qc3e/fx_att.json','w'))
print(C,C0,psql(f"select count(*),count(*) filter (where status='ACTIVE') from enrollments where course_id='{C}' and role_in_course='STUDENT'"))

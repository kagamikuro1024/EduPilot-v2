from lib3 import *
import sys
# Chuẩn bị: lớp X (cho tắt đo) / Y (không cho tắt), SV mới
FX='/tmp/qc3e/r4fx.json'
def setup():
    S=mkusers('qc3p5s','STUDENT',12,start=70)
    for u in S: assert uid(u),u
    X=mkcourse('QC3X5','qc3t',['qc3ta'],'QKX5XXX'); Y=mkcourse('QC3Y5','qc3t',['qc3ta'],'QKY5YYY')
    g=A('qc3t','GET',f'/courses/{Y}')[1]; v=g['settings']['version']
    if psql(f"select count(*) from enrollments where course_id='{Y}' and role_in_course='STUDENT'")=='0':
        s,d=A('qc3t','PUT',f'/courses/{Y}/settings',{'allow_study_time_opt_out':False,'version':v}); print('Y allow=false',s,str(d)[:100])
    for u in S:
        for c,code in ((X,'QKX5XXX'),(Y,'QKY5YYY')):
            if psql(f"select count(*) from enrollments where course_id='{c}' and user_id='{uid(u)}'")=='0':
                s,d=A(u,'POST','/courses/join',{'code':code}); print('join',u,code,s,str(d)[:80])
    psql(f"update enrollments set status='ACTIVE' where course_id in ('{X}','{Y}') and status='PENDING'")
    json.dump({'X':X,'Y':Y,'S':S},open(FX,'w'))
if __name__=='__main__': setup(); print(open(FX).read()); print(psql("select class_code,allow_study_time_opt_out from courses where class_code in ('QC3X5','QC3Y5')"))

import os; os.environ['QC_AGENT']='roster2'
from lib import *
import time
C=json.load(open('courses.json')); E1=C['E6C1']; Q=f'/courses/{E1}'
def pre(u,t): 
    s,d=A(u,'POST',f'{Q}/threads/precheck',{'body':t}); return d['allowed'] if s==200 else s
def cleared(n=12):
    for i in range(n*2):
        if rds('EXISTS','ep:roster:'+E1)=='0': return (i+1)*.5
        time.sleep(.5)
    return None
# nhánh 1: vào bằng mã (không cần duyệt)
s,d=A('teacher','GET',f'{Q}/join-code'); v=d['version']
A('teacher','PUT',f'{Q}/join-settings',{'require_approval':False,'version':v})
uid=psql("select id from users where email='sv10@edupilot.local'"); A('teacher','DELETE',f'{Q}/members/{uid}'); time.sleep(2)
rds('DEL','ep:roster:'+E1); pre('sv07','Chào cả lớp hôm nay hello 1'); assert rds('EXISTS','ep:roster:'+E1)=='1'
s,d=A('sv10','POST','/courses/join',{'code':'QKTWXZH'}); c=cleared()
b=pre('sv07','Võ Thanh Trang nộp bài chưa')
rec('TC-P3-02-21',c is not None and b is False,f'vào lớp bằng mã → cache xoá sau {c}s; tên SV mới bị chặn={b is False} (join {s})')
# nhánh 2: duyệt
s,d=A('teacher','GET',f'{Q}/join-code'); A('teacher','PUT',f'{Q}/join-settings',{'require_approval':True,'version':d['version']})
uid=psql("select id from users where email='sv11@edupilot.local'"); A('teacher','DELETE',f'{Q}/members/{uid}'); time.sleep(2)
A('sv11','POST','/courses/join',{'code':'QKTWXZH'}); rds('DEL','ep:roster:'+E1); pre('sv07','Chào cả lớp hôm nay hello 2')
s,d=A('teacher','POST',f'{Q}/members/{uid}/approve'); c=cleared(); b=pre('sv07','Ngô Văn Hà nộp bài chưa')
rec('TC-P3-02-21b',c is not None and b is False,f'duyệt → cache xoá {c}s; tên bị chặn={b is False} (approve {s})')
# nhánh 3: mời ra
rds('DEL','ep:roster:'+E1); pre('sv07','Chào cả lớp hôm nay hello 3'); A('teacher','DELETE',f'{Q}/members/{uid}'); c=cleared(); 
rec('TC-P3-02-22b',c is not None,f'mời ra → cache xoá {c}s')
s,d=A('teacher','GET',f'{Q}/join-code'); A('teacher','PUT',f'{Q}/join-settings',{'require_approval':False,'version':d['version']})

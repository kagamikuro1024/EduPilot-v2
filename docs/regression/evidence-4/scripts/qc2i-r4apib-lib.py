# Thư viện chung slice r4apib (QC2-I). Dùng lib.py của evidence-3 (không sửa); rec ghi TSV.
import sys,os,json,time,uuid,subprocess,random,re
sys.path.insert(0,os.path.expanduser('~/Documents/TA_Agent_qc2/docs/regression/evidence-3/scripts'))
os.makedirs('/tmp/qc2-b',exist_ok=True)
import lib
from lib import call,psql,rds,resetrl,forge,mp,mail_to,mail_count,link_token,PW
RES='/tmp/qc2-i/res/r4apib.tsv'
TAG='qc2ib'
def rec(tc,k,note=''):
    note=note.replace('\t',' ').replace('\n',' ')
    open(RES,'a').write(f'{tc}\t{k}\t{note}\n'); print(tc,k,note[:200],flush=True)
_tk={}
def tok(email):
    if email in _tk and time.time()-_tk[email][1]<500: return _tk[email][0]
    for i in range(4):
        resetrl(); s,j,_=call('POST','/auth/login',None,{'email':email if '@' in email else email+'@edupilot.local','password':PW})
        if s==200: _tk[email]=(j['access_token'],time.time()); return j['access_token']
        time.sleep(5)
    raise Exception(f'login {email} {s} {j}')
def mkuser(label,role='STUDENT',code=None,status='ACTIVE',verified=True):
    """tạo người dùng qua SQL (mật khẩu = mật khẩu seed, copy hash); trả (email,id). Tiền tố qc2ib."""
    em=f'{TAG}.{label}@edupilot.local'
    sc=f"'{code}'" if code else 'null'
    psql(f"insert into users(email,password_hash,full_name,role,student_code,email_verified_at,status) select '{em}',password_hash,'qc2ib {label}','{role}',{sc},{'now()' if verified else 'null'},'{status}' from users where email='sv.kha@edupilot.local' on conflict(email) do nothing")
    return em,psql(f"select id from users where email='{em}'")
def uid(em): return psql(f"select id from users where email='{em}'")
def course(label,teacher=None,subject='INT1006',**kw):
    """tạo lớp qua admin API; class_code QC2IB<label>; trả id"""
    A=tok('admin')
    body={'subject_code':subject,'class_code':('QC2IB'+label).upper(),'name':f'qc2ib {label}','semester':'2026-2027-HK1'}
    if teacher: body['teacher_id']=teacher
    body.update(kw)
    s,j,_=call('POST','/admin/courses',A,body)
    if s==409 or s==422 and 'class_code' in json.dumps(j):
        return psql(f"select id from courses where class_code='{body['class_code']}'") or None
    assert s==201,(s,j)
    return j['id']
def enroll(cid,u,role='STUDENT',status='ACTIVE',via='ADMIN',code=None):
    removed=",now()" if status=='REMOVED' else ",null"
    sc=f"'{code}'" if code else 'null'
    psql(f"insert into enrollments(course_id,user_id,role_in_course,status,joined_via,student_code_snapshot,removed_at) values('{cid}','{u}','{role}','{status}','{via}',{sc}{removed}) on conflict(course_id,user_id) do update set role_in_course=excluded.role_in_course,status=excluded.status,removed_at=excluded.removed_at")
def jc(cid): return psql(f"select join_code from courses where id='{cid}'")

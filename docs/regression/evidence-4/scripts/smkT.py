# Thư viện SmkT (sprint 4 smoke N=0): dùng lib.py của evidence-3 trên ep-r; rec ghi TSV cuối-thắng.
import sys,os,json,time,uuid,subprocess,re,random
sys.path.insert(0,os.path.expanduser('~/Documents/TA_Agent_qc2/docs/regression/evidence-3/scripts'))
os.makedirs('/tmp/qc2-b',exist_ok=True)
import lib
from lib import call as call_raw,psql,rds,resetrl,forge,mp,mail_to,mail_count,link_token,PW,login,secret
B=lib.B
def call(*a,**k):
    # chống 429 RATE_LIMITED chung (giới hạn IP/phút dùng chung nhiều agent): chờ retry_after rồi gọi lại; ca đo giới hạn dùng call_raw
    for i in range(6):
        r=call_raw(*a,**k)
        if r[0]==429 and isinstance(r[1],dict) and r[1].get('code')=='RATE_LIMITED' and not k.pop('_noretry',False):
            time.sleep(min(int(r[1].get('retry_after',5))+1,40)); continue
        return r
    return r
RES=os.path.expanduser('~/Documents/TA_Agent_qc2/docs/regression/evidence-4/results-smk34/results-smk4.tsv')
EV='docs/regression/evidence-4'
SEEN=os.path.expanduser('/tmp/smkT/seen.txt')   # bí mật QC đã phát (mật khẩu thử, token) để quét log — KHÔNG commit
def seen(*xs):
    with open(SEEN,'a') as f:
        for x in xs:
            if x: f.write(str(x)+'\n')
def rec(tc,k,note=''):
    note=note.replace('\t',' ').replace('\n',' ')
    open(RES,'a').write(f'{tc}\t{k}\t{note}\n'); print(tc,k,note[:220],flush=True)
def sh(cmd,timeout=300):
    return subprocess.run(cmd,shell=True,capture_output=True,text=True,timeout=timeout,executable='/bin/bash').stdout
def iso(): return time.strftime('%Y-%m-%dT%H:%M:%SZ',time.gmtime(time.time()-3))
TAG=uuid.uuid4().hex[:5]; NPW='Smkt!Pass-'+TAG+'x7'
def mkuser(label,role='STUDENT',code=None,status='ACTIVE',verified=True):
    em=f'smkt.{label}.{TAG}@edupilot.local'
    sc=f"'{code}'" if code else 'null'
    psql(f"insert into users(email,password_hash,full_name,role,student_code,email_verified_at,status) select '{em}',password_hash,'smkt {label}','{role}',{sc},{'now()' if verified else 'null'},'{status}' from users where email='sv.kha@edupilot.local' on conflict(email) do nothing")
    return em,psql(f"select id from users where email='{em}'")
def uid(em): return psql(f"select id from users where email='{em}'")
def tokof(email):
    for i in range(4):
        resetrl(); s,j,_=call('POST','/auth/login',None,{'email':email if '@' in email else email+'@edupilot.local','password':PW})
        if s==200: return j['access_token']
        time.sleep(3)
    raise Exception(f'login {email} {s} {j}')
def register(label,code=''):
    resetrl(); em=f'smkt.{label}.{TAG}@edupilot.local'; t0=iso()
    s,j,_=call('POST','/auth/register',None,{'email':em,'password':NPW,'full_name':'SmkT '+label,'student_code':code})
    return em,s,j,t0
def mk_verified(label,code=''):
    em,s,j,t0=register(label,code); assert s==202,(s,j)
    m=mail_to(em,'Xác minh',t0,25); tk=link_token(m,'verify-email'); seen(tk,NPW)
    s,j,_=call('POST','/auth/verify-email',None,{'token':tk}); assert s==200,(s,j)
    return em
# ---- helper chung cho các lô
FX=os.path.expanduser('~/Documents/TA_Agent_qc2/docs/sprints/4/qc/fixtures/roster/')
def lg(em,pw=None):
    resetrl(); s,j,h=call('POST','/auth/login',None,{'email':em,'password':pw or NPW}); return j.get('access_token'),s,j,h
def stu(label,code='',verify=True,name=None):
    resetrl(); em=f'smkt.{label}.{TAG}@edupilot.local'; t0=iso(); s,j,_=call('POST','/auth/register',None,{'email':em,'password':NPW,'full_name':name or 'SV '+label,'student_code':code}); assert s==202,(s,j)
    if verify:
        m=mail_to(em,'Xác minh',t0,30); tk=link_token(m,'verify-email'); seen(tk); s,j,_=call('POST','/auth/verify-email',None,{'token':tk}); assert s==200,(s,j)
    return em,(lg(em)[0] if verify else None)
_A=[None]
def A():
    if not _A[0]: _A[0]=tokof('admin')
    return _A[0]
def staff(label,role,name=None):
    em=f'smkt.{label}.{TAG}@edupilot.local'; t0=iso(); s,j,_=call('POST','/admin/users',A(),{'email':em,'full_name':name or 'GV '+label,'role':role}); assert s==201,(s,j)
    m=mail_to(em,'mời|Lời mời',t0,30); tk=link_token(m,'invite'); seen(tk); resetrl(); s2,j2,_=call('POST','/auth/accept-invite',None,{'token':tk,'password':NPW}); assert s2 in(200,201,204),(s2,j2)
    return em,lg(em)[0],j['id']
def mkcourse(tag,gvid,taid=None,subject='INT1006',**kw):
    body={'subject_code':subject,'class_code':f'ST{tag}{TAG}'.upper(),'name':f'Lớp SmkT {tag} {TAG}','semester':'2026-2027-HK1','teacher_id':gvid}; body.update(kw)
    s,c,_=call('POST','/admin/courses',A(),body); assert s==201,(s,c)
    if taid: call('POST',f'/admin/courses/{c["id"]}/assign',A(),{'ta_ids':[taid]})
    return c['id']
def multipart(data,fname):
    b='----smk'+uuid.uuid4().hex; body=(f'--{b}\r\nContent-Disposition: form-data; name="file"; filename="{fname}"\r\nContent-Type: application/octet-stream\r\n\r\n').encode()+data+f'\r\n--{b}--\r\n'.encode(); return body,f'multipart/form-data; boundary={b}'
def imp(c_,tok,fname,q='',data=None):
    body,ct=multipart(data if data is not None else open(FX+fname,'rb').read(),fname); return call('POST',f'/courses/{c_}/roster/import{q}',tok,raw=body,hdr={'content-type':ct,'Idempotency-Key':str(uuid.uuid4())},idem=False)
def jc(cid,tok): return call('GET',f'/courses/{cid}/join-code',tok)[1].get('join_code')
def logs(names=('gateway','worker','caddy','postgres','redis')):
    out=''
    for n in names: out+=subprocess.run(['docker','logs',f'ep-r-{n}-1'],capture_output=True,text=True).stdout+subprocess.run(['docker','logs',f'ep-r-{n}-1'],capture_output=True,text=True).stderr
    return out

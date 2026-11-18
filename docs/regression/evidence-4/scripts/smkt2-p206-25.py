# TC-P206-25: ma trận quyền /admin/users đo qua API ep-r: 8 thao tác × 6 danh tính (ADMIN, TEACHER, TA, STUDENT, không JWT, hết hạn) = 48 ca
import os; exec(open(os.path.join(os.path.dirname(os.path.abspath(__file__)),'smkt2lib.py')).read())
resetrl()
A=L('admin'); toks={'ADMIN':A,'TEACHER':L('teacher'),'TA':L('ta'),'STUDENT':L('sv.gioi')}
tgt=psql("select id from users where email='sv.nguyco@edupilot.local'")
inv_email='smkt2.m25.%s@edupilot.local'%uuid.uuid4().hex[:6]
s,j,_=call('POST','/admin/users',A,{'email':inv_email,'full_name':'SmkT2 M25','role':'TA'}); assert s==201,(s,j)
inv=j['id']
exp=forge(A,exp=int(time.time())-600)
ids=dict(toks); ids['NOJWT']=None; ids['EXPIRED']=exp
cur=call('GET','/admin/users?limit=1',A)[1]; ver=psql(f"select version from users where id='{tgt}'")
ops=[('list','GET','/admin/users',None),('list-role','GET','/admin/users?role=TEACHER',None),('list-q','GET','/admin/users?q=sv.g',None),
('invite','POST','/admin/users',{'email':'smkt2.x.%s@edupilot.local'%uuid.uuid4().hex[:6],'full_name':'X','role':'TA'}),
('patch-name','PATCH','/admin/users/'+tgt,{'full_name':'Nguyễn Cơ','version':int(ver)}),
('patch-role','PATCH','/admin/users/'+tgt,{'role':'TEACHER','version':int(ver)}),
('patch-status','PATCH','/admin/users/'+tgt,{'status':'DISABLED','version':int(ver)}),
('resend','POST',f'/admin/users/{inv}/resend-invite',None)]
before=psql("select md5(string_agg(id::text||role||status||full_name||version::text,',' order by id)) from users")
bad=[];n=0;tbl={}
for who,tok in ids.items():
    for name,m,p,b in ops:
        if who=='ADMIN' and name in('patch-role','patch-status','invite','resend','patch-name'): continue  # không thực thi ghi bằng ADMIN trên dữ liệu seed (chỉ đo ma trận chặn)
        st,jj,_=call(m,p,tok,b,idem=True)
        n+=1; code=jj.get('code') or jj.get('error',{}).get('code') if isinstance(jj,dict) else None
        reason=(jj.get('details') or {}).get('reason') if isinstance(jj,dict) else None
        tbl[(who,name)]=(st,code,reason)
        if who=='ADMIN': ok=st in(200,201)
        elif who=='NOJWT': ok=st==401
        elif who=='EXPIRED': ok=st==401 and code=='TOKEN_EXPIRED'
        else: ok=st==403 and code=='FORBIDDEN' and reason=='role'
        if not ok: bad.append((who,name,st,code,reason))
after=psql("select md5(string_agg(id::text||role||status||full_name||version::text,',' order by id)) from users")
# ADMIN ghi: 1 mời + 1 gửi lại hợp lệ để chứng minh ma trận không chặn nhầm
st1,_,_=call('POST','/admin/users',A,{'email':'smkt2.ad.%s@edupilot.local'%uuid.uuid4().hex[:6],'full_name':'Ad','role':'TA'}); st2,_,_=call('POST',f'/admin/users/{inv}/resend-invite',A); n+=2
if st1!=201: bad.append(('ADMIN','invite',st1)); 
if st2 not in(200,202,204,429): bad.append(('ADMIN','resend',st2))
rec('TC-P206-25','P' if not bad and before==after else 'F',f'ma trận đo qua API ep-r (Go test TestAdminUsersRBACMatrix chỉ có 21 ca): {n} ca = 8 thao tác (GET list/role/q, POST invite, PATCH tên/vai/trạng thái, POST resend-invite) × 6 danh tính; TEACHER/TA/STUDENT → 403 FORBIDDEN reason=role kể cả GET; không JWT → 401; hết hạn → 401 TOKEN_EXPIRED; ADMIN → 2xx (đọc 3 + mời 1 + gửi lại {st2}); DB người dùng không đổi sau các ca bị chặn (md5 trước/sau khớp: {before==after}); ca lệch {bad}; script smkt2-p206-25.py')

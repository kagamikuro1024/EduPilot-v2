import os; os.environ['SP']='4-t11'
from smkT import *
from smkT import call_raw
import html
XN='<script>alert(1)</script> "Nguyễn"'
res={}
def chk(label,m):
    if not m: res[label]=('KHÔNG CÓ THƯ',); return
    h=m.get('HTML') or ''; t=m.get('Text') or ''; subj=m.get('Subject','')
    res[label]=(subj,'<script>' in h, '&lt;script&gt;' in h, '&quot;' in h or '&#34;' in h, XN in t, 'Nguy' in subj or 'script' in subj)
# 1 verify_email + 2 email_exists
resetrl(); em=f'smkt.x25.{TAG}@edupilot.local'; t0=iso(); s,j,_=call('POST','/auth/register',None,{'email':em,'password':NPW,'full_name':XN})
print('register',s,j.get('code'),str(j.get('details'))[:100])
if s==202:
    m=mail_to(em,'Xác minh',t0,30); chk('verify_email',m); tk=link_token(m,'verify-email'); seen(tk); call('POST','/auth/verify-email',None,{'token':tk})
    t1=iso(); resetrl(); call('POST','/auth/register',None,{'email':em,'password':NPW,'full_name':XN}); chk('email_exists',mail_to(em,'đã có tài khoản',t1,30))
    t2=iso(); resetrl(); call('POST','/auth/forgot-password',None,{'email':em}); m=mail_to(em,'Đặt lại',t2,30); chk('reset_password',m); tk=link_token(m,'reset-password'); seen(tk)
    t3=iso(); resetrl(); s3,_,_=call('POST','/auth/reset-password',None,{'token':tk,'password':NPW+'q'}); chk('password_changed',mail_to(em,'đã được đổi',t3,30))
    t4=iso(); 
    for i in range(10): resetrl(); call_raw('POST','/auth/login',None,{'email':em,'password':'Sai-Mat-Khau-1'})
    chk('account_locked',mail_to(em,'khoá',t4,40))
# invite_staff với tên XSS
em2=f'smkt.x25gv.{TAG}@edupilot.local'; t5=iso(); s5,j5,_=call('POST','/admin/users',A(),{'email':em2,'full_name':XN,'role':'TEACHER'}); print('invite',s5,j5.get('code'))
if s5==201: m=mail_to(em2,'mời|Lời mời',t5,30); chk('invite_staff',m); tk=link_token(m,'invite'); seen(tk); call('POST','/auth/accept-invite',None,{'token':tk,'password':NPW}); gvt=lg(em2)[0]; gvid=j5['id']
# invite_student qua roster
if s5==201:
    cid=mkcourse('X',gvid); body=f'Email,Họ và tên,MSSV\r\nsmkt.x25sv.{TAG}@edupilot.local,"<script>alert(1)</script> ""Nguyễn""",20269901\r\n'.encode(); t6=iso()
    r=imp(cid,gvt,'x.csv',data=body); print('roster',r[0],str(r[1])[:120]); chk('invite_student',mail_to(f'smkt.x25sv.{TAG}@edupilot.local','thêm vào lớp|Bạn được thêm',t6,40))
bad=[k for k,v in res.items() if len(v)==1 or v[1] or not v[2] or v[5] or not v[4]]
rec('TC-P201-25','P' if res and not bad and len(res)>=6 else 'F',f'full_name={XN!r}: '+'; '.join(f'{k}: tiêu đề "{v[0]}"' + (f' | HTML có <script> thô={v[1]}, có &lt;script&gt;={v[2]}, thoát dấu "={v[3]}, Text giữ nguyên văn={v[4]}, tiêu đề chứa tên={v[5]}' if len(v)>1 else '') for k,v in res.items())+(f' LỆCH {bad}' if bad else ''))

import os; os.environ['SP']='4-t11b'
from smkT import *
from smkT import call_raw
XN='<script>alert(1)</script> "Nguyễn"'
def chk(m): 
    h=m.get('HTML') or ''; return (m['Subject'],'<script>' in h,'&lt;script&gt;' in h,'&quot;' in h or '&#34;' in h, XN in (m.get('Text') or ''),'Nguy' in m['Subject'])
em=f'smkt.x25b.{TAG}@edupilot.local'; resetrl(); t0=iso(); call('POST','/auth/register',None,{'email':em,'password':NPW,'full_name':XN}); m=mail_to(em,'Xác minh',t0,30); call('POST','/auth/verify-email',None,{'token':link_token(m,'verify-email')})
t2=iso(); resetrl(); call('POST','/auth/forgot-password',None,{'email':em}); m=mail_to(em,'Đặt lại',t2,30); tk=link_token(m,'reset-password')
t3=iso(); resetrl(); s3,j3,_=call('POST','/auth/reset-password',None,{'token':tk,'new_password':'Brand-New-Pw-2026!x'}); pc=mail_to(em,'mật khẩu.*đổi|đã được đổi',t3,40)
al=None; codes=[]
r={'password_changed':pc and chk(pc)}
bad=[k for k,v in r.items() if not v or v[1] or not v[2] or v[5] or not v[4]]
rec('TC-P201-25-b','P' if not bad else 'F',f'{r}')

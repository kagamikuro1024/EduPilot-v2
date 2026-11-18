import os,subprocess; exec(open(os.path.join(os.path.dirname(os.path.abspath(__file__)),'qc2i-r4apia-lib.py')).read())
OUT='/tmp/qc2-i/res/p4t12.tsv'
XN='<script>alert(1)</script> "Nguyễn"'
em=f'qc2ia-x25lock.{TAG}@edupilot.local'; resetrl(); t0=iso(); s,j,_=call('POST','/auth/register',None,{'email':em,'password':PWD,'full_name':XN}); m=mail_to(em,'Xác minh',t0,60); call('POST','/auth/verify-email',None,{'token':link_token(m,'verify-email')})
t4=iso(); n=0; end=time.time()+250
while time.time()<end and n<10:
    s,j,h=call('POST','/auth/login',None,{'email':em,'password':'Sai-Mat-Khau-Qc2ia-1'},retry=False)
    if s==401: n+=1
    elif s==429:
        ra=int(j.get('retry_after') or 1)
        if ra>=800: break
        time.sleep(ra+0.3)
m=mail_to(em,'khoá|khóa',t4,40)
h=m.get('HTML') if m else ''; tx=m.get('Text') if m else ''
ok=bool(m) and '<script>' not in h and '&lt;script&gt;' in h and ('&quot;' in h or '&#34;' in h) and XN in tx and 'Nguy' not in m['Subject']
rec('TC-P201-25-lock','P' if ok else 'F',f'account_locked sau {n} lần sai: tiêu đề {m and m["Subject"]!r}; HTML có <script> thô={"<script>" in (h or "")}, &lt;script&gt;={"&lt;script&gt;" in (h or "")}, Text nguyên văn={XN in (tx or "")}')

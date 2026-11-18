import os; exec(open(os.path.join(os.path.dirname(os.path.abspath(__file__)),'qc2i-r4apia-lib.py')).read())
em=reg('p05pend',verify=False); time.sleep(3); n0=mail_count(em); s1=[m['Subject'] for m in mails(em)]
resetrl(); s,j,_=call('POST','/auth/register',None,{'email':em,'password':PWD,'full_name':'dup'})
end=time.time()+40
while mail_count(em)==n0 and time.time()<end: time.sleep(1)
time.sleep(2); new=[m['Subject'] for m in sorted(mails(em),key=lambda m:m['Created'])][n0:]
us=psql(f"select count(*) from users where email='{em}'")
print(s1,s,new,us)
open('/tmp/smkT/p203-05b.txt','w').write(json.dumps([s1,s,new,us]))

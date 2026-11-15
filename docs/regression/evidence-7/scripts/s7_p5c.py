from lib import *
C=json.load(open('courses.json')); E1=C['E6C1']; Q=f'/courses/{E1}'
s,d=A('teacher','GET',f'/courses/{E1}'); st=d['settings']; v=st['version']; th=st['escalation_threshold']; print(st)
s1,d1=A('teacher','PUT',f'{Q}/settings',{'escalation_threshold':'0.85','version':v}); v2=d1['settings']['version'] if s1==200 else v+1
bad=[A('teacher','PUT',f'{Q}/settings',{'escalation_threshold':x,'version':v2})[0] for x in ('0.49','0.96','0.82','abc')]
rec('TC-P4-05-47',s1==200 and d1['settings']['escalation_threshold']=='0.85' and bad==[422,422,422,422],f'0.85 → {s1}; 0.49/0.96/0.82/abc → {bad}')
old=A('teacher','PUT',f'{Q}/settings',{'escalation_threshold':'0.80','version':v})[0]; r=[A(u,'PUT',f'{Q}/settings',{'escalation_threshold':'0.80','version':v2})[0] for u in ('ta','sv09','admin')]
rec('TC-P4-05-48',old==409 and all(x==403 for x in r),f'version cũ → {old}; TA/SV/ADMIN → {r}')
print(psql("select action from audit_log where action like '%settings%' or action like '%threshold%' order by created_at desc limit 2"))
A('teacher','PUT',f'{Q}/settings',{'escalation_threshold':th,'version':v2})

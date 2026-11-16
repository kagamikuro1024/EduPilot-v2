from smk3iso import *
V1='http://host.docker.internal:9751/v1'
T1=imkprov('SmkI-T1',V1,['m-ok','m-slow','m-slow'])
for t in ('CHAT','CLASSIFY','GRADING'): iput(t,[mid(T1,'m-slow')],{'retries':0},gw='gw3')
ipsql("delete from llm_audit where trace_id like 'ab%'")
def call(task,tid):
    ichat(task,gw='gw3',hdr=tp(tid))
ths=[]
def go(task,tid):
    t=threading.Thread(target=call,args=(task,tid)); t.start(); ths.append(t); time.sleep(0.06)
go('CHAT','abcc0'+'0'*27); time.sleep(0.4)
for lane,task in (('bb','GRADING'),('ee','CLASSIFY'),('11','CHAT')):
    for k in range(3): go(task,f'ab{lane}{k}'+'0'*27)
[t.join() for t in ths]; time.sleep(3)
rows=ipsql("select substr(trace_id,1,4),lane,queue_wait_ms,created_at from llm_audit where trace_id like 'ab%' order by created_at")
print(rows)

import sys
sys.argv=['x']
from lib3 import *
import fx as FXm
QA=FXm.QA; QQ=f'/courses/{QA}'
for u in ('sv06','teacher','admin'): login(u)
orig=fakeroute('qc3','Sprint7 fake-LLM bổ sung P4-03-24 (≤2 phút)')
try:
    fakeset(error_rate=0,latency_min_ms=0,latency_max_ms=30,stream_delay_ms=0)
    r1=FXm.ask('sv06','Quy định nộp bài muộn QCHIST của lớp là gì?'); sid=r1['sid']
    am=psql(f"select id from chat_messages where session_id='{sid}' and role='ASSISTANT' order by created_at desc limit 1")
    s,d=A('sv06','POST',f'/chat/messages/{am}/escalate'); tid=d['ticket']['id']
    A('teacher','POST',f'{QQ}/tickets/{tid}/claim'); sa,_=A('teacher','POST',f'{QQ}/tickets/{tid}/answer',{'body':'Chào em, QCSTAFF24 nộp muộn trừ 10 phần trăm mỗi ngày.'})
    sf,_=A('sv06','POST',f'/chat/tickets/{tid}/followup',{'content':'QCFOLLOW24 em chưa hiểu ý thầy','client_msg_id':str(uuid.uuid4())})
    r2=FXm.ask('sv06','Quy định về điểm danh của lớp là gì?',sid=sid)
    fl=FXm.flat(r2['payload'])
    gen=[i for i in r2['payload'] if i['task']!='EMBEDDING']
    R=(s,sa,sf,len(gen),'QCHIST' in fl,'QCSTAFF24' in fl,'QCFOLLOW24' in fl,r2['row'], psql(f"select count(*) from chat_messages where session_id='{sid}' and intent in ('STAFF_ANSWER','ESCALATION_FOLLOWUP')"))
finally:
    restoreroute('qc3',orig,'xong')
print(R)
rec('TC-P4-03-24',R[3]>=1 and R[4] and not R[5] and not R[6],f'provider=fake (CHAT→QC fake, cửa sổ khoá toàn cục): phiên có tin STAFF_ANSWER + ESCALATION_FOLLOWUP ({R[8]} tin intent đó); SV hỏi tiếp một câu thường → payload gửi LLM: {R[3]} lời gọi sinh văn bản, chứa câu hỏi trước của phiên (QCHIST)={R[4]}, chứa STAFF_ANSWER (QCSTAFF24)={R[5]}, chứa ESCALATION_FOLLOWUP (QCFOLLOW24)={R[6]}')

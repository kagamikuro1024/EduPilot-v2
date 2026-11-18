from lib3 import *
A_=C7['A']; Q=f'/courses/{A_}'
def info(am): return psql(f"select coalesce(intent,'-')||'/conf='||coalesce(confidence::text,'-')||'/low='||low_confidence||'/deg='||degraded||'/'||stream_status from chat_messages where id='{am}'")
s,d=A('qc3t','GET',f'/courses/{A_}'); v=d['settings']['version']
s1,d1=A('qc3t','PUT',f'{Q}/settings',{'escalation_threshold':0.95,'version':v}); print('threshold',s1,str(d1)[:120])
orig=fakeroute('qc3','Sprint7 P4-01 chat tự mở: CHAT→QC fake ≤5 phút')
try:
    fakeset(error_rate=0,latency_min_ms=0,latency_max_ms=30,stream_delay_ms=0)
    # 13 outbox lỗi
    psql(f"create or replace function qc3_fail() returns trigger as $$ begin if NEW.payload->>'course_id'='{A_}' then raise exception 'qc3 outbox lỗi'; end if; return NEW; end $$ language plpgsql")
    psql("create trigger qc3_t before insert on outbox for each row when (NEW.topic='escalation.created') execute function qc3_fail()")
    n0=psql(f"select count(*) from escalation_tickets where course_id='{A_}'")
    sid,am,st,ev,done=chat('qc3s15',A_,'Quy định về bảo lưu kết quả là gì?')
    n1=psql(f"select count(*) from escalation_tickets where course_id='{A_}'")
    psql("drop trigger qc3_t on outbox"); psql("drop function qc3_fail()")
    tr=psql("select count(*) from pg_trigger where tgname='qc3_t'")
    ss=psql(f"select stream_status||'/'||coalesce(error_code,'-') from chat_messages where id='{am}'")
    rec('TC-P4-01-13',n0==n1 and not (done and done.get('ticket')),f'outbox lỗi: SSE status={st} done={str(done)[:60]}; tin ASSISTANT={ss}; ticket {n0}→{n1} (không mồ côi); trigger gỡ ({tr} còn)')
    # 14 các intent
    qs=[('LIBRARY_SEARCH','qc3s16','Tìm giúp mình tài liệu về mạng máy tính trong thư viện lớp'),('PERSONAL','qc3s16','Điểm giữa kỳ của em là bao nhiêu?'),('WHAT_IF_GRADE','qc3s17','Nếu em được 8 điểm cuối kỳ thì tổng kết môn là bao nhiêu?'),('EXAM_SCHEDULE','qc3s17','Khi nào em thi cuối kỳ?'),('UPCOMING_EVENTS','qc3s18','Tuần này lớp có sự kiện hay hạn nộp gì không?')]
    rows=[]
    for name,u,q in qs:
        sid,am,st,ev,done=chat(u,A_,q); rows.append((name,info(am),tk(am) or 'không ticket')); print(rows[-1])
    ok=all(('low=t' in r[1])==(r[2]!='không ticket') for r in rows)
    rec('TC-P4-01-14',ok,'; '.join(f'{a}: {b} → {c}' for a,b,c in rows))
    # 15 nhánh không tự mở
    qs=[('CRISIS','Em không muốn sống nữa.'),('OTHER_PERSON','Bạn Nguyễn Văn A lớp mình được mấy điểm giữa kỳ?'),('SMALLTALK','chào bạn')]
    rows=[]
    for name,q in qs:
        sid,am,st,ev,done=chat('qc3s09',A_,q); rows.append((name,info(am),tk(am) or 'không ticket')); print(rows[-1])
    fakeset(error_rate=1.0,error_kind='SERVER')
    sid,am,st,ev,done=chat('qc3s09',A_,'Giải thích giúp mình khái niệm deadlock trong hệ điều hành'); rows.append(('degraded',info(am),tk(am) or 'không ticket')); print(rows[-1])
    fakeset(error_rate=0)
    # low_confidence=false: nhánh có công cụ conf=1
    ok15=all(r[2]=='không ticket' for r in rows)
    rec('TC-P4-01-15',ok15,'; '.join(f'{a}: {b} → {c}' for a,b,c in rows)+'; low_confidence=false: xem TC-14 (EXAM_SCHEDULE/PERSONAL không ticket nếu conf=1)')
    # 16 AUTO_NO_FORMULA
    sid,am,st,ev,done=chat('qc3s16',A_,'Điểm cuối kỳ tính thế nào?')
    tkk=psql(f"select origin from escalation_tickets where message_id='{am}'"); print(info(am),tkk,psql(f"select left(content,100) from chat_messages where id='{am}'"))
    rec('TC-P4-01-16',tkk=='AUTO_NO_FORMULA',f'{info(am)}; ticket origin={tkk or "KHÔNG"}; câu: {psql(f"select left(content,70) from chat_messages where id=chr(39)||0||chr(39)") if False else ""}')
    # 26 trần: s03 đã có 5 ticket mở
    cnt0=psql(f"select count(*) from escalation_tickets where student_id='{uid('qc3s03')}' and course_id='{A_}' and status in ('OPEN','CLAIMED')")
    lg0=subprocess.run("docker logs ep-r-gateway-1 2>&1 | grep -c suppressed",shell=True,capture_output=True,text=True).stdout.strip()
    sid,am,st,ev,done=chat('qc3s03',A_,'Cách tính học phí học lại là gì?')
    time.sleep(2)
    lg1=subprocess.run("docker logs ep-r-gateway-1 2>&1 | grep -c suppressed",shell=True,capture_output=True,text=True).stdout.strip()
    content=psql(f"select content from chat_messages where id='{am}'")
    rec('TC-P4-01-26',cnt0=='5' and not tk(am) and not (done and done.get('ticket')) and done and done.get('low_confidence'),f'trước: {cnt0} ticket mở; câu yếu low_confidence={done.get("low_confidence") if done else None}, ticket={done.get("ticket") if done else None}; log "suppressed" {lg0}→{lg1}; câu: {content[:70]}')
    # E3 canary
    sid,am,st,ev,done=chat('qc3s19',A_,'CANARY-9K3Z em muốn hỏi quy định về học bổng')
    s,d=esc('qc3s19',am); time.sleep(3)
    lg=subprocess.run("docker logs ep-r-gateway-1 2>&1 | grep -c CANARY-9K3Z; docker logs ep-r-worker-1 2>&1 | grep -c CANARY-9K3Z",shell=True,capture_output=True,text=True).stdout.split()
    rec('TC-P4-01-E3',lg==['0','0'],f'tin chứa CANARY-9K3Z đã gửi (st={st}); escalate {s}; grep log gateway/worker = {lg}')
finally:
    fakeset(error_rate=0)
    print(restoreroute('qc3',orig,'P4-01 chat tự mở xong'))
    A('qc3t','PUT',f'{Q}/settings',{'escalation_threshold':0.8,'version':A('qc3t','GET',f'/courses/{A_}')[1]['settings']['version']})
save('p401c')

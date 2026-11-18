from w7lib import *
import fx
fx.use('QC3S6C','QKTWQ3C'); Q=fx.QA
def log(m): open('/tmp/qc2-global-changes.log','a').write(time.strftime('%H:%M:%S')+' qc3: '+m+'\n')
tk=login('sv.gioi'); sid=newsess('sv.gioi',Q)
log('05-26/06-43: đổi tên tạm bảng exam_attempts ≤4 s rồi đổi lại')
res={}
try:
    psql("alter table exam_attempts rename to exam_attempts_qc3tmp"); time.sleep(0.3)
    s,t,h=raw('POST',f'/chat/sessions/{sid}/messages',tk,{'content':'hi'},timeout=20); res['chat']=(s,t[:300])
    s2,t2,h2=raw('POST',f'/courses/{Q}/threads',tk,{'title':'z','body':'AES'},timeout=20); res['thread']=(s2,t2[:200])
finally:
    psql("alter table exam_attempts_qc3tmp rename to exam_attempts"); log('exam_attempts đã đổi lại tên')
print(res, psql("select to_regclass('exam_attempts')"))
c=json.loads(res['chat'][1]) if res['chat'][1].startswith('{') else {}; c2=json.loads(res['thread'][1]) if res['thread'][1].startswith('{') else {}
ft=psql(f"select count(*) from forum_threads where title='z' and course_id='{Q}'")
rec('TC-P3-05-26',res['chat'][0]==503 and c.get('code')=='CHAT_UNAVAILABLE',f'IsLocked không tra được (Redis không có khoá của người này; bảng exam_attempts bị đổi tên tạm ~1 s → lỗi DB): gửi tin → HTTP {res["chat"][0]} {c.get("code")} retry_after={c.get("retry_after")}; thân={res["chat"][1][:120]}; (UI "Chat chưa dùng được lúc này…" ở nhóm UI); go test 5 ca locker (chat) đạt')
rec('TC-P3-06-43',res['thread'][0]==503 and c2.get('code')=='CHAT_UNAVAILABLE' and ft=='0',f'cùng điều kiện: đăng thread → HTTP {res["thread"][0]} {c2.get("code")}; forum_threads mới={ft}; go test 5 ca locker (thread) đạt')

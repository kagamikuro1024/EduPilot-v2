from w7lib import *
import fx
fx.use('QC3S6C','QKTWQ3C'); Q=fx.QA
def log(m): open('/tmp/qc2-global-changes.log','a').write(time.strftime('%H:%M:%S')+' qc3: '+m+'\n')
sh=lambda c: subprocess.run(c,shell=True,capture_output=True,text=True)
sid=newsess('sv.gioi',Q); tk=login('sv.gioi'); uid=psql("select id from users where email='sv.gioi@edupilot.local'")
log('05-26/06-43: khoá ACCESS EXCLUSIVE exam_attempts 22 s')
holder=subprocess.Popen(['docker','exec','-i','ep-r-postgres-1','psql','-U','edupilot','-d','edupilot','-c',"begin; lock table exam_attempts in access exclusive mode; select pg_sleep(22); rollback;"],stdout=subprocess.PIPE,stderr=subprocess.PIPE)
time.sleep(4)
try:
    t0=time.time(); s,t,h=raw('POST',f'/chat/sessions/{sid}/messages',tk,{'content':'hi'}); d1=time.time()-t0
    c1=json.loads(t) if t.startswith('{') else {}
    t0=time.time(); s2,t2,h2=raw('POST',f'/courses/{Q}/threads',tk,{'title':'x','body':'AES'}); d2=time.time()-t0
    c2=json.loads(t2) if t2.startswith('{') else {}
    ft=psql(f"select count(*) from forum_threads where title='x' and course_id='{Q}'")
finally:
    holder.wait(); time.sleep(2); log('khoá exam_attempts đã nhả (05-26/06-43 xong)')
rec('TC-P3-05-26',s==503 and c1.get('code')=='CHAT_UNAVAILABLE',f'Khoá bảng exam_attempts (Redis không có khoá của người này → tra DB lỗi; khi dừng cả Redis thì lớp bảo vệ chung trả 503 OVERLOADED/SERVICE_UNAVAILABLE trước khi tới bước kiểm khoá — đã thử): gửi tin → HTTP {s} {c1.get("code")} retry_after={c1.get("retry_after")} sau {d1:.1f}s (UI chữ "Chat chưa dùng được lúc này. Thử lại sau ít phút." ở nhóm UI); go test TestLockerErrorDeniesChat (lượt dưới)')
rec('TC-P3-06-43',s2==503 and c2.get('code')=='CHAT_UNAVAILABLE' and ft=='0',f'cùng điều kiện (chỉ khoá bảng): đăng thread → HTTP {s2} {c2.get("code")} sau {d2:.1f}s; forum_threads mới={ft}')
g=subprocess.run(GO+"go test -tags integration ./internal/chat ./internal/thread -run 'TestChatLockedDuringExam|TestLockCheckBeforeAnything|TestLockedWritesChatBlockedEvent|TestLockerErrorDeniesChat|TestUnlockedAfterSubmit|TestThreadCreateLockedDuringExam|TestThreadLockCheckBeforeFirewall|TestThreadLockerErrorDenies|TestThreadReadAllowedWhileLocked|TestThreadUnlockedAfterSubmit' -count=1 2>&1 | tail -4",shell=True,capture_output=True,text=True).stdout
print(g)
print(c1,c2)
json.dump(R,open('/tmp/qc3e/w22.json','w'),ensure_ascii=False)

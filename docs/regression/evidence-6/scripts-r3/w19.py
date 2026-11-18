from w7lib import *
import fx,threading
fx.use('QC3S6C','QKTWQ3C'); Q=fx.QA
uid=psql("select id from users where email='sv.gioi@edupilot.local'")
# ===== 02-50: bước che vượt hạn → MASK_FAILED
QM='Bùi Thanh Khải học nhóm với em và hỏi về mật mã'; ask('sv.gioi',QM,course=Q)
sid=newsess('sv.gioi',Q); tk=login('sv.gioi')
rds('del','ep:roster:'+Q); 
# xoá luôn cache roster mọi lớp để buộc nạp DB
subprocess.run(['docker','exec','ep-r-redis-1','sh','-c',"redis-cli --scan --pattern 'ep:roster:*' | xargs -r redis-cli del; redis-cli --scan --pattern 'ep:ans:*' | xargs -r redis-cli del"],capture_output=True)
holder=subprocess.Popen(['docker','exec','-i','ep-r-postgres-1','psql','-U','edupilot','-d','edupilot','-c',"begin; lock table enrollments in access exclusive mode; select pg_sleep(14); rollback;"],stdout=subprocess.PIPE,stderr=subprocess.PIPE)
time.sleep(1.5)
n0=int(rds('exists','x') or 0)
fk0=len(A('admin','GET','/_test/llm/payloads')[1]['items']); reset()
st,ev=sse('POST',f'/chat/sessions/{sid}/messages',tk,{'content':QM},maxsec=25)
its=pay(); holder.wait()
errf=[json.loads(d) for e,d,*_ in ev if e=='error']
aud=psql(f"select status||'|'||coalesce(error_kind,'') from llm_audit where user_id='{uid}' order by created_at desc limit 1")
um=psql(f"select content from chat_messages where session_id='{sid}' and role='USER'")
asst=psql(f"select stream_status||'|'||coalesce(error_code,'') from chat_messages where session_id='{sid}' and role='ASSISTANT'")
rec('TC-P3-02-50',len(nogen(its))==0 and errf and 'MASK_FAILED' in aud+json.dumps(errf)+asst,f'khoá ACCESS EXCLUSIVE bảng enrollments 14 s + xoá cache roster → bước che không nạp được trong hạn: lời gọi provider={len(nogen(its))} (0); khung error={errf}; hàng tin={asst}; llm_audit mới nhất (status|error_kind)={aud}; tin USER vẫn lưu={um[:30]!r}')
json.dump(R,open('/tmp/qc3e/w19.json','w'),ensure_ascii=False)

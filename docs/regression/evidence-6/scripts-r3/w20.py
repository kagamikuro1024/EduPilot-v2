from w7lib import *
import fx
fx.use('QC3S6C','QKTWQ3C'); Q=fx.QA
src="source /Users/kuro/Documents/TA_Agent_qc2/docs/regression/env/ep-r-env.sh; "
def dc(args,extra=''): 
    r=subprocess.run(['bash','-c',src+f'{os.environ.get("X","")} $EPR_C {extra} {args}'],capture_output=True,text=True); return r.returncode,(r.stdout+r.stderr)[-300:]
def ready(t=90):
    for i in range(t):
        r=subprocess.run(['curl','-sk','https://localhost:1143/readyz','-o','/dev/null','-w','%{http_code}'],capture_output=True,text=True).stdout
        if r=='200': return i
        time.sleep(1)
    return None
def log(m): open('/tmp/qc2-global-changes.log','a').write(time.strftime('%H:%M:%S')+' qc3: '+m+'\n')
QM='Bùi Thanh Khải học nhóm với em và hỏi về mật mã'; ask('sv.gioi',QM,course=Q)
open('/tmp/qc3-mask.override.yml','w').write('services:\n  gateway:\n    environment:\n      PRIVACY_MASK_TIMEOUT_MS: "1"\n')
log('gateway recreate với PRIVACY_MASK_TIMEOUT_MS=1 (02-50); hoàn lại ≤3 phút')
try:
    print(dc('up -d --no-deps gateway','-f /tmp/qc3-mask.override.yml'))
    time.sleep(12); print('ready',ready()); time.sleep(3)
    subprocess.run(['docker','exec','ep-r-redis-1','sh','-c',"redis-cli --scan --pattern 'ep:roster:*' | xargs -r redis-cli del; redis-cli --scan --pattern 'ep:ans:*' | xargs -r redis-cli del"],capture_output=True)
    uid=psql("select id from users where email='sv.gioi@edupilot.local'"); sid=newsess('sv.gioi',Q); tk=login('sv.gioi'); reset()
    st,ev=sse('POST',f'/chat/sessions/{sid}/messages',tk,{'content':QM},maxsec=30)
    its=pay(); errf=[json.loads(d) for e,d,*_ in ev if e=='error']
    time.sleep(1.5); aud=psql(f"select status||'|'||coalesce(error_kind,'') from llm_audit where user_id='{uid}' order by created_at desc limit 1")
    asst=psql(f"select stream_status||'|'||coalesce(error_code,'') from chat_messages where session_id='{sid}' and role='ASSISTANT'")
    um=psql(f"select count(*) from chat_messages where session_id='{sid}' and role='USER'")
    rec('TC-P3-02-50',len(nogen(its))==0 and 'MASK_FAILED' in aud+asst,f'PRIVACY_MASK_TIMEOUT_MS=1 (cách QC hạ thấp để ép lỗi, theo config.go): lời gọi provider={len(nogen(its))} (0); khung error={errf}; hàng tin={asst}; llm_audit mới nhất={aud}; tin USER vẫn lưu={um}')
finally:
    print(dc('up -d --no-deps gateway'))
    time.sleep(12); print('ready after restore',ready())
    log('gateway đã hoàn về cấu hình gốc (02-50 xong)')
json.dump(R,open('/tmp/qc3e/w20.json','w'),ensure_ascii=False)

from w7lib import *
import fx,threading
fx.use('QC3S6C','QKTWQ3C'); Q=fx.QA
def log(m): open('/tmp/qc2-global-changes.log','a').write(time.strftime('%H:%M:%S')+' qc3: '+m+'\n')
def sh(c): return subprocess.run(c,shell=True,capture_output=True,text=True)
def ready(t=90):
    for i in range(t):
        if sh("curl -sk https://localhost:1143/readyz -o /dev/null -w '%{http_code}'").stdout=='200': return i
        time.sleep(1)
uid=psql("select id from users where email='sv.gioi@edupilot.local'")
q05='Quy chế bảo mật mật khẩu nói gì về độ dài tối thiểu của mật khẩu'
ask('sv.gioi',q05,course=Q,title='Bài ghi chú của Vũ Hoàng Giang')
def start_stream(delay=700):
    sh("docker exec ep-r-redis-1 sh -c \"redis-cli --scan --pattern 'ep:ans:*' | xargs -r redis-cli del\"")
    A('admin','POST','/_test/llm/fake',{'stream_delay_ms':delay})
    sid=newsess('sv.gioi',Q); tk=login('sv.gioi'); res={}
    th=threading.Thread(target=lambda: res.setdefault('r',sse('POST',f'/chat/sessions/{sid}/messages',tk,{'content':q05},maxsec=40)),daemon=True); th.start()
    time.sleep(3.2); return sid,th
try:
    # ===== 05-50 (a): tắt có chủ đích (SIGTERM → drain)
    log('05-50: docker stop/kill gateway giữa lúc stream (≤4 phút), sẽ start lại')
    sidA,thA=start_stream(); mA=psql(f"select id from chat_messages where session_id='{sidA}' and role='ASSISTANT'")
    pre=psql(f"select stream_status||'|'||length(coalesce(partial_content,'')) from chat_messages where id='{mA}'")
    t0=time.time(); sh('docker stop -t 30 ep-r-gateway-1'); dt=time.time()-t0
    rowA=psql(f"select stream_status||'|'||coalesce(error_code,'')||'|'||(partial_content is not null)||'|'||length(coalesce(partial_content,'')) from chat_messages where id='{mA}'")
    busyA=rds('exists','ep:chat:active:'+uid)
    sh('docker start ep-r-gateway-1'); time.sleep(10); print('ready',ready())
    rec('TC-P3-05-50-a',rowA.startswith('FAILED|INTERRUPTED|true') and busyA=='0' and dt<31,f'tắt có chủ đích (docker stop → SIGTERM, drain) khi đang STREAMING ({pre}): sau {dt:.1f}s hàng tin={rowA} (FAILED/INTERRUPTED, partial giữ nguyên), khoá ep:chat:active nhả (exists={busyA}) — không chờ 150 s')
    # ===== (b) SIGKILL
    A('admin','POST','/_test/llm/fake',{'stream_delay_ms':700}); 
    sidB,thB=start_stream(); mB=psql(f"select id from chat_messages where session_id='{sidB}' and role='ASSISTANT'")
    sh('docker kill ep-r-gateway-1'); sh('docker start ep-r-gateway-1'); time.sleep(10); print('ready',ready())
    t0=time.time(); st=None
    for i in range(60):
        st=psql(f"select stream_status||'|'||coalesce(error_code,'')||'|'||(partial_content is not null) from chat_messages where id='{mB}'")
        if st.startswith('FAILED'): break
        time.sleep(5)
    dtB=time.time()-t0
    rec('TC-P3-05-50-b',st.startswith('FAILED|INTERRUPTED|true') and 100<dtB+10<=260,f'SIGKILL gateway giữa lượt STREAMING: sau khi gateway chạy lại, reaper đặt {st} sau {dtB+10:.0f}s (kỳ vọng >150 s kể từ lần ghi cuối; partial_content giữ nguyên)')
finally:
    sh('docker start ep-r-gateway-1'); ready()
    log('gateway đã chạy lại (05-50 xong)')
json.dump(R,open('/tmp/qc3e/w21.json','w'),ensure_ascii=False)

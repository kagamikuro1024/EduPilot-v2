from lib3 import *
A_=C7['A']; u='qc3s28'; login(u)
sid=A(u,'POST','/chat/sessions',{'course_id':A_,'title':'qc3'})[1]['id']
assert lockg('qc3','Sprint7 Redis chết ≤30 s (TC-P4-07-E2 câu khủng hoảng)')
R={}
try:
    subprocess.run(['docker','stop','ep-r-redis-1'],capture_output=True); time.sleep(2)
    st,ev=sse('POST',f'/chat/sessions/{sid}/messages',login(u),{'content':'Em không muốn sống nữa.'},maxsec=40)
    R['st']=st; R['ev']=[(e[0],(e[1] or '')[:120]) for e in ev][:6]
    # Threads: gửi câu dương
    R['thr']=A(u,'POST',f'/courses/{A_}/threads',{'title':'Em mệt quá QCE2','body':'Em không muốn sống nữa, em muốn chết.','tags':[]})
finally:
    subprocess.run(['docker','start','ep-r-redis-1'],capture_output=True)
    for _ in range(60):
        if subprocess.run("curl -sk https://localhost:1143/api/v1/readyz -o /dev/null -w '%{http_code}'",shell=True,capture_output=True,text=True).stdout=='200': break
        time.sleep(2)
    unlockg('qc3','Redis đã bật lại')
R['row']=psql(f"select intent||'/'||stream_status||'/'||left(content,70) from chat_messages where session_id='{sid}' and role='ASSISTANT' order by created_at desc limit 1")
print(R)
ok=R['st']==200 and R['row'].startswith('CRISIS/DONE/') and R['thr'][0]==422 and R['thr'][1].get('code')=='CRISIS_SUPPORT'
rec('TC-P4-07-E2',ok,f'Redis dừng ~15 s: chat gửi «Em không muốn sống nữa.» → HTTP {R["st"]}, tin ASSISTANT {R["row"]} (câu mẫu, không phụ thuộc Redis/LLM); Threads đăng bài khủng hoảng → {R["thr"][0]} {R["thr"][1].get("code")} (hộp hỗ trợ vẫn hiện, không mất lối hỗ trợ)')

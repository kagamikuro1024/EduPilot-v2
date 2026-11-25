import m, json, subprocess, time
def sql(q): return subprocess.run(['ssh','edupilot-vps',f'docker exec edupilot-postgres-1 psql -U edupilot -d edupilot -At -F"|" -c "{q}"'],capture_output=True,text=True).stdout.strip()
def rows_now(cid,sid): return sql(f"select count(*) from attendance_checkins c join users u on u.id=c.student_id where c.session_id='{sid}' and u.email='sv.qc2@edupilot.local'")
k=m.vps.course("QCMOB1"); cid=k["id"]
sid=sql(f"select s.id from class_sessions s where s.course_id='{cid}' and not exists (select 1 from checkin_windows w where w.session_id=s.id) order by s.starts_at limit 1").strip()
if not sid:
    import datetime; t=datetime.date.today()
    m.vps.call("teacher","POST",f"/courses/{cid}/sessions/generate",{"weekdays":[1,2,3,4,5,6,7],"start_time":"00:05","end_time":"23:50","from":str(t+datetime.timedelta(days=40)),"to":str(t+datetime.timedelta(days=70))})
    sid=sql(f"select s.id from class_sessions s where s.course_id='{cid}' and not exists (select 1 from checkin_windows w where w.session_id=s.id) order by s.starts_at limit 1").strip()
c,w=m.vps.call("teacher","POST",f"/courses/{cid}/sessions/{sid}/checkin-windows",{"minutes":30}); wid=w.get("details",{}).get("window_id") or w.get("id")
c,d=m.vps.call("teacher","GET",f"/courses/{cid}/sessions/{sid}/checkin-windows/{wid}/display")
url=d["qr_url"].replace("https://localhost",m.vps.HOST,1)
print("SID",sid[:8],"code",d["short_code"])
# mở link QR (đúng luồng) → điểm danh tự gửi
print(m.cdp(f"(()=>{{location.href={json.dumps(url)}; return 1}})()")); time.sleep(6)
t1=m.cdp("document.body.innerText.slice(0,120).replace(/\\n/g,' / ')")
n1=rows_now(cid,sid)
# NAV-02: back sau khi đã điểm danh
m.cdp("history.back()"); time.sleep(3)
after_back_path=m.cdp("location.pathname+' hash='+(location.hash?'yes':'no')")
hash_stash=m.cdp("JSON.stringify(Object.keys(sessionStorage))")
time.sleep(2); n2=rows_now(cid,sid)
# NAV-10: forward lại
m.cdp("history.forward()"); time.sleep(3)
fwd=m.cdp("document.body.innerText.slice(0,100).replace(/\\n/g,' / ')")
n3=rows_now(cid,sid)
print(json.dumps({"t1":t1,"n_after_submit":n1,"path_after_back":after_back_path,"storage":hash_stash,"n_after_back":n2,"fwd":fwd,"n_after_fwd":n3},ensure_ascii=False))

import m, json, subprocess, time
def sql(q): return subprocess.run(['ssh','edupilot-vps',f'docker exec edupilot-postgres-1 psql -U edupilot -d edupilot -At -F"|" -c "{q}"'],capture_output=True,text=True).stdout.strip()
def to_checkin():
    m.cdp("(()=>{const b=[...document.querySelectorAll('button')].find(x=>x.innerText.trim()==='Thêm'); b&&b.click(); return 1})()"); time.sleep(1.5)
    m.cdp("(()=>{const a=[...document.querySelectorAll('a,button')].find(x=>x.innerText.trim()==='Điểm danh'); if(!a) return 0; a.click(); return 1})()"); time.sleep(2.5)
def cnt(cid,sid): return sql(f"select count(*) from attendance_checkins c join users u on u.id=c.student_id where c.session_id='{sid}' and u.email='sv.qc2@edupilot.local'")
k=m.vps.course("QCMOB1"); cid=k["id"]
sid=sql(f"select s.id from class_sessions s where s.course_id='{cid}' and not exists (select 1 from checkin_windows w where w.session_id=s.id) order by s.starts_at limit 1").strip()
if not sid:
    import datetime; t=datetime.date.today()
    m.vps.call("teacher","POST",f"/courses/{cid}/sessions/generate",{"weekdays":[1,2,3,4,5,6,7],"start_time":"00:05","end_time":"23:50","from":str(t+datetime.timedelta(days=40)),"to":str(t+datetime.timedelta(days=70))})
    sid=sql(f"select s.id from class_sessions s where s.course_id='{cid}' and not exists (select 1 from checkin_windows w where w.session_id=s.id) order by s.starts_at limit 1").strip()
c,w=m.vps.call("teacher","POST",f"/courses/{cid}/sessions/{sid}/checkin-windows",{"minutes":30}); wid=w.get("details",{}).get("window_id") or w.get("id")
c,d=m.vps.call("teacher","GET",f"/courses/{cid}/sessions/{sid}/checkin-windows/{wid}/display")
code=d["short_code"]; print("code",code,"sid",sid[:8])
to_checkin()
print("path", m.cdp("location.pathname"))
before=cnt(cid,sid)
print("submit", m.cdp(f"(()=>{{const i=document.querySelector('input'); if(!i) return 'noinput'; const s=Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set; s.call(i,{json.dumps(code)}); i.dispatchEvent(new Event('input',{{bubbles:true}})); const b=[...document.querySelectorAll('button')].find(x=>x.innerText.trim()==='Điểm danh'&&!x.disabled); b&&b.click(); return 'ok'}})()"))
time.sleep(4)
txt=m.cdp("document.body.innerText.slice(0,120).replace(/\\n/g,' / ')")
n1=cnt(cid,sid)
# NAV-02
m.cdp("history.back()"); time.sleep(3)
back=m.cdp("location.pathname+' hash='+(location.hash?'yes':'no')")
stash=m.cdp("JSON.stringify(Object.keys(sessionStorage))")
n2=cnt(cid,sid)
# NAV-10
m.cdp("history.forward()"); time.sleep(3)
fwd=m.cdp("document.body.innerText.slice(0,90).replace(/\\n/g,' / ')")
n3=cnt(cid,sid)
print(json.dumps({"txt":txt,"n1":n1,"back":back,"stash":stash,"n2":n2,"fwd":fwd,"n3":n3},ensure_ascii=False))
m.shot("NAV-02-back-after-checkin")

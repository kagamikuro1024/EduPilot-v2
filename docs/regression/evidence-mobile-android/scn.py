import m, time, json, subprocess, segno, re
from PIL import Image
CID="01a21371-7998-75d8-bc50-58a8e2a3ff0b"
POST="/Users/kuro/.android/avd/ep-api36-ps.avd/poster.png"
def sql(q): return subprocess.run(['ssh','edupilot-vps',f'docker exec edupilot-postgres-1 psql -U edupilot -d edupilot -At -F"|" -c "{q}"'],capture_output=True,text=True).stdout.strip()
def policy(rot=60, geo=None):
    c,det=m.vps.call('teacher','GET',f"/courses/{CID}")
    att={"self_checkin":True,"require":"WEB","rotate_seconds":rot,"late_after_min":15}
    if geo: att["geofence"]=geo
    m.vps.call('teacher','PUT',f"/courses/{CID}/settings",{"version":det['settings']['version'],"attendance":att})
def newwin():
    sid=sql(f"select s.id from class_sessions s where s.course_id='{CID}' and not exists (select 1 from checkin_windows w where w.session_id=s.id) order by s.starts_at limit 1")
    c,w=m.vps.call('teacher','POST',f"/courses/{CID}/sessions/{sid}/checkin-windows",{"minutes":30})
    wid=w.get('id') or w['details']['window_id']
    c,d=m.vps.call('teacher','GET',f"/courses/{CID}/sessions/{sid}/checkin-windows/{wid}/display")
    return dict(cid=CID,sid=sid,wid=wid,d=d)
def poster(url, path=POST):
    segno.make(url,error='l').save('/tmp/_q.png',scale=14,border=4)
    im=Image.open('/tmp/_q.png').convert('RGB'); c=Image.new('RGB',(1024,1024),'white'); c.paste(im.resize((720,720),Image.NEAREST),(152,228)); c.save(path)
    m.adb(f"emu virtualscene-image wall {path}")
def poster_pair(urlA,urlB,path="/Users/kuro/.android/avd/ep-api36-ps.avd/poster2.png"):
    segno.make(urlA,error='l').save('/tmp/_a.png',scale=10,border=2); segno.make(urlB,error='l').save('/tmp/_b.png',scale=10,border=2)
    a=Image.open('/tmp/_a.png').convert('RGB').resize((420,420),Image.NEAREST); b=Image.open('/tmp/_b.png').convert('RGB').resize((420,420),Image.NEAREST)
    c=Image.new('RGB',(1024,1024),'white'); c.paste(b,(60,300)); c.paste(a,(540,300)); c.save(path)
    m.adb(f"emu virtualscene-image wall {path}")
def txt(): return m.cdp("document.body.innerText.replace(/\\n/g,' / ')")
def click(label):
    return m.cdp("(()=>{const b=[...document.querySelectorAll('button')].find(x=>x.innerText.trim()==="+json.dumps(label)+"); if(!b) return 'nobtn'; b.click(); return 'ok'})()")
def rows(w): return [x for x in (m.vps.checkins(w['cid'],w['sid']) or {}).get('items',[]) if x.get('window_id')==w['wid']]
def camclients(): 
    d=m.sh("dumpsys media.camera | grep -cE 'Client priority|Camera ID'"); return d
def goto_checkin():
    m.cdp(f"location.href='{m.vps.HOST}/checkin'"); time.sleep(3)

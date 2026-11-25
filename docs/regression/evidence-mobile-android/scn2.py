import m, scn, time, datetime, json
def rem(d):
    a=datetime.datetime.fromisoformat(d['step_expires_at'].replace('Z','+00:00'))
    b=datetime.datetime.fromisoformat(d['server_time'].replace('Z','+00:00'))
    return (a-b).total_seconds()
def refresh(w):
    c,d=m.vps.call('teacher','GET',f"/courses/{w['cid']}/sessions/{w['sid']}/checkin-windows/{w['wid']}/display")
    w['d']=d; return d
def fresh(min_rem=30):
    w=scn.newwin()
    for i in range(6):
        d=refresh(w)
        if rem(d)>=min_rem: return w
        time.sleep(1)
    return w
def entry():
    scn.goto_checkin(); time.sleep(2.5)
    if not m.cdp("!!document.querySelector('input')"):
        m.cdp("location.reload()"); time.sleep(3)
    return m.cdp("!!document.querySelector('input')")
def scan():
    m.cdp("(()=>{const b=[...document.querySelectorAll('button')].find(x=>x.innerText.trim()==='Quét mã điểm danh'); if(!b) return 'nobtn'; b.click(); return 'ok'})()")
    for i in range(20):
        if m.cdp("(()=>{const v=document.querySelector('video'); return !!(v&&v.readyState>=2)})()"): return True
        time.sleep(0.3)
    return False
def wait_result(w, secs=20):
    t0=time.time(); t=""
    while time.time()-t0<secs:
        t=scn.txt()
        if len(scn.rows(w))>0: return "done", t, time.time()-t0
        if "hết hạn" in t: return "expired", t, time.time()-t0
        if "không thuộc lớp" in t: return "otherclass", t, time.time()-t0
        if "Đã đóng" in t: return "closed", t, time.time()-t0
        if "Không mở được camera" in t: return "cameraerr", t, time.time()-t0
        time.sleep(0.25)
    return "timeout", t, time.time()-t0

def poster_url(url, tag):
    import segno, time
    from PIL import Image
    p=f"/tmp/qr-{tag}-{int(time.time()*1000)}.png"
    segno.make(url,error='l').save('/tmp/_q.png',scale=14,border=4)
    im=Image.open('/tmp/_q.png').convert('RGB'); c=Image.new('RGB',(1024,1024),'white'); c.paste(im.resize((720,720),Image.NEAREST),(152,228)); c.save(p)
    m.adb(f"emu virtualscene-image wall {p}"); return p
def poster_two(urlA,urlB,tag="two"):
    import segno, time
    from PIL import Image
    p=f"/tmp/qr2-{tag}-{int(time.time()*1000)}.png"
    segno.make(urlA,error='l').save('/tmp/_a.png',scale=10,border=2); segno.make(urlB,error='l').save('/tmp/_b.png',scale=10,border=2)
    a=Image.open('/tmp/_a.png').convert('RGB').resize((420,420),Image.NEAREST); b=Image.open('/tmp/_b.png').convert('RGB').resize((420,420),Image.NEAREST)
    c=Image.new('RGB',(1024,1024),'white'); c.paste(b,(40,300)); c.paste(a,(560,300)); c.save(p)
    m.adb(f"emu virtualscene-image wall {p}"); return p

def fresh_login(user='sv.qc2'):
    import subprocess, os, m as _m
    subprocess.run("adb shell pm clear vn.edupilot.app",shell=True,capture_output=True)
    _m.start_app(); time.sleep(2)
    os.environ["PW_"]=_m.acct(user); _m.login(user)

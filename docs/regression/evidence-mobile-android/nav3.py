import m, json, time
def navclick(txt):
    return m.cdp(f"(()=>{{const a=[...document.querySelectorAll('a,button')].find(x=>x.innerText.trim()==={json.dumps(txt)}); if(!a) return 0; a.click(); return 1}})()")
out={}
# NAV-11: ngăn xếp Hôm nay → Lịch → Điểm danh → Back×3
print(m.cdp("location.pathname"))
out["h"]=navclick("Hôm nay"); time.sleep(2)
name = "Lịch"
out["j"]=m.cdp("(()=>{const b=[...document.querySelectorAll('button')].find(x=>x.innerText.trim()==='Thêm'); b&&b.click(); return 1})()"); time.sleep(1.2)
out["l"]=navclick(name); time.sleep(2.5); out["p1"]=m.cdp("location.pathname")
m.cdp("(()=>{const b=[...document.querySelectorAll('button')].find(x=>x.innerText.trim()==='Thêm'); b&&b.click(); return 1})()"); time.sleep(1.2)
out["d"]=navclick("Điểm danh"); time.sleep(2.5); out["p2"]=m.cdp("location.pathname")
seq=[]
for i in range(3):
    m.sh("input keyevent KEYCODE_BACK"); time.sleep(1.5); seq.append(m.cdp("location.pathname"))
out["back_seq"]=seq
out["focus"]=m.sh("dumpsys window | grep -o 'mCurrentFocus=.*'")
m.shot("NAV-11-back-stack")
print(json.dumps(out,ensure_ascii=False)[:600])

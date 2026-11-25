# Các probe dùng ở lượt 6 (26/11). Chạy: python3 probes2.py <ten>
# cl14qcmob1: mã sai ở QCMOB1 -> chữ lỗi; lif03: khoá/mở màn; lif05: am kill; saf04: ngang+cutout
import m, json, time, sys
def cl14():
    m.nav("/checkin", 2200)
    m.cdp("(()=>{const i=document.querySelector('input');const s=Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set;s.call(i,'ZZZ999');i.dispatchEvent(new Event('input',{bubbles:true}));const b=[...document.querySelectorAll('button')].find(x=>x.innerText.trim()==='Điểm danh'&&!x.disabled);b&&b.click();return 1})()")
    time.sleep(4); print(m.cdp("document.body.innerText.slice(0,200)"))
def lif03():
    m.cdp("(()=>{const i=document.querySelector('input');i.focus();const s=Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set;s.call(i,'ABC12');i.dispatchEvent(new Event('input',{bubbles:true}));return 1})()")
    m.sh("input keyevent 26"); time.sleep(3); m.sh("input keyevent 26"); time.sleep(1); m.sh("input swipe 540 1800 540 600 250"); time.sleep(2)
    print(m.cdp("(document.querySelector('input')||{}).value"))
def saf04():
    m.sh("cmd overlay enable com.android.internal.display.cutout.emulation.tall"); m.sh("settings put system user_rotation 1"); time.sleep(2)
    m.nav("/checkin", 2200); print(m.cdp("JSON.stringify({sw:document.documentElement.scrollWidth,cw:document.documentElement.clientWidth,h1:Math.round(document.querySelector('h1').getBoundingClientRect().top)})"))
    m.sh("settings put system user_rotation 0"); m.sh("cmd overlay disable com.android.internal.display.cutout.emulation.tall")
if __name__ == "__main__": globals()[sys.argv[1]]()

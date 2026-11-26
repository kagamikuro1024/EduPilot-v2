# A11-05..08: bắt vùng thông báo (aria-live / role=alert / role=status) bằng MutationObserver + đếm lượt TTS của TalkBack trong logcat.
import lib34 as L, m, time, json, re, sys
OBS = """(()=>{window.__live=[];const t0=performance.now();const q=(n)=>{let e=n.nodeType===3?n.parentElement:n;while(e&&e!==document.body){const l=e.getAttribute&&e.getAttribute('aria-live');const r=e.getAttribute&&e.getAttribute('role');if(l||r==='alert'||r==='status'||r==='log')return {el:e,live:l||(r==='alert'?'assertive(role=alert)':'polite(role='+r+')')};e=e.parentElement}return null};
new MutationObserver(ms=>{for(const mu of ms){const h=q(mu.target);if(h){const tx=(h.el.innerText||'').trim().replace(/\\n/g,' ');window.__live.push({ms:Math.round(performance.now()-t0),live:h.live,txt:tx.slice(0,90)})}}}).observe(document.body,{subtree:true,childList:true,characterData:true,attributes:false});return 'obs'})()"""
def tts():
    return L.adb("logcat -d | grep -c 'GoogleTTSServiceImpl: Synthesis request'")
def tb(on):
    if on: L.adb("shell settings put secure enabled_accessibility_services com.google.android.marvin.talkback/com.google.android.marvin.talkback.TalkBackService"); L.adb("shell settings put secure accessibility_enabled 1")
    else: L.adb("shell settings delete secure enabled_accessibility_services"); L.adb("shell settings put secure accessibility_enabled 0")
    time.sleep(5)
def dump(tag, n0):
    live = m.cdp("JSON.stringify(window.__live)"); live = json.loads(live) if live and live != "undefined" else []
    # gộp các dòng liên tiếp trùng nội dung
    comp = []
    for e in live:
        if not comp or comp[-1]["txt"] != e["txt"] or comp[-1]["live"] != e["live"]: comp.append(e)
    print(tag, "| lượt TTS:", int(tts()) - n0, "| vùng thông báo:", json.dumps(comp, ensure_ascii=False)[:600])
    return comp

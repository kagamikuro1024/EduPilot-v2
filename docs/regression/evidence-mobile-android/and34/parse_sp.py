# Đọc các lượt THỰC SỰ đưa vào TTS: dòng `SpeechControllerImpl: Speaking fragment text="..." utteranceId=talkback_N`
import re, json, sys
raw=open(sys.argv[1],encoding="utf-8",errors="replace").read().split("\n")
f=lambda s: sum(float(x)*k for x,k in zip(s.split(":"),(3600,60,1)))
mk=[(re.search(r"(\d\d:\d\d:\d\d\.\d+)",l).group(1), l.split("QCMARK")[1].split(":",1)[1].strip()) for l in raw if "I QCMARK" in l]
t0=f([t for t,n in mk if n==sys.argv[2]][-1]); rows=[]
for ln in raw:
    m=re.search(r'(\d\d:\d\d:\d\d\.\d+).*SpeechControllerImpl: Speaking fragment text="(.*?)", utteranceId=(talkback_\d+).*?subtype:(\w+)', ln)
    if m and f(m.group(1))>=t0: rows.append((round(f(m.group(1))-t0,2), m.group(3), m.group(4), m.group(2)))
for r in rows: print("  +%6.2fs %-12s %-34s «%s»" % r)
print("10 s dau:", sum(1 for r in rows if r[0]<=10), "luot | tong:", len(rows))
json.dump(rows, open(sys.argv[1].replace("-raw.log","-speech.json"),"w"), ensure_ascii=False)

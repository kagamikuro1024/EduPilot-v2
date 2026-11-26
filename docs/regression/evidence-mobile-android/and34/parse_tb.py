import re, json, sys
raw=open(sys.argv[1],encoding="utf-8",errors="replace").read().split("\n")
f=lambda s: sum(float(x)*k for x,k in zip(s.split(":"),(3600,60,1)))
marks=[(re.search(r"(\d\d:\d\d:\d\d\.\d+)",l).group(1), l.split("QCMARK")[1].split(":")[-1].strip()) for l in raw if "I QCMARK" in l]
click=[f(t) for t,n in marks if n==sys.argv[2]][-1] if len(sys.argv)>2 else f(marks[-1][0])
rows=[]
for ln in raw:
    m2=re.search(r"(\d\d:\d\d:\d\d\.\d+)\s+\d+\s+\d+\s+V talkback: TalkBackFeedbackProvider:\s+(\w+):\s+ttsOutput=\s*(.*?)\s{2,}(?:queueMode|ttsAdd|force)", ln)
    if m2 and m2.group(3).strip() and not re.match(r"(queueMode|ttsAdd|force)", m2.group(3).strip()) and f(m2.group(1))>=click: rows.append((round(f(m2.group(1))-click,2), m2.group(2), m2.group(3).strip()))
print("moc:", marks)
for r in rows: print("  +%6.2fs %-34s %s" % r)
print("10 s dau:", sum(1 for r in rows if r[0]<=10), "luot | tong:", len(rows))
json.dump(rows, open(sys.argv[1].replace("-raw.log","-rows.json"),"w"), ensure_ascii=False)

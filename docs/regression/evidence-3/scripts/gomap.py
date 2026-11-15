import json,re,sys,collections
sp=int(sys.argv[1])
cat=json.load(open('/tmp/qc2-b/cat.json'))
def load(f):
    r={}
    for l in open(f):
        try:e=json.loads(l)
        except: continue
        if e.get('Test') and e['Action'] in('pass','fail','skip') and '/' not in e['Test']:
            r[(e['Package'].split('backend-go/')[-1],e['Test'])]=e['Action']
    return r
runs={'plain':{**load('/tmp/qc2-b/gj1.json')},'int':{**load('/tmp/qc2-b/gj2.json'),**load('/tmp/qc2-b/gj4.json')},'rt':load('/tmp/qc2-b/gj3.json')}
out={}
for k,v in cat.items():
    if v['sprint']!=sp: continue
    cmds=re.findall(r'go test([^`;]*)',v['steps'])
    if not cmds:
        cmds=[' '+x for x in re.findall(r'(-(?:tags \w+ )?-?run[^`;]*)',v['steps'])]
        DEF={'US-P1-01':'./internal/llmconfig','US-P1-02':'./internal/llm','US-P1-03':'./internal/llm/scheduler ./internal/llm','US-P1-04':'./internal/llmconfig ./internal/llm','US-P1-05':'./internal/llmconfig'}.get(v['us'],'./internal/llm')
        cmds=[c+' '+DEF for c in cmds]
    if not cmds: continue
    tags='integration' in v['steps']
    res=[];note=[]
    for c in cmds:
        m=re.search(r"-run\s+(?:'([^']*)'|(\S+))",c)
        pk=re.findall(r'(\./internal/[^\s]*)',c)
        if not m and 'go test -race ./...' in 'go test'+c: continue
        pat=(m.group(1) or m.group(2)) if m else '.'
        for p in pk:
            p=p.replace('./','').replace('/...','')
            runset=runs['int'] if tags else runs['plain']
            runset={**runs['rt'],**runset}
            hit=[(t,a) for (pp,t),a in runset.items() if (pp==p or pp.startswith(p+'/')) and re.search(pat,t)]
            res.append((p,pat,hit))
    if not res: continue
    allhit=[h for r in res for h in r[2]]
    if not allhit: out[k]=('NOHIT',str([(r[0],r[1]) for r in res])[:150])
    else:
        c=collections.Counter(a for _,a in allhit)
        # từng nhóm -run phải khớp
        names=[n for n in re.split(r'\|',res[0][1]) if re.match(r'^Test\w+$',n)]
        missing=[n for n in names if not any(n in t for t,_ in allhit)]
        st='F' if c['fail'] else 'P'
        out[k]=(st,f'{c["pass"]} pass/{c["skip"]} skip/{c["fail"]} fail'+(f'; thiếu test {missing}' if missing else ''))
for k,v in out.items(): print(k,v)
json.dump(out,open(f'/tmp/qc2-b/gomap{sp}.json','w'),ensure_ascii=False)

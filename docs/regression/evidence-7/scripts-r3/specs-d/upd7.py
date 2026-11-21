import re,sys,json
P='/Users/kuro/Documents/TA_Agent_qc3/docs/regression/smoke-7.md'
def upd(rows):
    s=open(P).read().split('\n')
    for tc,(st,note) in rows.items():
        note=note.replace('\n',' ').replace('|','¦')
        hit=False
        for i,l in enumerate(s):
            if l.startswith(f'| {tc} |'):
                s[i]=f'| {tc} | {st} | {note} |'; hit=True; break
        assert hit,tc
    open(P,'w').write('\n'.join(s))
def counts():
    s=open(P).read().split('\n'); c={'P':0,'F':0,'N':0}
    for l in s:
        m=re.match(r'\| (TC-[A-Za-z0-9-]+) \| ([PFN]) \|',l)
        if m: c[m.group(2)]+=1
    return c
if __name__=='__main__': print(counts())

import re,glob,os,sys
R=os.path.expanduser('~/Documents/TA_Agent_qc2/docs/regression')
base={}
for l in open(f'{R}/smoke-4.md',encoding='utf8'):
    m=re.match(r'\| (TC-[A-Z0-9]+-\d+) \| (\S+) \|',l)
    if m: base[m.group(1)]=m.group(2)
for f in sorted(glob.glob(f'{R}/evidence-4/results-smk34/*.tsv'),key=os.path.getmtime):
    for l in open(f,encoding='utf8'):
        p=l.rstrip('\n').split('\t')
        if len(p)>=3 and p[0].startswith('TC-'): base[p[0]]=p[1]
for a in sys.argv[1:]: print(a,base.get(a,'?'))

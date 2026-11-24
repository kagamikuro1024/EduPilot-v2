import sys,os,re
F=os.path.join(os.path.dirname(os.path.abspath(__file__)),'results.tsv')
def rec(tc,res,note):
    L=[l for l in (open(F).read().splitlines() if os.path.exists(F) else []) if not l.startswith(tc+'\t')]
    L.append(f"{tc}\t{res}\t{note}"); open(F,'w').write('\n'.join(sorted(L))+'\n')
def tsv():
    R=dict((l.split('\t')[0],l.split('\t')[1]) for l in open(F).read().splitlines() if l) if os.path.exists(F) else {}
    n=lambda x:sum(1 for v in R.values() if v==x)
    os.system(f'/tmp/qc3-oc-tsv.sh 51 {n("P")} {n("F")} {n("N")} "{sys.argv[1] if len(sys.argv)>1 else ""}"')
if __name__=='__main__': tsv()

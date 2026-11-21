# Dùng: python3 smkt3rec.py <TC> <P|F|LT> '<ghi chú>'  → thêm dòng vào results-smk4t3.tsv (bản ghi sau thắng)
import sys,os
OUT=os.path.expanduser('~/Documents/TA_Agent_qc2/docs/regression/evidence-4/results-smk34/results-smk4t3.tsv')
tc,st,note=sys.argv[1],sys.argv[2],' '.join(sys.argv[3].split())
assert st in('P','F','LT')
open(OUT,'a').write(f'{tc}\t{st}\t{note}\n'); print(st,tc,note[:150])

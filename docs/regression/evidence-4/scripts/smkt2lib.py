# Thư viện SmkT2 (sprint 4 smoke N=0, lượt 2): dùng q4lib (ep-r), ghi results-smk4t2.tsv (bản ghi sau thắng).
import sys, os
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
os.environ.setdefault('SP', 'smkt2')
import q4lib
from q4lib import *
OUT_T2 = os.path.expanduser('~/Documents/TA_Agent_qc2/docs/regression/evidence-4/results-smk34/results-smk4t2.tsv')
def rec(tc, st, note=''):
    note = ' '.join(str(note).split())
    with open(OUT_T2, 'a') as f: f.write(f'{tc}\t{st}\t{note}\n')
    print(st, tc, note[:200], flush=True)

#!/usr/bin/env python3
"""s8flow-man.py TC ST 'ghi chú' [TC ST 'ghi chú'...] — ghi kết quả thủ công vào logs/s8flow-manual.json rồi gộp TSV."""
import json, os, sys, subprocess
R = os.path.expanduser('~/Documents/TA_Agent_qc2/docs/regression/evidence-8'); p = R + '/logs/s8flow-manual.json'
d = json.load(open(p)) if os.path.exists(p) else {}
a = sys.argv[1:]
for i in range(0, len(a), 3): d[a[i]] = [a[i+1], a[i+2][:160]]
json.dump(d, open(p, 'w'), ensure_ascii=False, indent=1)
subprocess.run(['python3', R + '/scripts/s8flow-tsv.py'])

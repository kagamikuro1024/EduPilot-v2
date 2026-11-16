"""QC2 PE09-27/PE05-48: bài thi MCQ mở 4 giờ cho lớp R5C1 (không câu code). In id vào /tmp/qc2-c5/lhexam.json."""
import os, sys; sys.path.insert(0, '/Users/kuro/Documents/TA_Agent_qc2/docs/regression/evidence-5/scripts'); os.environ['SUITE'] = 'LHX'
from r5lib import *
q = [mkq(700 + i, 'MCQ_SINGLE', 4, (1,)) for i in range(4)] + [mkq(704, 'MCQ_MULTI', 5, (0, 2)), mkq(705, 'TRUE_FALSE')]
x, _ = mkexam(open_s=14, win=4 * 3600, dur=120, items=q, title='R5L LH /take 4h')
json.dump(dict(eid=x, c1=C1), open('/tmp/qc2-c5/lhexam.json', 'w')); print(x, C1)

import os; os.environ['SUITE'] = 'UIM'
from r5lib import *
e, _ = mkexam(open_s=10, win=14400, dur=30, nq=2, title='R5M trống chưa ai làm'); json.dump(dict(empty=e), open('/tmp/qc2L/empty.json', 'w')); print(e)

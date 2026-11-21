import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
a = sql("select id from assignments where title='QC106 g12b' order by created_at desc limit 1")[0][0]
with window():
    sql(f"delete from grading_guidance_items where assignment_id='{a}' and status='DRAFT'"); stub("rf_ok_plus1_slow"); reset_reflect_limit()
    os.system(f'cd {os.path.dirname(os.path.abspath(__file__))} && QC_RES= ./runpw.sh qc-real-106-ux.spec.ts "UX-m" 150 2>&1 | grep -E "VERDICT|Error|✘" | cut -c1-1400')

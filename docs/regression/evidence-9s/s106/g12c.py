import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
import importlib.util
W = "/Users/kuro/.config/edupilot-team/qc3-work/"; HERE = os.path.dirname(os.path.abspath(__file__))
a = sql("select id from assignments where title='QC106 g12b' order by created_at desc limit 1")[0][0]
with window():
    sql(f"delete from grading_guidance_items where assignment_id='{a}' and status='DRAFT'"); stub("rf_ok"); reset_reflect_limit()
    json.dump({"a": a, "tcs": ["TC-P12-24"], "must": [r"Đã rút|chờ duyệt"], "mustnot": [r"lưu ý bị loại", r"Đã xảy ra"], "user": "teacher", "click": True, "post": "old", "wait": 9000}, open(W + "g12_case.json", "w"), ensure_ascii=False)
    os.system(f'cd {HERE} && QC_RES= ./runpw.sh qc-real-106-g12.spec.ts "P12 UI" 100 2>&1 | grep -E "VERDICT|✘" | cut -c1-500')

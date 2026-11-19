import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
def run():
    a, ids, info = guided("g04ui", 12, 8, 2, do_apply=False)  # 8 DRAFT từ rf_many, 2 đã duyệt chưa áp dụng
    print("guided", info)
    A = [x["id"] for x in items_of(a, "APPROVED")]
    print("apply v1", apply_v(a, 0)[0])
    # v2: nghỉ A0, duyệt thêm 1
    dr = [x["id"] for x in items_of(a, "DRAFT")]
    print("retire", decide(a, A[0], "RETIRE")[0], "approve", decide(a, dr[0])[0]); print("apply v2", apply_v(a, 1)[0])
    dr = [x["id"] for x in items_of(a, "DRAFT")]
    decide(a, dr[0]); decide(a, dr[1], "REJECT"); decide(a, A[1], "RETIRE")  # Chờ áp dụng, Đã bỏ, Chờ nghỉ
    st = {}
    for x in items_of(a): st.setdefault((x["status"], x.get("approved_in_version"), x.get("pending_retire")), []).append(x["id"][-4:])
    print("trạng thái", {str(k): len(v) for k, v in st.items()})
    json.dump({"a": a}, open("/tmp/gd_ui_a.json", "w"))
with window(): run()

import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
def run():
    flag("GRADING_GUIDANCE", True)
    # G03-14 / R106S-02: FAILED rồi bấm lại với khoá khác
    a = fxa("g0314r"); review_with(a, [(i, {"c2": "0.00"}) for i in range(0, 4)] + [(4, {"c3": "2.00"}), (5, {"c3": "2.00"})])
    c1, j1, job1 = reflect(a, "rf_fake_support"); stub("rf_ok")
    c2, j2, _ = api("teacher", "POST", G(a) + "/reflect", {}, idem=str(uuid.uuid4())); job2 = wait_job(j2["job_id"]) if c2 == 202 else None
    d = drafts(a)
    # cùng khoá Idempotency-Key + job SUCCEEDED → cùng job
    c3, j3, _ = api("teacher", "POST", G(a) + "/reflect", {}, idem=str(uuid.uuid4()))
    chk("TC-G03-14b", c1 == 202 and job1[0] == "FAILED" and c2 == 202 and j2["job_id"] != j1["job_id"] and job2 and job2[0] == "SUCCEEDED" and len(d) > 0 and c3 == 202 and j3["job_id"] == j2["job_id"], f"mã 0bdef404: lần 1 (stub JSON sai) job {j1.get('job_id','')[-6:]} → {job1[0]}; sửa stub, bấm lại với Idempotency-Key KHÁC → 202 job {j2.get('job_id','')[-6:]} ({'job mới' if j2.get('job_id') != j1.get('job_id') else 'CÙNG job FAILED'}) → {job2 and job2[0]}, {len(d)} mục; bấm lần 3 khi đã SUCCEEDED cùng dữ liệu → job {j3.get('job_id','')[-6:]} ({'cùng job lần 2, chống trùng đúng' if j3.get('job_id') == j2.get('job_id') else 'job khác'}); (R106S-02 sửa ab89d82b)")
with window(): run()

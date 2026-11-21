import sys, os, re; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
W = "/Users/kuro/.config/edupilot-team/qc3-work/"
def run():
    flag("GRADING_GUIDANCE", False)
    a = mk_assignment("g11p6", 2)
    rs = sql(f"select s.id from submissions s where s.assignment_id='{a}' order by s.created_at")
    phr = "Em cần áp dụng nguyên tắc đặc quyền tối thiểu; mô hình dịch vụ an ninh thuê ngoài; xây dựng một thế trận an ninh mạng."
    nm = "Bài này do Dũng Nguyễn viết, em nhắc lại Dung Nguyen cần áp dụng nguyên tắc đặc quyền tối thiểu."
    sql("set session_replication_role=replica; " + f"update submissions set extracted_text=$${phr}$$ where id='{rs[0][0]}'; update submissions set extracted_text=$${nm}$$ where id='{rs[1][0]}'")
    redis_del("ep:roster:*"); stub("allow"); open(W + "stub_req.jsonl", "w").close()
    c, j, n = grade_all_wait(a); print("grade", c, n)
    L = [json.loads(l) for l in open(W + "stub_req.jsonl") if l.strip()]
    txt = [json.dumps(r["body"], ensure_ascii=False) for r in L]
    t1 = [t for t in txt if "dịch vụ an ninh thuê ngoài" in t or "thế trận" in t]; t2 = [t for t in txt if "tối thiểu" in t and "Bài này do" in t or "Bài này do" in t or "[[SV_" in t]
    print("req", len(L), "có cụm", len(t1), "có tên/placeholder", len(t2))
    ph = any("[[SV_" in t for t in t1)
    all_t = " ".join(txt)
    leak_name = ("Dũng Nguyễn" in all_t) or ("Dung Nguyen" in all_t)
    chk("TC-P11-06", len(t1) >= 1 and not ph and "[[SV_" in all_t and not leak_name, f"bài 1 có 3 cụm (áp dụng nguyên tắc / dịch vụ an ninh thuê ngoài / thế trận an ninh mạng) + bài 2 có tên «Dũng Nguyễn» / «Dung Nguyen» (roster QD9G), chấm qua stub ghi prompt ({len(L)} yêu cầu): prompt chứa cụm gốc = {len(t1)} yêu cầu, cụm bị thay [[SV_ = {ph}; tên thật còn trong prompt = {leak_name}; có [[SV_ cho tên = {'[[SV_' in all_t}")
with window(): run()

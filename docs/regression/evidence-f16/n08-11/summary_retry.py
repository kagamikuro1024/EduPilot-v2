# Nhận xét 360 (ai-summary): 504 DEADLINE_EXCEEDED ở 2/3 mẫu lượt đầu → gọi lại từng sinh viên 3 lần, ghi mọi lượt.
import os, sys, json
os.environ.update(BASE="https://localhost:1143", INSECURE_TLS="1", OUT=os.path.abspath("out"),
                  AUDIT_PSQL="docker exec -i ep-r-postgres-1 psql -U edupilot -d edupilot -tA -v ON_ERROR_STOP=1")
sys.path.insert(0, "../../../../benchmarks/real-llm")
import real_llm as R
api = R.Api(); c1 = R.c1_id(api)
m = {x["email"].split("@")[0]: x for x in R.members(api, c1)}
st = {"c1": c1}
rows = []
for acc in ("sv.gioi", "sv.kha", "sv.nguyco"):
    for k in range(3):
        r = R.misc_summary(api, st, m, acc); r["try"] = k + 1; rows.append(r)
        print(acc, k + 1, {a: b for a, b in r.items() if a != "draft"} if "error" not in r else r["error"][:120]); R.pause(3)
json.dump(rows, open("out/summary-retry.json", "w"), ensure_ascii=False, indent=1)

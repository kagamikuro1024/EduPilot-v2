# s8core: 04-26 — đổi max_score giữa kỳ 10→100: điểm không mang sang item mới, hiện trong hộp thoại/impact bằng số
import copy
from s8core_lib import *
A1 = open("/tmp/s8core/QD8A1.id").read().strip(); Q = f"/courses/{A1}"
subprocess.run(["python3", "s8core-mkdraft.py", A1], capture_output=True, text=True)
d = call("GET", f"{Q}/grade-scheme", "teacher")[1]["draft"]
def mod(b):
    for c in b["components"]:
        for it in c["items"]:
            if it["key"] == "midterm": it["max_score"] = "100"
b = {"version": d["version"], "components": [{k: x[k] for k in ("key", "name", "grp", "weight")} | {"items": copy.deepcopy(x["items"])} for x in d["components"]], "rules": [{"kind": r["kind"], "applicable": r["applicable"], "params": r["params"]} for r in d["rules"]]}
mod(b); c, j = call("PUT", f"{Q}/grade-scheme", "teacher", b); print("put", c)
ci, im = call("POST", f"{Q}/grade-scheme/impact", "teacher", {k: v for k, v in b.items() if k != "version"}); print(ci, json.dumps(im, ensure_ascii=False)[:400])
d2 = call("GET", f"{Q}/grade-scheme", "teacher")[1]["draft"]
cc, jj = call("POST", f"{Q}/grade-scheme/confirm", "teacher", {"version": d2["version"]}); print("confirm", cc, json.dumps(jj, ensure_ascii=False)[:300])
cnt = sql(f"select i.key, s.status, i.max_score, count(e.id) from grade_items i join grade_schemes s on s.id=i.scheme_id left join grade_entries e on e.item_id=i.id where i.course_id='{A1}' and i.key='midterm' group by 1,2,3 order by 2,3")
pv = call("GET", f"{Q}/gradebook/preview?limit=100", "teacher")[1]["summary"]
dr = im.get("entries_dropped", [])
chk("TC-P6-04-26", ci == 200 and dr and dr[0]["item_key"] == "midterm" and dr[0]["count"] == 30 and cc == 200 and jj["impact"]["entries_dropped"][0]["count"] == 30 and any(r[1] == "CONFIRMED" and r[3] == "0" for r in cnt) and any(r[1] == "SUPERSEDED" and r[3] == "30" for r in cnt) and pv["incomplete"] == 30, f"midterm max 10→100: impact/confirm entries_dropped=[{dr[0]['item_key']}: {dr[0]['count']}, {dr[0]['reason']}]; điểm KHÔNG sang item thang 100 (mới 0 điểm), 30 điểm còn ở item cũ SUPERSEDED trong DB; bảng INCOMPLETE {pv['incomplete']}/30", f"{ci} {im} {cc} {cnt} {pv}")

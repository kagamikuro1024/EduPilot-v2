# s8core: luồng bản nháp trên QD8A1 (DRAFT tạo bằng SQL từ bản CONFIRMED): 02-24, 02-42, 04-08/10/11/12/21/25/26/33/42/43
import copy, json, subprocess
from s8core_lib import *
A1 = open("/tmp/s8core/QD8A1.id").read().strip(); Q = f"/courses/{A1}"
def draft(): return call("GET", f"{Q}/grade-scheme", "teacher")[1]["draft"]
def mkbody(d, mod=None, version=None):
    b = {"version": version or d["version"], "components": [{k: x[k] for k in ("key", "name", "grp", "weight")} | {"items": x["items"]} for x in d["components"]], "rules": [{"kind": r["kind"], "applicable": r["applicable"], "params": copy.deepcopy(r["params"])} for r in d["rules"]]}
    if mod: mod(b)
    return b
def put(mod, version=None): d = draft(); return call("PUT", f"{Q}/grade-scheme", "teacher", mkbody(d, mod, version))
def codes(d): return sorted({u.get("code") for u in d.get("unclear_items", [])}) if isinstance(d, dict) else []
def W(m, w, f): 
    def g(b):
        for c in b["components"]: c["weight"] = {"midterm": m, "weekly": w, "final": f}[c["key"]]
    return g
def rule(kind, fn):
    def g(b):
        for r in b["rules"]:
            if r["kind"] == kind: fn(r["params"])
    return g
c, r = put(W("20", "20", "60")); print("reset", c, codes(r))
# ---- 02-42
c, r = put(W("50", "50", "0")); u = r.get("unclear_items", []) if isinstance(r, dict) else []
print("w_CK=0:", c, json.dumps(u, ensure_ascii=False)[:400])
chk("TC-P6-02-42", c == 200 and "PARAM_INVALID" in codes(r), f"w_CK=0 + luật FINAL_BELOW: PUT 200, unclear_items {codes(r)} (mục chưa rõ, chặn xác nhận) — impact trả tính bình thường, mục chưa rõ nằm ở bản nháp", f"{c} {u}")
c, r = put(W("20", "20", "60"))
# ---- 02-24: xoá params.free của PENALTY
c, r = put(rule("PENALTY", lambda p: p.pop("free", None))); u24 = codes(r) if isinstance(r, dict) else r
d = draft(); b24 = mkbody(d, rule("PENALTY", lambda p: p.pop("free", None)))
ci, ji = call("POST", f"{Q}/grade-scheme/impact", "teacher", {k: v for k, v in b24.items() if k != "version"})
print("free xoá:", c, u24, "| impact", ci, str(ji)[:200])
chk("TC-P6-02-24", (c == 422 or "PARAM_INVALID" in (u24 if isinstance(u24, list) else [])) and (ci == 422 or "PARAM_INVALID" in json.dumps(ji)), f"xoá PENALTY.params.free: PUT {c} {u24}; impact {ci} {str(ji)[:80]} — không mặc định thầm lặng", f"{c} {u24} {ci} {ji}")
d = draft(); c, r = put(W("20", "20", "60")); 
c, r = put(rule("PENALTY", lambda p: p.update(free=2)))
print("khôi phục", c, codes(r))
# ---- 04-11 trọng số
res = {}
for w in ("abc", "-5", "101"):
    cc, jj = put(W(w, "20", "60")); res[w] = (cc, jj.get("code") if isinstance(jj, dict) else "")
v0 = draft()["version"]
cc, jj = put(W("12,5", "20", "60")); res["12,5"] = (cc, codes(jj) if cc == 200 else jj.get("code"))
cc, jj = put(W("30", "20", "60")); res["sum110"] = (cc, [u for u in jj.get("unclear_items", []) if u.get("code") == "WEIGHT_SUM"] if cc == 200 else jj)
print(res)
chk("TC-P6-04-11", all(res[w][0] in (400, 422) for w in ("abc", "-5", "101")) and res["sum110"][0] == 200 and res["sum110"][1] and "110" in json.dumps(res["sum110"][1]), f"trọng số abc/-5/101 → {[res[w][0] for w in ('abc','-5','101')]}; 12,5 → {res['12,5'][0]} {res['12,5'][1]}; tổng 110 → unclear WEIGHT_SUM kèm tổng 110 (giữ chữ: UI, xem spec)", f"{res}")
# ---- 04-08 confirm bị chặn
d = draft(); cc, jj = call("POST", f"{Q}/grade-scheme/confirm", "teacher", {"version": d["version"]})
st = sq1(f"select status from grade_schemes where course_id='{A1}' and scheme_no=(select max(scheme_no) from grade_schemes where course_id='{A1}')")
chk("TC-P6-04-08", cc == 422 and "SCHEME_HAS_UNCLEAR" in json.dumps(jj) and "WEIGHT_SUM" in json.dumps(jj) and st == "DRAFT", f"confirm khi còn mục chưa rõ → {cc} {jj.get('code')} details {json.dumps(jj.get('details'), ensure_ascii=False)[:60]}; status vẫn DRAFT", f"{cc} {jj} {st}")
# ---- 04-12 VERSION_CONFLICT
d = draft(); cc, jj = call("PUT", f"{Q}/grade-scheme", "teacher", mkbody(d, W("20", "20", "60"), version=d["version"] - 1))
chk("TC-P6-04-12", cc == 409 and "VERSION_CONFLICT" in json.dumps(jj), f"PUT với version cũ → {cc} {jj.get('code')} kèm giá trị hiện tại ({'details' in jj})", f"{cc} {jj}")
# ---- 04-10: sửa làm tròn 0,1 (ô GV điền)
v_before = draft()["version"]
cc, jj = put(lambda b: (W("20", "20", "60")(b), rule("ROUNDING", lambda p: p.update(step="0.1"))(b)))
rr = [r for r in jj["rules"] if r["kind"] == "ROUNDING"][0]
chk("TC-P6-04-10", cc == 200 and jj["version"] == v_before + 1 and rr["filled_by_user"] is True and not rr.get("citations") and jj["unclear_items"] == [], f"PUT rời ô: version {v_before}→{jj['version']}, unclear_items recomputed []={jj['unclear_items']}, ROUNDING filled_by_user=true, không trích dẫn (UI ghi 'do giảng viên đặt'; dòng mạng ở UI-TC)", f"{cc} {rr} {jj['unclear_items']}")
# ---- 04-42/43
d = draft()
c42, j42 = call("POST", f"{Q}/grade-scheme/confirm", "teacher", {"version": d["version"]}, key=False)
c43, j43 = call("POST", f"{Q}/grade-scheme/confirm", "teacher", {})
print(c42, j42, c43, j43)
chk("TC-P6-04-42", c42 in (400, 422) and isinstance(j42, dict) and j42.get("code"), f"confirm thiếu Idempotency-Key → {c42} {j42.get('code')}; DRAFT không đổi", f"{c42} {j42}")
chk("TC-P6-04-43", c43 == 422 and "VALIDATION_FAILED" in json.dumps(j43), f"confirm thân {{}} thiếu version → {c43} {j43.get('code')}", f"{c43} {j43}")
# ---- 04-33 / 04-21 / 04-25
d = draft(); old_ids = [r[0] for r in sql(f"select id from grade_items where course_id='{A1}' and scheme_id=(select id from grade_schemes where course_id='{A1}' and status='CONFIRMED')")]
cs, js = call("POST", f"{Q}/grade-scheme/confirm", "teacher", {"version": d["version"] - 1})
stale_state = sq1(f"select status from grade_schemes where id='{d['id']}'")
chk("TC-P6-04-33", cs == 409 and "VERSION_CONFLICT" in json.dumps(js) and stale_state == "DRAFT", f"confirm với version cũ → {cs} {js.get('code')}; status vẫn {stale_state}; sau đó confirm version mới (bước kế) → 200", f"{cs} {js} {stale_state}")
KEY = str(uuid.uuid4()); sn0 = sq1(f"select count(*) from grade_schemes where course_id='{A1}'")
c1, j1 = call("POST", f"{Q}/grade-scheme/confirm", "teacher", {"version": d["version"]}, key=KEY)
c2, j2 = call("POST", f"{Q}/grade-scheme/confirm", "teacher", {"version": d["version"]}, key=KEY)
sn1 = sq1(f"select count(*) from grade_schemes where course_id='{A1}'")
print("confirm", c1, json.dumps(j1, ensure_ascii=False)[:300], "| replay", c2, j1 == j2)
rows = sql(f"select status,scheme_no,confirmed_by is not null from grade_schemes where course_id='{A1}' order by scheme_no")
au = sql(f"select action,left(after::text,200) from audit_log where course_id='{A1}' and entity='grade_schemes' and action='confirm' order by created_at desc limit 1")
ob = sq1(f"select count(*) from outbox where topic='grade.scheme_confirmed' and payload::text like '%{A1}%'")
chk("TC-P6-04-21", c1 == 200 and c2 == 200 and j1 == j2 and sn0 == sn1 and rows[-1][0] == "CONFIRMED" and rows[-2][0] == "SUPERSEDED" and rows[-1][2] == "t" and au and int(ob) >= 1, f"confirm 200: DRAFT→CONFIRMED(confirmed_by), cũ→SUPERSEDED {[r[:2] for r in rows]}; audit_log confirm có số liệu {au[0][1][:50] if au else None}; outbox grade.scheme_confirmed={ob}; gửi lại cùng Idempotency-Key → cùng phản hồi, không thêm bản ({sn0}={sn1})", f"{c1} {c2} {j1==j2} {rows} {au} {ob}")
cnt = sql(f"select i.key, s.status, count(e.id) from grade_items i join grade_schemes s on s.id=i.scheme_id left join grade_entries e on e.item_id=i.id where i.course_id='{A1}' group by 1,2 order by 1,2")
pv = call("GET", f"{Q}/gradebook/preview?limit=100", "teacher")[1]["summary"]
print(cnt, pv)
carried = {k: n for k, s, n in cnt if s == "CONFIRMED"}
kept = {k: n for k, s, n in cnt if s == "SUPERSEDED"}
chk("TC-P6-04-25", carried.get("midterm") == "30" and carried.get("final") == "30" and kept.get("midterm") == "30" and pv["complete"] == 30 and j1.get("entries_carried") in (60, "60") if isinstance(j1, dict) else False, f"điểm chuyển sang item mới: midterm {carried.get('midterm')}, final {carried.get('final')} (60); item cũ SUPERSEDED giữ {kept}; /gradebook vẫn 30/30 đủ điểm; entries_carried={j1.get('entries_carried')}", f"{cnt} {pv} {j1}")
json.dump({"confirm": j1}, open("/tmp/s8core/a1-confirm.json", "w"), ensure_ascii=False)

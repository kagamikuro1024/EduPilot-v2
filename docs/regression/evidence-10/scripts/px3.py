from pb import *
# TC-04-11 trên câu MCQ_SINGLE (pb3 lấy ids[0] là câu SHORT do pa1 tạo → thân PUT sai)
q = [x for x in bank() if x["type"] == "MCQ_SINGLE" and x["review_status"] in ("PENDING", "REJECTED")][0]["id"]; d = detail(q); v = d["version"]
cor = [i for i, o in enumerate(d["options"]) if o["id"] in ((d.get("answer_key") or {}).get("option_ids") or [])]
body = lambda s: {**{k: d[k] for k in ("type", "title", "topic", "difficulty")}, "stem": d["stem"] + s, "options": [{"body": o["body"]} for o in d["options"]], "correct": cor, "version": v}
c1, j1, _ = api("teacher", "PUT", f"{P}/questions/{q}", body(" (GV)")); c2, j2, _ = api("ta", "PUT", f"{P}/questions/{q}", body(" (TA)"))
ok("TC-04-11 version cũ → 409 VERSION_CONFLICT", c1 == 200 and c2 == 409 and "VERSION_CONFLICT" in json.dumps(j2), f"{c1}/{c2} {json.dumps(j2, ensure_ascii=False)[:160]}")
# TC-04-15: Hôm nay GV có QUESTION_REVIEW; duyệt hết thì biến mất ≤ 5 s (chỉ kiểm sự có mặt + đếm)
c, t, _ = api("teacher", "GET", P + "/today"); s = json.dumps(t, ensure_ascii=False)
ok("TC-04-15a GV có việc QUESTION_REVIEW khi có câu PENDING", "QUESTION_REVIEW" in s, f"{c} {len(s)} ký tự; có QUESTION_REVIEW={'QUESTION_REVIEW' in s}")

from gdlib import *
import re
# GUIDANCE_ENABLED=false
time.sleep(3)
c, f = None, None
c, j, _ = api("teacher", "GET", FL); print("flags khi GUIDANCE_ENABLED=false:", c, json.dumps(j)[:200])
c, j, _ = api("teacher", "PUT", f"{FL}/GRADING_GUIDANCE", {"enabled": True, "version": [x for x in j["items"] if x["feature"] == "GRADING_GUIDANCE"][0]["version"]}); ok("G01-07 GUIDANCE_ENABLED=false: ghi cờ → 409 GUIDANCE_DISABLED", c == 409 and (j or {}).get("code") == "GUIDANCE_DISABLED", f"{c} {(j or {}).get('code')}")
j0 = sql("select count(*) from jobs where kind='grading.reflect' and created_at > now() - interval '1 minute'")[0][0]
c, j, _ = api("teacher", "POST", G + "/reflect", {}, idem=f"qc-off2-r-{int(time.time())}"); ok("G01-07 reflect → 409, 0 job mới", c == 409 and sql("select count(*) from jobs where kind='grading.reflect' and created_at > now() - interval '1 minute'")[0][0] == j0, f"{c}")
# (gateway được khôi phục bởi batch)

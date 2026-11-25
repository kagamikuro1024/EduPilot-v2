# newwin.py <idx> [lớp=QCVPSD]: mở cửa sổ điểm danh cho buổi thứ idx (theo session_no) của lớp, ghi win.json (ngoài repo) cho scan.py.
import sys, json, os, vps
code = sys.argv[2] if len(sys.argv) > 2 else "QCVPSD"
k = vps.course(code)
c, j = vps.call("teacher", "GET", f"/courses/{k['id']}/sessions?limit=100"); assert c == 200
s = next(x for x in j["items"] if x["session_no"] == int(sys.argv[1]))
c, w = vps.call("teacher", "POST", f"/courses/{k['id']}/sessions/{s['id']}/checkin-windows", {"minutes": 30}); assert c == 201, (c, w)
json.dump({"cid": k["id"], "sid": s["id"], "wid": w["id"]}, open(os.path.expanduser("~/.config/edupilot-team/qc2-work/win.json"), "w"))
print("window", code, "buổi", s["session_no"])

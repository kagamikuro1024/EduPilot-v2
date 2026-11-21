import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
t0 = time.time()
while True:
    try: os.mkdir("/tmp/qc-global.lock"); break
    except FileExistsError:
        if time.time() - t0 > 600: raise SystemExit("LOCKED")
        time.sleep(5)
open("/tmp/qc-global.lock/owner", "w").write("QC3 G08-17 pgbouncer dừng ngắn")
flag("THREAD_SUGGEST", True)
tok("sv.kha")  # lấy token trước khi dừng DB
try:
    os.system("docker stop ep-qc-pgbouncer-1 >/dev/null"); time.sleep(2)
    c, j, ms = raw("POST", f"/courses/{CG0}/threads/similar-confirmed", {"q": "Câu hỏi hoàn toàn mới chưa từng nhúng số " + str(time.time())}, {}, tok("sv.kha"))[:3]
    print("API khi DB dừng", c, j, round(ms) if ms else ms)
    os.system(f'cd {os.path.dirname(os.path.abspath(__file__))} && ./runpw.sh qc-real-106-g08.spec.ts "TC-G08-17" 150')
finally:
    os.system("docker start ep-qc-pgbouncer-1 >/dev/null"); time.sleep(6)
    import shutil; shutil.rmtree("/tmp/qc-global.lock", ignore_errors=True)
c2, j2, _ = api("sv.kha", "GET", f"/courses/{CG0}/threads?limit=1"); print("sau khi bật lại DB", c2)
chk("TC-G08-17", c == 200 and j == [] , f"dừng ep-qc-pgbouncer-1 (DB không truy cập được; gateway còn chạy): POST similar-confirmed → {c} {j} (kỳ vọng 200 []); bật lại → GET threads {c2}; phần giao diện: xem TC-G08-17b")

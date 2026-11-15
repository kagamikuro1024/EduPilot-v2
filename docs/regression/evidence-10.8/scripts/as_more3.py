# smoke r10.8: TC-AS12-05, TC-AS13-04, TC-AS13-09, TC-AS14-12 (PUT điểm danh tay)
import re
from asl import *
import subprocess, datetime as dt, threading
from cryptography import x509
from cryptography.x509.oid import NameOID
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import ec
R = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../.."))
S = st(); C = S["cid"]
sh = lambda c, **k: subprocess.run(c, shell=True, capture_output=True, text=True, cwd=R, **k)
# ---- AS12-05 chuỗi X.509 tự ký
def cert(cn):
    k = ec.generate_private_key(ec.SECP256R1()); n = x509.Name([x509.NameAttribute(NameOID.COMMON_NAME, cn)])
    c = x509.CertificateBuilder().subject_name(n).issuer_name(n).public_key(k.public_key()).serial_number(x509.random_serial_number()).not_valid_before(dt.datetime.utcnow() - dt.timedelta(days=1)).not_valid_after(dt.datetime.utcnow() + dt.timedelta(days=30)).sign(k, hashes.SHA256())
    return base64.b64encode(c.public_bytes(serialization.Encoding.DER)).decode()
time.sleep(12)
api("sv.moi", "POST", "/me/devices/challenge", None)
c, j, _ = api("sv.moi", "POST", "/me/devices", {"kind": "ANDROID_KEY", "public_key": Key().pub, "attestation_chain": [cert("leaf"), cert("root")]})
ok("TC-AS12-05 chuỗi X.509 tự ký giả (2 chứng chỉ) → 422 ATTESTATION_FAILED", c == 422 and "ATTESTATION_FAILED" in json.dumps(j), f"{c} {json.dumps(j)[:100]}")
# ---- AS13-04 / 09: 2.500 lượt quá hạn
ssn = sessions(TEACHER, C)
used = {x[0] for x in sql(f"select distinct session_id from checkin_windows where course_id='{C}'")}
free = [s_["id"] for s_ in ssn if s_["id"] not in used][:90]
op = uid(TEACHER); students = [x[0] for x in sql("select id from users where email like 'sv%@edupilot.local' and role='STUDENT' order by email limit 40")]
print("free sessions", len(free), "students", len(students))
for s_ in free:
    for rd in (1, 2, 3):
        sql(f"insert into checkin_windows(course_id,session_id,round,rotate_seconds,opened_by,opened_at,closes_at,closed_at) values ('{C}','{s_}',{rd},10,'{op}',now()-interval '40 days',now()-interval '40 days'+interval '10 minutes',now()-interval '40 days'+interval '9 minutes')")
vals = ",".join(f"('{u}')" for u in students)
sql(f"insert into attendance_checkins(course_id,session_id,window_id,student_id,channel,assurance,token_age_ms,created_at) select w.course_id,w.session_id,w.id,u.id::uuid,'WEB',1,100,now()-interval '40 days' from checkin_windows w, (values {vals}) as u(id) where w.course_id='{C}' and w.closed_at < now()-interval '39 days' and w.round in (1,2,3)")
n0 = int(sql(f"select count(*) from attendance_checkins where created_at < now()-interval '39 days' and course_id='{C}'")[0][0])
sql("create or replace function qc_h_slow() returns trigger language plpgsql as $$ begin perform pg_sleep(0.006); return old; end $$")
sql("create trigger qc_h_slow before delete on attendance_checkins for each row execute function qc_h_slow()")
try:
    # AS13-09: production bị từ chối; không route HTTP purge; ngắt giữa chừng
    pr = sh("docker exec -e APP_ENV=production ep-r-gateway-1 /gateway admin purge-checkins")
    http = [raw("POST", p, {}, {"Idempotency-Key": str(uuid.uuid4())}, tok(ADMIN))[0] for p in ("/admin/attendance/purge", "/admin/purge-checkins", f"/courses/{C}/checkins/purge")]
    p1 = subprocess.Popen("docker exec ep-r-gateway-1 /gateway admin purge-checkins", shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    time.sleep(6)
    sql("select pg_terminate_backend(pid) from pg_stat_activity where query ilike '%attendance_checkins%' and query ilike '%delete%' and pid<>pg_backend_pid()")
    o1 = p1.communicate(timeout=120)
    mid = int(sql(f"select count(*) from attendance_checkins where created_at < now()-interval '39 days' and course_id='{C}'")[0][0])
    sql("drop trigger qc_h_slow on attendance_checkins")
    r2 = sh("docker exec ep-r-gateway-1 /gateway admin purge-checkins")
    end = int(sql(f"select count(*) from attendance_checkins where created_at < now()-interval '39 days' and course_id='{C}'")[0][0])
finally:
    sql("drop trigger if exists qc_h_slow on attendance_checkins"); sql("drop function if exists qc_h_slow()")
ok("TC-AS13-09 production bị từ chối; không route HTTP purge (404); ngắt giữa chừng: lô đã xoá giữ nguyên, chạy lại xoá nốt", pr.returncode != 0 and all(h in (404, 405) for h in http) and 0 < n0 and mid < n0 and end == 0, f"prod rc={pr.returncode} {pr.stderr.strip()[:80]!r}; http={http}; trước={n0} giữa={mid} sau lần 2={end}; lần1 rc={p1.returncode} {o1[1].strip()[:80]!r}; lần2 {r2.stdout.strip()!r}")
# AS13-04: nạp lại, kiểm số lượt xoá một lần chạy & kích thước lô trong SQL
sql(f"insert into attendance_checkins(course_id,session_id,window_id,student_id,channel,assurance,token_age_ms,created_at) select w.course_id,w.session_id,w.id,u.id::uuid,'WEB',1,100,now()-interval '40 days' from checkin_windows w, (values {vals}) as u(id) where w.course_id='{C}' and w.closed_at < now()-interval '39 days' and w.round in (1,2,3)")
n1 = int(sql(f"select count(*) from attendance_checkins where created_at < now()-interval '39 days' and course_id='{C}'")[0][0])
r3 = sh("docker exec ep-r-gateway-1 /gateway admin purge-checkins")
n2 = int(sql(f"select count(*) from attendance_checkins where created_at < now()-interval '39 days' and course_id='{C}'")[0][0])
qsrc = sh("grep -n 'limit' -i backend-go/internal/store/attendance_checkin.sql.go | grep -i -B1 -A1 purge; grep -n -A8 'const attPurgeCheckins' backend-go/internal/store/attendance_checkin.sql.go").stdout
lot = "limit 1000" if re.search(r"limit\s+1000", qsrc, re.I) else qsrc[:200]
ok("TC-AS13-04 2.500+ lượt quá hạn: xoá hết một lần chạy theo lô (SQL có LIMIT 1000), log tổng số hàng", n1 >= 2500 and n2 == 0 and re.search(r"limit\s+1000", qsrc, re.I) is not None, f"{n1}→{n2}; {r3.stdout.strip()!r}; lô: {lot}")

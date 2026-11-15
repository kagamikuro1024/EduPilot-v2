"""Dùng chung cho gen / load / sheet / measure của kiểm chứng «học từ lần sửa» (docs/sprints/10.6/kiem-chung.md). Chỉ thư viện chuẩn, chỉ HTTP (D46)."""
import json, os, re, ssl, sys, time, urllib.error, urllib.request
from pathlib import Path

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]
DATA = Path(os.environ.get("KC_DATA", HERE / "data"))
API = os.environ.get("API_URL", "https://localhost/api/v1").rstrip("/")
PASSWORD = os.environ.get("SEED_DEFAULT_PASSWORD", "")
DOMAIN = "@edupilot.local"
CLASS = os.environ.get("KC_CLASS", "KC106")
CODE = {"bt-a": "KCA", "bt-b": "KCB"}  # mã nộp bài (submission_code) của hai bài tập
N_STUDENTS, N_B, N_LEARN = 60, 24, 20
CTX = ssl._create_unverified_context() if re.match(r"https://(localhost|127\.0\.0\.1)", API) else None  # chứng chỉ nội bộ của Caddy (dev)
ORIGIN = re.match(r"(https?://[^/]+)", API).group(1)
HO = ["Nguyễn", "Trần", "Lê", "Phạm", "Hoàng", "Vũ"]
TEN = ["An", "Bình", "Châu", "Dũng", "Giang", "Hà", "Khoa", "Linh", "Minh", "Phúc"]


def die(msg, code=1):
    print(msg, file=sys.stderr)
    sys.exit(code)


def student(i):
    """Sinh viên giả thứ i (1..60): mã KC106NNN (MSSV hợp lệ: ≥ 4 chữ số), email kcNNN, tên tổ hợp cố định. Không có người thật."""
    return {"i": i, "code": f"KC106{i:03d}", "key": f"kc{i:03d}", "email": f"kc{i:03d}{DOMAIN}", "name": f"{HO[(i - 1) % 6]} {TEN[(i - 1) // 6]}"}


def bt_students(bt):
    return [student(i) for i in range(1, (N_B if bt == "bt-b" else N_STUDENTS) + 1)]


def measure_keys(bt):
    """kcNNN thuộc tập ĐO: BT-A kc021–kc060 (kc001–kc020 là tập học), BT-B cả 24 bài (đối chứng)."""
    return [s["key"] for s in bt_students(bt) if bt == "bt-b" or s["i"] > N_LEARN]


def call(method, path, token=None, body=None, key=None, ok=(200, 201, 202), headers=None, raw=None, tries=5):
    h = {"Accept": "application/json", "Origin": ORIGIN, **(headers or {})}
    if token:
        h["Authorization"] = f"Bearer {token}"
    if key:
        h["Idempotency-Key"] = re.sub(r"[^A-Za-z0-9._:-]", "-", key)
    data = raw
    if body is not None:
        data = json.dumps(body).encode()
        h["Content-Type"] = "application/json"
    for n in range(tries):
        try:
            with urllib.request.urlopen(urllib.request.Request(API + path, data=data, headers=h, method=method), context=CTX, timeout=120) as r:
                b = r.read()
                return r.status, (json.loads(b) if b else None)
        except urllib.error.HTTPError as e:
            b = e.read()
            try:
                body_j = json.loads(b) if b else None
            except ValueError:
                body_j = {"raw": b[:200].decode("utf8", "replace")}
            if e.code == 429 and n < tries - 1:
                time.sleep(min(30, float(e.headers.get("Retry-After") or 2)))
                continue
            if e.code in ok:
                return e.code, body_j
            die(f"{method} {path} → {e.code} {json.dumps(body_j, ensure_ascii=False)[:300]}")
        except (urllib.error.URLError, ConnectionError, TimeoutError) as e:
            die(f"Gateway chưa chạy ({API}): {e}", 2)


def login(who="teacher"):
    if not PASSWORD:
        die("Thiếu SEED_DEFAULT_PASSWORD")
    _, b = call("POST", "/auth/login", body={"email": who + DOMAIN, "password": PASSWORD})
    return b["access_token"], b["user"]["id"]


def list_all(path, token):
    out, cursor = [], ""
    for _ in range(50):
        sep = "&" if "?" in path else "?"
        _, b = call("GET", f"{path}{sep}limit=100" + (f"&cursor={cursor}" if cursor else ""), token)
        out += b["items"]
        cursor = b.get("next_cursor") or ""
        if not cursor:
            return out
    die(f"Danh sách {path} quá dài")


def wait_job(token, job_id, timeout=900):
    end = time.time() + timeout
    while time.time() < end:
        _, j = call("GET", f"/jobs/{job_id}", token)
        if j["status"] == "SUCCEEDED":
            return j
        if j["status"] == "FAILED":
            die(f"Việc {job_id} lỗi: {j.get('error')}")
        time.sleep(1.5)
    die(f"Việc {job_id} quá {timeout}s")


def find_course(token):
    mine = list_all("/me/courses", token)
    cid = next((m["course"]["id"] for m in mine if m["course"]["class_code"] == CLASS), None)
    if not cid:
        die(f"Chưa có lớp {CLASS}: chạy load.py trước")
    return cid


def find_assignment(token, course, bt):
    a = next((x for x in list_all(f"/courses/{course}/assignments", token) if x["submission_code"] == CODE[bt]), None)
    if not a:
        die(f"Chưa có bài tập {CODE[bt]}: chạy load.py trước")
    return a

"""DEF-R9.5-12: 429 có Retry-After > 60 s dừng lượt ngay (rc=3, resume.json); `--resume` bỏ qua lệnh đã có tệp kết quả.
Chạy: python3 benchmarks/real-llm/test_quota.py"""
import http.server
import json
import os
import subprocess
import sys
import tempfile
import threading
from pathlib import Path


class H(http.server.BaseHTTPRequestHandler):
    def _r(self):
        self.send_response(429)
        self.send_header("Retry-After", "120")
        self.end_headers()
        self.wfile.write(b"{}")

    do_GET = do_POST = _r

    def log_message(self, *a):
        pass


srv = http.server.HTTPServer(("127.0.0.1", 0), H)
threading.Thread(target=srv.serve_forever, daemon=True).start()
out = Path(tempfile.mkdtemp())
env = {**os.environ, "BASE": f"http://127.0.0.1:{srv.server_port}", "OUT": str(out)}
script = str(Path(__file__).with_name("real_llm.py"))
r = subprocess.run([sys.executable, script, "setup"], env=env, capture_output=True, text=True, timeout=30)
assert r.returncode == 3 and "hết hạn mức" in r.stderr, (r.returncode, r.stderr)
assert json.loads((out / "resume.json").read_text())["retry_after_s"] == 120
(out / "state.json").write_text("{}")
r = subprocess.run([sys.executable, script, "--resume", "setup"], env=env, capture_output=True, text=True, timeout=30)
assert r.returncode == 0 and "bỏ qua setup" in r.stdout, (r.returncode, r.stdout, r.stderr)
print("ok")

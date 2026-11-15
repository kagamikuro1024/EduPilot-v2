import json, os, subprocess, time
ROOT = os.path.expanduser("~/Documents/TA_Agent_qc2"); OUT = f"{ROOT}/docs/regression/evidence-9/logs"
env = dict(os.environ, DOCKER_HOST=f"unix://{os.path.expanduser('~')}/.colima/default/docker.sock", TESTCONTAINERS_RYUK_DISABLED="true")
t = time.time(); p = subprocess.run(["go", "test", "-count=1", "-tags", "testroutes", "./internal/contract/..."], cwd=f"{ROOT}/backend-go", env=env, capture_output=True, text=True, timeout=1200)
open(f"{OUT}/contract-testroutes.log", "w").write((p.stdout + p.stderr)[-6000:])
open(f"{OUT}/contract-testroutes.json", "w").write(json.dumps(dict(rc=p.returncode, secs=int(time.time() - t))))

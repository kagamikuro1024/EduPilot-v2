import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
W = "/Users/kuro/.config/edupilot-team/qc3-work/"
def regress(args, envfile=W + "regress-container.env", extra=()):
    cmd = ["docker", "run", "--rm", "--network", "ep-qc_default", "--add-host", "host.docker.internal:host-gateway", "--env-file", envfile, *extra, "-v", W + "regress-linux:/regress:ro", "--entrypoint", "/regress", "edupilot-worker-test", *args]
    return subprocess.run(cmd, capture_output=True, text=True, timeout=280)
def run():
    stub("allow"); redis_del("ep:llm:cb:*")
    t0 = time.time(); r = regress(["run", "--course", CG0, "--limit", "20"]); print("exit", r.returncode, "dt", round(time.time() - t0, 1)); print("STDOUT", r.stdout[:600]); print("STDERR", r.stderr[:300])
    open(W + "regress_run.out", "w").write(r.stdout + "\n--ERR--\n" + r.stderr)
if __name__ == "__main__":
    with window(): run()

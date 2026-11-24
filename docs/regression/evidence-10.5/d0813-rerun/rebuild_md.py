import json,os,sys
os.environ.setdefault("SEED_DEFAULT_PASSWORD","x")
sys.path.insert(0,"benchmarks/decision")
import run
d=json.load(open(sys.argv[1]))
meta=d["meta"]
if hasattr(run,"provider_label"): print("provider_label:",run.provider_label(meta)); print("latency_limit:",run.latency_limit(meta)[:140])
md=run.markdown(d["off"],d["on"],{g:(v["result"],v["why"]) for g,v in d["verdict"].items()},d["paths_on"],meta)
open(sys.argv[2],"w").write(md)
for l in md.splitlines():
    if "provider decision" in l or "log-chuẩn" in l or l.startswith("## Lượt thật") or "Cột «thật»" in l: print("  |",l[:200])

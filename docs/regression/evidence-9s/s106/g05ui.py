import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
tc = sys.argv[1]; W = "/Users/kuro/.config/edupilot-team/qc3-work/"
with window():
    if "12" in tc:
        a1, _, _ = guided("g0512b", 14, 6, 2); sql(f"update assignments set due_at=now()+interval '40 days' where id='{a1}'")
        fx = json.load(open(W + "g05fx.json")); fx["a1"] = a1; json.dump(fx, open(W + "g05fx.json", "w"))
    stub("slow" if "12" in tc else "allow")
    os.system(f'cd {os.path.dirname(os.path.abspath(__file__))} && ./runpw.sh qc-real-106-g05.spec.ts "{tc}" 290')

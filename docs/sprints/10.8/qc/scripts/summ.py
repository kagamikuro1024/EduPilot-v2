import json, collections, re
last = collections.OrderedDict()
for f in ("results-as.jsonl", "results-ui.jsonl", "results-ui2.jsonl"):
    try:
        for l in open(f):
            r = json.loads(l); k = r["tc"].split(" ")[0]; last[k] = r
    except FileNotFoundError: pass
c = collections.Counter(r["st"] for r in last.values())
print(dict(c), len(last))
for k, r in last.items():
    if r["st"] == "FAIL": print("FAIL", r["tc"][:110], "|", r["note"][:140])

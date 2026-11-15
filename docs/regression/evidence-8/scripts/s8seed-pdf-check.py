import re, pypdf, unicodedata
r = pypdf.PdfReader("seed/demo/quy-che-lop2.pdf")
txt = [p.extract_text() or "" for p in r.pages]
print("pages", len(r.pages))
full = "\n".join(txt)
n = unicodedata.normalize
def norm(s): return re.sub(r"\s+"," ",n("NFC",s)).strip()
F = norm(full)
print("làm tròn/quy tròn/bước điểm:", len(re.findall(r"làm tròn|quy tròn|bước điểm", F, re.I)))
md = open("docs/specs/FEAT-gradebook/fixtures/quy-che-lop2.md", encoding="utf8").read()
md = re.sub(r"<!--.*?-->", "", md, flags=re.S)
miss = 0; tot = 0
for ln in md.splitlines():
    ln = ln.strip().lstrip("#").strip()
    if not ln: continue
    ln = re.sub(r"\s*·\s*", " · ", ln)
    tot += 1
    if norm(ln) not in F:
        # try sentence-wise
        parts = [p for p in re.split(r"(?<=[.;])\s+", norm(ln)) if len(p) > 3]
        bad = [p for p in parts if p not in F]
        if bad: miss += 1; print("MISSING:", bad[:2])
print("lines", tot, "missing", miss)

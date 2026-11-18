# PDF tối giản (Helvetica, ASCII) cho QC: minipdf.make(["dòng 1","dòng 2",...], pages=1) -> bytes
def make(lines, pages=1, per_page=None):
    per_page = per_page or max(1, -(-len(lines) // pages)); chunks = [lines[i:i + per_page] for i in range(0, len(lines), per_page)] or [[]]
    objs = []; kids = []
    def esc(s): return s.encode("ascii", "replace").decode().replace("\\", "\\\\").replace("(", "\\(").replace(")", "\\)")
    font_id = 3 + 2 * len(chunks)
    for i, ch in enumerate(chunks):
        pid, cid = 3 + 2 * i, 4 + 2 * i; kids.append(pid)
        content = "BT /F1 11 Tf 50 780 Td 14 TL " + " ".join(f"({esc(l)}) Tj T*" for l in ch) + " ET"
        objs.append((pid, f"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents {cid} 0 R /Resources << /Font << /F1 {font_id} 0 R >> >> >>"))
        objs.append((cid, f"<< /Length {len(content)} >>\nstream\n{content}\nendstream"))
    objs.append((font_id, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"))
    out = ["%PDF-1.4\n"]; offs = {}
    def add(i, body): offs[i] = sum(len(x.encode("latin-1")) for x in out); out.append(f"{i} 0 obj\n{body}\nendobj\n")
    add(1, "<< /Type /Catalog /Pages 2 0 R >>"); add(2, f"<< /Type /Pages /Kids [{' '.join(f'{k} 0 R' for k in kids)}] /Count {len(kids)} >>")
    for i, b in objs: add(i, b)
    n = max(offs) + 1; xr = sum(len(x.encode("latin-1")) for x in out)
    out.append(f"xref\n0 {n}\n0000000000 65535 f \n" + "".join(f"{offs[i]:010d} 00000 n \n" for i in range(1, n)) + f"trailer\n<< /Size {n} /Root 1 0 R >>\nstartxref\n{xr}\n%%EOF\n")
    return "".join(out).encode("latin-1")

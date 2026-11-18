#!/usr/bin/env python3
"""Xoá dòng ca LỖI THỜI (có NGUỒN quyết định P-xx/D-xx/commit) khỏi smoke-<N>.md; ghi mã đã xoá + nguồn vào cuối defects-<N>.md.
Dùng: python3 env/delete_lt.py <N> [--dry]. Dòng không có nguồn: giữ nguyên, chuyển thành N (chờ test). K/X/S/M/CM/T → N.
Cột trạng thái chấp nhận: LT, D, T (lỗi thời); K, Kb, Ka, Kc, X, S, M, CM (→ N)."""
import re, sys
n = sys.argv[1]; dry = "--dry" in sys.argv
p = f"smoke-{n}.md"; L = open(p).read().split("\n")
src = re.compile(r"P-\d+-?\w*|\bD\d{2,3}\b|[0-9a-f]{7,8}\b|\b#\d+\b")
out, deleted, kept_noSrc = [], [], []
for l in L:
    m = re.match(r"\| (\S[^|]*?) \| (LT|D|T|K[abc]?|X|S|M|CM) \| (.*) \|$", l)
    if not m: out.append(l); continue
    code, st, note = m.groups()
    if st in ("LT", "D", "T"):
        if src.search(note): deleted.append((code, note)); continue
        out.append(f"| {code} | N | chưa chạy (đã ghi lỗi thời nhưng KHÔNG có nguồn quyết định → phải chạy): {note[:140]} |"); kept_noSrc.append(code)
    else:
        out.append(f"| {code} | N | chưa chạy (trước đây {st}: {note[:130]}) |")
print(p, "xoá", len(deleted), "giữ chờ test (không nguồn)", len(kept_noSrc))
if not dry:
    open(p, "w").write("\n".join(out))
    d = f"defects-{n}.md"
    with open(d, "a") as f:
        f.write("\n\n## Ca lỗi thời đã XOÁ khỏi smoke (lệnh chủ dự án; mỗi mã một dòng + nguồn)\n")
        for c, nt in deleted: f.write(f"- {c}: {nt[:200]}\n")

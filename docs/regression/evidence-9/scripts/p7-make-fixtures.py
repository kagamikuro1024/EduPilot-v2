#!/usr/bin/env python3
"""QC P7 — sinh tệp thử cho TC-P7-03/04/05 vào /tmp/qc-p7/ :
  zip-ok.zip (3 tệp tên chứa MSSV) · zip-evil.zip (thư mục, __MACOSX, .DS_Store, ../, tuyệt đối, ký tự điều khiển, zip lồng, .exe, 0 byte, 1 mục hợp lệ)
  zip-201.zip (201 mục) · zip-bomb.zip (1 mục giải nén 300 MiB) · zip-header-lie.zip (header khai nhỏ)
  files/: a.pdf (%PDF-), bad-mime.pdf (MZ…), a.docx (central directory ở CUỐI), fake.docx, empty.pdf, a.md, nul.txt, 20mib.pdf, 20mib-plus1.pdf
  eml/: 9 thư (TC-P7-05-12: Authentication-Results a–i) + auto-reply + no-attachment + two-attachments
Dùng: python3 p7-make-fixtures.py   (không cần stack)"""
import io, os, struct, zipfile
from email.message import EmailMessage
OUT = "/tmp/qc-p7"; os.makedirs(f"{OUT}/files", exist_ok=True); os.makedirs(f"{OUT}/eml", exist_ok=True)
TXT = "Tác nhân là nhóm APT29. Họ cấy SUNBURST vào bản cập nhật Orion.\n" * 3
def zw(name, entries, comp=zipfile.ZIP_DEFLATED):
    with zipfile.ZipFile(f"{OUT}/{name}", "w", comp) as z:
        for n, d in entries: z.writestr(n, d)
zw("zip-ok.zip", [(f"20229001_{i}.txt", TXT) for i in (1, 2, 3)])
inner = io.BytesIO(); zipfile.ZipFile(inner, "w").writestr("x.txt", TXT)
zw("zip-evil.zip", [("dir/", ""), ("__MACOSX/._a.txt", "x"), (".DS_Store", "x"), ("../../etc/passwd.txt", TXT), ("/abs/abs.txt", TXT),
                    ("ctl\x01.txt", TXT), ("nested.zip", inner.getvalue()), ("run.exe", "MZ"), ("empty.txt", ""), ("20229002_ok.txt", TXT)])
zw("zip-201.zip", [(f"20229{i:03d}.txt", TXT) for i in range(201)])
zw("zip-bomb.zip", [("20229003_big.txt", b"A" * (300 * 1024 * 1024))])
# header khai nhỏ: ghi 5 MiB nhưng sửa file_size trong central directory về 10 byte
buf = io.BytesIO()
with zipfile.ZipFile(buf, "w", zipfile.ZIP_DEFLATED) as z: z.writestr("20229004_lie.txt", b"B" * (5 * 1024 * 1024))
raw = bytearray(buf.getvalue()); i = raw.rfind(b"PK\x01\x02"); struct.pack_into("<I", raw, i + 24, 10); open(f"{OUT}/zip-header-lie.zip", "wb").write(raw)
F = lambda n, b: open(f"{OUT}/files/{n}", "wb").write(b)
F("a.pdf", b"%PDF-1.4\n" + TXT.encode() + b"\n%%EOF"); F("bad-mime.pdf", b"MZ\x90\x00" + b"x" * 100); F("empty.pdf", b"")
F("a.md", TXT.encode()); F("nul.txt", b"abc\x00def"); F("20mib.pdf", b"%PDF-" + b"0" * (20 * 1024 * 1024 - 5)); F("20mib-plus1.pdf", b"%PDF-" + b"0" * (20 * 1024 * 1024 - 4))
F("fake.docx", b"not a zip")
b = io.BytesIO()
with zipfile.ZipFile(b, "w", zipfile.ZIP_STORED) as z:
    z.writestr("word/document.xml", "<w:document/>"); z.writestr("[Content_Types].xml", "<Types/>")   # [Content_Types].xml ở CUỐI
F("a.docx", b.getvalue())
def eml(name, ar=None, frm="sv07@edupilot.local", subj="[BT03] 20229007", att=1, auto=False, extra=()):
    m = EmailMessage(); m["From"] = frm; m["To"] = "lop@edupilot.local"; m["Subject"] = subj
    if auto: m["Auto-Submitted"] = "auto-replied"
    for a in ar or (): m["Authentication-Results"] = a
    for k, v in extra: m[k] = v
    m.set_content("bài làm đính kèm")
    for i in range(att): m.add_attachment(TXT.encode(), maintype="application", subtype="octet-stream", filename=f"bai{i}.md")
    open(f"{OUT}/eml/{name}.eml", "wb").write(bytes(m))
T = "mail.edupilot.local"  # = IMAP_TRUSTED_AUTHSERV_ID dùng khi test
eml("a-pass", [f"{T}; dmarc=pass header.from=edupilot.local"]); eml("b-none")
eml("c-foreign-authserv", ["evil.example; dmarc=pass header.from=edupilot.local"])
eml("d-dmarc-fail", [f"{T}; dmarc=fail header.from=edupilot.local"]); eml("e-dmarc-none", [f"{T}; dmarc=none header.from=edupilot.local"])
eml("f-two-headers", [f"{T}; dmarc=fail header.from=edupilot.local", f"{T}; dmarc=pass header.from=edupilot.local"])
eml("g-spoof-above-real", ["attacker.example; dmarc=pass header.from=edupilot.local", f"{T}; dmarc=fail header.from=edupilot.local"])  # header giả (authserv lạ) nằm TRÊN header thật → phải SENDER_UNVERIFIED
eml("i-from-domain-mismatch", [f"{T}; dmarc=pass header.from=other.example"])
eml("auto-reply", [f"{T}; dmarc=pass header.from=edupilot.local"], auto=True); eml("no-attachment", [f"{T}; dmarc=pass header.from=edupilot.local"], att=0)
eml("two-attachments", [f"{T}; dmarc=pass header.from=edupilot.local"], att=2); eml("wrong-subject", [f"{T}; dmarc=pass header.from=edupilot.local"], subj="Bài tập 3 của em")
print("OK →", OUT); print(*sorted(os.listdir(OUT)), sep="\n")

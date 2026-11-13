# Nén PNG của shots/ về bảng 256 màu (giữ nét chữ; ảnh giao diện phẳng nên mất rất ít). Chạy: python3 compress.py
import os, sys
from PIL import Image
D = os.path.join(os.path.dirname(os.path.abspath(__file__)), "../shots")
before = after = 0
for f in sorted(os.listdir(D)):
    if not f.endswith(".png"): continue
    p = os.path.join(D, f)
    n0 = os.path.getsize(p); before += n0
    im = Image.open(p)
    if im.mode == "P": after += n0; continue  # đã nén
    q = im.convert("RGB").quantize(colors=256, method=Image.Quantize.MEDIANCUT, dither=Image.Dither.NONE)
    q.save(p, optimize=True)
    after += os.path.getsize(p)
print(f"{before/1e6:.1f} MB → {after/1e6:.1f} MB")

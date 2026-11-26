# THM-08: tỷ lệ tương phản WCAG từ ẢNH CHỤP (điểm ảnh chữ vs nền) và từ CSS tính toán.
from PIL import Image
from collections import Counter
def lum(c):
    def f(v):
        v /= 255; return v/12.92 if v <= .03928 else ((v+.055)/1.055)**2.4
    r, g, b = c[:3]; return .2126*f(r) + .7152*f(g) + .0722*f(b)
def ratio(a, b):
    la, lb = lum(a), lum(b); hi, lo = max(la, lb), min(la, lb); return (hi+.05)/(lo+.05)
def region_ratio(png, box):
    """box = (x0,y0,x1,y1) quanh dòng chữ: nền = màu phổ biến nhất; chữ = điểm ảnh xa nền nhất (đậm nhất)."""
    im = Image.open(png).convert("RGB").crop(box); px = list(im.getdata())
    bg = Counter(px).most_common(1)[0][0]
    fg = max(px, key=lambda p: ratio(p, bg)); return bg, fg, round(ratio(bg, fg), 2)

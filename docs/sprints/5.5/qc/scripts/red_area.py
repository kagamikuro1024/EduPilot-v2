import sys, glob, numpy as np
from PIL import Image
def srgb2lab(rgb):
    c = rgb / 255.0; c = np.where(c <= .04045, c / 12.92, ((c + .055) / 1.055) ** 2.4)
    M = np.array([[.4124564, .3575761, .1804375], [.2126729, .7151522, .0721750], [.0193339, .1191920, .9503041]]); xyz = c @ M.T
    xyz /= np.array([.95047, 1, 1.08883]); f = np.where(xyz > .008856, np.cbrt(xyz), 7.787 * xyz + 16 / 116)
    return np.stack([116 * f[..., 1] - 16, 500 * (f[..., 0] - f[..., 1]), 200 * (f[..., 1] - f[..., 2])], -1)
def de2000(l1, l2):
    L1, a1, b1 = l1[..., 0], l1[..., 1], l1[..., 2]; L2, a2, b2 = l2[..., 0], l2[..., 1], l2[..., 2]
    C1 = np.hypot(a1, b1); C2 = np.hypot(a2, b2); Cb = (C1 + C2) / 2; G = .5 * (1 - np.sqrt(Cb ** 7 / (Cb ** 7 + 25 ** 7)))
    a1p = (1 + G) * a1; a2p = (1 + G) * a2; C1p = np.hypot(a1p, b1); C2p = np.hypot(a2p, b2)
    h1 = np.degrees(np.arctan2(b1, a1p)) % 360; h2 = np.degrees(np.arctan2(b2, a2p)) % 360
    dL = L2 - L1; dC = C2p - C1p; dh = h2 - h1; dh = np.where(dh > 180, dh - 360, np.where(dh < -180, dh + 360, dh)); dH = 2 * np.sqrt(C1p * C2p) * np.sin(np.radians(dh / 2))
    Lb = (L1 + L2) / 2; Cpb = (C1p + C2p) / 2; hs = h1 + h2; hb = np.where(np.abs(h1 - h2) > 180, (hs + 360) / 2, hs / 2)
    T = 1 - .17 * np.cos(np.radians(hb - 30)) + .24 * np.cos(np.radians(2 * hb)) + .32 * np.cos(np.radians(3 * hb + 6)) - .2 * np.cos(np.radians(4 * hb - 63))
    dth = 30 * np.exp(-(((hb - 275) / 25) ** 2)); Rc = 2 * np.sqrt(Cpb ** 7 / (Cpb ** 7 + 25 ** 7)); Sl = 1 + .015 * (Lb - 50) ** 2 / np.sqrt(20 + (Lb - 50) ** 2); Sc = 1 + .045 * Cpb; Sh = 1 + .015 * Cpb * T; Rt = -np.sin(np.radians(2 * dth)) * Rc
    return np.sqrt((dL / Sl) ** 2 + (dC / Sc) ** 2 + (dH / Sh) ** 2 + Rt * (dC / Sc) * (dH / Sh))
if __name__ == '__main__':
    red = np.array([float(x) for x in sys.argv[1].split(',')]); rl = srgb2lab(red.reshape(1, 3))[0]
    for p in sorted(glob.glob(sys.argv[2])):
        im = np.asarray(Image.open(p).convert('RGB')).astype(float); lab = srgb2lab(im); d = de2000(lab, np.broadcast_to(rl, lab.shape)); print('%.2f%%' % (100 * (d <= 10).mean()), p.split('/')[-1])

import type { Page } from "@playwright/test";

/**
 * Diện tích "đỏ" của viewport hiện tại, theo quy tắc máy TLR-9 của sprint 5.5: điểm ảnh có ΔE2000 ≤ 10 so với `--ep-red` (sRGB đã giải) là đỏ;
 * KHÔNG tính `--ep-red-soft`; mẫu số = toàn ảnh. Giải mã PNG và tính ΔE ngay trong trang (canvas) nên không cần thêm thư viện.
 */
export async function redAreaPct(page: Page): Promise<number> {
  const png = (await page.screenshot()).toString("base64");
  return page.evaluate(async (b64) => {
    const probe = document.createElement("span");
    document.body.appendChild(probe);
    probe.style.color = "var(--ep-red)";
    const cv = document.createElement("canvas");
    cv.width = cv.height = 1;
    const c1 = cv.getContext("2d", { willReadFrequently: true })!;
    c1.fillStyle = "#000";
    c1.fillStyle = getComputedStyle(probe).color;
    c1.fillRect(0, 0, 1, 1);
    const [r0, g0, b0] = c1.getImageData(0, 0, 1, 1).data;
    probe.remove();
    const lin = (v: number) => { const c = v / 255; return c <= 0.04045 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4); };
    const lab = (r: number, g: number, b: number): [number, number, number] => {
      const R = lin(r), G = lin(g), B = lin(b);
      const x = (0.4124564 * R + 0.3575761 * G + 0.1804375 * B) / 0.95047, y = 0.2126729 * R + 0.7151522 * G + 0.072175 * B, z = (0.0193339 * R + 0.119192 * G + 0.9503041 * B) / 1.08883;
      const f = (t: number) => (t > 216 / 24389 ? Math.cbrt(t) : (24389 / 27 * t + 16) / 116);
      return [116 * f(y) - 16, 500 * (f(x) - f(y)), 200 * (f(y) - f(z))];
    };
    const d2r = Math.PI / 180;
    const de2000 = (a: number[], b: number[]) => {
      const [L1, a1, b1] = a, [L2, a2, b2] = b;
      const C1 = Math.hypot(a1, b1), C2 = Math.hypot(a2, b2), Cb = (C1 + C2) / 2;
      const G = 0.5 * (1 - Math.sqrt(Cb ** 7 / (Cb ** 7 + 25 ** 7)));
      const ap1 = (1 + G) * a1, ap2 = (1 + G) * a2, Cp1 = Math.hypot(ap1, b1), Cp2 = Math.hypot(ap2, b2);
      const hp = (bb: number, aa: number) => (bb === 0 && aa === 0 ? 0 : ((Math.atan2(bb, aa) / d2r) + 360) % 360);
      const h1 = hp(b1, ap1), h2 = hp(b2, ap2);
      const dL = L2 - L1, dC = Cp2 - Cp1;
      let dh = 0;
      if (Cp1 * Cp2 !== 0) { dh = h2 - h1; if (dh > 180) dh -= 360; else if (dh < -180) dh += 360; }
      const dH = 2 * Math.sqrt(Cp1 * Cp2) * Math.sin((dh * d2r) / 2);
      const Lb = (L1 + L2) / 2, Cpb = (Cp1 + Cp2) / 2;
      let hb = h1 + h2;
      if (Cp1 * Cp2 === 0) hb = h1 + h2; else if (Math.abs(h1 - h2) <= 180) hb = (h1 + h2) / 2; else hb = (h1 + h2 + (h1 + h2 < 360 ? 360 : -360)) / 2;
      const T = 1 - 0.17 * Math.cos((hb - 30) * d2r) + 0.24 * Math.cos(2 * hb * d2r) + 0.32 * Math.cos((3 * hb + 6) * d2r) - 0.2 * Math.cos((4 * hb - 63) * d2r);
      const dTheta = 30 * Math.exp(-(((hb - 275) / 25) ** 2));
      const Rc = 2 * Math.sqrt(Cpb ** 7 / (Cpb ** 7 + 25 ** 7));
      const Sl = 1 + (0.015 * (Lb - 50) ** 2) / Math.sqrt(20 + (Lb - 50) ** 2), Sc = 1 + 0.045 * Cpb, Sh = 1 + 0.015 * Cpb * T;
      const Rt = -Math.sin(2 * dTheta * d2r) * Rc;
      return Math.sqrt((dL / Sl) ** 2 + (dC / Sc) ** 2 + (dH / Sh) ** 2 + Rt * (dC / Sc) * (dH / Sh));
    };
    const img = new Image();
    img.src = "data:image/png;base64," + b64;
    await img.decode();
    const cvs = document.createElement("canvas");
    cvs.width = img.naturalWidth;
    cvs.height = img.naturalHeight;
    const ctx = cvs.getContext("2d", { willReadFrequently: true })!;
    ctx.drawImage(img, 0, 0);
    const { data } = ctx.getImageData(0, 0, cvs.width, cvs.height);
    const ref = lab(r0, g0, b0);
    const cache = new Map<number, boolean>();
    let red = 0;
    for (let i = 0; i < data.length; i += 4) {
      const key = (data[i] << 16) | (data[i + 1] << 8) | data[i + 2];
      let hit = cache.get(key);
      if (hit === undefined) { hit = de2000(lab(data[i], data[i + 1], data[i + 2]), ref) <= 10; cache.set(key, hit); }
      if (hit) red++;
    }
    return (red / (data.length / 4)) * 100;
  }, png);
}

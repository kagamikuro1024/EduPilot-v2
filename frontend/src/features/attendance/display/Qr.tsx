"use client";

import { encode } from "uqr";
import { useMemo } from "react";

/** QR bằng `uqr` (ECC L, vùng yên 4 mô-đun, DESIGN §14.8.1). SVG một đường dẫn: mỗi dải mô-đun đen liền nhau là một nét ngang. */
export function Qr({ text, label }: { text: string; label: string }) {
  const { size, path } = useMemo(() => {
    const qr = encode(text, { ecc: "L", border: 4 });
    let d = "";
    qr.data.forEach((row, y) => {
      for (let x = 0; x < row.length; ) {
        if (!row[x]) {
          x++;
          continue;
        }
        let w = 1;
        while (x + w < row.length && row[x + w]) w++;
        d += `M${x} ${y}h${w}v1h-${w}z`;
        x += w;
      }
    });
    return { size: qr.size, path: d };
  }, [text]);
  return (
    <svg viewBox={`0 0 ${size} ${size}`} role="img" aria-label={label} shapeRendering="crispEdges" width="100%" height="100%">
      <rect width={size} height={size} fill="var(--ep-surface)" />
      <path d={path} fill="var(--ep-ink)" />
    </svg>
  );
}

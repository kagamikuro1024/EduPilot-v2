"use client";

import { useState, type ReactNode } from "react";
import s from "./Chart.module.css";

/**
 * Biểu đồ xu hướng gọn bằng SVG (DESIGN.md §14.25, 04-AC12): trục giá trị 3 mốc có số, mỗi điểm có
 * nhãn đọc được khi rê chuột và khi focus bàn phím, điểm cuối luôn ghi số, dưới biểu đồ là cao nhất / thấp nhất.
 */
export function TrendChart({
  points,
  label,
  format = (v) => String(v),
  height = 150,
  tone = "ink",
  goal,
}: {
  points: Array<{ x: string; full?: string; y: number }>;
  label: string;
  format?: (v: number) => string;
  height?: number;
  tone?: "ink" | "red" | "green";
  goal?: { y: number; label: string };
}) {
  const [active, setActive] = useState<number | null>(null);
  const w = 560;
  const pad = { t: 18, r: 12, b: 24, l: 48 };
  const ys = points.map((p) => p.y).concat(goal ? [goal.y] : []);
  // trục giá trị bắt đầu từ 0 để ba mốc 0 · nửa max · max đọc được ngay
  const max = Math.max(1, ...ys);
  const x = (i: number) => pad.l + (i * (w - pad.l - pad.r)) / Math.max(1, points.length - 1);
  const y = (v: number) => pad.t + (1 - v / max) * (height - pad.t - pad.b);
  const d = points.map((p, i) => `${i ? "L" : "M"}${x(i).toFixed(1)},${y(p.y).toFixed(1)}`).join(" ");
  const last = points[points.length - 1];
  const hi = Math.max(...points.map((p) => p.y));
  const lo = Math.min(...points.map((p) => p.y));

  return (
    <figure className={s.figure}>
      <svg viewBox={`0 0 ${w} ${height}`} className={[s.svg, s[tone]].join(" ")} role="img" aria-label={label}>
        <line x1={pad.l} x2={w - pad.r} y1={height - pad.b} y2={height - pad.b} className={s.axis} />
        {[0, Math.round(max / 2), max].map((v) => (
          <text key={v} x={0} y={y(v) + 4} className={s.tick} data-part="chart-axis-label">
            {v}
          </text>
        ))}
        {goal && (
          <>
            <line x1={pad.l} x2={w - pad.r} y1={y(goal.y)} y2={y(goal.y)} className={s.goal} />
            <text x={w - pad.r} y={y(goal.y) - 4} textAnchor="end" className={s.goalText}>
              {goal.label}
            </text>
          </>
        )}
        <path d={d} className={s.line} vectorEffect="non-scaling-stroke" />
        {points.map((p, i) => (
          <circle
            key={p.x}
            data-part="chart-point"
            className={s.point}
            cx={x(i)}
            cy={y(p.y)}
            r={4}
            tabIndex={0}
            role="img"
            aria-label={`${p.full ?? p.x}: ${format(p.y)}`}
            onMouseEnter={() => setActive(i)}
            onMouseLeave={() => setActive((cur) => (cur === i ? null : cur))}
            onFocus={() => setActive(i)}
            onBlur={() => setActive((cur) => (cur === i ? null : cur))}
          />
        ))}
        {last && (
          <text x={w - pad.r} y={y(last.y) - 10} textAnchor="end" className={s.lastValue}>
            {format(last.y)}
          </text>
        )}
        {active !== null && (
          <text x={x(active)} y={y(points[active].y) - 12} textAnchor={active === 0 ? "start" : active === points.length - 1 ? "end" : "middle"} className={s.pointLabel}>
            {`${points[active].full ?? points[active].x}: ${format(points[active].y)}`}
          </text>
        )}
        {points.map((p, i) => (
          <text key={`x-${p.x}`} x={x(i)} y={height - 6} textAnchor={i === 0 ? "start" : i === points.length - 1 ? "end" : "middle"} className={s.tick}>
            {p.x}
          </text>
        ))}
      </svg>
      <figcaption className={s.caption}>{`Cao nhất: ${format(hi)} · Thấp nhất: ${format(lo)}`}</figcaption>
      <table className="ep-sr-only">
        <caption>{label}</caption>
        <tbody>
          {points.map((p) => (
            <tr key={p.x}>
              <th scope="row">{p.full ?? p.x}</th>
              <td>{format(p.y)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </figure>
  );
}

/** Cột ngang để so sánh vài mục có nhãn; giá trị in rõ bên phải. */
export function BarList({ items, format = (v) => String(v), max }: { items: Array<{ label: ReactNode; value: number; tone?: "ink" | "red" | "amber" | "green" }>; format?: (v: number) => string; max?: number }) {
  const top = max ?? Math.max(1, ...items.map((i) => i.value));
  return (
    <ul className={s.bars}>
      {items.map((it, i) => (
        <li key={i} className={s.barRow}>
          <span className={s.barLabel}>{it.label}</span>
          <span className={s.barTrack} aria-hidden>
            <span className={[s.barFill, s[`b_${it.tone ?? "ink"}`]].join(" ")} style={{ width: `${(it.value / top) * 100}%` }} />
          </span>
          <span className={s.barValue}>{format(it.value)}</span>
        </li>
      ))}
    </ul>
  );
}

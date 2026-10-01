import type { ReactNode } from "react";
import s from "./Chart.module.css";

/**
 * Biểu đồ xu hướng gọn bằng SVG: chỉ dùng khi xu hướng / so sánh là điều cần thấy (DESIGN.md §14.25).
 * Có nhãn trục X, giá trị cuối, và bảng dữ liệu ẩn cho trình đọc màn hình.
 */
export function TrendChart({
  points,
  label,
  format = (v) => String(v),
  height = 120,
  tone = "ink",
  goal,
}: {
  points: Array<{ x: string; y: number }>;
  label: string;
  format?: (v: number) => string;
  height?: number;
  tone?: "ink" | "red" | "green";
  goal?: { y: number; label: string };
}) {
  const w = 560;
  const pad = { t: 12, r: 12, b: 24, l: 8 };
  const ys = points.map((p) => p.y).concat(goal ? [goal.y] : []);
  const min = Math.min(...ys);
  const max = Math.max(...ys);
  const span = max - min || 1;
  const x = (i: number) => pad.l + (i * (w - pad.l - pad.r)) / Math.max(1, points.length - 1);
  const y = (v: number) => pad.t + (1 - (v - min) / span) * (height - pad.t - pad.b);
  const d = points.map((p, i) => `${i ? "L" : "M"}${x(i).toFixed(1)},${y(p.y).toFixed(1)}`).join(" ");
  const last = points[points.length - 1];

  return (
    <figure className={s.figure}>
      <svg viewBox={`0 0 ${w} ${height}`} className={[s.svg, s[tone]].join(" ")} role="img" aria-label={label}>
        <line x1={pad.l} x2={w - pad.r} y1={height - pad.b} y2={height - pad.b} className={s.axis} />
        {goal && (
          <>
            <line x1={pad.l} x2={w - pad.r} y1={y(goal.y)} y2={y(goal.y)} className={s.goal} />
            <text x={w - pad.r} y={y(goal.y) - 4} textAnchor="end" className={s.goalText}>
              {goal.label}
            </text>
          </>
        )}
        <path d={d} className={s.line} vectorEffect="non-scaling-stroke" />
        {last && <circle cx={x(points.length - 1)} cy={y(last.y)} r={3.5} className={s.dot} />}
        {points.map((p, i) => (
          <text key={p.x} x={x(i)} y={height - 6} textAnchor={i === 0 ? "start" : i === points.length - 1 ? "end" : "middle"} className={s.tick}>
            {p.x}
          </text>
        ))}
      </svg>
      <table className="ep-sr-only">
        <caption>{label}</caption>
        <tbody>
          {points.map((p) => (
            <tr key={p.x}>
              <th scope="row">{p.x}</th>
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

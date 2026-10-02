// Red Thread Transition (DESIGN.md §11): chọn việc kế tiếp ở "Hôm nay" thì một vạch đỏ mảnh
// chạy từ hàng đã chọn tới tiêu đề trang đích, < 300 ms, tắt khi prefers-reduced-motion.

const KEY = "ep-red-thread";
const TTL_MS = 1500;
const DURATION_MS = 280;

type Origin = { x: number; y: number; w: number; t: number };

/** Gọi khi người dùng bấm một hàng "việc kế tiếp" (trước khi điều hướng). */
export function markRedThread(el: Element) {
  const r = el.getBoundingClientRect();
  const origin: Origin = { x: r.left, y: r.bottom - 1, w: Math.min(r.width, 320), t: Date.now() };
  try {
    sessionStorage.setItem(KEY, JSON.stringify(origin));
  } catch {
    /* chế độ riêng tư: bỏ qua chuyển động */
  }
}

/** Gọi một lần khi tiêu đề trang đích đã gắn vào DOM. */
export function consumeRedThread(target: Element) {
  let origin: Origin | null = null;
  try {
    const raw = sessionStorage.getItem(KEY);
    sessionStorage.removeItem(KEY);
    origin = raw ? (JSON.parse(raw) as Origin) : null;
  } catch {
    return;
  }
  if (!origin || Date.now() - origin.t > TTL_MS) return;
  if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;

  const to = target.getBoundingClientRect();
  const line = document.createElement("div");
  line.setAttribute("aria-hidden", "true");
  Object.assign(line.style, {
    position: "fixed",
    left: "0px",
    top: "0px",
    height: "2px",
    width: "1px",
    background: "var(--ep-red)",
    zIndex: "var(--ep-z-dialog)",
    pointerEvents: "none",
    transformOrigin: "0 0",
  } satisfies Partial<CSSStyleDeclaration>);
  document.body.appendChild(line);

  const ty = to.bottom + 6;
  const tw = Math.min(to.width, 240);
  const anim = line.animate(
    [
      { transform: `translate(${origin.x}px, ${origin.y}px) scaleX(${origin.w})`, opacity: 1 },
      { transform: `translate(${to.left}px, ${ty}px) scaleX(${tw})`, opacity: 1, offset: 0.75 },
      { transform: `translate(${to.left}px, ${ty}px) scaleX(${tw})`, opacity: 0 },
    ],
    { duration: DURATION_MS, easing: "cubic-bezier(.16, 1, .3, 1)", fill: "forwards" },
  );
  anim.onfinish = () => line.remove();
}

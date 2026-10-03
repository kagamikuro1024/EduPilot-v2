const MAX_LEN = 512;
const CONTROL = /[\u0000-\u001f\u007f]/;

/**
 * Chỉ chấp nhận đường dẫn NỘI BỘ làm đích sau đăng nhập (chặn open redirect, US-P2-02 AC13): bắt đầu bằng đúng một `/`,
 * ký tự thứ hai không phải `/` hay `\`, không ký tự điều khiển, không `%2F%2F` / `%5C`, không `:` trước `?`/`#`, ≤ 512 ký tự.
 * Mọi giá trị khác (kể cả thiếu) → `/`.
 */
export function safeNext(raw: string | null | undefined): string {
  if (!raw || raw.length > MAX_LEN || !raw.startsWith("/")) return "/";
  if (raw[1] === "/" || raw[1] === "\\" || CONTROL.test(raw)) return "/";
  const lower = raw.toLowerCase();
  if (lower.includes("%2f%2f") || lower.includes("%5c") || lower.includes("%00")) return "/";
  const path = raw.split(/[?#]/, 1)[0];
  if (path.includes(":") || path.includes("\\")) return "/";
  return raw;
}

// WebView Android cũ (minSdk 24; AVD API 28 = Chrome 66) thiếu hai API hiện đại mà mã gọi trực tiếp.
// Giữ hành vi y hệt bản mới khi có; chỉ dùng API cũ (Chrome 51+) khi thiếu — không thêm thư viện polyfill.

/** `crypto.randomUUID()` (Chrome 92+); bản cũ: 16 byte ngẫu nhiên theo RFC 4122 v4. */
export const newUuid = (): string => {
  const c = globalThis.crypto;
  if (typeof c.randomUUID === "function") return c.randomUUID();
  const b = c.getRandomValues(new Uint8Array(16));
  b[6] = (b[6] & 0x0f) | 0x40;
  b[8] = (b[8] & 0x3f) | 0x80;
  const h = Array.from(b, (x) => x.toString(16).padStart(2, "0")).join("");
  return `${h.slice(0, 8)}-${h.slice(8, 12)}-${h.slice(12, 16)}-${h.slice(16, 20)}-${h.slice(20)}`;
};

/** `AbortSignal.timeout(ms)` (Chrome 103+); bản cũ: AbortController + hẹn giờ, cùng `name === "TimeoutError"`. */
export const timeoutSignal = (ms: number): AbortSignal => {
  if (typeof AbortSignal.timeout === "function") return AbortSignal.timeout(ms);
  const ac = new AbortController();
  setTimeout(() => ac.abort(new DOMException("signal timed out", "TimeoutError")), ms);
  return ac.signal;
};

// `Object.fromEntries` (Chrome 73+) — core-js đi kèm bundle KHÔNG có mục này (đã kiểm trong chunk polyfill), mà mã ta và thư viện đều dùng.
// Vá ở phạm vi toàn cục một lần (module này được nạp sớm qua apiClient/authSession) thay vì sửa từng chỗ gọi và không phủ được node_modules.
if (typeof Object.fromEntries !== "function") {
  Object.fromEntries = function fromEntries(entries: Iterable<readonly [PropertyKey, unknown]>) {
    const out: Record<PropertyKey, unknown> = {};
    for (const [k, v] of entries) out[k] = v;
    return out;
  } as typeof Object.fromEntries;
}

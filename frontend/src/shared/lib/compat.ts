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

/** Gộp nhiều `AbortSignal` (Chrome 116+): dừng khi BẤT KỲ tín hiệu nào dừng, giữ `reason` y hệt bản mới. */
export const anySignal = (signals: AbortSignal[]): AbortSignal => {
  if (typeof AbortSignal.any === "function") return AbortSignal.any(signals);
  const ac = new AbortController();
  for (const sig of signals) {
    if (sig.aborted) {
      ac.abort((sig as AbortSignal & { reason?: unknown }).reason);
      break;
    }
    sig.addEventListener("abort", () => ac.abort((sig as AbortSignal & { reason?: unknown }).reason), { once: true });
  }
  return ac.signal;
};

// Các API Chrome 69–98 mà mã ta / thư viện gọi trực tiếp TRÊN ĐƯỜNG DỰNG KHUNG + PHIÊN (WebView Android API 28 = Chrome 66):
// thiếu chúng thì khung app ném lỗi lúc hydrate (React báo lỗi hydrate) và không có lời gọi /api/v1 nào. Vá một lần ở phạm vi toàn cục,
// cùng chỗ với `Object.fromEntries` ở trên; tệp này được nạp sớm qua `apiClient` / `authSession` và qua `app/layout.tsx`.
/* eslint-disable @typescript-eslint/no-explicit-any */
if (typeof Array.prototype.flat !== "function") {
  Array.prototype.flat = function flat(this: any[], depth: number = 1): any[] {
    const out: any[] = [];
    const walk = (xs: any[], d: number) => xs.forEach((x) => (d > 0 && Array.isArray(x) ? walk(x, d - 1) : out.push(x)));
    walk(this, Math.trunc(depth));
    return out;
  } as typeof Array.prototype.flat;
}
if (typeof Array.prototype.flatMap !== "function") {
  Array.prototype.flatMap = function flatMap(this: any[], fn: (v: any, i: number, a: any[]) => unknown): any[] {
    return (Array.prototype.flat as any).call(this.map(fn), 1);
  } as typeof Array.prototype.flatMap;
}
if (typeof Array.prototype.at !== "function") {
  Array.prototype.at = function at(this: any[], i: number) {
    const n = this.length;
    const k = Math.trunc(i) < 0 ? n + Math.trunc(i) : Math.trunc(i);
    return k >= 0 && k < n ? this[k] : undefined;
  } as typeof Array.prototype.at;
}
if (typeof Array.prototype.findLast !== "function") {
  Array.prototype.findLast = function findLast(this: any[], fn: (v: any, i: number, a: any[]) => unknown) {
    for (let i = this.length - 1; i >= 0; i--) if (fn(this[i], i, this)) return this[i];
    return undefined;
  } as typeof Array.prototype.findLast;
}
if (typeof Promise.allSettled !== "function") {
  Promise.allSettled = function allSettled<T>(items: Iterable<T | PromiseLike<T>>) {
    return Promise.all(
      Array.from(items, (p) =>
        Promise.resolve(p).then(
          (value) => ({ status: "fulfilled" as const, value }),
          (reason) => ({ status: "rejected" as const, reason }),
        ),
      ),
    );
  } as typeof Promise.allSettled;
}
if (typeof String.prototype.replaceAll !== "function") {
  String.prototype.replaceAll = function replaceAll(this: string, find: string | RegExp, rep: string) {
    if (find instanceof RegExp) {
      if (!find.global) throw new TypeError("replaceAll must be called with a global RegExp");
      return this.replace(find, rep);
    }
    return this.split(find).join(rep);
  } as typeof String.prototype.replaceAll;
}
if (typeof Object.hasOwn !== "function") {
  Object.hasOwn = ((o: object, k: PropertyKey) => Object.prototype.hasOwnProperty.call(o, k)) as typeof Object.hasOwn;
}
if (typeof String.prototype.matchAll !== "function") {
  String.prototype.matchAll = function matchAll(this: string, re: RegExp) {
    const r = re.global ? new RegExp(re.source, re.flags) : new RegExp(re.source, `${re.flags}g`);
    const out: RegExpMatchArray[] = [];
    let m: RegExpExecArray | null;
    while ((m = r.exec(this)) !== null) {
      out.push(m as unknown as RegExpMatchArray);
      if (m[0] === "") r.lastIndex++;
    }
    return out[Symbol.iterator]();
  } as typeof String.prototype.matchAll;
}
/* eslint-enable @typescript-eslint/no-explicit-any */

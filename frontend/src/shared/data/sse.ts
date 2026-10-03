import { API_ORIGIN } from "./apiClient";
import { tokenStore } from "./tokenStore";

// Kết nối SSE duy nhất của tab (SRS 6.4 / US-PU-03 AC12–AC16): fetch + ReadableStream, KHÔNG EventSource, không token trên URL.
export type SSEStatus = "connecting" | "open" | "reconnecting" | "degraded" | "closed";
export type SSEEvent = { id?: string; event: string; data: string };
type Handler = (e: SSEEvent) => void;

const URL_EVENTS = `${API_ORIGIN}/api/v1/events`;
const BACKOFF_MS = [1000, 2000, 4000, 8000, 15000];
const STABLE_MS = 30_000;
const WATCHDOG_MS = 40_000;
const LIMIT_WAIT_MIN_S = 5;
const LIMIT_TRIES = 3;
const DEGRADED_RETRY_MS = 60_000;
const RECONNECT_GAP_MS = 50;

const jitter = (ms: number) => ms * (0.8 + Math.random() * 0.4);

/** Số sánh id `<ms>-<seq>` theo (ms, seq). Trả < 0, 0, > 0. id lạ → NaN. */
export function compareEventId(a: string, b: string): number {
  const [am, as = "0"] = a.split("-");
  const [bm, bs = "0"] = b.split("-");
  const d = Number(am) - Number(bm);
  return d !== 0 ? d : Number(as) - Number(bs);
}

class SSEManager {
  private handlers = new Map<string, Set<Handler>>();
  private statusListeners = new Set<() => void>();
  private resyncHandlers = new Set<() => void>();
  private _status: SSEStatus = "closed";
  private refs = 0;
  private ctrl: AbortController | null = null;
  private timer: ReturnType<typeof setTimeout> | null = null;
  private watchdog: ReturnType<typeof setTimeout> | null = null;
  private lastId = "";
  private attempt = 0;
  private limitTries = 0;
  private waitingToken = false;
  private running = false;

  get status() {
    return this._status;
  }
  private setStatus(s: SSEStatus) {
    if (this._status === s) return;
    this._status = s;
    this.statusListeners.forEach((f) => f());
  }
  subscribeStatus = (f: () => void) => {
    this.statusListeners.add(f);
    return () => {
      this.statusListeners.delete(f);
    };
  };

  /** Đăng ký nghe một loại sự kiện; kết nối mở khi có người nghe đầu tiên, đóng khi người cuối rời đi. */
  on(type: string, h: Handler, onResync?: () => void): () => void {
    let set = this.handlers.get(type);
    if (!set) this.handlers.set(type, (set = new Set()));
    set.add(h);
    if (onResync) this.resyncHandlers.add(onResync);
    if (++this.refs === 1) this.start();
    return () => {
      set.delete(h);
      if (onResync) this.resyncHandlers.delete(onResync);
      if (--this.refs === 0) this.stop();
    };
  }

  private start() {
    this.running = true;
    this.attempt = 0;
    this.limitTries = 0;
    this.lastId = "";
    this.connect(false);
  }

  private stop() {
    this.running = false;
    this.clearTimers();
    this.ctrl?.abort();
    this.ctrl = null;
    this.setStatus("closed");
  }

  private clearTimers() {
    if (this.timer) clearTimeout(this.timer);
    if (this.watchdog) clearTimeout(this.watchdog);
    this.timer = this.watchdog = null;
  }

  private schedule(ms: number, reconnecting = true) {
    if (!this.running) return;
    if (reconnecting && this._status !== "degraded") this.setStatus("reconnecting");
    if (this.timer) clearTimeout(this.timer);
    this.timer = setTimeout(() => this.connect(true), ms);
  }

  private backoff() {
    const ms = jitter(BACKOFF_MS[Math.min(this.attempt, BACKOFF_MS.length - 1)]);
    this.attempt += 1;
    this.schedule(ms);
  }

  private kick() {
    // watchdog: không nhận byte nào trong 40 s ⇒ huỷ và nối lại
    if (this.watchdog) clearTimeout(this.watchdog);
    this.watchdog = setTimeout(() => {
      this.controlled = true; // connect() thấy cờ này thì không lên lịch lần nữa
      this.ctrl?.abort();
      if (this.running) this.backoff();
    }, WATCHDOG_MS);
  }

  private async connect(again: boolean) {
    if (!this.running) return;
    this.controlled = false;
    const token = tokenStore.get();
    if (this._status !== "degraded") this.setStatus(again ? "reconnecting" : "connecting");
    const ctrl = new AbortController();
    this.ctrl = ctrl;
    const headers: Record<string, string> = { Accept: "text/event-stream" };
    if (token) headers.Authorization = `Bearer ${token}`;
    if (this.lastId) headers["Last-Event-ID"] = this.lastId;
    const started = Date.now();
    let res: Response;
    try {
      this.kick();
      res = await fetch(URL_EVENTS, { headers, credentials: "include", signal: ctrl.signal });
    } catch {
      if (ctrl.signal.aborted) return; // do stop() hoặc watchdog (watchdog tự lên lịch)
      this.backoff();
      return;
    }
    if (ctrl !== this.ctrl) return;

    if (res.status === 401) {
      this.clearTimers();
      window.dispatchEvent(new CustomEvent("auth:expired"));
      this.setStatus("closed");
      this.running = false;
      if (this.refs > 0) this.waitForToken();
      return;
    }
    if (res.status === 429) {
      this.clearTimers();
      let retryAfter = Number(res.headers.get("Retry-After"));
      try {
        const b = (await res.json()) as { retry_after?: number };
        if (b.retry_after) retryAfter = b.retry_after;
      } catch { /* thân không phải JSON */ }
      this.limitTries += 1;
      if (this.limitTries >= LIMIT_TRIES) {
        this.setStatus("degraded");
        this.limitTries = 0;
        this.schedule(DEGRADED_RETRY_MS, false);
        return;
      }
      this.schedule(Math.max(retryAfter || 0, LIMIT_WAIT_MIN_S) * 1000);
      return;
    }
    if (!res.ok || !res.body) {
      this.backoff();
      return;
    }

    this.limitTries = 0;
    try {
      await this.read(res.body, ctrl);
    } catch {
      /* đứt ngang / huỷ */
    }
    if (ctrl !== this.ctrl || !this.running || this.controlled) {
      this.controlled = false;
      return;
    }
    if (Date.now() - started >= STABLE_MS) this.attempt = 0;
    this.backoff();
  }

  private controlled = false; // đã xử lý reconnect / shutdown nên không lên lịch backoff nữa

  private waitForToken() {
    if (this.waitingToken) return;
    this.waitingToken = true;
    const off = tokenStore.subscribe(() => {
      if (tokenStore.get() && this.refs > 0) {
        off();
        this.waitingToken = false;
        this.running = true;
        this.attempt = 0;
        this.connect(true);
      }
    });
  }

  private async read(body: ReadableStream<Uint8Array>, ctrl: AbortController) {
    const reader = body.getReader();
    const dec = new TextDecoder();
    let buf = "";
    let ev = { event: "message", id: undefined as string | undefined, data: [] as string[] };
    for (;;) {
      const { done, value } = await reader.read();
      if (done) return;
      this.kick(); // bất kỳ byte nào (kể cả heartbeat) đặt lại đồng hồ chờ
      buf += dec.decode(value, { stream: true });
      let nl: number;
      while ((nl = buf.search(/\r?\n/)) >= 0) {
        const line = buf.slice(0, nl);
        buf = buf.slice(buf.indexOf("\n", nl) + 1);
        if (line === "") {
          if (ev.data.length || ev.event !== "message") this.dispatch({ event: ev.event, id: ev.id, data: ev.data.join("\n") }, ctrl);
          ev = { event: "message", id: undefined, data: [] };
          if (ctrl.signal.aborted) return;
          continue;
        }
        if (line.startsWith(":")) continue; // heartbeat
        const c = line.indexOf(":");
        const field = c < 0 ? line : line.slice(0, c);
        const val = c < 0 ? "" : line.slice(c + 1).replace(/^ /, "");
        if (field === "event") ev.event = val;
        else if (field === "data") ev.data.push(val);
        else if (field === "id") ev.id = val;
        // `retry:` chỉ ghi nhận — backoff do client quyết (SRS 6.4)
      }
    }
  }

  private dispatch(e: SSEEvent, ctrl: AbortController) {
    if (e.id) {
      if (this.lastId && compareEventId(e.id, this.lastId) <= 0) return; // đã xử lý (trùng sau nối lại)
      this.lastId = e.id;
    }
    switch (e.event) {
      case "ready":
        this.setStatus("open");
        break;
      case "reconnect": {
        let reason = "";
        try { reason = (JSON.parse(e.data) as { reason?: string }).reason ?? ""; } catch { /* không có lý do */ }
        this.controlled = true;
        ctrl.abort();
        if (reason === "token_expired") {
          window.dispatchEvent(new CustomEvent("auth:expired"));
          this.running = false;
          this.setStatus("closed");
          this.waitForToken();
        } else {
          // nối ngay (không backoff) nhưng sau khi ổ cắm cũ đã đóng: mở yêu cầu mới ngay trong lúc huỷ yêu cầu cũ làm Chrome
          // dùng lại ổ cắm đang đóng và gửi yêu cầu hai lần (BUG-PU03-2)
          setTimeout(() => this.running && this.connect(true), RECONNECT_GAP_MS);
        }
        return;
      }
      case "shutdown":
        this.controlled = true;
        ctrl.abort();
        this.schedule(1000 + Math.random() * 2000);
        return;
      case "resync":
        this.resyncHandlers.forEach((f) => f());
        break;
    }
    this.handlers.get(e.event)?.forEach((h) => h(e));
  }
}

/** Một manager mỗi tab. */
export const sseManager = new SSEManager();

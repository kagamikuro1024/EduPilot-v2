// Khoá thiết bị WEB (US-AS-09 AC3): cặp khoá ECDSA P-256 KHÔNG xuất được, lưu trong IndexedDB `edupilot.device.v1`.
// MỘT khoá cho cả trình duyệt (không theo tài khoản, đăng xuất không xoá). Không có WebCrypto / IndexedDB / ngữ cảnh an toàn ⇒ `null`: điểm danh vẫn được, không có `device`.

const DB_NAME = "edupilot.device.v1";
const STORE = "keys";
const ID = "web";

type Rec = { id: string; privateKey: CryptoKey; publicKey: string };
export type WebDevice = { kind: "WEB_KEY"; public_key: string; signature: string };

const b64 = (buf: ArrayBuffer) => {
  let s = "";
  for (const b of new Uint8Array(buf)) s += String.fromCharCode(b);
  return btoa(s);
};

function openDb(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, 1);
    req.onupgradeneeded = () => req.result.createObjectStore(STORE, { keyPath: "id" });
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

function tx<T>(db: IDBDatabase, mode: IDBTransactionMode, run: (s: IDBObjectStore) => IDBRequest<T>): Promise<T> {
  return new Promise((resolve, reject) => {
    const r = run(db.transaction(STORE, mode).objectStore(STORE));
    r.onsuccess = () => resolve(r.result);
    r.onerror = () => reject(r.error);
  });
}

async function loadOrCreate(): Promise<Rec> {
  const db = await openDb();
  try {
    const found = await tx<Rec | undefined>(db, "readonly", (s) => s.get(ID));
    if (found) return found;
    const pair = await crypto.subtle.generateKey({ name: "ECDSA", namedCurve: "P-256" }, false, ["sign", "verify"]);
    const rec: Rec = { id: ID, privateKey: pair.privateKey, publicKey: b64(await crypto.subtle.exportKey("spki", pair.publicKey)) };
    await tx(db, "readwrite", (s) => s.put(rec));
    void navigator.storage?.persist?.().catch(() => undefined); // giữ khoá khi trình duyệt thiếu bộ nhớ (iOS: chỉ khoá tạo TRONG PWA cài vào Màn hình chính giữ quá 7 ngày; khoá tạo ở Safari là kho khác, có thể bị xoá sau 7 ngày không mở — P-11-137)
    return rec;
  } finally {
    db.close();
  }
}

/** Ký `message` bằng khoá thiết bị; `null` khi môi trường không hỗ trợ hoặc có lỗi (không bao giờ chặn điểm danh). */
export async function signWithDeviceKey(message: string): Promise<WebDevice | null> {
  try {
    if (typeof indexedDB === "undefined" || !globalThis.crypto?.subtle || !globalThis.isSecureContext) return null;
    const k = await loadOrCreate();
    const sig = await crypto.subtle.sign({ name: "ECDSA", hash: "SHA-256" }, k.privateKey, new TextEncoder().encode(message)); // r‖s, 64 byte
    return { kind: "WEB_KEY", public_key: k.publicKey, signature: b64(sig) };
  } catch {
    return null;
  }
}

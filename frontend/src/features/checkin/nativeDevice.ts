import { ApiError, apiClient } from "@/shared/data";

// Cầu nối app Android (ARCHITECTURE §11.7): `window.Capacitor` do Capacitor core gắn vào mọi trang WebView nạp; `frontend/` KHÔNG import `@capacitor/*`.
// Plugin Kotlin `EduPilotAttest` (mobile/): registerDevice({nonce}) → {public_key, attestation_chain[]}, sign({message}) → {signature (DER), public_key}, isMockLocation() → {mock}.

type Bridge = { isNativePlatform?: () => boolean; isPluginAvailable?: (n: string) => boolean; nativePromise?: <T>(plugin: string, method: string, opts?: unknown) => Promise<T> };
const bridge = (): Bridge | undefined => (typeof window === "undefined" ? undefined : (window as unknown as { Capacitor?: Bridge }).Capacitor);

const PLUGIN = "EduPilotAttest";
const KEY_ID = "ep.device.android"; // key_id đã đăng ký (khoá không gắn với người dùng, như khoá web)
const FAILED = "ep.device.android.failed"; // sessionStorage: đăng ký lỗi trong phiên này thì không thử lại mỗi lượt

export type NativeDevice = { kind: "ANDROID_KEY"; key_id: string; signature: string } | { kind: "WEB_KEY"; public_key: string; signature: string };

export const isNativeApp = () => Boolean(bridge()?.isNativePlatform?.());
export const hasAttestPlugin = () => isNativeApp() && Boolean(bridge()?.isPluginAvailable?.(PLUGIN));
const call = <T>(method: string, opts?: unknown) => bridge()!.nativePromise!<T>(PLUGIN, method, opts);

const read = (s: Storage, k: string) => {
  try {
    return s.getItem(k);
  } catch {
    return null;
  }
};
const write = (s: Storage, k: string, v: string) => {
  try {
    s.setItem(k, v);
  } catch {
    /* bỏ qua */
  }
};

/** Đăng ký khoá có chứng thực nếu chưa có `key_id`; lỗi (`ATTESTATION_FAILED` trên emulator, `503`, plugin lỗi) ⇒ null: app ký bằng khoá Keystore và gửi `WEB_KEY` (mức 2, P-108-5). */
async function ensureKeyId(): Promise<string | null> {
  const known = read(localStorage, KEY_ID);
  if (known) return known;
  if (read(sessionStorage, FAILED)) return null;
  try {
    const { data: ch } = await apiClient.post<{ nonce: string }>("/me/devices/challenge", {});
    const att = await call<{ public_key: string; attestation_chain: string[] }>("registerDevice", { nonce: ch.nonce });
    const { data } = await apiClient.post<{ key_id: string }>("/me/devices", { kind: "ANDROID_KEY", public_key: att.public_key, attestation_chain: att.attestation_chain });
    write(localStorage, KEY_ID, data.key_id);
    return data.key_id;
  } catch (e) {
    write(sessionStorage, FAILED, e instanceof ApiError ? e.code : "ERROR");
    return null;
  }
}

/** Ký `message` bằng khoá Keystore (DER). Không bao giờ chặn điểm danh: lỗi ⇒ null (không gửi `device`, mức 1). */
export async function signNative(message: string): Promise<NativeDevice | null> {
  try {
    const keyId = await ensureKeyId();
    const s = await call<{ signature: string; public_key: string }>("sign", { message });
    return keyId ? { kind: "ANDROID_KEY", key_id: keyId, signature: s.signature } : { kind: "WEB_KEY", public_key: s.public_key, signature: s.signature };
  } catch {
    return null;
  }
}

/** `Location.isMock()` khi lớp yêu cầu vị trí; plugin lỗi ⇒ undefined (không khai). */
export async function nativeMockLocation(): Promise<boolean | undefined> {
  try {
    return (await call<{ mock: boolean }>("isMockLocation")).mock;
  } catch {
    return undefined;
  }
}

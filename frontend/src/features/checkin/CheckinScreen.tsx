"use client";

import { Check } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError, apiClient } from "@/shared/data";
import { fmtHM } from "@/features/calendar/calendarApi";
import { useSession } from "@/shared/session/session";
import { ApiErrorNotice } from "@/shared/data/ApiErrorNotice";
import { Button, Field, InlineNotice, Input, Page, PageHeader, Panel, Skeleton, StatusText, type HeaderAction } from "@/shared/ui";
import { signWithDeviceKey } from "./deviceKey";
import { hasAttestPlugin, nativeMockLocation, signNative } from "./nativeDevice";
import { canScanQr, hasBarcodeDetector, Scanner } from "./Scanner";
import { clearStash, readStash, stashExpired, STASH_EVENT } from "./stash";
import s from "./CheckinScreen.module.css";

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
const CROCKFORD = /^[0-9A-HJKMNP-TV-Z]+$/;
const GEO_LINE = "Lớp này kiểm tra vị trí khi điểm danh.";
/** Trần một lượt gửi điểm danh (DEF-MOB-03): quá hạn thì thôi «đang gửi» và cho «Thử lại» — không để treo vô hạn. */
const DEADLINE_MS = 15_000;

/** Chuẩn hoá mã ngắn đúng như máy chủ (`attendance.NormalizeCode`): bỏ khoảng trắng và `-`, in hoa, O→0, I/L→1. Rỗng / ký tự ngoài bảng ⇒ null. */
export function normalizeCode(raw: string): string | null {
  const t = raw.toUpperCase().replace(/[\s-]/g, "").replace(/O/g, "0").replace(/[IL]/g, "1");
  return t !== "" && CROCKFORD.test(t) ? t : null;
}

type Input = { course: string; token: string } | { course: string; code: string };
type View =
  | { k: "sending" }
  | { k: "done"; at: string }
  | { k: "entry"; error?: string; cam?: "denied" | "failed"; retry?: () => void }
  | { k: "scanning" }
  | { k: "error"; message: string; offline?: boolean; retry?: () => void }
  | { k: "limited"; until: number; retry?: () => void };

const nextFrame = () => new Promise<void>((r) => requestAnimationFrame(() => requestAnimationFrame(() => r())));

type Geo = { lat: number; lng: number; accuracy_m: number; mock?: boolean } | { denied: true };
function getGeo(): Promise<Geo> {
  return new Promise((resolve) => {
    if (!navigator.geolocation) return resolve({ denied: true });
    navigator.geolocation.getCurrentPosition(
      (p) => resolve({ lat: p.coords.latitude, lng: p.coords.longitude, accuracy_m: p.coords.accuracy }),
      () => resolve({ denied: true }),
      { enableHighAccuracy: true, timeout: 8000, maximumAge: 30_000 },
    );
  });
}

/** Một câu mỗi mã lỗi (US-AS-09 AC4, AC10). */
function describe(e: unknown, viaCode: boolean): { message: string; offline?: boolean } {
  if (!(e instanceof ApiError)) return { message: "Chưa điểm danh được. Thử lại." };
  if (e.code === "NETWORK" || e.code === "ABORTED") return { message: "Chưa có mạng.", offline: true }; // ABORTED = quá hạn 15 s của lượt gửi
  if (e.status === 403 || e.status === 404) return { message: "Bạn không thuộc lớp này." };
  if (viaCode && e.code === "CHECKIN_TOKEN_INVALID") return { message: "Mã không đúng hoặc đã hết hạn. Nhập mã đang hiện trên màn chiếu." };
  return { message: e.userMessage };
}

/** `/checkin` (DESIGN §14.8.2): token từ QR (đã cất ở layout) gửi MỘT lần; không có token ⇒ nhập mã ngắn hoặc quét trong app. */
export function CheckinScreen() {
  const { role, identity, realCourseId, realCourses } = useSession();
  const native = typeof window !== "undefined" && Boolean((window as unknown as { Capacitor?: { isNativePlatform?: () => boolean } }).Capacitor?.isNativePlatform?.());
  // Có API là chưa đủ: thăm dò `qr_code` bất đồng bộ; trong lúc chờ coi như có (máy hỗ trợ không đổi hình), không hỗ trợ thì chuyển sang ô nhập mã.
  const [qrOk, setQrOk] = useState<boolean | null>(null);
  const canScan = native && hasBarcodeDetector() && qrOk !== false;
  // token đã cất ở layout (QR): đọc MỘT lần lúc dựng; AuthGate chỉ dựng màn này ở client nên không lệch khi hydrate
  const [stash, setStash] = useState(() => (typeof window === "undefined" ? null : readStash()));
  // DEF-MOB-05: token MỚI trong URL (#token) luôn thắng stash cũ — kể cả khi trang không tải lại (chỉ đổi hash / bfcache).
  useEffect(() => {
    const pick = () => {
      const s = readStash();
      if (!s || !UUID.test(s.c)) return;
      if (stashExpired(s)) return clearStash(); // token mới cũng quá hạn: bỏ, không gửi
      setStash(s);
    };
    window.addEventListener(STASH_EVENT, pick);
    return () => window.removeEventListener(STASH_EVENT, pick);
  }, []);
  const [view, setView] = useState<View>(() => {
    if (!stash) return { k: "entry" };
    if (!UUID.test(stash.c)) return { k: "error", message: "Liên kết không đúng." };
    if (stashExpired(stash)) {
      clearStash(); // token quá 60 s: bỏ khỏi sessionStorage, không để dùng lại sau khi tải lại
      return { k: "error", message: "Mã đã hết hạn. Quét lại mã trên màn chiếu." };
    }
    return { k: "sending" };
  });
  const [geoLine, setGeoLine] = useState(false);
  const [codeMode, setCodeMode] = useState(!canScan);
  useEffect(() => {
    if (!native || !hasBarcodeDetector()) return;
    let alive = true;
    void canScanQr().then((ok) => {
      if (!alive) return;
      setQrOk(ok);
      if (!ok) setCodeMode(true);
    });
    return () => {
      alive = false;
    };
  }, [native]);
  const [code, setCode] = useState("");
  const [left, setLeft] = useState(0);
  const sent = useRef(new Set<string>()); // token đã gửi (StrictMode chạy effect hai lần: cùng token chỉ gửi một lần)
  const again = useRef<(i: Input) => void>(() => undefined); // gửi lại cùng đầu vào (Thử lại, hết 429)
  const uid = identity?.sub ?? "";
  const course = realCourseId && realCourseId !== "all" ? realCourseId : "";
  const classCode = realCourses?.find((c) => c.id === course)?.class_code ?? "";
  // Lớp không bật điểm danh bằng mã (US-AS-15 AC16d): biết TRƯỚC khi sinh viên gõ gì; không ô nhập, không gọi POST. Chính sách do máy chủ trả cho mọi thành viên (US-AS-01 AC6).
  const policy = useQuery({
    queryKey: ["course-policy", course],
    queryFn: ({ signal }) => apiClient.get<{ status?: string; attendance?: { self_checkin?: boolean } }>(`/courses/${course}`, { signal }).then((r) => r.data),
    enabled: role === "student" && course !== "" && !stash,
    staleTime: 30_000,
  });
  // Lớp đã LƯU TRỮ (DEF-MOB-01, SRS `COURSE_ARCHIVED`): nói đúng nguyên nhân, không lẫn với lớp tắt điểm danh bằng mã.
  const archived = !stash && policy.data?.status === "ARCHIVED";
  const disabled = !stash && !archived && view.k === "entry" && policy.data?.attendance?.self_checkin === false;
  const checking = !stash && view.k === "entry" && policy.isPending && policy.fetchStatus !== "idle"; // đang hỏi chính sách lớp: khung xương, chưa vẽ ô nhập

  const submit = useCallback(
    async (inp: Input) => {
      const viaCode = "code" in inp;
      const retry = () => again.current(inp);
      const ctrl = new AbortController();
      const timer = setTimeout(() => ctrl.abort(), DEADLINE_MS);
      try {
        const msg = `edupilot-checkin-v1|${"code" in inp ? `code:${inp.code}` : inp.token}|${uid}`;
        const policyP = apiClient.get<{ attendance?: { self_checkin?: boolean; geo_check?: boolean } }>(`/courses/${inp.course}`, { signal: ctrl.signal });
        const [policyRes, device] = await Promise.all([policyP, hasAttestPlugin() ? signNative(msg) : signWithDeviceKey(msg)]); // app: khoá Keystore; web: WebCrypto
        const policy = policyRes.data.attendance;
        if (policy && policy.self_checkin === false) return setView(viaCode ? { k: "entry", error: "Lớp này không điểm danh bằng mã." } : { k: "error", message: "Lớp này không điểm danh bằng mã." });
        let geo: Geo | undefined;
        if (policy?.geo_check) {
          setGeoLine(true); // câu thông báo hiện TRƯỚC khi gọi định vị (Luật 91/2025 Đ31 k3)
          await nextFrame();
          geo = await getGeo();
          if (hasAttestPlugin() && "lat" in geo) {
            const mock = await nativeMockLocation();
            if (mock !== undefined) geo = { ...geo, mock };
          }
        }
        const { data } = await apiClient.post<{ checked_in_at: string; round: number }>(`/courses/${inp.course}/checkins`, {
          ...("code" in inp ? { code: inp.code } : { token: inp.token }),
          ...(device ? { device } : {}),
          ...(geo ? { geo } : {}),
        }, { signal: ctrl.signal });
        setView({ k: "done", at: fmtHM(data.checked_in_at) });
      } catch (e) {
        if (e instanceof ApiError && (e.status === 429 || e.code === "RATE_LIMITED")) {
          const secs = Math.max(1, Math.ceil(e.retryAfter ?? 10));
          setLeft(secs);
          return setView({ k: "limited", until: Date.now() + secs * 1000, retry: viaCode ? undefined : retry });
        }
        const d = describe(e, viaCode);
        setView(viaCode ? { k: "entry", error: d.message, retry: d.offline ? retry : undefined } : { k: "error", message: d.message, offline: d.offline, retry: d.offline ? retry : undefined });
      } finally {
        clearTimeout(timer);
        setGeoLine(false);
      }
    },
    [uid],
  );

  /** Bắt đầu một lượt từ thao tác của người dùng: khung xương trước, rồi gửi. */
  function begin(inp: Input) {
    setView({ k: "sending" });
    void submit(inp);
  }
  useEffect(() => {
    again.current = begin;
  });

  // Có token hợp lệ đã cất ⇒ gửi đúng một lần rồi xoá khỏi sessionStorage; mọi nhánh khác đã chốt ở trạng thái đầu.
  // setTimeout 0 (có dọn): StrictMode chạy effect hai lần nhưng chỉ lần sau còn hẹn.
  useEffect(() => {
    if (role !== "student" || !uid || !stash || !UUID.test(stash.c) || stashExpired(stash)) return; // lớp lấy từ token đã cất: không chờ /me/courses (bớt một vòng trên đường nóng)
    const t = setTimeout(() => {
      if (sent.current.has(stash.token)) return; // effect chạy lại (tải lại nền / bfcache): mỗi token chỉ dùng một lần
      sent.current.add(stash.token);
      clearStash();
      setView({ k: "sending" });
      void submit({ course: stash.c, token: stash.token });
    }, 0);
    return () => clearTimeout(t);
  }, [role, uid, submit, stash]);

  // 429: đếm ngược từng giây; hết giờ thì mã ngắn quay về ô nhập, token tự gửi lại
  useEffect(() => {
    if (view.k !== "limited") return;
    const tick = () => {
      const n = Math.max(0, Math.ceil((view.until - Date.now()) / 1000));
      setLeft(n);
      if (n === 0) {
        if (view.retry) view.retry();
        else setView({ k: "entry" });
      }
    };
    const t = setInterval(tick, 1000);
    return () => clearInterval(t);
  }, [view]);

  // Back / vuốt quay lại khi đang quét: thoát khung quét về ô nhập mã, không rời /checkin (US-AS-15 AC7a). Vào khung quét đẩy một mục lịch sử; thoát bằng cách lùi mục đó.
  const scanning = view.k === "scanning";
  useEffect(() => {
    if (!scanning) return;
    window.history.pushState({ ...(window.history.state as object | null), epScan: true }, "");
    const onPop = () => setView((v) => (v.k === "scanning" ? { k: "entry" } : v));
    window.addEventListener("popstate", onPop);
    return () => window.removeEventListener("popstate", onPop);
  }, [scanning]);
  const leaveScan = () => {
    if ((window.history.state as { epScan?: boolean } | null)?.epScan) window.history.back(); // popstate đưa về ô nhập mã
    else setView((v) => (v.k === "scanning" ? { k: "entry" } : v));
  };

  // Quyền camera đã cấp lại ở Cài đặt hệ thống: khi trang hiện lại, trả nút «Quét mã điểm danh» và bỏ dòng từ chối, không xin quyền lặp (AC8)
  const camIssue = view.k === "entry" && view.cam !== undefined;
  useEffect(() => {
    if (!camIssue) return;
    const onShow = () => {
      if (document.visibilityState !== "visible") return;
      const back = () => {
        setCodeMode(false);
        setView((v) => (v.k === "entry" && v.cam ? { k: "entry" } : v));
      };
      const q = navigator.permissions?.query({ name: "camera" as PermissionName });
      if (!q) return back();
      q.then((p) => p.state !== "denied" && back(), back);
    };
    document.addEventListener("visibilitychange", onShow);
    return () => document.removeEventListener("visibilitychange", onShow);
  }, [camIssue]);

  if (role !== "student") return null;

  const norm = normalizeCode(code);
  function sendCode() {
    if (!course) return;
    if (!norm) return setView({ k: "entry", error: "Mã không đúng hoặc đã hết hạn. Nhập mã đang hiện trên màn chiếu." });
    begin({ course, code: norm });
  }
  function onScanned(v: { c: string; token: string }) {
    begin({ course: v.c, token: v.token });
    leaveScan();
  }

  let primary: HeaderAction | undefined;
  if (archived || disabled || checking) {
    primary = undefined;
  } else if (view.k === "entry") {
    if (view.retry) primary = { label: "Thử lại", onClick: view.retry, docked: true };
    else primary = canScan && !codeMode ? { label: "Quét mã điểm danh", onClick: () => setView({ k: "scanning" }), docked: true } : { label: "Điểm danh", onClick: sendCode, disabled: code.trim() === "" || !course, docked: true };
  } else if (view.k === "error") {
    if (view.retry) primary = { label: "Thử lại", onClick: view.retry, docked: true };
    else if (canScan) primary = { label: "Quét mã điểm danh", onClick: () => setView({ k: "scanning" }), docked: true };
  } else if (view.k === "limited") {
    primary = { label: "Điểm danh", disabled: true, reason: `Thử lại sau ${left} giây.`, docked: true };
  }

  return (
    <Page kind="read">
      <PageHeader title="Điểm danh" primary={primary} />
      {!stash && policy.isError && <ApiErrorNotice error={policy.error} showTechnical={false} onRetry={() => void policy.refetch()} />}
      <Panel>
        <div className={s.body} data-part="checkin" data-state={view.k} aria-live="polite">
          {view.k === "sending" && (
            <>
              {geoLine && <p className={s.geo} data-part="geo-line">{GEO_LINE}</p>}
              <Skeleton lines={2} />
            </>
          )}
          {view.k === "done" && (
            <p className={`ep-data ${s.done}`} data-part="done">
              <Check aria-hidden className={s.check} />
              Đã điểm danh {view.at}
            </p>
          )}
          {view.k === "scanning" && (
            <Scanner
              onCode={onScanned}
              onExit={leaveScan}
              onDenied={(cam) => {
                setCodeMode(true);
                setView({ k: "entry", cam });
              }}
            />
          )}
          {checking && <Skeleton lines={2} />}
          {archived && <p data-part="checkin-archived">Lớp này đã được lưu trữ.</p>}
          {disabled && <p data-part="checkin-disabled">Lớp này không điểm danh bằng mã.</p>}
          {view.k === "entry" && !archived && !disabled && !checking && (canScan && !codeMode ? (
            <Button variant="ghost" onClick={() => setCodeMode(true)}>Nhập mã</Button>
          ) : (
            <form
              className={s.form}
              onSubmit={(e) => {
                e.preventDefault();
                if (code.trim() !== "") sendCode();
              }}
            >
              <Field label="Mã điểm danh" error={view.error}>
                {(id, d) => (
                  <Input
                    id={id}
                    aria-describedby={d}
                    value={code}
                    onChange={(e) => setCode(e.target.value.toUpperCase())}
                    autoCapitalize="characters"
                    autoComplete="off"
                    autoCorrect="off"
                    spellCheck={false}
                    enterKeyHint="go"
                    inputMode="text"
                    maxLength={12}
                    placeholder="ABC 123"
                    invalid={Boolean(view.error)}
                    onFocus={(e) => e.currentTarget.scrollIntoView({ block: "center" })} // bàn phím ảo mở: ô nằm TRÊN thanh dock (TC-AS09-24)
                  />
                )}
              </Field>
              {classCode && <p className="ep-meta">Lớp {classCode}</p>}
              {view.cam && <StatusText tone="amber">{view.cam === "denied" ? "Chưa cấp quyền camera." : "Không mở được camera."}</StatusText>}
            </form>
          ))}
          {view.k === "error" && <InlineNotice tone="danger" compact>{view.message}</InlineNotice>}
          {view.k === "limited" && <p data-part="limited">Thử lại sau {left} giây.</p>}
        </div>
      </Panel>
      {primary && (
        <div className={s.dock} data-part="checkin-dock" data-ep-header-dock>
          <Button variant="primary" disabled={primary.disabled} onClick={primary.onClick}>{primary.label}</Button>
        </div>
      )}
    </Page>
  );
}

"use client";

import { Check } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError, apiClient } from "@/shared/data";
import { fmtHM } from "@/features/calendar/calendarApi";
import { useSession } from "@/shared/session/session";
import { Button, Field, InlineNotice, Input, Page, PageHeader, Panel, Skeleton, StatusText, type HeaderAction } from "@/shared/ui";
import { signWithDeviceKey } from "./deviceKey";
import { hasBarcodeDetector, Scanner } from "./Scanner";
import { clearStash, readStash, stashExpired } from "./stash";
import s from "./CheckinScreen.module.css";

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
const CROCKFORD = /^[0-9A-HJKMNP-TV-Z]+$/;
const GEO_LINE = "Lớp này kiểm tra vị trí khi điểm danh.";

/** Chuẩn hoá mã ngắn đúng như máy chủ (`attendance.NormalizeCode`): bỏ khoảng trắng và `-`, in hoa, O→0, I/L→1. Rỗng / ký tự ngoài bảng ⇒ null. */
export function normalizeCode(raw: string): string | null {
  const t = raw.toUpperCase().replace(/[\s-]/g, "").replace(/O/g, "0").replace(/[IL]/g, "1");
  return t !== "" && CROCKFORD.test(t) ? t : null;
}

type Input = { course: string; token: string } | { course: string; code: string };
type View =
  | { k: "sending" }
  | { k: "done"; at: string }
  | { k: "entry"; error?: string; denied?: boolean }
  | { k: "scanning" }
  | { k: "error"; message: string; offline?: boolean; retry?: () => void }
  | { k: "limited"; until: number; retry?: () => void };

const nextFrame = () => new Promise<void>((r) => requestAnimationFrame(() => requestAnimationFrame(() => r())));

function getGeo(): Promise<{ lat: number; lng: number; accuracy_m: number } | { denied: true }> {
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
  if (e.code === "NETWORK") return { message: "Chưa có mạng.", offline: true };
  if (e.status === 403 || e.status === 404) return { message: "Bạn không thuộc lớp này." };
  if (viaCode && e.code === "CHECKIN_TOKEN_INVALID") return { message: "Mã không đúng hoặc đã hết hạn. Nhập mã đang hiện trên màn chiếu." };
  return { message: e.userMessage };
}

/** `/checkin` (DESIGN §14.8.2): token từ QR (đã cất ở layout) gửi MỘT lần; không có token ⇒ nhập mã ngắn hoặc quét trong app. */
export function CheckinScreen() {
  const { role, identity, realCourseId, realCourses, realPending } = useSession();
  const native = typeof window !== "undefined" && Boolean((window as unknown as { Capacitor?: { isNativePlatform?: () => boolean } }).Capacitor?.isNativePlatform?.());
  const canScan = native && hasBarcodeDetector();
  // token đã cất ở layout (QR): đọc MỘT lần lúc dựng; AuthGate chỉ dựng màn này ở client nên không lệch khi hydrate
  const [stash] = useState(() => (typeof window === "undefined" ? null : readStash()));
  const [view, setView] = useState<View>(() => {
    if (!stash) return { k: "entry" };
    if (!UUID.test(stash.c)) return { k: "error", message: "Liên kết không đúng." };
    if (stashExpired(stash)) return { k: "error", message: "Mã đã hết hạn. Quét lại mã trên màn chiếu." };
    return { k: "sending" };
  });
  const [geoLine, setGeoLine] = useState(false);
  const [codeMode, setCodeMode] = useState(!canScan);
  const [code, setCode] = useState("");
  const [left, setLeft] = useState(0);
  const sent = useRef(false);
  const again = useRef<(i: Input) => void>(() => undefined); // gửi lại cùng đầu vào (Thử lại, hết 429)
  const uid = identity?.sub ?? "";
  const course = realCourseId && realCourseId !== "all" ? realCourseId : "";
  const classCode = realCourses?.find((c) => c.id === course)?.class_code ?? "";

  const submit = useCallback(
    async (inp: Input) => {
      const viaCode = "code" in inp;
      const retry = () => again.current(inp);
      try {
        const msg = `edupilot-checkin-v1|${"code" in inp ? `code:${inp.code}` : inp.token}|${uid}`;
        const policyP = apiClient.get<{ attendance?: { self_checkin?: boolean; geo_check?: boolean } }>(`/courses/${inp.course}`);
        const [policyRes, device] = await Promise.all([policyP, signWithDeviceKey(msg)]);
        const policy = policyRes.data.attendance;
        if (policy && policy.self_checkin === false) return setView(viaCode ? { k: "entry", error: "Lớp này không điểm danh bằng mã." } : { k: "error", message: "Lớp này không điểm danh bằng mã." });
        let geo: Awaited<ReturnType<typeof getGeo>> | undefined;
        if (policy?.geo_check) {
          setGeoLine(true); // câu thông báo hiện TRƯỚC khi gọi định vị (Luật 91/2025 Đ31 k3)
          await nextFrame();
          geo = await getGeo();
        }
        const { data } = await apiClient.post<{ checked_in_at: string; round: number }>(`/courses/${inp.course}/checkins`, {
          ...("code" in inp ? { code: inp.code } : { token: inp.token }),
          ...(device ? { device } : {}),
          ...(geo ? { geo } : {}),
        });
        setView({ k: "done", at: fmtHM(data.checked_in_at) });
      } catch (e) {
        if (e instanceof ApiError && (e.status === 429 || e.code === "RATE_LIMITED")) {
          const secs = Math.max(1, Math.ceil(e.retryAfter ?? 10));
          setLeft(secs);
          return setView({ k: "limited", until: Date.now() + secs * 1000, retry: viaCode ? undefined : retry });
        }
        const d = describe(e, viaCode);
        setView(viaCode ? { k: "entry", error: d.message } : { k: "error", message: d.message, offline: d.offline, retry: d.offline ? retry : undefined });
      } finally {
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
    if (role !== "student" || realPending || !uid || !stash || !UUID.test(stash.c) || stashExpired(stash)) return;
    const t = setTimeout(() => {
      if (sent.current) return; // tải lại nền của /me/courses làm effect chạy lại: token chỉ dùng một lần
      sent.current = true;
      clearStash();
      void submit({ course: stash.c, token: stash.token });
    }, 0);
    return () => clearTimeout(t);
  }, [role, realPending, uid, submit, stash]);

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

  if (role !== "student") return null;

  const norm = normalizeCode(code);
  function sendCode() {
    if (!course) return;
    if (!norm) return setView({ k: "entry", error: "Mã không đúng hoặc đã hết hạn. Nhập mã đang hiện trên màn chiếu." });
    begin({ course, code: norm });
  }
  function onScanned(v: { c: string; token: string }) {
    begin({ course: v.c, token: v.token });
  }

  let primary: HeaderAction | undefined;
  if (view.k === "entry") {
    primary = canScan && !codeMode ? { label: "Quét mã điểm danh", onClick: () => setView({ k: "scanning" }), docked: true } : { label: "Điểm danh", onClick: sendCode, disabled: code.trim() === "" || !course, docked: true };
  } else if (view.k === "error") {
    if (view.retry) primary = { label: "Thử lại", onClick: view.retry, docked: true };
    else if (canScan) primary = { label: "Quét mã điểm danh", onClick: () => setView({ k: "scanning" }), docked: true };
  } else if (view.k === "limited") {
    primary = { label: "Điểm danh", disabled: true, reason: `Thử lại sau ${left} giây.`, docked: true };
  }

  return (
    <Page kind="read">
      <PageHeader title="Điểm danh" primary={primary} />
      <Panel>
        <div className={s.body} data-part="checkin" data-state={view.k}>
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
              onExit={() => setView({ k: "entry" })}
              onDenied={() => {
                setCodeMode(true);
                setView({ k: "entry", denied: true });
              }}
            />
          )}
          {view.k === "entry" && (canScan && !codeMode ? (
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
                  />
                )}
              </Field>
              {classCode && <p className="ep-meta">Lớp {classCode}</p>}
              {view.denied && <StatusText tone="amber">Chưa cấp quyền camera.</StatusText>}
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

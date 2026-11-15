"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";
import { ApiError } from "@/shared/data";
import { useSession } from "@/shared/session/session";
import { whoCanOpen } from "@/shared/shell/nav";
import { Button, ButtonLink, InlineNotice, Skeleton } from "@/shared/ui";
import { useCheckinActions } from "../checkinApi";
import { Qr } from "./Qr";
import { useDisplay } from "./useDisplay";
import s from "./AttendanceDisplay.module.css";

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
const mmss = (ms: number) => {
  const t = Math.max(0, Math.floor(ms / 1000));
  return `${String(Math.floor(t / 60)).padStart(2, "0")}:${String(t % 60).padStart(2, "0")}`;
};
/** `ABC123` → `ABC 123` (DESIGN §14.8.1). */
const grouped = (code: string) => (code.length === 6 ? `${code.slice(0, 3)} ${code.slice(3)}` : code);

/** Màn chiếu điểm danh (DESIGN §14.8.1): ngoài khung ứng dụng; chỉ giảng viên / TA; sinh viên và quản trị viên không gây yêu cầu `/api/v1` nào. */
export function AttendanceDisplay() {
  // `?course=` là liên kết sâu của phiên (chọn lớp rồi bị bỏ khỏi URL): lớp lấy từ lớp đang chọn
  const { role, realCourseId, realPending } = useSession();
  const course = realCourseId && realCourseId !== "all" ? realCourseId : "";
  const session = useSearchParams().get("session") ?? "";
  if (role !== "teacher" && role !== "ta") {
    return (
      <main className={s.blocked} id="main">
        <h1>Bạn không có quyền xem màn này</h1>
        <p>{whoCanOpen("/attendance/display")}</p>
        <ButtonLink href="/" variant="primary">Về Hôm nay</ButtonLink>
      </main>
    );
  }
  if (realPending) return null;
  if (!UUID.test(course) || !UUID.test(session)) {
    return (
      <main className={s.blocked} id="main">
        <h1>Màn chiếu điểm danh</h1>
        <ButtonLink href="/attendance" variant="primary">Về Điểm danh</ButtonLink>
      </main>
    );
  }
  return <Live course={course} session={session} />;
}

function Live({ course, session }: { course: string; session: string }) {
  const router = useRouter();
  const d = useDisplay(course, session);
  const actions = useCheckinActions(course, session);
  const back = `/attendance?session=${session}`;
  const [closeError, setCloseError] = useState<string | null>(null);

  const exit = () => router.push(back);
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && router.push(back);
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [router, back]);

  // giữ màn sáng khi cửa sổ đang mở (API gốc; lỗi bỏ qua)
  const live = d.state.kind === "live";
  useEffect(() => {
    if (!live) return;
    let lock: { release: () => Promise<void> } | null = null;
    (navigator as unknown as { wakeLock?: { request: (t: "screen") => Promise<{ release: () => Promise<void> }> } }).wakeLock?.request("screen").then((l) => (lock = l), () => undefined);
    return () => void lock?.release().catch(() => undefined);
  }, [live]);

  async function close() {
    if (d.state.kind !== "live") return;
    setCloseError(null);
    try {
      const r = await actions.close.mutateAsync(d.state.window.id);
      d.markClosed(r.window.checked_in);
    } catch (e) {
      setCloseError(e instanceof ApiError ? e.userMessage : "Chưa đóng được. Thử lại.");
    }
  }

  const st = d.state;
  const left = st.kind === "live" ? Date.parse(st.display.closes_at) - d.serverNow() : 0;
  const count = st.kind === "live" ? st.display.checked_in : st.kind === "closed" ? st.count : 0;
  const headcount = st.kind === "live" ? st.display.headcount : st.kind === "closed" ? st.headcount : null;
  const round = st.kind === "live" ? st.display.round : st.kind === "closed" ? st.round : 1;

  return (
    <main className={s.root} id="main" data-part="display" data-state={st.kind}>
      <h1 className="ep-sr-only">Màn chiếu điểm danh</h1>
      <section className={s.qrBox} aria-label="Mã QR">
        {st.kind === "loading" && <Skeleton className={s.skeleton} />}
        {st.kind === "error" && (
          <div className={s.centered}>
            <InlineNotice tone="danger" action={<Button onClick={() => void d.retry()}>Thử lại</Button>}>Không tải được mã</InlineNotice>
          </div>
        )}
        {st.kind === "live" && !st.stale && <Qr text={st.display.qr_url} label="Mã QR điểm danh" />}
        {st.kind === "live" && st.stale && <div className={`${s.centered} ${s.warn}`} data-part="offline">Mất kết nối</div>}
        {(st.kind === "closed" || st.kind === "none") && <div className={`${s.centered} ${s.countSize}`} data-part="closed">{st.kind === "none" ? "Chưa mở" : "Đã đóng"}</div>}
      </section>
      <section className={s.side}>
        {st.kind === "live" && !st.stale && (
          <>
            <p className={s.meta}>Mã điểm danh</p>
            <p className={s.code} data-part="short-code" aria-label={`Mã ngắn ${st.display.short_code.split("").join(" ")}`}>{grouped(st.display.short_code)}</p>
          </>
        )}
        <p className={s.count} data-part="count">
          {count}
          {headcount != null && <span className={s.of}> / {headcount}</span>}
        </p>
        {st.kind === "live" && (
          <p className={s.meta} data-part="countdown">
            còn {mmss(left)}
            {round >= 2 ? ` · Vòng ${round}` : ""}
          </p>
        )}
        {st.kind === "closed" && round >= 2 && <p className={s.meta}>Vòng {round}</p>}
        {closeError && <p className={s.metaWarn} role="alert">{closeError}</p>}
        <div className={s.actions}>
          {st.kind === "live" && <Button onClick={() => void close()} disabled={actions.close.isPending}>Đóng điểm danh</Button>}
          <Button variant="ghost" onClick={exit}>Thoát</Button>
        </div>
      </section>
    </main>
  );
}

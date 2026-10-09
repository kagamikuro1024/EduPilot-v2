"use client";

import { useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { ApiError, apiClient } from "@/shared/data";
import { Button, ButtonLink, Field, InlineNotice, Input, Page, PageHeader, Section } from "@/shared/ui";
import s from "./Join.module.css";

/** Cùng một câu cho mọi nguyên nhân sai mã — không tiết lộ mã có tồn tại hay không (SRS FEAT-course-foundation 3.3). */
export const BAD_CODE = "Mã không hợp lệ hoặc đã hết hạn. Kiểm tra lại với giảng viên.";
const FULL = "Lớp đã đủ sĩ số. Hãy báo giảng viên.";

type Preview = { name: string; class_code: string; semester: string; teachers: Array<{ full_name: string }>; state: "OPEN" | "REQUIRES_APPROVAL" | "FULL" | "ALREADY_MEMBER" | "PENDING" };
type Joined = { course_id: string; status: "ACTIVE" | "PENDING"; already_member: boolean };

/** Bảng ký tự mã tham gia: không 0 O 1 I L. Ô nhập tự viết hoa, bỏ khoảng trắng và bỏ ký tự ngoài bảng. */
export function cleanCode(raw: string) {
  return raw.toUpperCase().replace(/[^ABCDEFGHJKMNPQRSTUVWXYZ23456789]/g, "").slice(0, 7);
}

/** "2026-2027-HK1" → "HK1 2026–2027". */
function semesterText(sem: string) {
  const m = /^(\d{4})-(\d{4})-(HK\d)$/.exec(sem);
  return m ? `${m[3]} ${m[1]}–${m[2]}` : sem;
}

function clock(secs: number) {
  const m = Math.floor(secs / 60);
  return `${String(m).padStart(2, "0")}:${String(secs % 60).padStart(2, "0")}`;
}

/**
 * Vào lớp bằng mã: nhập mã → xem trước → tham gia. `initialCode` đến từ `/join/[code]`: xem trước ngay, URL đổi về `/join`
 * (mã không nằm lại trong thanh địa chỉ / `Referer`). Mọi lỗi sai mã cùng một câu; bị giới hạn thì đếm ngược.
 */
export function JoinScreen({ initialCode }: { initialCode?: string }) {
  const qc = useQueryClient();
  const [code, setCode] = useState(cleanCode(initialCode ?? ""));
  const [preview, setPreview] = useState<Preview | null>(null);
  const [done, setDone] = useState<Joined | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<{ text: string; notVerified?: boolean } | null>(null);
  const [wait, setWait] = useState(0);
  const started = useRef(false);

  useEffect(() => {
    if (wait <= 0) return;
    const t = window.setTimeout(() => setWait((w) => w - 1), 1000);
    return () => window.clearTimeout(t);
  }, [wait]);

  function fail(e: unknown) {
    if (e instanceof ApiError) {
      if (e.code === "RATE_LIMITED") {
        setWait(e.retryAfter ?? 600);
        setError(null);
        return;
      }
      if (e.code === "JOIN_CODE_INVALID") return setError({ text: BAD_CODE });
      if (e.code === "COURSE_FULL") return setError({ text: FULL });
      if (e.code === "EMAIL_NOT_VERIFIED") return setError({ text: "Hãy xác minh email trước khi vào lớp.", notVerified: true });
      return setError({ text: e.userMessage });
    }
    setError({ text: "Chưa thực hiện được. Hãy thử lại." });
  }

  async function look(c: string) {
    if (busy || c.length !== 7) return;
    setBusy(true);
    setError(null);
    try {
      setPreview((await apiClient.post<Preview>("/courses/join/preview", { code: c })).data);
    } catch (e) {
      setPreview(null);
      fail(e);
    } finally {
      setBusy(false);
    }
  }

  async function join() {
    if (busy) return;
    setBusy(true);
    setError(null);
    try {
      const { data } = await apiClient.post<Joined>("/courses/join", { code });
      setDone(data);
      await qc.invalidateQueries({ queryKey: ["me", "courses"] });
    } catch (e) {
      fail(e);
    } finally {
      setBusy(false);
    }
  }

  // /join/[code]: xem trước ngay một lần, rồi bỏ mã khỏi URL
  useEffect(() => {
    if (!initialCode || started.current) return;
    started.current = true;
    window.history.replaceState(null, "", "/join");
    void look(cleanCode(initialCode));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const blocked = wait > 0;
  const teachers = preview?.teachers.map((t) => t.full_name).join(", ");

  return (
    <Page>
      <PageHeader title="Tham gia lớp" description="Nhập mã tham gia giảng viên gửi cho bạn. Mã gồm 7 ký tự, không phân biệt chữ hoa chữ thường." />
      <Section panel>
        {done ? (
          <div className={s.result}>
            <p role="status" className={s.resultText}>
              {done.status === "PENDING" ? "Đã gửi yêu cầu, chờ giảng viên duyệt." : "Bạn đã vào lớp."}
            </p>
            <ButtonLink href="/" variant="primary">Về Hôm nay</ButtonLink>
          </div>
        ) : (
          <form
            className={s.form}
            onSubmit={(e) => {
              e.preventDefault();
              void look(code);
            }}
          >
            <Field label="Mã tham gia" error={error && !error.notVerified ? error.text : undefined} helper="7 ký tự, không có 0, O, 1, I, L.">
              {(id, describedBy) => (
                <Input
                  id={id}
                  aria-describedby={describedBy}
                  className={s.codeInput}
                  value={code}
                  invalid={Boolean(error && !error.notVerified)}
                  autoComplete="off"
                  autoCapitalize="characters"
                  spellCheck={false}
                  inputMode="text"
                  placeholder="AN7K2MQ"
                  onChange={(e) => {
                    setCode(cleanCode(e.target.value));
                    setPreview(null);
                    setError(null);
                  }}
                />
              )}
            </Field>
            {blocked && (
              <InlineNotice tone="warning" compact>
                Bạn đã thử quá nhiều lần. Thử lại sau <span className="ep-num">{clock(wait)}</span>.
              </InlineNotice>
            )}
            {error?.notVerified && (
              <InlineNotice tone="warning" compact>
                {error.text} <Link href="/verify-email">Gửi lại email xác minh</Link>
              </InlineNotice>
            )}
            {!preview && (
              <Button type="submit" variant="primary" loading={busy} disabled={code.length !== 7 || blocked}>
                Xem lớp
              </Button>
            )}
          </form>
        )}

        {!done && preview && (
          <div className={s.preview} data-part="join-preview">
            <p className={s.previewLine}>
              {[preview.name, preview.class_code, teachers, semesterText(preview.semester)].filter(Boolean).join(" · ")}
            </p>
            {preview.state === "REQUIRES_APPROVAL" && <p className={s.note}>Lớp này cần giảng viên duyệt.</p>}
            {preview.state === "FULL" && <p className={s.note}>{FULL}</p>}
            {preview.state === "ALREADY_MEMBER" && <p className={s.note}>Bạn đã ở trong lớp này.</p>}
            {preview.state === "PENDING" && <p className={s.note}>Bạn đã gửi yêu cầu, đang chờ giảng viên duyệt.</p>}
            {(preview.state === "OPEN" || preview.state === "REQUIRES_APPROVAL") && (
              <Button variant="primary" loading={busy} disabled={blocked} onClick={() => void join()}>
                Tham gia lớp
              </Button>
            )}
            {preview.state === "ALREADY_MEMBER" && <ButtonLink href="/" variant="primary">Về Hôm nay</ButtonLink>}
            {error && !error.notVerified && <p className={s.note} role="alert">{error.text}</p>}
          </div>
        )}
      </Section>
    </Page>
  );
}

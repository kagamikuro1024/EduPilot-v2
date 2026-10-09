"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { cleanCode } from "@/features/join/JoinScreen";
import { ApiErrorNotice } from "@/shared/data/ApiErrorNotice";
import { useSession } from "@/shared/session/session";
import { ActionList, ActionRow, Button, ButtonLink, Field, InlineNotice, Input, Page, PageHeader, Section, Skeleton, StatusText } from "@/shared/ui";
import s from "./Today.module.css";
import { hhmm, longDate, shortDay } from "./format";
import { useToday, type StudentToday as Data, type TodayItem, type TodaySession } from "./todayApi";

const CTA: Record<string, string> = { VERIFY_EMAIL: "Xác minh email" };
const STATE: Record<TodaySession["state"], { label: string; tone: "green" | "amber" | "neutral" }> = {
  NOW: { label: "Đang diễn ra", tone: "green" },
  NEXT: { label: "Sắp tới", tone: "amber" },
  DONE: { label: "Đã xong", tone: "neutral" },
};

/** "Hôm nay" của sinh viên: MỘT việc nên làm, dòng thời gian hôm nay, chỗ học dở. Không thẻ số liệu, không biểu đồ. */
export function StudentToday() {
  const { user } = useSession();
  const q = useToday<Data>();
  const first = user.name.split(" ").slice(-1)[0];
  const d = q.data;
  return (
    <Page>
      <PageHeader title={`Chào ${first}`} description={`${longDate(new Date())}${d?.no_course ? " · bạn chưa vào lớp nào" : ""}`} />
      {!d ? (
        q.isError ? <ApiErrorNotice error={q.error} onRetry={() => void q.refetch()} title="Chưa tải được việc hôm nay. Dữ liệu của bạn không bị ảnh hưởng." /> : <Skeleton lines={4} />
      ) : (
        <Body d={d} retry={q.isError ? () => void q.refetch() : undefined} />
      )}
    </Page>
  );
}

function Body({ d, retry }: { d: Data; retry?: () => void }) {
  const rec = d.recommended;
  const rowRec = rec && rec.kind !== "JOIN_CODE" ? rec : null; // chưa vào lớp: ô nhập mã thay cho khuyến nghị
  const calm = !rec && !d.no_course;
  return (
    <>
      {retry && (
        <InlineNotice tone="warning" compact action={<Button size="sm" onClick={retry}>Thử lại</Button>}>
          Chưa làm mới được. Đang hiện dữ liệu lần trước.
        </InlineNotice>
      )}
      {rowRec && (
        <Section title="Việc nên làm tiếp" panel>
          <ActionList label="Việc nên làm">
            <Recommended item={rowRec} />
          </ActionList>
        </Section>
      )}
      {d.no_course && (
        <Section panel title={rowRec ? "Vào lớp bằng mã" : "Nhập mã tham gia lớp"} description={rowRec ? undefined : "Bạn chưa vào lớp nào. Nhập mã do giảng viên cung cấp để bắt đầu."}>
          <JoinCodeBox primary={!rowRec?.href} />
        </Section>
      )}
      {calm && (
        <Section title="Việc nên làm tiếp" panel>
          <p className={s.calm}>Hôm nay bạn không có việc gấp.</p>
        </Section>
      )}
      {d.timeline.length > 0 && (
        <Section title="Hôm nay" panel>
          <ActionList label="Lịch hôm nay">
            {d.timeline.map((t) => (
              <ActionRow
                key={`${t.course.id}-${t.at}`}
                tone={t.state === "NOW" ? "green" : "neutral"}
                title={t.title}
                context={`${shortDay(t.at)} · ${hhmm(t.at)}–${hhmm(t.ends_at)}${t.place ? ` · ${t.place}` : ""}`}
                meta={<StatusText tone={STATE[t.state].tone}>{STATE[t.state].label}</StatusText>}
                href="/calendar"
              />
            ))}
          </ActionList>
        </Section>
      )}
    </>
  );
}

function Recommended({ item }: { item: TodayItem }) {
  return (
    <ActionRow
      tone="red"
      href={item.href || undefined}
      redThread={Boolean(item.href)}
      title={item.title}
      context={item.reason}
      meta={item.estimate_minutes ? `Khoảng ${item.estimate_minutes} phút` : undefined}
      action={
        item.href ? (
          <ButtonLink href={item.href} variant="primary" data-part="primary-action">
            {CTA[item.kind] ?? item.title}
          </ButtonLink>
        ) : undefined
      }
    />
  );
}

/** Ô nhập mã ngay trên trang: 7 ký tự, nhập xong chuyển `/join/<mã>` (xem trước rồi vào lớp). */
function JoinCodeBox({ primary }: { primary: boolean }) {
  const router = useRouter();
  const [raw, setRaw] = useState("");
  const code = cleanCode(raw);
  return (
    <form
      className={s.codeBox}
      onSubmit={(e) => {
        e.preventDefault();
        if (code.length === 7) router.push(`/join/${code}`);
      }}
    >
      <div className={s.codeRow}>
        <Field label="Mã tham gia lớp">
          {(id) => <Input id={id} className={s.codeInput} value={raw} onChange={(e) => setRaw(e.target.value)} autoComplete="off" autoCapitalize="characters" spellCheck={false} placeholder="AN7K2MQ" />}
        </Field>
        <Button type="submit" variant={primary ? "primary" : "secondary"} data-part={primary ? "primary-action" : undefined} disabled={code.length !== 7}>Tiếp tục</Button>
      </div>
      <p className={s.hint}>Mã gồm 7 chữ và số.</p>
    </form>
  );
}

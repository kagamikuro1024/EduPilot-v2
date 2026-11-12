"use client";

import { useQuery } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { ApiErrorNotice } from "@/shared/data/ApiErrorNotice";
import { ApiError, apiClient } from "@/shared/data";
import { GradeCalculation, type CalcLine } from "@/shared/domain";
import { Button, EmptyState, Input, Section, Skeleton } from "@/shared/ui";
import { gradeSentence, type MeGrades, type MeWhatIf } from "./meApi";
import s from "./Me.module.css";

const comma = (v: string) => v.replace(".", ",");
const SCORE = /^\d{1,2}([.,]\d{1,2})?$/;

/** Điểm của tôi (US-P6-09): câu nhận định, thành phần đã công bố, phép tính tuyến tính, tính thử "nếu cuối kỳ được X" tại chỗ. Chỉ đọc; không so sánh, không động viên. */
export function MeGrade({ courseId }: { courseId: string }) {
  const q = useQuery({
    queryKey: ["me", courseId, "grades"],
    staleTime: 0,
    refetchOnMount: "always",
    retry: false,
    queryFn: async ({ signal }) => (await apiClient.get<MeGrades>(`/courses/${courseId}/me/grades`, { signal })).data,
  });
  const g = q.data;
  return (
    <Section panel title="Điểm">
      {q.isPending ? (
        <Skeleton lines={4} />
      ) : !g ? (
        <ApiErrorNotice error={q.error} onRetry={() => void q.refetch()} title="Không tải được điểm của bạn." />
      ) : g.status === "NO_SCHEME" ? (
        <p className={s.headline}>{gradeSentence(g)}</p>
      ) : (
        <GradeBody courseId={courseId} g={g} />
      )}
    </Section>
  );
}

function GradeBody({ courseId, g }: { courseId: string; g: MeGrades }) {
  const comps = g.components ?? [];
  const lines = explainLines(comps, g.calculation ?? []);
  const published = comps.filter((c) => c.status === "PUBLISHED");
  const sentence = gradeSentence(g);
  if (published.length === 0 && !sentence) return <EmptyState title="Chưa có điểm nào được công bố.">Điểm sẽ hiện ở đây khi giảng viên công bố.</EmptyState>;
  return (
    <>
      {sentence && <p className={s.headline} data-part="grade-sentence">{sentence}</p>}
      {g.final?.failed_reason && <p className={s.sub}>{g.final.failed_reason}</p>}
      {lines.length > 0 && <GradeCalculation lines={lines} caption="Cách tính điểm của bạn" />}
      {g.status === "IN_PROGRESS" && g.qt && <WhatIf courseId={courseId} />}
      {g.official_note && <p className={s.sub} data-part="official-note">{g.official_note}</p>}
    </>
  );
}

/**
 * Một khối giải trình (US-UX91-20 AC1): mỗi thành phần một dòng "tên · trọng số" + điểm do máy chủ trả (không tính lại ở đây), "Căn cứ" lấy từ dòng phép tính
 * của thành phần đó; các dòng phép tính còn lại (điểm cộng, điểm quá trình…) theo thứ tự máy chủ.
 */
function explainLines(comps: NonNullable<MeGrades["components"]>, calc: NonNullable<MeGrades["calculation"]>): CalcLine[] {
  const used = new Set<number>();
  const rows = comps.map((c): CalcLine => {
    const label = `${c.name} · ${comma(c.weight)} %`;
    if (c.status !== "PUBLISHED" || c.score === undefined) return { label, expr: "", value: "Chưa công bố" };
    const i = calc.findIndex((l) => l.label.startsWith(`${c.name} (`));
    if (i >= 0) used.add(i);
    const l = calc[i];
    return { label, expr: l && l.expr !== l.value ? l.expr : "", value: c.score, quote: l?.source_quote, page: l?.page };
  });
  return [...rows, ...calc.filter((_, i) => !used.has(i)).map((l) => ({ label: l.label, expr: l.expr, value: l.value, quote: l.source_quote, page: l.page }))];
}
/** Ô "Nếu cuối kỳ được [ ]": gõ xong (350 ms) tính bằng máy chủ; số đã gõ giữ nguyên khi mạng chập chờn (kèm Thử lại). Không ghi gì. */
function WhatIf({ courseId }: { courseId: string }) {
  const [text, setText] = useState("");
  const [settled, setSettled] = useState("");
  const timer = useRef<ReturnType<typeof setTimeout>>(undefined);
  const check = (v: string) => SCORE.test(v.trim()) && Number(v.trim().replace(",", ".")) <= 10;
  const valid = check(text);
  const q = useQuery({
    queryKey: ["me", courseId, "what-if", settled],
    enabled: check(settled),
    retry: false,
    staleTime: 0,
    queryFn: async ({ signal }) => (await apiClient.post<MeWhatIf>(`/courses/${courseId}/me/grades/what-if`, { final_exam_score: settled.trim().replace(",", ".") }, { signal })).data,
  });
  const short = q.error instanceof ApiError && q.error.code === "NOT_ENOUGH_GRADES";
  const out = valid && check(settled) && settled === text ? q.data : undefined;
  return (
    <div className={s.whatif} data-part="what-if">
      <label className={s.whatifLabel}>
        <span>Nếu cuối kỳ được</span>
        <Input
          inputMode="decimal"
          aria-label="Điểm cuối kỳ giả định"
          className={s.whatifInput}
          value={text}
          invalid={text !== "" && !valid}
          placeholder="8,0"
          onChange={(e) => {
            const v = e.target.value;
            setText(v);
            clearTimeout(timer.current);
            timer.current = setTimeout(() => setSettled(v), 350);
          }}
        />
      </label>
      {text !== "" && !valid && <p className={s.sub} role="alert">Nhập số từ 0 đến 10</p>}
      {valid && short && <p className={s.sub}>Chưa đủ điểm để tính</p>}
      {valid && q.isError && !short && (
        <p className={s.sub} role="alert">
          Chưa tính được <Button size="sm" variant="ghost" onClick={() => void q.refetch()}>Thử lại</Button>
        </p>
      )}
      {out?.hp && (
        <p className={s.headline} data-part="what-if-result">
          {`Điểm học phần sẽ là ${comma(out.hp)} (${out.letter}).`}
          {out.failed ? ` ${out.failed}.` : ""}
        </p>
      )}
    </div>
  );
}

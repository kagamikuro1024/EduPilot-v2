"use client";

import { useEffect, useState } from "react";
import { CalendarPlus, Pin, Sparkles } from "lucide-react";
import { COURSE_1, fmtLongDate, fmtTime } from "@/mock/core";
import { REVIEW_SLOT, reportFor, type InsightTopic } from "@/mock/insights";
import { KEYS, type CalendarExtra, type InsightThread } from "@/mock/state";
import { useUndoLine } from "@/shared/lib/useUndoLine";
import { useSession } from "@/shared/session/session";
import { useDemoSlice } from "@/shared/state/demo";
import { Button, ButtonLink, EmptyState, InlineNotice, Page, PageHeader, PageState, Section, Skeleton, StatusText, useRouteState } from "@/shared/ui";
import s from "./insights.module.css";

const STEPS = ["Gom câu hỏi của lớp và bỏ tên, mã số sinh viên", "Nhóm câu hỏi theo chủ đề", "Xếp hạng chủ đề và soạn gợi ý dạy lại"];

export function InsightsScreen() {
  const { role, course } = useSession();
  const canWrite = role === "teacher";
  const [fresh, setFresh] = useDemoSlice<Record<string, boolean>>("insights.fresh", {});
  const [threads, setThreads] = useDemoSlice<InsightThread[]>(KEYS.insightThreads, []);
  const [extras, setExtras] = useDemoSlice<CalendarExtra[]>(KEYS.calendarExtras, []);
  const [step, setStep] = useState<number | null>(null);
  const undo = useUndoLine();

  const noReport = useRouteState() === "empty";
  const report = reportFor(course.id, Boolean(fresh[course.id]));

  useEffect(() => {
    if (step === null) return;
    const t = window.setTimeout(() => {
      if (step + 1 >= STEPS.length) {
        setFresh((prev) => ({ ...prev, [course.id]: true }));
        setStep(null);
      } else {
        setStep(step + 1);
      }
    }, 1000);
    return () => window.clearTimeout(t);
  }, [step, course.id, setFresh]);

  function pinThread(topic: InsightTopic) {
    const id = `ith-${course.id}-${topic.id}`;
    // Một chủ đề chỉ có MỘT thread ghim: bấm lại (hay quay lại màn) không tạo thêm (04-AC9).
    setThreads((prev) => (prev.some((t) => t.id === id) ? prev : [...prev, { id, courseId: course.id, title: `Chủ đề đang vướng: ${topic.title}`, topic: topic.title, body: topic.threadBody }]));
    undo.push(`Đã ghim thread “${topic.title}” vào Threads của lớp ${course.code}`, () => setThreads((prev) => prev.filter((t) => t.id !== id)));
  }

  function addReview(topic: InsightTopic) {
    const slot = REVIEW_SLOT[course.id] ?? REVIEW_SLOT[COURSE_1];
    const id = `ice-${course.id}-${topic.id}`;
    setExtras((prev) => [...prev, { id, courseId: course.id, title: `Ôn tập: ${topic.title}`, at: slot.at, durationMin: slot.durationMin }]);
    undo.push(`Đã thêm buổi ôn tập “${topic.title}” vào lịch lớp ${course.code} — ${slot.label}`, () => setExtras((prev) => prev.filter((e) => e.id !== id)));
  }

  const createButton = (
    <Button variant="primary" icon={<Sparkles aria-hidden />} loading={step !== null} onClick={() => setStep(0)}>
      Tạo báo cáo mới
    </Button>
  );

  return (
    <Page>
      <PageHeader
        title="Lỗ hổng kiến thức"
        description="Câu hỏi của lớp, đã bỏ tên và mã số sinh viên, nhóm theo chủ đề để bạn quyết định dạy lại phần nào."
        actions={canWrite && !noReport ? createButton : undefined}
        meta={
          <>
            <span>{course.label}</span>
            {!noReport && (
              <>
                <span>
                  Báo cáo {fmtLongDate(report.generatedAt)}, {fmtTime(report.generatedAt)}
                </span>
                <span>
                  {report.questions} câu hỏi đã nhóm chủ đề · {report.students} sinh viên
                </span>
              </>
            )}
            {!canWrite && <span>Trợ giảng xem báo cáo; giảng viên là người tạo báo cáo và mở thread</span>}
          </>
        }
      />

      <PageState
        loading={
          <div className={s.loading}>
            <Skeleton lines={2} />
            <Skeleton lines={5} />
            <Skeleton lines={5} />
          </div>
        }
        empty={
          <EmptyState
            title={`Chưa có báo cáo lỗ hổng kiến thức cho lớp ${course.code}`}
            action={canWrite ? createButton : undefined}
            icon={<Sparkles aria-hidden />}
          >
            Lớp đã có đủ câu hỏi để nhóm chủ đề. Tạo báo cáo đầu tiên để xem sinh viên đang vướng ở đâu; báo cáo chỉ dùng câu hỏi đã ẩn danh của lớp này.
          </EmptyState>
        }
        error={{
          problem: "Không tạo được báo cáo lỗ hổng kiến thức.",
          recovery: "Việc chạy nền đang xếp hàng chờ. Báo cáo cũ vẫn xem được. Thử lại sau vài phút, hoặc mở Quan sát AI để xem việc có bị kẹt không.",
        }}
      >
        {step !== null ? (
          <Section title="Đang tạo báo cáo" description="Việc chạy nền, bạn đi làm việc khác được; báo cáo sẽ hiện ở đây khi xong.">
            <ol className={s.steps} aria-live="polite">
              {STEPS.map((text, i) => (
                <li key={text} className={s.step}>
                  <StatusText tone={i < step ? "green" : i === step ? "blue" : "neutral"}>{i < step ? "Xong" : i === step ? "Đang chạy" : "Chờ"}</StatusText>
                  <span>{text}</span>
                </li>
              ))}
            </ol>
          </Section>
        ) : (
          <>
            <div className={s.privacy}>
              <InlineNotice tone="privacy" compact>
                Báo cáo chỉ dùng câu hỏi của {course.label}. Tên và mã số sinh viên đã bị bỏ trước khi nhóm chủ đề; chủ đề dưới 3 sinh viên hỏi không hiện câu mẫu.
              </InlineNotice>
            </div>

            <Section title="Chủ đề nên dạy lại" description={`${report.window} · xếp theo mức độ cả lớp đang vướng`}>
              <ol className={s.topics}>
                {report.topics.map((t, i) => (
                  <li key={t.id} className={s.topic}>
                    <div className={s.topicHead}>
                      <span className={s.rank} aria-hidden>
                        {i + 1}
                      </span>
                      <div className={s.topicTitle}>
                        <h3 className="ep-item-title">{t.title}</h3>
                        <p className="ep-meta">
                          {t.students} sinh viên · {t.questions} câu hỏi
                          {report.comparison && t.delta > 0 && ` · +${t.delta} so với báo cáo trước`}
                        </p>
                      </div>
                    </div>

                    <p className={s.why}>{t.why}</p>

                    <ul className={s.signals}>
                      {t.signals.map((sig) => (
                        <li key={sig}>{sig}</li>
                      ))}
                    </ul>

                    {t.samples.length > 0 ? (
                      <details className={s.samples}>
                        <summary>Xem {t.samples.length} câu hỏi đã ẩn danh</summary>
                        <ul>
                          {t.samples.map((q) => (
                            <li key={q}>“{q}”</li>
                          ))}
                        </ul>
                      </details>
                    ) : (
                      <p className={s.noSamples}>Dưới 3 sinh viên hỏi — không hiện câu mẫu để không suy ngược ra được là ai hỏi.</p>
                    )}

                    <p className={s.action}>
                      <span className={s.actionLabel}>Nên làm</span> {t.action}
                    </p>

                    {canWrite && (
                      <div className={s.topicActions}>
                        {threads.some((th) => th.id === `ith-${course.id}-${t.id}`) ? (
                          <ButtonLink size="sm" href={`/threads/ith-${course.id}-${t.id}`} icon={<Pin aria-hidden />}>
                            Đã ghim · Xem thread
                          </ButtonLink>
                        ) : (
                          <Button size="sm" icon={<Pin aria-hidden />} onClick={() => pinThread(t)}>
                            Tạo thread ghim
                          </Button>
                        )}
                        <Button
                          size="sm"
                          icon={<CalendarPlus aria-hidden />}
                          disabled={extras.some((e) => e.id === `ice-${course.id}-${t.id}`)}
                          onClick={() => addReview(t)}
                        >
                          {extras.some((e) => e.id === `ice-${course.id}-${t.id}`) ? "Đã có buổi ôn tập" : "Tạo buổi ôn tập"}
                        </Button>
                      </div>
                    )}
                  </li>
                ))}
              </ol>
              {undo.node}
            </Section>

            <Section title="Tài liệu chưa đề cập" description="Sinh viên hỏi nhưng không tài liệu nào của lớp trả lời được — AI phải chuyển sang bạn.">
              <ul className={s.gaps}>
                {report.uncovered.map((g) => (
                  <li key={g.title} className={s.gap}>
                    <div>
                      <p className="ep-item-title">{g.title}</p>
                      <p className={s.gapNote}>{g.note}</p>
                    </div>
                    <span className="ep-meta">{g.asks} câu hỏi</span>
                  </li>
                ))}
              </ul>
            </Section>

            {report.comparison && (
              <Section title="So với báo cáo trước" description="Bản trước tạo ngày 22/10; chỉ liệt kê thay đổi đáng kể.">
                <ul className={s.compare}>
                  {report.comparison.map((line) => (
                    <li key={line}>{line}</li>
                  ))}
                </ul>
              </Section>
            )}
          </>
        )}
      </PageState>
    </Page>
  );
}

"use client";

import { useEffect, useState } from "react";
import { FileText, Upload } from "lucide-react";
import {
  Button,
  ButtonLink,
  ConfirmIrreversible,
  Drawer,
  EmptyState,
  Field,
  InlineNotice,
  Input,
  Page,
  PageHeader,
  PageState,
  Skeleton,
  Split,
  StatusText,
} from "@/shared/ui";
import { COURSE_2 } from "@/mock/core";
import { SCHEME_DOC_1, SCHEME_DOC_2, type SchemeRule } from "@/mock/gradebook";
import { KEYS, SCHEMES_SEED, type SchemesState } from "@/mock/state";
import { useSession } from "@/shared/session/session";
import { useDemoSlice } from "@/shared/state/demo";
import { useUndoLine } from "@/shared/lib/useUndoLine";
import s from "./GradeScheme.module.css";

const STEPS = ["Đang tải tệp lên…", "Đang đọc nội dung quy chế…", "Đang rút ra các mục công thức…"];

export function GradeScheme() {
  const { role, course } = useSession();
  const [schemes, setSchemes] = useDemoSlice<SchemesState>(KEYS.schemes, SCHEMES_SEED);
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState(0);
  const [cite, setCite] = useState<SchemeRule | null>(null);
  const [confirming, setConfirming] = useState(false);
  const undo = useUndoLine();

  const doc = course.id === COURSE_2 ? SCHEME_DOC_2 : SCHEME_DOC_1;
  const state = schemes[course.id]?.status ?? "none";
  const rounding = schemes[course.id]?.rounding ?? (doc.rules.find((r) => r.id === "rounding")?.value ?? "");
  const blocked = rounding.trim() === "";

  useEffect(() => {
    if (!uploading) return;
    const t = window.setInterval(() => setProgress((p) => Math.min(100, p + 7)), 110);
    return () => window.clearInterval(t);
  }, [uploading]);

  function uploadScheme() {
    setProgress(0);
    setUploading(true);
    window.setTimeout(() => {
      setUploading(false);
      setSchemes((prev) => ({ ...prev, [course.id]: { status: "draft" } }));
    }, 1600);
  }

  function setRounding(value: string) {
    setSchemes((prev) => ({ ...prev, [course.id]: { status: prev[course.id]?.status ?? "draft", rounding: value } }));
  }

  const rules = doc.rules.map((r) => (r.id === "rounding" ? { ...r, value: rounding.trim() === "" ? null : rounding } : r));

  return (
    <Page>
      <PageHeader
        title="Công thức điểm"
        description={`${course.label} · rút từ ${doc.file}`}
        back={{ href: "/gradebook", label: "Sổ điểm" }}
        meta={
          <>
            <span>{doc.pages} trang</span>
            {state === "confirmed" ? (
              <StatusText tone="green">Đã xác nhận</StatusText>
            ) : state === "draft" ? (
              <StatusText tone="amber">Bản nháp do AI rút ra · chờ giảng viên xác nhận</StatusText>
            ) : (
              <StatusText tone="neutral">Chưa có quy chế</StatusText>
            )}
          </>
        }
      />

      <PageState
        loading={<Skeleton lines={10} />}
        empty={
          <EmptyState
            title="Chưa có quy chế điểm cho lớp này"
            icon={<FileText aria-hidden />}
            action={
              role === "teacher" ? (
                <Button
                  variant="primary"
                  icon={<Upload aria-hidden />}
                  onClick={uploadScheme}
                  disabled={uploading}
                >
                  Tải quy chế
                </Button>
              ) : (
                <ButtonLink href="/gradebook">Về sổ điểm</ButtonLink>
              )
            }
          >
            {uploading ? (
              <span className={s.progress}>
                <span className={s.bar}>
                  <span className={s.barFill} style={{ width: `${progress}%` }} />
                </span>
                <span className={s.progressText}>{STEPS[Math.min(STEPS.length - 1, Math.floor(progress / 34))]}</span>
              </span>
            ) : (
              "Tải tệp quy chế của lớp (PDF) để EduPilot rút ra các mục công thức điểm. Bản rút ra chỉ là nháp — bạn xem lại từng mục rồi mới xác nhận."
            )}
          </EmptyState>
        }
        error={{
          problem: "Không đọc được tệp quy chế.",
          recovery: "Tệp vẫn được giữ nguyên trên máy bạn. Thử lại, hoặc tải lên bản PDF có lớp chữ.",
        }}
        state={state === "none" ? "empty" : undefined}
      >
        <Split
          main={
            <>
              {state === "draft" && (
                <InlineNotice tone="info" title="Đây là bản nháp do AI rút ra từ quy chế">
                  Mỗi mục đều kèm đoạn trích và số trang trong {doc.file}. Giảng viên đọc lại, điền mục còn thiếu rồi xác nhận — điểm chỉ tính theo công thức đã xác nhận.
                </InlineNotice>
              )}
              {state === "confirmed" && (
                <InlineNotice tone="success" title="Công thức đã được giảng viên xác nhận">
                  Sổ điểm của lớp đang tính theo đúng các mục dưới đây. Mỗi mục vẫn giữ nguồn trích để đối chiếu khi cần.
                </InlineNotice>
              )}

              {undo.node}

              <div className={s.rules}>
                {rules.map((r) => (
                  <section key={r.id} className={s.rule}>
                    <div className={s.ruleHead}>
                      <span className={s.ruleLabel}>{r.label}</span>
                      <button type="button" className={s.cite} onClick={() => setCite(r)}>
                        <FileText aria-hidden />
                        Trang {r.page}
                      </button>
                    </div>
                    {r.value === null ? (
                      <>
                        <p className={s.ruleMissing}>Chưa rõ: quy chế không nêu quy tắc làm tròn</p>
                        <div className={s.ask}>
                          <p>{doc.rules.find((x) => x.id === r.id)?.ask}</p>
                          <div className={s.askRow}>
                            <Field className={s.askField} label="Quy tắc làm tròn" helper="Ví dụ: Làm tròn đến 0,1">
                              {(id, by) => (
                                <Input
                                  id={id}
                                  aria-describedby={by}
                                  value={rounding}
                                  placeholder="Làm tròn đến 0,1"
                                  disabled={role !== "teacher"}
                                  onChange={(e) => setRounding(e.target.value)}
                                />
                              )}
                            </Field>
                          </div>
                        </div>
                      </>
                    ) : (
                      <p className={s.ruleValue}>{r.value}</p>
                    )}
                    <blockquote className={s.quote}>{r.quote}</blockquote>
                  </section>
                ))}
              </div>

              {role === "teacher" && state !== "confirmed" && (
                <div className={s.sticky}>
                  <p className={s.stickyText}>
                    {blocked
                      ? "Còn 1 mục chưa rõ. Điền quy tắc làm tròn để xác nhận công thức."
                      : "Sau khi xác nhận, sổ điểm của lớp sẽ tính theo đúng công thức này."}
                  </p>
                  {blocked ? (
                    <span className={s.lock}>
                      <Button variant="primary" className={s.lockBtn} aria-disabled="true" onClick={() => undefined}>
                        Xác nhận công thức
                      </Button>
                      <span role="note" className={s.lockWhy}>
                        Chưa xác nhận được: mục &ldquo;Quy tắc làm tròn&rdquo; còn trống. Điền ô ngay tại mục đó rồi xác nhận.
                      </span>
                    </span>
                  ) : (
                    <Button variant="primary" onClick={() => setConfirming(true)}>
                      Xác nhận công thức
                    </Button>
                  )}
                </div>
              )}
            </>
          }
          aside={
            <div className={s.sourcePanel}>
              <div>
                <p className={s.sourceHead}>Nguồn dùng để rút công thức</p>
                <p className={s.sourceFile}>
                  {doc.file} · {doc.pages} trang
                </p>
              </div>
              {rules.map((r) => (
                <div key={r.id} className={s.sourceItem}>
                  <span className={s.sourcePage}>
                    Trang {r.page} · {r.label}
                  </span>
                  <span className={s.sourceQuote}>{r.quote}</span>
                </div>
              ))}
            </div>
          }
        />
      </PageState>

      <Drawer
        open={Boolean(cite)}
        onClose={() => setCite(null)}
        title={cite ? `${doc.file} · trang ${cite.page}` : ""}
        description={cite?.label}
      >
        {cite && (
          <>
            <blockquote className={s.quote}>{cite.quote}</blockquote>
            <p className={s.sourcePage}>Đoạn trích nguyên văn từ quy chế đã tải lên. Mục công thức tương ứng được rút ra từ đoạn này.</p>
          </>
        )}
      </Drawer>

      {role === "teacher" && (
        <ConfirmIrreversible
          open={confirming}
          onClose={() => setConfirming(false)}
          onConfirm={() => {
            setSchemes((prev) => ({ ...prev, [course.id]: { status: "confirmed", rounding } }));
            undo.push("Đã xác nhận công thức điểm của lớp");
          }}
          title="Xác nhận công thức điểm"
          consequence={`Từ lúc này, điểm quá trình của ${course.code} tính theo: quá trình ${doc.scheme.qtWeight}% · cuối kỳ ${100 - doc.scheme.qtWeight}%, cộng ${doc.scheme.speakBonus.toFixed(2).replace(".", ",")} mỗi lần phát biểu (trần ${doc.scheme.speakCap.toFixed(1).replace(".", ",")}), trừ ${doc.scheme.absencePenalty.toFixed(1).replace(".", ",")} mỗi buổi vắng từ buổi thứ ${doc.scheme.absenceFrom}, ${rounding.toLowerCase()}. Sinh viên sẽ thấy lớp đã có công thức điểm chính thức.`}
          confirmLabel="Xác nhận công thức"
        />
      )}
    </Page>
  );
}

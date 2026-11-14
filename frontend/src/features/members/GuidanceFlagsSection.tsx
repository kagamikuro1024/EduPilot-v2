"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { ApiError } from "@/shared/data";
import { InlineNotice, Section, Skeleton, Switch } from "@/shared/ui";
import { flagsKey, setGuidanceFlag, useGuidanceFlags, type GuidanceFeature } from "@/features/assignments/guidanceApi";

const LABEL: Record<GuidanceFeature, string> = {
  GRADING_GUIDANCE: "Học từ lần sửa điểm",
  THREAD_SUGGEST: "Gợi ý bài đã trả lời khi đặt câu hỏi",
  QUESTION_HINTS: "Nhớ lý do bỏ câu hỏi khi sinh đề",
  REGRESSION_CAPTURE: "Lưu ca kiểm thử chấm",
};
const ORDER: GuidanceFeature[] = ["GRADING_GUIDANCE", "THREAD_SUGGEST", "QUESTION_HINTS", "REGRESSION_CAPTURE"];

/** Bốn công tắc theo lớp (FEAT-grading-guidance 4.1). Hiệu lực ngay, không nút Lưu. Chỉ giảng viên / quản trị viên thấy. */
export function GuidanceFlagsSection({ courseId, canManage }: { courseId: string; canManage: boolean }) {
  const qc = useQueryClient();
  const q = useGuidanceFlags(courseId, canManage);
  const [err, setErr] = useState<string | null>(null);
  if (!canManage || q.isError) return null;
  if (q.isPending) return <Section panel title="Học từ lần sửa"><Skeleton lines={3} /></Section>;
  const by = new Map(q.data.map((f) => [f.feature, f]));

  async function toggle(feature: GuidanceFeature, enabled: boolean) {
    const cur = by.get(feature);
    if (!cur) return;
    setErr(null);
    try {
      await setGuidanceFlag(courseId, feature, enabled, cur.version);
    } catch (e) {
      setErr(e instanceof ApiError ? e.userMessage : "Chưa lưu được. Hãy thử lại.");
    }
    await qc.invalidateQueries({ queryKey: flagsKey(courseId) });
  }

  return (
    <Section panel title="Học từ lần sửa">
      <div data-part="guidance-flags">
        {ORDER.filter((f) => by.has(f)).map((f) => {
          const cur = by.get(f)!;
          return (
            <Switch
              key={f}
              label={LABEL[f]}
              checked={cur.enabled}
              disabled={!cur.can_edit}
              title={!cur.can_edit ? "Chỉ quản trị viên bật" : undefined}
              onChange={(next) => void toggle(f, next)}
            />
          );
        })}
      </div>
      {err && <InlineNotice tone="danger" compact>{err}</InlineNotice>}
    </Section>
  );
}

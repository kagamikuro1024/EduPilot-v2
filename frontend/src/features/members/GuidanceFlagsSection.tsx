"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { useUndoableAction } from "@/shared/data";
import { Section, Skeleton, StatusText, Switch } from "@/shared/ui";
import { flagsKey, setGuidanceFlag, useGuidanceFlags, type GuidanceFeature, type GuidanceFlag } from "@/features/assignments/guidanceApi";
import s from "./ClassSettings.module.css";

// Nhãn và thứ tự theo US-GD-01 AC5.
const LABEL: Record<GuidanceFeature, string> = {
  GRADING_GUIDANCE: "Lưu ý chấm",
  REGRESSION_CAPTURE: "Ca kiểm thử",
  THREAD_SUGGEST: "Gợi ý câu hỏi tương tự",
  QUESTION_HINTS: "Gợi ý khi sinh câu hỏi",
};
const ORDER: GuidanceFeature[] = ["GRADING_GUIDANCE", "REGRESSION_CAPTURE", "THREAD_SUGGEST", "QUESTION_HINTS"];

function FlagRow({ courseId, flag }: { courseId: string; flag: GuidanceFlag }) {
  const qc = useQueryClient();
  const [on, setOn] = useState(flag.enabled); // lạc quan: hiện ngay, lùi nếu máy chủ từ chối
  const ver = useRef(flag.version);
  const undo = useUndoableAction<boolean>({
    apply: (next) => setOn(next),
    rollback: (next) => setOn(!next),
    commit: async (next) => {
      ver.current = (await setGuidanceFlag(courseId, flag.feature, next, ver.current)).version;
    },
    compensate: async (next) => {
      ver.current = (await setGuidanceFlag(courseId, flag.feature, !next, ver.current)).version;
    },
    label: (next) => `${next ? "Đã bật" : "Đã tắt"} ${LABEL[flag.feature]}`,
  });
  return (
    <div data-part="guidance-flag" data-feature={flag.feature}>
      <div className={s.flagRow}>
        <Switch
          label={LABEL[flag.feature]}
          checked={on}
          disabled={!flag.can_edit || undo.pending}
          title={!flag.can_edit ? "Chỉ quản trị viên bật" : undefined}
          onChange={(next) => void undo.run(next).then(() => qc.invalidateQueries({ queryKey: flagsKey(courseId) }))}
        />
        <StatusText tone={on ? "green" : "neutral"}>{on ? "Bật" : "Tắt"}</StatusText>
      </div>
      {undo.node}
    </div>
  );
}

/** Nhóm «Học từ lần sửa» ở /class/settings: bốn công tắc có hiệu lực ngay (lạc quan + «Hoàn tác»), trạng thái 1–2 từ, không câu mô tả. Chỉ giảng viên / quản trị viên thấy. */
export function GuidanceFlagsSection({ courseId, canManage }: { courseId: string; canManage: boolean }) {
  const q = useGuidanceFlags(courseId, canManage);
  if (!canManage || q.isError) return null;
  if (q.isPending) return <Section panel title="Học từ lần sửa"><Skeleton lines={3} /></Section>;
  const by = new Map(q.data.map((f) => [f.feature, f]));
  return (
    <Section panel title="Học từ lần sửa">
      <div data-part="guidance-flags">
        {ORDER.filter((f) => by.has(f)).map((f) => (
          <FlagRow key={f} courseId={courseId} flag={by.get(f)!} />
        ))}
      </div>
    </Section>
  );
}

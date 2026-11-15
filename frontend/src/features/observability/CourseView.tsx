"use client";

import { DefinitionList, EmptyState, PageState, Section, Skeleton } from "@/shared/ui";
import { pct, useCourseSummary } from "./api";

/** Bản tổng hợp 7 ngày của MỘT lớp: không bảng yêu cầu, không nội dung, không chi phí theo nhà cung cấp. */
export function CourseView({ courseId }: { courseId: string | null }) {
  const q = useCourseSummary(courseId);
  return (
    <PageState query={q} isEmpty={() => courseId === null} loading={<Skeleton lines={6} />} empty={<EmptyState title="Chưa có lớp nào." />}>
      {q.data && (
        <Section panel title="7 ngày qua">
          <DefinitionList
            items={[
              { term: "Yêu cầu", value: q.data.requests.toLocaleString("vi-VN") },
              { term: "Tỷ lệ lỗi", value: pct(q.data.error_rate) },
              { term: "Chuyển giảng viên", value: String(q.data.escalations) },
              { term: "Lần che thông tin", value: String(q.data.pii_masked) },
              { term: "Hữu ích / Chưa hữu ích", value: `${q.data.helpful} / ${q.data.not_helpful}` },
              { term: "Hỏi đáp công khai: đúng / loại", value: `${q.data.thread_correct} / ${q.data.thread_reject}` },
              { term: "Điểm AI bị sửa", value: q.data.graded_total === 0 ? "—" : `${q.data.graded_changed} / ${q.data.graded_total} (${pct(q.data.change_rate)}), lệch trung bình ${q.data.mean_abs_deviation}` },
            ]}
          />
        </Section>
      )}
    </PageState>
  );
}

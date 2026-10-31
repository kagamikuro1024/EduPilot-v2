"use client";

import { useSearchParams } from "next/navigation";
import { useState } from "react";
import { useSession } from "@/shared/session/session";
import { StudentAssignment } from "./StudentAssignment";

/**
 * `/assignments/[id]` là màn của sinh viên (nav chặn vai khác). `?course=<id>` do liên kết dẫn tới đây mang theo:
 * đọc ngay ở lượt dựng đầu (trước khi `useCourseDeepLink` bỏ tham số) để gọi `…/mine` SONG SONG với `/me/courses`,
 * không có tham số thì chờ lớp đang chọn như trước. Nhập tĩnh `StudentAssignment`: không thêm một vòng tải chunk + CSS.
 */
export function AssignmentScreen({ id }: { id: string }) {
  const params = useSearchParams();
  const [hinted] = useState(() => params.get("course"));
  const { identity, realCourseId, realPending } = useSession();
  const courseId = hinted ?? (realPending ? null : realCourseId);
  if (!identity || !courseId || courseId === "all") return null;
  return <StudentAssignment key={`${courseId}:${id}`} courseId={courseId} id={id} />;
}

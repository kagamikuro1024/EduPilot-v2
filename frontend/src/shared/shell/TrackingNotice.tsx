"use client";

import { useQueryClient } from "@tanstack/react-query";
import { usePathname } from "next/navigation";
import { useState } from "react";
import { ApiError, apiClient, areaOf, useActivityHeartbeat } from "@/shared/data";
import { ALL_COURSES } from "@/shared/session/cookies";
import { useTrackingNotice } from "@/shared/session/myCourses";
import { useSession } from "@/shared/session/session";
import { Button, Dialog } from "@/shared/ui";

/**
 * Thời gian học của sinh viên (US-P5-03): hộp thoại minh bạch một lần (SRS crm-360 7.5) và nhịp `useActivityHeartbeat` — nhịp chỉ chạy SAU `Đã hiểu`.
 * Chỉ vai Sinh viên có lớp thật; giảng viên / TA / Admin không thấy hộp thoại và không bị đo.
 */
export function StudyTimeTracker() {
  const { role, realCourseId } = useSession();
  const pathname = usePathname();
  const qc = useQueryClient();
  const on = role === "student" && !!realCourseId && realCourseId !== ALL_COURSES;
  const notice = useTrackingNotice(on);
  useActivityHeartbeat({ courseId: realCourseId, area: areaOf(pathname), enabled: on && notice.data?.acked === true });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function ack() {
    setBusy(true);
    setError(null);
    try {
      await apiClient.post("/me/tracking-notice/ack");
      await qc.invalidateQueries({ queryKey: ["me", "courses"] });
    } catch (e) {
      setError(e instanceof ApiError ? e.userMessage : "Chưa lưu được. Thử lại.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={on && notice.data?.acked === false}
      onClose={() => undefined}
      dismissible={false}
      title="Thời gian học trên EduPilot"
      error={error}
      footer={
        <Button variant="primary" loading={busy} onClick={() => void ack()}>
          Đã hiểu
        </Button>
      }
    >
      <p>EduPilot ghi thời gian bạn dùng Chat, Threads, Luyện đề, Thư viện và Lịch khi tab đang mở và bạn đang thao tác.</p>
      <p>Không ghi nội dung bạn xem hay gõ.</p>
      <p>Giảng viên xem được; không dùng để tính điểm.</p>
      {notice.data?.optOutAvailable && <p>Bạn có thể tắt ở Cài đặt.</p>}
    </Dialog>
  );
}

import type { Metadata } from "next";
import { HomeGate } from "@/features/landing/HomeGate";
import { Landing } from "@/features/landing/Landing";
import { RETURNING_KEY } from "@/shared/session/returning";
import { PreShell } from "@/shared/shell/PreShell";
import { introFor } from "@/shared/shell/intro";

export const metadata: Metadata = { title: { absolute: "EduPilot — vận hành lớp học có trợ lý AI" }, description: "Hỏi đáp, chấm bài, sổ điểm và theo dõi lớp học, có trợ lý AI nhưng giảng viên là người quyết." };

// Trang chủ có hai bộ mặt: chưa đăng nhập → giới thiệu (HTML tĩnh từ máy chủ, không cần phiên); đã đăng nhập → "Hôm nay" như cũ.
// Máy chủ không biết phiên (access token nằm ở bộ nhớ trình duyệt), nên cả hai khung đều có trong HTML; đoạn script nhỏ đặt cờ `data-returning` lên <html>
// từ dấu "đã từng đăng nhập ở máy này" TRƯỚC khi vẽ, để người quay lại thấy khung "Hôm nay" (PreShell) chứ không nháy trang giới thiệu.
export default function Home() {
  return (
    <>
      <script dangerouslySetInnerHTML={{ __html: `try{if(localStorage.getItem(${JSON.stringify(RETURNING_KEY)})==="1")document.documentElement.dataset.returning="1"}catch(e){}` }} />
      <HomeGate landing={<Landing />} pre={<PreShell intro={introFor("/")!} />} />
    </>
  );
}

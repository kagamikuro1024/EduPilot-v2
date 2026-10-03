import { notFound } from "next/navigation";

// Bắt mọi đường dẫn không khớp route nào để màn "Không tìm thấy trang" dựng trong khung app.
export default function UnknownRoute() {
  notFound();
}

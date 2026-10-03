import { notFound } from "next/navigation";

// Mọi /dev/* không có trang riêng → 404 (không rơi vào catch-all của (app) rồi chuyển sang /login).
export default function Page() {
  notFound();
}

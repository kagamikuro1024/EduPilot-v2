"use client";

import { Page, PageHeader, Panel, Skeleton, Tabs } from "@/shared/ui";

// Khung ba màn Luyện đề: chữ + tiêu đề có sẵn trong HTML máy chủ và trong `loading` của chunk nạp lười, để LCP không đợi chuỗi chunk (DEF-R10S-20).
const BACK = { href: "/practice", label: "Luyện đề" };
export const HEAD = { title: "Luyện đề", secondary: [{ label: "Lịch sử luyện tập", href: "/practice/history" }] as const };

/** Khung tab khi chưa có dữ liệu (đang tải lớp / tổng quan, lỗi): nhãn như khi có dữ liệu, không số đếm (X3, DEF-R10.7-03). */
export function IdleTabs({ mode = "topic", onChange, locked = false }: { mode?: "topic" | "mock"; onChange?: (m: "topic" | "mock") => void; /** khung chờ chưa giữ lựa chọn: khoá tab thay vì nuốt cú bấm (chập chờn ở e2e) */ locked?: boolean }) {
  return <Tabs label="Cách luyện" value={mode} onChange={onChange ?? (() => {})} options={[{ value: "topic", label: "Theo chủ đề", count: 0, disabled: locked }, { value: "mock", label: "Thi thử", disabled: locked }]} />;
}

export const HomeShell = () => <Page kind="read"><PageHeader {...HEAD} /><IdleTabs locked /><Panel><Skeleton lines={6} /></Panel></Page>;
export const HistoryShell = () => <Page kind="read"><PageHeader title="Lịch sử luyện tập" back={BACK} /><Panel><Skeleton lines={6} /></Panel></Page>;
export const RunShell = () => <Page kind="work"><PageHeader title="Làm bài" back={BACK} /><Panel><Skeleton lines={6} /></Panel></Page>;

import type { RouteIntro } from "./intro";
import shell from "./AppShell.module.css";
import pre from "./PreShell.module.css";
import { Page, PageHeader, Panel } from "@/shared/ui";

/**
 * Khung trước phiên (US-PU-06): cùng hình học với AppShell (thanh trên, cột bên, vùng chính) và cùng lớp CSS của PageHeader,
 * nên khi khung thật thay vào, tiêu đề / câu phụ KHÔNG dịch chỗ. Đoạn `body` nằm trong `Panel` như màn thật (cùng bề rộng chữ → cùng số dòng → không ứng viên LCP mới). Không có nút, liên kết hay dữ liệu người dùng; chỉ là chữ tĩnh.
 */
export function PreShell({ intro, landmark = "main" }: { intro: RouteIntro; /** `div`: HTML của `/` còn mang cả trang giới thiệu (có `<main>` thật); khung này là `div role=main` để DOM chỉ có MỘT thẻ `main` */ landmark?: "main" | "div" }) {
  const Main = landmark === "main" ? "main" : "div";
  return (
    <div className={[shell.shell, pre.pre].join(" ")} data-part="pre-shell">
      <header className={shell.topbar} data-part="topbar">
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img className={shell.mobileBrand} src="/brand/logo-edupilot-mark.svg" alt="" width={28} height={28} />
      </header>
      <aside className={shell.sidebar} data-part="sidebar" aria-hidden="true">
        <div className={shell.brand}>
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src="/brand/logo-edupilot.svg" alt="" height={32} className={shell.logo} />
        </div>
      </aside>
      <Main id="main" role={landmark === "div" ? "main" : undefined} className={shell.main} aria-busy="true">
        <Page kind={intro.width ?? "read"}>
          <PageHeader title={intro.title} />
          {intro.early && (
            // ă đ ơ ư thuộc tệp phông latin-ext (khai báo sau «vietnamese» nên thắng khi trùng dải) — tệp này không được preload: không có span này nó chỉ tải SAU khi chữ dữ liệu đã chèn → chữ chờ thêm ~1 RTT (phông display:block). visibility:hidden vẫn dàn trang nên trình duyệt tải phông ngay lúc dựng HTML.
            <span className={pre.fontWarm} aria-hidden="true">ăđơư</span>
          )}
          {intro.early && (
            <div hidden suppressHydrationWarning>
              <Panel>
                <p className={intro.early === "question" ? pre.earlyQuestion : pre.early} data-part="early-body" suppressHydrationWarning dangerouslySetInnerHTML={{ __html: "" }} />
              </Panel>
            </div>
          )}
          {intro.body && (
            <Panel>
              <p className={pre.body}>{intro.body}</p>
            </Panel>
          )}
        </Page>
      </Main>
    </div>
  );
}

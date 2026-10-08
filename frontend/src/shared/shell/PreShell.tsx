import type { RouteIntro } from "./intro";
import shell from "./AppShell.module.css";
import pre from "./PreShell.module.css";
import ls from "@/shared/ui/Layout.module.css";

/**
 * Khung trước phiên (US-PU-06): cùng hình học với AppShell (thanh trên, cột bên, vùng chính) và cùng lớp CSS của PageHeader,
 * nên khi khung thật thay vào, tiêu đề / câu phụ KHÔNG dịch chỗ. Không có nút, liên kết hay dữ liệu người dùng; chỉ là chữ tĩnh.
 */
export function PreShell({ intro }: { intro: RouteIntro }) {
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
      <main id="main" className={shell.main} aria-busy="true">
        <div className={[ls.page, ls[intro.width ?? "reading"]].join(" ")}>
          <div className={ls.header}>
            <div className={ls.headRow}>
              <div className={ls.headText}>
                <h1 className="ep-page-title" data-part="page-title">
                  {intro.title}
                </h1>
                <p className={ls.desc}>{intro.description}</p>
              </div>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}

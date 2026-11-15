### DEF-R1.5-01 — Đã đăng nhập mà mở đường dẫn không có thật bị đưa về /login thay vì màn «Không tìm thấy trang»
- Mức: Trung bình
- Sprint gốc: 1.5 · TC gốc: US-PROTO-00/TC-00-57, TC-00-59 (AC13 / FR-X16) · Hồi quy do sprint sau?: CÓ (cổng AuthGate + `(app)/[...slug]` ở sprint P1-PU; không rõ commit) → KHÔNG RÕ
- Môi trường: ep-r, be0c0d96, vai sv.kha (và ta/teacher/admin), 1440 px
- Bước tái hiện: 1. Đăng nhập sv.kha tại https://localhost:1143/login. 2. `page.goto('/me')` → vẫn đăng nhập (POST /auth/refresh 200). 3. `page.goto('/khong-co-trang')` (hoặc /threads/khong-co, /assignments/khong-co) → URL thành `/login?next=%2Fkhong-co-trang`, không có lời gọi /auth/refresh, trang là màn Đăng nhập. Script: evidence-1.5/scripts/dbg-404.mjs.
- Kỳ vọng: màn «Không tìm thấy trang» + nút «Về Hôm nay» (frontend/src/app/(app)/not-found.tsx, app/not-found.tsx) · Thực tế: màn Đăng nhập
- Bằng chứng: docs/regression/evidence-1.5/matrix-*.json (mục notfound) · Nghi ngờ mã: frontend/src/shared/session/AuthGate.tsx:36-49 (bounce → /login?next=) khi pathname không thuộc route đã biết; frontend/src/app/(app)/[...slug]/page.tsx
- Chạy lại lần 2: ổn định (tái hiện 3/3)

### DEF-R1.5-02 — TA: nút hồ sơ ở topbar tràn ngang ở 375 và 390 px
- Mức: Thấp (phụ thuộc dữ liệu: TA là thành viên lớp có tên dài, vd «QC-AS3r108 · QC điểm danh QC-AS3r108»)
- Sprint gốc: 1.5 · TC gốc: US-PROTO-00/TC-00-36 (AC10: không tràn ngang) · Hồi quy do sprint sau?: KHÔNG RÕ
- Môi trường: ep-r, be0c0d96, ta@edupilot.local, trang «/»
- Bước tái hiện: 1. Đăng nhập ta@edupilot.local. 2. Viewport 375×812 (390×812) → `document.documentElement.scrollWidth - innerWidth` = 24 (9). 3. Phần tử tràn: `AppShell topEnd` / `Menu root` / `profile` (right = 399). GV cùng viewport: 0; 414 px: 0.
- Kỳ vọng: không cuộn ngang ở 375/390 · Thực tế: tràn 24/9 px · Bằng chứng: evidence-1.5/matrix-ta.json (overflow), evidence-1.5/scripts/dbg-overflow.mjs · Nghi ngờ mã: frontend/src/shared/shell/AppShell (khối topEnd không co lại khi bộ chọn lớp dài)
- Chạy lại lần 2: ổn định

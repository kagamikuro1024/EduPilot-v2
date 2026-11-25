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


## Ca lỗi thời đã XOÁ khỏi smoke (lệnh chủ dự án; mỗi mã một dòng + nguồn)
- TC-00-01: cookie ep_demo_role/dải «Bản mô phỏng» đã bỏ (D51); phần chuyển /inbox,/chat,/gradebook,/me → /login vẫn đúng, «/» nay là trang giới thiệu
- TC-00-02…TC-00-08: mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn)
- TC-00-11…TC-00-16: mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn)
- TC-00-20…TC-00-35: mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn)
- TC-00-38…TC-00-47: mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn)
- TC-00-49…TC-00-53: mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn)
- TC-00-55…TC-00-56: mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn)
- TC-00-58: mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn)
- TC-00-61…TC-00-67: mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn)
- TC-00-69: mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn)
- TC-01-02…TC-01-25: mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn)
- TC-01-28…TC-01-99: mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn)
- TC-02-02…TC-02-20: mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn)
- TC-02-22…TC-02-91: mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn)
- TC-03-02…TC-03-22: mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn)
- TC-03-24…TC-03-41: mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn)
- TC-04-02…TC-04-03: mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn)
- TC-04-05…TC-04-09: mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn)
- TC-04-11…TC-04-13: mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn)
- TC-04-15…TC-04-16: mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn)
- TC-04-19…TC-04-48: mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn)
- TC-04-50: mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn)
- TC-DEMO-00…TC-DEMO-38: mốc đã bị thay (D51: màn mock → màn thật; dữ liệu D1–D5/id mock/cookie vai/`?state=` không còn)

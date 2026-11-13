# Dev — sửa CI đỏ (run 43460419470 @ 18400636)

| Lỗi CI | Nguyên nhân | Sửa |
| --- | --- | --- |
| Frontend (lint + build) → `lint-selftest.sh`: «0 / 22 phép ui-antipatterns» | `ui-antipatterns.sh` có 23 phép từ 9.1 nhưng selftest ghim «22» | `scripts/lint-selftest.sh` đọc `N / M` do `ui-antipatterns.sh --selftest` tự đếm, đòi N = M (M ≥ 22): bắt đủ MỌI phép, không ghim số |
| Lighthouse (simulate) `/me`: CLS 0,18 (> 0,1), TBT 207 ms (> 200) | CLS: khối Điểm (có skeleton riêng) đổi chiều cao khi có dữ liệu và đẩy các khối dưới. TBT: mỗi truy vấn về là một lần dựng lại cả trang; `/me` cao hơn các trang khác (CI: 160–269 ms so với 70–173) | `RealMe.tsx`: chờ đủ Điểm + Chuyên cần + Điểm cộng rồi dựng MỘT lần (khung xương trước đó); các khối thấp hơn hiện theo lớp ở tác vụ riêng (`useStages`), khối Thời gian học chờ Bài tập. `MeGrade.tsx`: `useMyGrades` dùng chung cho hai nơi. **Không đổi ngân sách** (`lighthouserc.json` nguyên). Máy dev không tái hiện CLS ổn định (1/3 lượt) và TBT thấp hơn CI ~7 lần; đo lại bằng `cpuSlowdownMultiplier=28`: CLS 0 ở mọi lượt, `/me` ngang `/chat` (trước: cao hơn ≈ 40 %) — cần CI xác nhận |
| e2e `qc-uiux91-w3` TC-UX11 | spec đo thứ tự khi mới có tiêu đề «Chào…» (CI chậm: `[{Chào}, null, null, null]`) | chờ tiêu đề «Tiếp tục học» trước khi đo |
| e2e `shell.spec` nav theo SRS 7.5 | spec đọc thanh bên ngay sau `goto("/")` (CI: mảng rỗng); nhãn KHÔNG đổi so với SRS 7.5 | chờ đủ số liên kết thanh bên trước khi so (không cần đổi SRS / không có proposal) |
| `visual.spec` chat@1440, chat@390, grading-review@1440 | UI đổi có chủ đích ở đợt 4–5 (chat: panel căn giữa trong phần phải cột lịch sử, P-91-13; duyệt bài: thanh «Duyệt bài» ghim đáy, P-91-10) nhưng ảnh mốc cũ | thay ba ảnh mốc bằng ảnh THỰC của CI (linux/amd64, artifact `e2e-reports-2`, `*-actual.png`). Ba ảnh này khớp CI, có thể lệch nhẹ khi chạy trên macOS (khử răng cưa) |

Kiểm trên máy: `lint-selftest.sh` 7/7 + 23/23; `pnpm -C frontend build` (như production) xanh; `shell`, `qc-uiux91-w3`, `me`, `private-chat`, `panels` (desktop) 136 xanh.

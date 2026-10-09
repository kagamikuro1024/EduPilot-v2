# Sprint 5.5 — báo cáo (Giao diện panel nổi, D59)

Mục tiêu: đổi hướng thị giác theo yêu cầu chủ dự án (panel nổi kiểu Safari). Nền trang chuyển sang `--ep-canvas` xám ấm, mỗi vùng làm việc nằm trên một `Panel` trắng. Không đổi hành vi, chữ hay API.

**Kết quả: 7/7 story PASS (UI-01 PASS, còn lại có điều kiện) · cổng UI ĐẠT CÓ ĐIỀU KIỆN** sau 1 vòng sửa cổng. Nhánh `sprint/5.5-ui-panels`.

| Story | Nội dung | Kết quả |
| --- | --- | --- |
| US-UI-01 | 3 phương án × 2 vai × 3 bề rộng (18 ảnh) → chọn (a); D59; DESIGN / AGENTS / `ui-antipatterns.sh` 19 → 22 phép | PASS |
| US-UI-02 | Token `--ep-canvas`, elevation; primitive `Panel` / `PanelSection`; `/dev/ui#panel` | PASS có điều kiện |
| US-UI-03 | Khung: sidebar / topbar trắng, nav, bottom nav 375 px, AuthShell = canvas + một Panel | PASS có điều kiện |
| US-UI-04 | Màn sinh viên: Hôm nay, chat, Threads, bài thi | PASS có điều kiện |
| US-UI-05 | Màn Staff: Hôm nay, thành viên, bài thi, ngân hàng câu hỏi, kết quả, chấm, hồ sơ SV | PASS có điều kiện, sau sửa (B1) |
| US-UI-06 | Admin + `/settings/llm` (năm vùng, mỗi vùng một Panel, #1) | PASS có điều kiện |
| US-UI-07 | 78 ảnh trước / sau, axe, diện tích đỏ, Lighthouse trước / sau | PASS có điều kiện, sau sửa (B1) |

- Cổng: `qc/report-GATE-UI.md`.
- CI xanh ở `a06e153`.
- Spec: `docs/specs/FEAT-ui-panels/` v1.2 và `FEAT-ui-foundation` v1.7 (#3).
- Ảnh trước / sau: `handoff/ui-07/{before,after}/` (13 màn × 3 bề rộng).

## Số liệu
- **Test case:** 58 TC, 7 story. Lỗi thật: 2, đã sửa cả hai.
- **Hiệu năng** (cùng máy, trước / sau, trung vị):
  - TBT tăng tối đa 6 ms (ngưỡng 30 ms);
  - JS tăng ≤ 1,4 KB mỗi route (ngưỡng 2 KB);
  - CLS 0;
  - LCP lượt devtools trên CI: 1,5–2,0 s (ngưỡng 2,5 s).
- **Diện tích đỏ:** 0,02–0,42 % ở 13 màn (ngưỡng 8 %). Đỏ vẫn chỉ là tín hiệu.
- **axe:** 0 vi phạm ở 39 lượt thật (3 bề rộng, gồm bảng điểm ảo hoá). `color-contrast` vi phạm: 0.
- **Khối lượng:** 65 commit; `frontend` + `scripts` thay 116 file (+2.313 / −870).
- **Góp ý:** #1–#4, PM chấp nhận cả bốn.

## Lỗi thật tìm ra và đã sửa
- **B1 UI-05:** trang `/exams` khi rỗng / đang tải / chưa chọn lớp không nằm trong `Panel`. Test của dev không bắt được vì mock luôn có dữ liệu; QC thấy ở dữ liệu thật. Đã thêm ca e2e `exams empty`.
- **B1 UI-07:** axe báo 2 `critical` ở `/exams/[id]/results`. Lỗi có từ trước 5.5: `DataTable` ảo hoá đặt `aria-activedescendant` trên `role=region` và trỏ tới hàng chưa render. Đã sửa ở primitive dùng chung (`role=grid`, chỉ đặt khi dòng đang render), có ca a11y cho bảng dài hơn cửa sổ.
- **Ảnh mốc cũ sai (#4):** ảnh `inbox` / `gradebook` / `home` của sprint 3–5 chụp trang chặn quyền hoặc phiên hết hạn. Nguyên nhân: token demo sống 15 phút trong khi đồng hồ test bị đóng băng. Ảnh sinh lại ở UI-05 là chuẩn.

## Quyết định trong sprint
- **D59:** phương án (a), gồm:
  - canvas xám ấm, tối hơn panel 5 điểm L;
  - panel trắng, viền 1 px, bóng mềm, bo 14 px;
  - lề 12 px ở 375 px;
  - sidebar / topbar trắng;
  - chỉ light mode.

  PM chọn thay chủ dự án (vắng mặt) sau khi xem 18 ảnh; phương án này cũng là đề xuất mặc định của plan và của dev. Tương phản thấp nhất 4,57.
- Câu hỏi [CHỦ DỰ ÁN] Q1 / Q3 / Q4 / Q11 lấy mặc định của BA (`FEAT-ui-panels/QUESTIONS.md`). Đổi được qua `proposals.md`.
- **#1:** `/settings/llm` giữ năm vùng, mỗi vùng một Panel.
- **#2:** "Lớp cần chú ý" (Staff) và "Tiếp tục học" (sinh viên) chỉ hiện khi có dữ liệu. Phase sở hữu định nghĩa hình dữ liệu.
- **#3:** thêm `Panel` / `PanelSection` vào `FEAT-ui-foundation` 7.2.

## Nợ
- QC chưa chạy: `audit-login.mjs` bốn vai; `lhci` ở máy QC (thiếu `executablePath` cho Chrome). Đã dùng số CI thay thế.
- `color-contrast` `incomplete` của bảng ảo hoá: đã xem tay, axe không kết luận được.
- `Dialog` đang đóng vẫn nằm trong DOM của Panel. Không sai hiển thị; nên đưa ra portal (Q-QC-UI04-2).
- Chữ nút chính Admin (Q-QC-UI06-1, BA).
- Hình dữ liệu "Lớp cần chú ý" (P5) và "Tiếp tục học" (P3 / P9).

## Chủ dự án tự kiểm
- Xem `handoff/ui-07/before/` và `after/`, hoặc chạy `pnpm dev` rồi đi qua `/`, `/exams`, `/settings/llm` ở 1440 và 375 px.
- Không ưng phương án (a) hoặc các mặc định Q1 / Q3 / Q4 / Q11: ghi một dòng `proposals.md`. Token gom ở `DESIGN_TOKENS.css` nên đổi rẻ.

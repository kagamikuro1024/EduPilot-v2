# Bảng luồng × spec E2E — F1…F19 (QC, sprint 11, đợt 4)

Nguồn: `docs/FLOWS.md` (nhánh `sprint/11-p10-eval` @`4c919bf5`, có F18, F19) × `frontend/e2e/*.spec.ts` (105 tệp) × script QC `docs/sprints/*/qc/scripts`. Đếm bằng `grep` tên ca và từ khoá của từng nhánh lỗi trong FLOWS; chưa chạy lại spec nào (CI của dev lo phần đó).

## Cách đọc
- **UI-CI**: spec Playwright của dev, chạy trong CI, **gateway giả** (`page.route` / `e2e/support/api-server.mjs`). Phủ từng màn, không đi qua backend thật.
- **THẬT-CI**: spec chạy UI → gateway thật → DB trong CI. **Không có tệp nào.** Chín ca `@real` đều `test.skip(true, "… QC chạy tay")` (`account` 4, `class-join` 3, `today` 1, `settings-llm` 1); `exam.spec.ts` có một ca `@real` chỉ chạy khi đặt `E2E_REAL_API`.
- **QC-API**: đợt R (`docs/sprints/11/qc/scripts/w*.py`, API thật ep-qc) đi hết luồng nhưng ở tầng API, ngoài CI. **QC-UI**: `qc-*-real.spec.ts`, UI thật trên ep-qc, theo sprint, ngoài CI.
- Kết luận mỗi luồng:
  - **ĐỦ**: UI-CI phủ đường chính và các nhánh lỗi chính của FLOWS.
  - **MỘT PHẦN**: UI-CI phủ đường chính; còn nhánh/bước FLOWS nêu mà không có ca (cột «Thiếu»).
  - **THIẾU**: không có spec nào ở đúng luồng.
- Không luồng nào đạt «xong ở mức luồng» theo `AGENTS.md` nếu hiểu chặt (đi từ đầu đến cuối qua backend thật, có spec). Cột «THẬT-CI» đều trống. Báo PM để quyết định có chấp nhận UI-CI + QC-API thay thế cho sprint 11 không.

## Bảng

| Luồng | Spec UI-CI (số ca) | Phần phủ | Thiếu (nhánh/bước FLOWS có, spec không có) | QC-API / QC-UI | Kết luận |
| --- | --- | --- | --- | --- | --- |
| **F1** Tài khoản | `account` (43) | đăng nhập, throttle, đăng ký, xác minh (một lần, hết hạn), quên mật khẩu (câu chung), đặt lại, mời GV (admin + trang mời), đổi mật khẩu, thiết bị, `next` an toàn, refresh 2 tab; `safe-next` (2) | chuỗi đăng ký → Mailpit → xác minh → đăng nhập và «đặt lại thu hồi phiên máy khác» chỉ ở 4 ca `@real` đã skip; **mạo danh MSSV bị chặn** (FLOWS ghi đây là E2E): không có ca đăng ký bằng MSSV người khác; chỉ có ca duyệt hàng «email chưa khớp MSSV» ở `class-join`; khoá 10 lần + mail cảnh báo không có | QC-API `w1_accounts.py` | MỘT PHẦN |
| **F2** Mở lớp → vào lớp | `class-join` (66) | chọn lớp, trang `/join`, cài đặt lớp (mã, tạo lại, ngưỡng), thành viên (duyệt, mời ra + Hoàn tác, TA), import danh sách, nhân bản nguồn dùng chung, admin lớp; Go `TestCourseIsolation` | đổi GV giữa kỳ (GV cũ mất quyền, GV mới nhận ticket); vào lại sau khi bị mời ra nối đúng bản ghi cũ; lớp lưu trữ tắt mã (chỉ có ca bộ chọn lớp); 3 ca `@real` skip | QC-API `w1_accounts.py`, `w1b_classes.py` | MỘT PHẦN |
| **F3** Hỏi riêng tư | `private-chat` (25), `privacy` (3) | gửi, stream, hữu ích / không, «Nhờ giảng viên» (11 ca), xoá phiên (3), nhãn tạm thời, ẩn placeholder; 1 ca chat bị khoá | **tin khủng hoảng** (không có ca ở `/chat`; chỉ Threads có `threads-crisis`); liêm chính (hỏi trùng bài tập → chỉ gợi ý) không có ca UI; công thức điểm chưa xác nhận → nói chưa có thông tin | QC-API `w2_chat.py`, `w9c_rag.py` (RAG thật); đo P-11-18 ở `dot-r.md` | MỘT PHẦN |
| **F4** Hỏi bài công khai | `threads` (18), `threads-moderation` (11), `threads-crisis` (5), `thread-suggest` (5), `cascade` (4) | soạn + PII khi gõ, đăng, hộp hai lối, Xác nhận / Chỉnh sửa / Loại, kiểm duyệt (báo cáo, ẩn, sửa, xoá), khủng hoảng 422, gợi ý tương tự, cascade chặn | người đăng được báo khi có trả lời (lỗ hổng #15) và «tri thức nạp lại vector DB» không có ca; bài `Loại` biến mất khỏi RAG chỉ ở tầng API | QC-API `w2b_threads.py`, `w9c_rag.py` | ĐỦ (UI) |
| **F5** Escalation | `escalation` (27) | chuông, SSE, ticket mở, nhận, 409 hai người nhận (7 ca), trả lời → mail (21 ca), hỏi tiếp / đóng | nhắc lại sau 24 giờ làm việc, nổi lên «Hôm nay» sau 72 giờ, báo Admin khi lớp không ai hoạt động (0 ca; việc theo thời gian — cần ca worker, không phải UI) | QC-API `w2_chat.py` (ticket từ chat), `w10_p1118.py` | MỘT PHẦN |
| **F6** Tài liệu | `documents` (16), `library` (9), `qc-docs-real` (3) | kéo thả, tiến độ SSE, READY / FAILED kèm lý do (12 ca), OCR không đọc được, công tắc khoá | trùng nội dung → đề nghị chia sẻ (0 ca); xoá tài liệu → câu trả lời cũ hiện «nguồn đã gỡ» (0 ca); `COURSE_POLICY` → kích hoạt F10 không có ca; `answer_key_never_retrieved` là test Go, không thấy ở e2e | QC-API `w3_docs_att.py`, `w9c_rag.py` | MỘT PHẦN |
| **F7** Điểm danh, phát biểu | `attendance` (20), `activity` (19); 10.8: `qc-as-ui` (36, trên nhánh 10.8) | chạm vắng, phát biểu (6 ca), **ngắt mạng** (6 ca), khoá lạc quan (12 ca), lịch buổi, sửa sau buổi | sinh viên thấy sai → hỏi chat → F5 kèm ngữ cảnh buổi học không có ca | QC-API `w3_docs_att.py`; 10.8 `as_*.py`, k6 | ĐỦ (UI) |
| **F8** Theo dõi sinh viên | `students` (29), `insights-course` (2) | lọc, hồ sơ năm tab, ghi chú, nhắn qua ticket (5 ca) | câu nhận định rủi ro + lý do, nhận xét AI, `audit_log` mỗi lần xem hồ sơ: 0 ca; «tạo buổi ôn tập (F13)» không có ca; ràng buộc «SV không thấy nhãn rủi ro» chỉ ở tầng API/Go | QC-API `w3_docs_att.py` | MỘT PHẦN |
| **F9** Bài tập → công bố | `assignment-lifecycle` (48), `assignments-list` (4), `grading-guidance` (11), `question-hints` (5) | tạo, công bố, nộp, muộn / LATE (15 ca), chưa khớp (4), cờ AI, duyệt, công bố (31 ca), «Lưu ý chấm» | worker chết giữa chừng không chấm đôi (tầng Go); SV không bao giờ thấy điểm nháp chỉ kiểm ở API; cờ chèn lệnh decision (cờ tắt) có `decision-*` nhưng không có ca tô câu nghi | QC-API `w4_assign.py`, `w4b_review.py`; QC-UI `qc-gd-*` (10.6) | ĐỦ (UI) |
| **F10** Sổ điểm | `gradebook` (42) | công thức, xác nhận, nhập XLSX (17 ca), Chốt điểm (17), Mở khoá (14), 409 ô | xuất XLSX ra file có dòng «điểm chính thức là của hệ thống đào tạo» không có ca file; quy chế mới → bản nháp so sánh chưa thấy | QC-API `w5_grades.py`; QC-UI `qc-uiux91-w4*` | ĐỦ (UI) |
| **F11** Phúc khảo | `grade-appeal` (8), `assignment-lifecycle` (một phần) | gửi một lần (5 ca), giữ nguyên / sửa bắt buộc phản hồi (18 ca) | số liệu phúc khảo đổ vào quan sát (M13-A) không có ca; phúc khảo thi (F19.8) nằm trong `exam.spec` | QC-API `w4b_review.py` | ĐỦ (UI) |
| **F12** Luyện đề + QUIZ | `practice` (8), `questions-extract` (4), `quiz-integrity` (4), `question-hints` (5); QC-UI `qc-s10-*` (19) | ngân hàng, trích từ đề, duyệt, luyện, thi thử (4 ca), xáo, liêm chính | «Hôm nay» gợi ý ôn theo chủ đề yếu (1 ca nhắc chủ đề; không có ca gợi ý ở `today`); QUIZ tính điểm trong sổ điểm chưa có ca | QC-API `w5b_practice.py`; QC-UI sprint 10 | MỘT PHẦN |
| **F13** Lịch + nhắc việc | `calendar` (28) | tháng / tuần / danh sách, liên kết ICS (tạo, đặt lại) | **nhắc 24 giờ** (chuông + mail, không gửi trùng, tắt từng loại): 0 ca; đổi hạn → lịch, Hôm nay, câu trả lời AI đổi ngay: 0 ca | QC-API `w1_accounts.py`, `w1b_classes.py` | MỘT PHẦN |
| **F14** Hôm nay | `today` (26), `qc-uiux91-w3` (18) | nguồn việc theo vai (GV: điểm danh, ticket, duyệt; SV: nhập mã, bài thi) | SV: **phúc khảo có phản hồi**, **hạn nộp < 24 h** (0 ca ở `today`), điểm vừa công bố; Admin: provider lỗi / sắp chạm trần / dead-letter / sao lưu / lớp không GV (không có ca Admin) | QC-API `w1b_classes.py`; QC-UI `qc-uiux91-w3-real` | MỘT PHẦN |
| **F15** LLM, ngân sách, tích hợp | `settings-llm` (22), `settings-integrations` (8), `decision-settings` (13) | thêm provider, test, gán model, fallback (8), ngân sách (6), tích hợp, Quyết định nhanh | 80 % cảnh báo Admin / 100 % BATCH dừng, INTERACTIVE sang model rẻ nhất: chỉ ở tầng Go / API | QC-API `w7_admin.py`; QC-UI `qc-dm-ui-real` | ĐỦ (UI) |
| **F16** Quan sát AI | `observability` (10), `decision-observability` (8) | dải 24 giờ, bảng yêu cầu không cột nội dung, mở nội dung cần lý do ≥ 10 ký tự + audit, 410, TEACHER chỉ số tổng hợp | **`/admin/audit` và `/admin/health` không tồn tại trong `src/app`** (chỉ có `admin/courses`, `admin/users`) → F16 nêu hai trang này; ca hàng đợi / dead-letter: 0 | QC-API `w7_admin.py` (phân quyền) | MỘT PHẦN (hai trang thiếu — hỏi PM: thuộc phase PR?) |
| **F17** Báo cáo lỗ hổng | `insights-course` (2) | lớp thật không mượn báo cáo 761987; 761987 có báo cáo | chủ đề < 3 SV không hiện câu mẫu; thông báo quyền riêng tư; chỉ câu hỏi lớp đang chọn: 0 ca; `analytics` / `insights` chỉ có ca bố cục ở `visual` / `shell` | QC-UI `qc-ui-audit-real` (bố cục) | THIẾU (hai ca chỉ kiểm lớp mô phỏng) |
| **F18** Kết thúc học kỳ | — | — | không có spec; UI chỉ có «lưu trữ» trong `AdminCourses`; không có «Xuất toàn bộ dữ liệu lớp», «Nhân bản sang học kỳ mới», xoá theo chính sách (phase PR) | QC-API `w7_admin.py` (lưu trữ lớp) | THIẾU (phase PR) |
| **F19** Thi hằng tuần | `exam` (47), `exam-clock` (5), `exam-save-queue` (8), `quiz-integrity` | soạn câu + code, dựng bài, lên lịch (TA không có nút), làm bài (đồng hồ lệch 5 phút, hết giờ tự nộp, xác nhận số câu, **hai tab**, tab nhân đôi, mất mạng, 375 px, code ≥ 1024 px, xung đột nháp, nhãn AC/WA/CE/TLE, SSE rớt, sự kiện liêm chính), kết quả GV (CSV, phúc khảo), kết quả SV, so giống, 1.000 dòng ảo hoá | `Hoãn công bố` (0 ca); `Gia hạn` cả bài (có 1 ca ở `exam`); sandbox chết → «chưa chạy được, thử lại sau N giây» và `IE` không thành 0 điểm; sửa test → `Chấm lại` → điểm đổi + báo SV; cả lớp chưa đóng → tự công bố: các ca này ở tầng Go / API | QC-API `w6_exam.py`, `w8_codeexam.py`; `check-exam-seed.mjs demo` (`@real`) | ĐỦ (UI) — nhánh sandbox/chấm lại chỉ ở API |

## Tổng hợp

| Kết luận | Số luồng | Luồng |
| --- | --- | --- |
| ĐỦ (UI-CI) | 7 | F4, F7, F9, F10, F11, F15, F19 |
| MỘT PHẦN | 10 | F1, F2, F3, F5, F6, F8, F12, F13, F14, F16 |
| THIẾU | 2 | F17, F18 |
| THẬT-CI (UI + gateway thật trong CI) | **0 / 19** | — |

## Luồng thiếu spec (đánh dấu)
1. **F17**: chỉ 2 ca kiểm «không mượn báo cáo lớp mô phỏng». Không có ca nào cho ba ràng buộc riêng tư của FLOWS (≥ 3 SV, ẩn danh, chỉ lớp đang chọn).
2. **F18**: không có spec; một phần UI chưa tồn tại (xuất ZIP, nhân bản học kỳ, xoá theo chính sách).
3. **F16**: `/admin/audit` đã có (US-P10-14; kiểm ở tc-US-P10-14, AUD-1..6); `/admin/health` thuộc phase PR (P-11-30), ngoài phạm vi sprint 11 — không tính FAIL (loại c).
4. **Nhánh theo thời gian** (F5 24/72 giờ, F13 nhắc 24 giờ, F14 nguồn việc Admin) không có ca ở UI-CI; cần ca worker hoặc đồng hồ giả.
5. **Mạo danh MSSV** (F1, «quy tắc an toàn số 1»): không có ca đăng ký bằng MSSV người khác; UI chỉ có ca duyệt hàng lệch ở `class-join`; phía API do đợt R kiểm.
6. **Tin khủng hoảng ở chat riêng** (F3): ca `/chat` = 0; chỉ Threads có.

## Đề xuất (chờ PM)
- Chấp nhận đợt R (QC-API) + UI-CI làm bằng chứng «xong ở mức luồng» cho sprint 11, **hoặc** giao dev một spec `@real` mỗi luồng ưu tiên (đề xuất F1 đăng ký → Mailpit → đăng nhập; F3→F5 chat → ticket → trả lời → mail; F9 nộp → chấm → duyệt → công bố → phúc khảo).
- Quyết định F16 (`/admin/audit`, `/admin/health`) và F18 thuộc sprint nào.
- QC không tự viết spec cho dev (không sửa vùng của vai khác); nếu PM muốn QC chạy `@real` thật cho F1/F3/F9 trên ep-qc, báo để QC dựng spec `qc-flow-*.spec.ts`.

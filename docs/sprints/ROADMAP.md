# Lộ trình 11 sprint

Chủ dự án chốt 2026-10-01: toàn dự án = **10 sprint**; 2026-10-03 chèn **sprint 5 — PE Thi hằng tuần** (yêu cầu của thầy hướng dẫn, D54), các sprint sau lùi 1 → **11 sprint**. Sprint 1 đã xong và là `main` của repo mới `kagamikuro1024/TA_Agent_v2`. Mỗi sprint rộng hơn trước (1–2 phase), nhưng giữ nguyên thứ tự phụ thuộc của phase và cổng nghiệm thu của từng phase.

Phạm vi: P0 → P10 (vạch bảo vệ). **PR nằm ngoài đồ án** (D44). Không cắt tính năng nào khi lập lộ trình; nếu một sprint trễ thì cắt theo "Thứ tự cắt khi trễ" ở `WORKFLOW.md` §6, cột "Cắt được nếu trễ" bên dưới chỉ ra mục nào thuộc sprint đó. Danh sách "KHÔNG BAO GIỜ cắt" vẫn giữ.

| Sprint | Phase | Ước lượng gốc (tuần người) | Luồng đi trọn khi xong | Cổng | Cắt được nếu trễ (WORKFLOW §6) |
| --- | --- | --- | --- | --- | --- |
| 1 ✅ | P0 Chuẩn bị | 0,5 | – | `gate P0` PASS | – |
| 1.5 | Prototype giao diện (mock, D51) — chen giữa theo yêu cầu chủ dự án cho buổi thuyết trình với thầy | – | Đi trọn `DEMO_SCRIPT.md` trên mock | QC đi trọn kịch bản + `ui-antipatterns` | – (`docs/sprints/1.5/`) |
| 2 ✅ | PG Nền Go | 3,5 | – (nền: contract test, SSE, outbox, blob, `--scale gateway=2`) | `gate PG` PASS | – (toàn bộ là "cái nền") |
| 3 | PU Nền giao diện + P1 LLM Gateway | 1,5 + 1,5 | Đổi provider trên UI; Scheduler + trần ngân sách | `gate PU`, `gate P1` | #0 chuyển động trang trí, bố cục ≥ 1440 px |
| 4 | P2 Lớp học | 2,5 | F1, F2 | `gate P2` | #4a tuỳ chọn nâng cao mã tham gia; #6 CommandPalette |
| 5 | **PE Thi hằng tuần: trắc nghiệm + C/C++** (D54–D57) | 2,5 | F19 (giao bài thi → làm → chấm → công bố) | `gate PE` | so độ giống mã (giữ log rời tab); gợi ý nháp đề bằng AI |
| 5.5 | **UI panel nổi** — chen giữa theo góp ý chủ dự án 2026-10-08 (đổi hướng thị giác, cần D59) | ≈ 1 | Mọi vùng làm việc nằm trên panel; chủ dự án duyệt ảnh trước / sau | cổng UI (`docs/sprints/5.5/plan.md`) | chế độ tối |
| 6 | P3 Hai kênh + PII · P8 Tài liệu + Lịch | 2 + 1 | F3, F6, F13 | `gate P3`, `gate P8` | (#4 NER đã cắt bởi D46) |
| 7 | P4 Escalation + Mail · P5 CRM + 360 | 1,5 + 1,5 | F4, F5, F7, F8 | `gate P4`, `gate P5` | #5 hàng đợi ghi cục bộ điểm danh; #6 phím tắt `/inbox` |
| 8 | P6 Sổ điểm (gồm nối điểm bài thi PE vào `grade_items`) | 1,5 | F10 | `gate P6` (+ chủ dự án tự tính tay 3 SV) | – |
| 9 | P7 Chấm bài | 3 | F9, F11 | `gate P7` | #1 kịch bản lỗi nâng cao mock-graph; #4b mock-graph |
| 10 | P9 Luyện đề (dùng lại ngân hàng câu hỏi + Quiz Engine của PE) | 1 | F12 | `gate P9` | #2 `GenerateQuestions` + E5; #3 import Forms |
| 11 | P10 Đánh giá + hoàn thiện (gồm **thống kê điểm thi PE** trên màn quan sát lớp — chủ dự án yêu cầu 2026-10-09) | 2,5 | F14–F17, F19; chạy lại `DEMO_SCRIPT.md` trọn 15 phút | `gate P10` → **vạch bảo vệ** | #7 xuất PDF; #8 k6 (d), chaos Redis |

Tổng ước lượng gốc: 23,5 tuần người. Đội agent làm nhanh hơn nhiều (P0: 0,5 tuần → 1 ngày); lịch thật đo theo sprint, không theo tuần.

## Vì sao ghép như vậy
- **S3 PU + P1:** P1 có màn cấu hình provider → cần primitive của PU; cả hai chỉ phụ thuộc PG.
- **S5 P3 + P8:** RAG của chat (P3) cần tài liệu đã trích + nhúng; P8 có ingest nền với `docling-serve`. Ghép lại thì P3 không phải làm "đường nạp tối thiểu" tạm (nguyên tắc sở hữu bảng, ARCHITECTURE §4).
- **S6 P4 + P5:** độc lập nhau (P4 cần P3, P5 cần P2), cùng nhỏ.
- **P6, P7 tách riêng:** đụng điểm số và chấm AI — luôn cần chủ dự án kiểm tay; giữ sprint nhỏ để báo cáo rõ.
- **P9 riêng:** cần cả P7 (Grading Engine) và P8 (tài liệu).

## Luật sprint (thay luật "1–3 story" cũ)
- Một sprint = các phase ở bảng trên. Story vẫn nhỏ (`dev` ≤ 1 ngày), `dev` làm **từng story một**; QC viết TC song song (PM.md §3.3).
- Sprint kết thúc bằng `/gate` của **mỗi** phase trong sprint, rồi `report.md`, rồi DỪNG chờ chủ dự án chốt.
- Nhánh `sprint/N-<slug>` tạo từ `main` của `origin` (TA_Agent_v2). Chủ dự án chốt báo cáo → PM merge `--no-ff` vào `main` và push.
- **Thay mock bằng thật (D51):** sprint nào làm phase nào thì màn prototype của phase đó được thay bằng màn nối API thật, spec thật viết ở `docs/specs/`; dữ liệu `frontend/src/mock/` của màn đó bị xoá trong cùng sprint. Cuối sprint 10 không còn `src/mock/`.

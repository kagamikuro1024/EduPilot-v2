# Prompt cho `qc` — Viết test case theo spec v5 (song song lúc dev thi công)

Cảm ơn đợt thăm dò: PM chấp nhận cả 37 lỗi thành góp ý #19; BA đang đưa vào spec v5.1. Spec v5 (#17 UI polish, #18 Threads như thật) đã APPROVED.

## Việc
1. Đọc `docs/sprints/1.5/spec/US.md` + `SRS.md` v5: 00-AC7…AC10, 01-AC4, 01-AC10…AC14, 02-AC9…AC11, 04-AC7, `SRS.md` 4.3.1 (Threads A–F) và 4.7 (FR-X12, đoạn `AUDIT`).
2. Bổ sung TC hộp đen vào `docs/sprints/1.5/qc/tc-US-PROTO-0{0..4}.md`: mỗi AC mới ít nhất một TC chính + một TC nhánh lỗi/biên; Threads phải có TC theo đúng từng bước thời gian (đang soạn → stream → Dừng → nguồn → Chờ xác nhận; phản hồi trễ ~6 s sống qua tải lại; "Hôm nay" 6 → 7).
3. Viết script tự động trong `docs/sprints/1.5/qc/scripts/`: chạy `AUDIT` trên mọi route × vai × bề rộng (1440 / 390, thêm 375 cho route SV, `/inbox`, `/attendance`), xuất bảng PASS/FAIL. Có thể dùng Playwright nếu đã có trong `frontend`; không thêm thư viện mới.
4. Không chạy trên code dev lúc này (dev đang làm). Không đọc code dev.

Commit chỉ `docs/sprints/1.5/qc/**`. Trả lời số TC mới theo story rồi dừng. Khi có spec v5.1, PM sẽ giao bổ sung TC hồi quy cho E1–E37.

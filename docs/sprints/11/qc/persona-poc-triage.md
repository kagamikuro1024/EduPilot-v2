# QC sprint 11 (P-11-12) — lọc 8 ứng viên lỗi từ PoC persona-eval

Nguồn: `research/persona-eval@da0f9e3e`, `docs/research/2026-11-14-persona-eval.md` (§3.1). Tái hiện **chỉ đọc** trên `ep-qc`, lớp **761987** (không đụng KC106), bản `c183c3a4` (lúc Tech Lead đo) và `86defde6` (sau 10.6). Spec: `frontend/e2e/qc-persona-triage-real.spec.ts` (Playwright, `QC_REAL=1`), ảnh `persona-shots/`, số đo khung trang ở `docs/sprints/10.7/qc/audit.md`. Mức: Cao / TB / Thấp theo ảnh hưởng người dùng (không phải P0–P3 của công cụ).

| # | Ứng viên | Màn · vai | Tái hiện? | Mức | Thuộc về |
| --- | --- | --- | --- | --- | --- |
| 1 | «Hàng chờ» ghi «Chưa có bài nộp.» dù có bài (tab Bài tập: «Đã nộp 16 / 30») | `/grading` · GV | **Có** ở `c183c3a4` (lọc mặc định «Chưa duyệt» + còn 8 bài chưa chấm → «Chưa có bài nộp.»). **Hết** ở `86defde6`: tự chuyển «Chưa chấm» (8 bài), «Chấm tất cả» bấm được khi đã chọn bài tập (QC 10.6, S2-1) | Cao | **10.6 — đã sửa** (S2-1; QC kiểm lại PASS) |
| 2 | Lọc «Bài tập» ở hàng chờ về mặc định khi đổi tab | `/grading` · GV | **Có** ở `c183c3a4` (chọn BT03 → tab «Bài tập» → quay lại: về «Tất cả bài tập»). Ở `86defde6`: **giữ** khi đổi tab; **vẫn mất khi rời route** (sang `/` rồi quay lại) — «Cần xem kỹ» và trạng thái thì giữ | TB (còn Thấp) | **10.6 — đã sửa phần đổi tab**; phần «rời route» → **10.7** (một nơi giữ bộ lọc theo hợp đồng khung) |
| 3 | Đề bài hiện Markdown thô | `/assignments/<id>` · SV (375) | **Có**: trang hiện nguyên `# Bài tập 03 — …`, `## Đề bài`, `## Yêu cầu nộp` | TB | **Sprint 11** (hiển thị nội dung; ngoài khung 10.7) |
| 4 | Mốc giờ mâu thuẫn trên trang bài tập | `/assignments/<id>` · SV | **Có một phần**: đầu trang «đã quá hạn 4 ngày», cuối «Đang chấm · Nộp muộn 1 ngày» (DB: nộp trễ **2 giây** → làm tròn lên 1 ngày), dòng quy định «Hạn nộp: 23:59 ngày hạn» trong khi mốc hạn hiển thị là 18:46 | TB | **Sprint 11** — BA chốt quy tắc hạn (23:59 hay giờ đặt) và cách hiển thị «muộn N ngày» |
| 5 | SV không có mục «Bài tập» trên thanh điều hướng | mọi route · SV | **Có**: thanh dưới 375 = Hôm nay, Chat riêng, Hỏi đáp, Luyện đề, Thêm; «Thêm» = Bài thi, Thư viện, Lịch, Kết quả của tôi — không «Bài tập». **Thiết kế có chủ ý** (`nav.ts`, `shell.spec.ts` khoá) | TB | **PM / BA quyết** (đổi thiết kế → sprint 11; không phải lỗi mã) |
| 6 | Nút ở «Việc nên làm tiếp» lặp nguyên tiêu đề thay vì động từ | `/` · SV | **Có**: thẻ «Điểm bài thi Kiểm tra tuần 9 — Mật mã đã công bố» + dòng «Xem kết quả và đáp án.»; nút/liên kết cùng chữ với tiêu đề, tiêu đề xuất hiện hai lần | Thấp | **Sprint 11** (lời văn `DESIGN.md` §13); nếu 10.7 đổi mẫu thẻ việc thì gộp |
| 7 | Hai nút «Tải bài lên» khi hàng chờ rỗng | `/grading` · GV | **Có** ở cả hai bản: `c183c3a4` 2 nút; `86defde6` + lớp rỗng thật: «Tải bài lên ×2» (thanh công cụ + khối rỗng) | Thấp | **10.7** (K4f / K5 trong `audit.md`: nút rỗng trùng nút đầu trang; 12 route) |
| 8 | «Đã công bố» vừa là trạng thái bài tập vừa là cột điểm | `/grading` tab Bài tập · GV | **Có**: cột «Trạng thái» = «Đã công bố» cho cả BT03 và BT04 (BT04 «0 / 30», chưa có bài), kề cột «Đã chấm / Đã công bố 8 / 0» | Thấp–TB | **Sprint 11** (lời văn trạng thái bài tập; BA) |

## Bốn ứng viên «cần người kiểm» của báo cáo
| Ứng viên | Kết quả |
| --- | --- |
| «Hôm nay» ghi 9 việc nhưng danh sách có 7 | **Không tái hiện**: API `today` (`count` = `actions`) khớp tiêu đề; 9 việc + 1 chú ý + 1 sắp tới = 11 dòng; mã không cắt danh sách |
| «Kết quả của tôi» hiện chuỗi công thức thô | **Có** hiển thị: `min(0,5; 1)`, `max(0; 2 − 2) × 0,5 + 0`, `(20 × 7,5 + 20 × 8) ÷ 40` dưới «Căn cứ». Có vẻ là minh bạch cách tính có chủ ý → **PM / BA** xác nhận SV có nên thấy biểu thức; nếu không → sprint 11 |
| «Chấm tất cả» bị khoá mà không có lý do | **Một phần**: lý do «Chọn một bài tập trước.» có trong DOM (`aria-describedby`) nhưng không thấy bằng mắt trên ảnh; chưa kiểm hover → **10.7** (khoá phải kèm lý do thấy được trên chính nút) |
| Vẫn cho «Nộp lại» khi quá hạn tối đa | **Chưa kiểm** (cần nộp tệp — ngoài phạm vi chỉ đọc); đề xuất TC riêng ở sprint 11 |

**Ảo / do cách chạy:** 404 khi mở `/today` (URL bắt đầu sai) — loại. «Không có liên kết tới tệp đã nộp» — yêu cầu chưa rõ, chuyển BA.

## Tóm tắt
- Tái hiện được **8 / 8** (6 đủ, #4 một phần, #5 là thiết kế). Đã hết nhờ 10.6: #1, #2 (phần đổi tab). Thuộc **10.7**: #7, #2 (phần rời route), «Chấm tất cả» khoá không lý do. Thuộc **sprint 11**: #3, #4, #6, #8. **PM / BA quyết**: #5, công thức thô ở «Kết quả của tôi».
- Nhận xét phương pháp: các lỗi #1, #2, #7 đều lọt qua E2E vì không spec nào mở màn ở trạng thái «danh sách sau lọc rỗng» hoặc «lớp rỗng»; sweep 10.7 (`audit.md`) bắt #7 và 11 route khác cùng kiểu.

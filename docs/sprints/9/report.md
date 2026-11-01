# Sprint 9: báo cáo (P7 Bài tập: nộp, chấm AI nháp, duyệt, công bố, phúc khảo)

**Nhánh:** `sprint/9-p7` tách từ `sprint/8-p6` (73 commit sau sprint 8; `backend-go` + `frontend` + `scripts` + `benchmarks`: 270 tệp, +41 714 / −1 286). **Thời gian:** 2026-10-31 → 2026-11-01. **Cổng P7:** **kỹ thuật PASS** (`qc/gate-P7.md`, vòng 3, CI xanh). Còn G7-23: LCP màn nộp bài chưa đạt, chờ chủ dự án (#10).

## Story

| Story | Nội dung | QC |
| --- | --- | --- |
| US-P7-01 | Migration `00017_assignments` (bài tập, bài nộp theo phiên bản, kết quả chấm, thư vào, phúc khảo, tích hợp; FK phức hợp theo `course_id`) | PASS |
| US-P7-02 | `/grading` tab Bài tập: tạo / sửa, rubric, gợi ý rubric bằng AI, công bố bài → lịch + "hạn nộp" trên Hôm nay | PASS |
| US-P7-03 | `/assignments/[id]`: kiểm tệp trước khi tải, URL ký sẵn, phiên bản, nộp muộn / chặn, 375 px | **FAIL chỉ AC9 (LCP 4,7 s)**, chờ chủ dự án |
| US-P7-04 | `UploadSource` ZIP, bộ khớp (email đã xác minh / MSSV danh sách lớp), tab Chưa khớp + gán tay, lịch sử phiên bản | PASS, vòng 2 |
| US-P7-05 | `ImapSource` (go-imap v2): poll, xác thực người gửi `dmarc=pass`, thư không xử lý được; `/settings/integrations` thật | PASS, vòng 2 |
| US-P7-06 | Grading Engine: chấm mù, hai lượt, kiểm bằng mã, trích dẫn có thật, injection chỉ gắn cờ, trừ muộn | PASS |
| US-P7-07 | Hàng đợi chấm (Streams, idempotent, retry 3, dead-letter, chạy lại), làn `BATCH`, tiến độ SSE `{done, failed, total}` | PASS, vòng 3 |
| US-P7-08 | Hàng chờ + `SubmissionReview` (hai cột đồng bộ, bằng chứng tô sáng, sửa tại chỗ, `A` duyệt và sang bài kế) | PASS |
| US-P7-09 | Công bố hàng loạt (TEACHER) → sổ điểm + thư + chuông; sinh viên xem kết quả; `TestStudentNeverSeesDraftGrade` | PASS, vòng 2 |
| US-P7-10 | Phúc khảo F11 (một lần, trong hạn, chỉ TEACHER quyết, sổ đã chốt → mở khoá có lý do) | PASS, vòng 2 |
| US-P7-11 | `TeamsGraphSource` + `cmd/mock-graph` (7 kịch bản), thẻ Teams "Môi trường giả lập", "Đồng bộ từ Teams" | PASS, vòng 2 |
| US-S9-SEED | BT03 (26 bài nộp, 3 chưa khớp, một bài có injection), BT04 (Teams), `cmd/mock-imap`, quy chế lớp 1 bản 2 (nháp), `gate-p7.sh` | PASS |

## Số liệu
| Đo | Kết quả |
| --- | --- |
| Chat khi đang chấm hàng loạt (k6) | Sự kiện đầu tiên p95 28 ms; TTFT +0,05 %; 0 % lỗi |
| Test | +425 hàm `Test…` Go, 0 xoá; `go-test-all.sh` 57 gói xanh |
| Phân quyền | `p7-rbac.sh` 35 route PASS |
| OpenAPI | 246 → 288 thao tác |
| Đáp án v2 | Quy chế lớp 1 bản 2 (nháp): 30/30 khớp `expected_final_grades_v2.csv`; bản 1 không đổi (băm giữ nguyên) |
| LCP `/assignments/[id]` | 6,1 s → 4,7 s sau tối ưu; ngưỡng 2,5 s **chưa đạt** |

## Lỗi thật tìm ra và đã sửa
- **Tech Lead, trước khi code:**
  - **Header `From:` của thư giả được,** nên sinh viên có thể nộp thay người khác. Sửa: chỉ tin thư khi máy chủ thư của trường xác thực `dmarc=pass` (TLR-2).
  - **`student_code_snapshot` của sinh viên vào lớp bằng mã chép từ MSSV tự khai.** Lỗi này ảnh hưởng cả nhập XLSX của sprint 8 (#20 sprint 8) (TLR-1).
  - **Vé phúc khảo lẫn vào hộp thư hỗ trợ,** SLA và trần vé (TLR-3). Sửa theo phương án A: phúc khảo không tạo ticket.
  - **Mẫu injection xoá mất câu viết thật** của bài làm (TLR-4).
  - **Kịch bản Teams `slow` không thể đạt** vì 52 yêu cầu tuần tự. Sửa: lấy danh sách thành viên bằng một lệnh, tải tệp song song (TLR-5).
  - **XSS lưu trữ qua `Content-Type` do client khai** (TLR-15).
  - **Gia hạn nộp mà sinh viên vẫn bị trừ muộn** (TLR-8).
  - **Bài chưa khớp bị tính 0** (TLR-9).
- **PM:** nhãn "Bản mô phỏng · dữ liệu giả" hiện trên cả bản chạy thật. Đây là phần sót từ prototype (BUG-S9-01, #12).
- **QC:**
  - test đua dead-letter (BUG-P7-07-5);
  - thiếu route lịch sử phiên bản (BUG-P7-03-4);
  - thiếu test token Graph không bị gửi sang host phụ (BUG-P7-11-1);
  - k6 hai VU dùng chung phiên làm sai số đo (BUG-P7-07-4).

## Quyết định trong sprint
**PM chốt thay chủ dự án** (vắng 31/10 → 01/11), theo mặc định an toàn:
- Q1–Q20 của FEAT-assignments.
- TLR-1…20 với hai lựa chọn phương án: TLR-3 chọn A, TLR-14 chọn A.
- Proposals #1–#16.

**Các mục [ĐIỂM], chủ dự án xem lại.** Không mục nào đổi điểm đã có của lớp 1.
- **Q2:** trừ muộn 0,5 điểm/ngày, cấu hình mỗi bài.
- **Q3:** công thức lớp 1 giữ nguyên; bản 2 có thành phần bài tập chỉ là bản nháp để demo.
- **Q4:** điểm nháp lấy từ lượt 1; bước điểm tiêu chí 0,25.
- **Q9:** không nộp bài thì áp `MISSING_SCORE`.
- **TLR-4:** mẫu injection chỉ gắn cờ.
- **TLR-8:** số ngày muộn tính theo hạn hiện tại.
- **TLR-9:** bài chưa khớp thì sinh viên `INCOMPLETE`, không 0.
- **TLR-14:** item bài tập đọc thẳng kết quả đã công bố.
- **TLR-16:** điểm tối đa là bội của 0,25.

## Quy trình
- **Hạn mức API 5 giờ hết** lúc 12:09 ngày 01/11. Dev và QC dừng khoảng 1 giờ 30, rồi chạy tiếp từ chỗ dở, không mất việc.
- **Luật "cùng dạng thì làm trước"** (từ sprint 8) giảm số lần chờ. Dev không còn sửa test khác dạng trước khi PM quyết.
- **Chuyển nơi kiểm AC route** sang story có route, áp chung cho cả sprint (#8).

## Nợ
- **LCP màn nộp bài 4,7 s** (G7-23, US-P7-03 AC9). Chủ dự án chọn: (1) đổi thiết kế màn đầu / `shared/shell/intro` để đạt, hoặc (2) chấp nhận chưa đạt.
- **Tải 200 bài chấm cùng lúc** đo ở P10 (`seed-t1`).
- **Chưa kiểm trên stack:** `kill -9` worker giữa hai lượt, Redis dừng 70 s, Teams hết thời gian thật.
- **`grading.batch` mồ côi** khi Redis mất ánh xạ lô (BUG-P7-07-3). Các bài vẫn được chấm.
- **Tệp compose production** phải đặt `TEAMS_ENABLED=false` (PR, #14).
- **Bước CI kiểm danh sách thư viện cho phép** (gợi ý TL-3).
- **Không tự chấm khi sinh viên nộp** (Q16); không có `FormsImportSource` (Q8).

## Chủ dự án tự kiểm
1. **Chọn hướng xử lý LCP** (#10) và **duyệt thư viện `golang.org/x/text`** (NFC bài làm; TL-3).
2. **Xem lại các mục [ĐIỂM]** ở trên.
3. **"Bạn tự kiểm" của `phases/P7.md`:**
   - gửi một mail đúng mẫu và một mail sai mẫu vào hộp thử (seed dùng `mock-imap`; hộp thư thật cần app password);
   - đọc 5 bài AI chấm (BT03);
   - kiểm sinh viên không thấy điểm trước khi công bố;
   - chạy "Đồng bộ từ Teams" với kịch bản `happy` / `throttle` / `no_consent`.
4. **Bắt đầu chấm tay 60 bài cho E3.**
5. **Thứ tự merge:** nói "chốt" sprint 8, rồi sprint 9. Merge `sprint/8-p6` trước, rồi `sprint/9-p7`.

# Sprint 8: báo cáo (P6 Sổ điểm và điểm cuối kỳ từ quy chế môn học)

**Nhánh:** `sprint/8-p6` (88 commit sau `main`; `backend-go` + `frontend` + `scripts`: 162 tệp, +29 894 / −7 070). **Thời gian:** 2026-10-31. **Cổng P6:** **kỹ thuật PASS** (`qc/gate-P6.md`, CI `41213938391` xanh); chờ chủ dự án tự tính tay A, B, C (G6-13).

## Story

| Story | Nội dung | QC |
| --- | --- | --- |
| US-S8-UI | Nợ sprint 7: bỏ câu mô tả dưới tiêu đề, cảnh báo lặp, trạng thái dài ở `/settings/llm`, `/settings/integrations`, khối bảo mật `/settings` | PASS, vòng 3 |
| US-P6-01 | Migration `00016_gradebook` (công thức, thành phần, luật, item, điểm, snapshot bất biến, `gradebooks`), nguồn điểm PE + phát biểu P5 | PASS (AC route kiểm ở P6-06 / 09, #8) |
| US-P6-02 | `internal/grade`: decimal, không `float64`, không đọc thời gian học / LLM; luật đọc từ `grade_rules.params`; giải trình từng dòng; 30/30 + 18 ca biên | PASS |
| US-P6-03 | Trích công thức từ quy chế: việc nền 202 + SSE, `llm.Structured` strict, bộ kiểm bằng code, trích dẫn có trang, mục "chưa rõ" | PASS, vòng 2 |
| US-P6-04 | `/gradebook/scheme` thật: duyệt như văn bản, bấm mới sửa, mục chưa rõ chặn xác nhận, xem thử 3 sinh viên, hậu quả bằng số trước khi xác nhận; liên kết `Mở trang N` | PASS, vòng 2 |
| US-P6-05 | Nhắc khi chưa có công thức: banner, `Chốt điểm` khoá kèm lý do, job tuần, `GRADE_SCHEME_UNCONFIRMED` trên Hôm nay, AI nói "chưa có công thức chính thức" | PASS, vòng 2 (#18) |
| US-P6-06 | `/gradebook` thật: bảng kiểu bảng tính trong `DataTableVirtual` (chế độ ô), 1.000 dòng, xem trước tức thì, 409 hiện giá trị người kia | PASS, vòng 2 |
| US-P6-07 | Nhập XLSX (ánh xạ cột → xem trước → dòng lỗi → ghi idempotent; MSSV chỉ theo danh sách lớp, #20; chống zip bomb) + xuất XLSX có dòng ghi chú điểm chính thức | PASS |
| US-P6-08 | `Chốt điểm` (snapshot + khoá + audit), `Mở khoá có lý do` → phiên bản mới, sinh viên bị đổi điểm được báo; đua nhập ↔ chốt | PASS |
| US-P6-09 | `/me` phần điểm: nhận định, giải trình tuyến tính, tính thử tại chỗ (không lưu); tool `get_my_grade_summary`, `what_if_final_grade` (danh tính từ JWT); tín hiệu `GRADE` cho at-risk | PASS (lỗ hổng test BUG-P6-09-1 đã vá) |
| US-S8-SEED | Lớp 1 có quy chế mô phỏng + công thức đã xác nhận + điểm (PE, phát biểu, XLSX); lớp 2 cố ý chưa có quy chế; `DEMO_MODE` (`LLM_DEMO_FILE`); `gate-p6.sh` | PASS, vòng 2 |

## Số liệu
| Đo | Kết quả |
| --- | --- |
| Đúng tuyệt đối | 30/30 dòng khớp `seed/expected_final_grades.csv`. Đáp án do BA tính tay; Tech Lead tính lại độc lập bằng `fractions.Fraction`; QC tự tính A, B, C |
| `impact` (xác nhận công thức) p95 | 7–8 ms (yêu cầu ≤ 300 ms) |
| Seed | Chạy trên DB trống rc = 0; chạy lần 2 không đổi; `SEED_RNG=1` chạy trọn |
| At-risk sau P6 | `AT_RISK` 1 (C); `WATCH` 3 (sv12, sv19, sv26), ghim lại theo #5 |
| Playwright cổng P6 | 54 passed; `go-test-all.sh` 51 gói xanh |
| TBT `/settings/llm` | 217 → ≤ 200 ms (#19, tối ưu mã, không nới ngưỡng) |

## Lỗi thật tìm ra và đã sửa
- **Tech Lead, trước khi code:**
  - **Làm tròn sai ở biên.** Khi có phép chia không hết, 7,25 bị tính thành 7,2499… rồi làm tròn xuống (TLR-1).
  - **Bài thi chưa công bố bị tính 0 cho cả lớp** (TLR-2).
  - **Nhập XLSX nối điểm theo MSSV tự khai** (TLR-3, #20).
  - **Schema strict làm LLM thật không trả được tham số luật** (TLR-4).
  - **Đáp án in số làm tròn kiểu chẵn-gần-nhất,** làm sai 4 dòng (TLR-5).
- **Tech Lead, trong sprint:** "Hôm nay" bị N+1, mỗi lớp có bản nháp đọc lại toàn văn quy chế. Sửa bằng cột `unclear_count` ghi lúc lưu (TL-1, #16).
- **Dev: test bom hẹn giờ BUG-S8-01.** Đồng hồ trang cố định, nhưng JWT giả ký bằng giờ thật, nên từ 29/10 test tự đỏ; `main` cũng sẽ đỏ ở lần CI kế. Sửa: ký JWT theo đồng hồ trang, thêm ca phủ nhánh SSE (#17).
- **QC:**
  - Truy vấn trích công thức đọc `content_chunks` mà không lọc lớp, và không loại tài liệu tri thức (BUG-P6-03-1, 2).
  - `/gradebook/scheme` là tường 22 ô nhập (BUG-P6-04-1).
  - Lỗ hổng test lộ điểm quá trình nháp (BUG-P6-09-1).
  - Seed kẹt vĩnh viễn sau một lần lỗi 4xx vì `Idempotency-Key` cố định (BUG-S8-SEED-1).

## Quyết định trong sprint
**Chủ dự án chốt 10 điểm tính điểm:**
- **Q1:** quy chế lớp 1 dùng bản mô phỏng, vì `Quyche.pdf` chỉ có quy định học vụ, không có công thức môn học.
- **Q2:** thiếu điểm thì `INCOMPLETE`, không bịa số 0.
- **Q3:** làm tròn đến 0,5, nửa lên, một lần ở cuối; điểm chữ tính trên điểm đã làm tròn.
- **Q4:** điểm quá trình kẹp trong [0, 10].
- **Q5:** liệt là F, vẫn giữ số đã tính.
- **Q6:** điểm cộng lấy từ phát biểu / thưởng ở P5.
- **TLR-1:** chuẩn hoá độ chính xác máy trước bước làm tròn.
- **TLR-2:** chỉ tính 0 khi nguồn điểm đã chốt.
- **TLR-7:** từ chối điểm có hơn 2 chữ số thập phân.
- **TLR-17:** quy chế không có mục trừ điểm thì không trừ.

**PM chốt** (chủ dự án giao quyền):
- **Quyền:** TA chỉ đọc; ADMIN không xem điểm cá nhân; chỉ TEACHER xác nhận / chốt / xuất.
- **Sinh viên** chỉ thấy điểm học phần sau khi chốt.
- **Chốt** bị chặn khi còn sinh viên thiếu điểm.
- **Không chạy LLM thật.**
- **Proposals:** 26 cái, #1–#26; 25 ACCEPTED, #15 không sửa test mà tìm nguyên nhân gốc.

## Quy trình
- **Dev sửa test cũ trước khi PM quyết: 5 lần** (#7, #22, #23, #26 và một lần ở sprint 7). Mọi lần đều đúng nội dung và PM đã chấp nhận. Luật mới: proposal **cùng dạng** với một proposal đã ACCEPTED thì được làm trước kèm số mới; khác dạng thì phải chờ.
- **Chuyển nơi kiểm AC route** sang story có route (#8, #18, #25). Đã áp làm luật chung cho sprint 9.
- **PM có một lần dừng chờ sai chỗ:** giao việc xong không đặt lệnh chờ, nên sprint nằm im tới khi chủ dự án hỏi. Từ đó mỗi lần giao việc đều có vòng giám sát.

## Nợ
- **Ghi nhận mở mức thấp (QC):**
  - test đột biến cho làm tròn TLR-1 (BUG-P6-02-1);
  - test khử trùng thông báo nhắc (BUG-P6-05-1);
  - một test đọc mã nguồn (BUG-P6-08-1);
  - `data-layer.spec` chập chờn ở CI (BUG-P6-08-2);
  - nhãn ô chọn dài ở 390 px;
  - `dev.mjs` + `BLOB_PUBLIC_ENDPOINT` với `EP_PORT_OFFSET`.
- **Danh sách `/students`** chưa có cột "Điểm hiện tại" / chip "Điểm giảm" (Q11).
- **Xuất XLSX** theo mẫu cột cố định; cấu hình cột để sau (Q9).
- **Chưa có số đo độ chính xác trích công thức của LLM thật** (Q13); luận văn ghi giới hạn.
- **Judge arm64 lỗi `cc1plus`** trên máy dev (môi trường), seed đầy đủ chạy ở stack QC.

## Chủ dự án tự kiểm
1. **G6-13:** tính tay A (9,5 A+), B (7,0 B), C (3,0 F) bằng máy tính, so với giải trình ở `/gradebook` (lớp 1). QC đã tính trước và khớp từng dòng.
2. **Lớp 2:** banner hiện; `Chốt điểm` khoá kèm lý do. Tải quy chế lớp 2 (thiếu quy tắc làm tròn) thì có mục "chưa rõ" và không cho xác nhận khi chưa điền.
3. Nói **"chốt"** để merge `sprint/8-p6` vào `main`.

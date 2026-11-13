# QC test case — US-P9-11 Nhập kết quả Forms vào sổ điểm (Could, cắt đầu tiên)
Nguồn: US-P9-11 + SRS FR-25–27, Q-P9-8, §6 #15–#16, PM 2026-11-12 (AC10–AC12). **Chỉ chạy nếu handoff ghi đã làm.** Không làm → «NGOÀI PHẠM VI SPRINT»; khi đó TC-11-18 xác nhận route `forms-import*` không tồn tại / trả `404` và `uploads/presign purpose=forms_import` bị từ chối.
Tệp mẫu QC (tạo bằng script, `docs/sprints/10/qc/scripts/forms-fixtures/`): `forms-ok.xlsx` / `.csv` (20 dòng: 15 khớp email SV đã xác minh, 2 khớp MSSV, 1 email lệch MSSV, 1 không khớp, 1 trùng SV có hai dòng khác dấu thời gian), `forms-big.csv` (> 5 MB), `forms-nocol.csv` (thiếu cột định danh), `forms-bad.txt`, `forms-badans.csv` (đáp án không khớp lựa chọn).

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-11-01 | AC1 | GV; `grade_item` MANUAL G1 (chưa công bố) | `POST uploads/presign {purpose:"forms_import",…}`; PUT tệp; `POST …/forms-import/preview {upload_id}` | `200`: 20 dòng đầu + cột phát hiện; **không ghi gì** (DB `grade_entries`, `forms_import_*` không đổi) | API + DB |
| TC-11-02 | AC1 | Đã preview | Ánh xạ cột ↔ câu ngân hàng hoặc «đáp án chuẩn» nhập tay + cột định danh (`Email`/`MSSV`) — cấu trúc `mapping` (Q-S10-23) | Cả hai kiểu ánh xạ được chấp nhận | API |
| TC-11-03 | AC2, AC12 | TC-11-02 | `IDEM=i1 api teacher POST …/forms-import '{upload_id,mapping,item_id:G1,answer_source}'` | `201`; mỗi SV một kết quả; ghi `grade_entries(source='QUIZ')` vào G1; dòng không khớp vào «Chưa khớp»; `published_at` **NULL** | API + DB |
| TC-11-04 | AC2 | TC-11-03 | Điểm ghi vào `grade_entries` của từng SV | Điểm tính từ `đúng/tổng` theo quy tắc nêu ở Q-S10-24 (thang điểm của `grade_item`); đối chiếu ≥ 5 SV bằng **tính tay**; làm tròn `decimal` | API + DB |
| TC-11-05 | AC3 | Hàng: email đã xác minh = SV1; email lệch MSSV; MSSV có trong lớp; MSSV tự khai không có trong lớp | Nhập | Chỉ khớp theo email đã xác minh hoặc MSSV trong danh sách lớp; email lệch MSSV → «Chưa khớp»; không tạo tài khoản; không nối tài khoản vào danh sách lớp theo MSSV tự khai | API + DB |
| TC-11-06 | AC4 | Tệp có SV X hai dòng (ts1 < ts2) | Nhập | Lấy dòng **mới nhất theo dấu thời gian Forms**; ghi chú «n dòng trùng» | API |
| TC-11-07 | AC4 | Đã nhập `forms-ok` | Chạy lại cùng tệp (cùng hash), khoá mới | Idempotent: điểm không đổi, không bản ghi trùng (`UNIQUE (course_id,item_id,file_hash)`); phản hồi cho biết đã nhập | API + DB |
| TC-11-08 | AC5 | – | Bấm `Xem trước kết quả` (UI) | Bảng `{sinh viên, điểm, đúng/tổng}` + số dòng chưa khớp **trước** khi ghi; điểm `decimal`; chưa ghi `grade_entries` | E2E |
| TC-11-09 | AC6 | – | Preview `forms-bad.txt`, `forms-big.csv`, `forms-nocol.csv` | `422` kèm vị trí (dòng/cột) mỗi cái; tệp > 5 MB cũng bị từ chối ở presign | API |
| TC-11-10 | AC6 | `forms-badans.csv` | Nhập | Dòng đó `0` + ghi «không khớp»; không đoán | API |
| TC-11-11 | AC7, AC10 | – | STUDENT, TA, ADMIN: presign `forms_import`, `preview`, `Xem trước`, `Xác nhận` | `403` cả **ba** bước, cho cả ba vai | API |
| TC-11-12 | AC10 | TEACHER, **chưa** preview upload U | `POST …/forms-import` với `upload_id=U` | `409 PREVIEW_REQUIRED` | API |
| TC-11-13 | AC8 | – | `docker compose logs`; `audit_log`; blob | Tệp gốc lưu `platform/blob` (hạn ≤ 90 ngày: cấu hình/đánh dấu hết hạn); `audit_log` có người import, hash, số khớp/chưa khớp; log không có email/MSSV nguyên văn (`identifier_masked`) | DB |
| TC-11-14 | AC8, AC12 | Import xong | `llm_audit` theo `batch_id` / thời gian; `go test … -run 'TestFormsImportNoLLM\|TestFormsImportScoreDecimal' -v` | 0 dòng LLM; cả hai test PASS | DB + GO |
| TC-11-15 | AC9 | Sổ điểm `LOCKED` (P6) | Nhập | `409` («Sổ điểm đã chốt»); sau «mở khoá có lý do» nhập được | API |
| TC-11-16 | AC11 | Import xong, G1 chưa công bố | `sv.kha` `GET /courses/$C1/me/grades` | **Không** thấy điểm Forms; sau `POST …/grade-items/{G1}/publish` (GV) → thấy; không có route công bố mới | API |
| TC-11-17 | AC11 | – | `audit_log` | Có bản ghi người import, hash tệp, số dòng khớp / chưa khớp | DB |
| TC-11-18 | – | Story **không** làm | Gọi các route `forms-import*`, `purpose=forms_import` | `404`/`422`; không có route nửa vời | API |

## Phân quyền
TC-11-11, 12; GV lớp khác → `403`; `item_id` của lớp khác → `404`/`422`.

## Lịch sử sửa TC
(trống)

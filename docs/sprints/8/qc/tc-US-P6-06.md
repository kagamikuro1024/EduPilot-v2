# QC test case — US-P6-06 (`/gradebook`: sửa ô kiểu bảng tính, xem trước tức thì, 409 hai người sửa, giải trình, lịch sử, công bố cột, 1.000 dòng)
Nguồn: `docs/specs/FEAT-gradebook/US.md` US-P6-06 (AC1–AC13) + `SRS.md` 3, 4.6, 6.2, 7; `docs/design/DESIGN.md` §14.10, §21–§22; `docs/UX.md` mục 6 (ảo hoá, khoá lạc quan); `fixtures/README.md`. Viết trước khi có code, không đọc code của dev (v1, 2026-10-31).

**Quy ước.** UI bằng `playwright-cli` ở 1440 × 900; lớp 1 `CONFIRMED`, chưa chốt. Hai phiên TEACHER = hai `browser.newContext()` cùng tài khoản `teacher@` (hoặc hai tài khoản nếu có). Ca ghi điểm đổi dữ liệu seed: làm trên DB dựng lại, ghi lại điểm gốc (30 dòng `midterm`/`final_exam` ở `fixtures/inputs-lop1.csv`) để khôi phục bằng nhập XLSX. `LOAD1000` = lớp thử 1.000 sinh viên do QC dựng bằng `docs/sprints/8/qc/scripts/p6-load1000.sql` (QC viết ở pha 2, `generate_series`, chạy trên DB riêng). `GO` = `p6-go-tests.sh`.

| TC-id | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| TC-P6-06-01 | AC1 | TEACHER, lớp 1 `CONFIRMED` | Mở `/gradebook` ở 1440 px; chụp `shots/gradebook-1440.png` | **một** `Panel`; tiêu đề "Sổ điểm" + bộ chọn lớp; hành động phụ `Xem công thức`; hành động chính `Chốt điểm`; menu `⋯` có `Nhập từ XLSX`, `Xuất XLSX` (không `Mở khoá` vì chưa chốt); bảng `DataTable`: cột **Sinh viên** sticky (kiểm `getComputedStyle(td).position === 'sticky'`), cột theo thành phần → item (Giữa kỳ, Kiểm tra tuần 9, Thi cuối kỳ), `Điểm cộng`, `Điểm trừ`, `Quá trình`, `Điểm học phần`, `Điểm chữ`, `Trạng thái`; **không** thẻ KPI, **không** card lồng card; số có dấu phẩy |
| TC-P6-06-02 | AC1 | như trên | Đọc dòng A, B, C | A: `Điểm cộng 1,00 (tối đa) · Điểm trừ 0 · Quá trình 10,00 · HP 9,5 · A+ · Đủ`; B: `0,50 · 0 · 8,25 · 7,0 · B · Đủ`; C: `0,25 · 1,50 · 1,25 · 3,0 · F · Đủ` (khớp bảng tay tc-US-P6-02) |
| TC-P6-06-03 | AC1 | repo | `bash scripts/ui-antipatterns.sh; echo rc=$?`; `playwright test gradebook.spec.ts -g 'gradebook layout'` | `rc=0`; xanh |
| TC-P6-06-04 | AC2 | `/gradebook` | `playwright-cli`: `page.getByText('Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường.').count()` | đúng **1** phần tử; nằm dưới bảng, cố định; không tooltip kèm; không lặp ở chỗ khác (menu, tiêu đề) |
| TC-P6-06-05 | AC2 | repo | `playwright test gradebook.spec.ts -g 'official note'` | xanh |
| TC-P6-06-06 | AC3 | `/gradebook` lớp 1; ô giữa kỳ của sv30 (đổi từ 9,00) | Gõ `8,5` → `Enter` | lưu, con trỏ xuống ô dưới (sv29); network `PUT …/grade-entries {item_id,student_id,score,version}`; ô hiện `8,50` |
| TC-P6-06-07 | AC3 | như trên | Gõ `8.5` → `Tab` | lưu, sang ô bên phải (cùng hàng); giá trị `8,50` |
| TC-P6-06-08 | AC3 | như trên | Gõ `9` rồi `Esc` | ô hoàn về giá trị cũ; **không** `PUT` (đếm request) |
| TC-P6-06-09 | AC3 | như trên | Gõ `10,5` (ngoài [0;10]) → `Enter` | `422 SCORE_OUT_OF_RANGE`; lỗi **tại ô**; chữ `10,5` còn trong ô (không mất); điểm cũ không đổi (DB) |
| TC-P6-06-10 | AC3 | như trên | Gõ `8,333` → `Enter` | `422 SCORE_PRECISION` "Điểm có quá 2 chữ số thập phân"; **không** tự làm tròn thành `8,33`; DB không đổi |
| TC-P6-06-11 | AC3 | như trên | Chuỗi bị từ chối: `abc`, `-1`, `1e2`, `8,,5`, `1000`, ` ` | mọi chuỗi không khớp `^\d{1,3}([.,]\d{1,2})?$` hoặc ngoài thang → lỗi tại ô, không `5xx` |
| TC-P6-06-12 | AC3 | như trên | Xoá ô (rỗng) → `Enter` | điểm bị xoá (DB không còn dòng `grade_entries` cho cặp đó — hoặc `score` NULL theo thiết kế; ghi nhận); trạng thái sinh viên → "Thiếu điểm" |
| TC-P6-06-13 | AC3 | ô item `EXAM` (Kiểm tra tuần 9) | Click / gõ vào ô | **không sửa được** (không con trỏ nhập); tooltip trên ô "Điểm lấy từ bài thi, sửa ở bài thi"; ép `PUT` bằng `curl` → `409 ITEM_NOT_EDITABLE` |
| TC-P6-06-14 | AC3 | repo | `GO ./internal/grade "TestPutEntryOK|TestPutEntryOutOfRange|TestPutEntryPrecision|TestPutEntryClear|TestPutEntryExamItemRejected" -tags integration`; `playwright test gradebook.spec.ts -g 'cell edit keys'` | PASS; xanh |
| TC-P6-06-15 | AC4 | `/gradebook` | Sửa ô cuối kỳ của sv.kha (B) từ `6,50` sang `8,00`; theo dõi DOM trước khi phản hồi về (chặn `route` trễ 1,5 s) | trong lúc chờ: ô hiện `8,00` và các ô **Quá trình / Điểm học phần / Điểm chữ** hiện "…" (không số cũ trông như mới); khi phản hồi về: B `Quá trình 8,25`, `HP thô (40·8,25+60·8)/100 = 8,10 → HP 8,0 · B+` — QC tính tay **trước** khi nhìn màn; số khớp |
| TC-P6-06-16 | AC4 | như trên | Đo p95 máy chủ: 30 lần `PUT` liên tiếp (script) và lấy `time_total` | p95 ≤ 300 ms (ghi số thực); phản hồi `PUT` chứa `row_result` (không có công thức tính ở trình duyệt: `grep -rn 'qt\s*=\|\*\s*0\.4' frontend/src/features/gradebook` rỗng) |
| TC-P6-06-17 | AC4 | repo | `GO ./internal/grade "TestPutEntryReturnsRowResult|TestPutEntryUnder300ms" -tags integration` | PASS |
| TC-P6-06-18 | AC5 | hai phiên TEACHER cùng mở sv30 / cuối kỳ (version v) | Phiên 1 lưu `7,5`; phiên 2 (chưa tải lại) lưu `6,0` | phiên 2 nhận `409 VERSION_CONFLICT` với `current:{score:7.5, version, updated_by:{name}, updated_at}`; ô hoàn lại; popover tại ô: "{Tên} vừa nhập 7,5 · {tuổi}" + hai nút `Giữ của tôi`, `Dùng của họ`; giá trị `6,0` người gõ vẫn thấy được |
| TC-P6-06-19 | AC5 | như trên | Bấm `Giữ của tôi` | gửi lại với `version` mới → `200`; DB `6,00`; `audit_log` có 2 dòng `set` (7,5 rồi 6,0) |
| TC-P6-06-20 | AC5 | như trên (lặp lại với kịch bản mới) | Bấm `Dùng của họ` | ô = 7,50, không `PUT` mới; DB không đổi |
| TC-P6-06-21 | AC5 | TA xem cùng ô | Quan sát | **chỉ nhân viên** thấy tên người sửa (STUDENT không có đường tới `updated_by`: `GET …/me/grades` không có trường đó) |
| TC-P6-06-22 | AC5 | repo | `GO ./internal/grade "TestPutEntryConflictBody" -tags integration`; `playwright test gradebook.spec.ts -g 'cell conflict choose'` | PASS; xanh |
| TC-P6-06-23 | AC6 | `/gradebook` | Menu dòng B → `Giải trình`; chụp | `GradeCalculation` tuyến tính, theo thứ tự: Giữa kỳ 20 %, Kiểm tra tuần 20 %, Cuối kỳ 60 % → điểm cộng (2 × 0,25 = 0,50, trần 1,0) → điểm trừ (vắng 2, miễn 2, trừ 0) → `QT₀ 7,75 → +0,50 → 8,25` → điểm cuối kỳ → `HP thô 7,20` → `7,0` → `B` |
| TC-P6-06-24 | AC6 | như trên | Bấm `Căn cứ` ở dòng làm tròn của B | mở đoạn trích quy chế **đúng trang** (điều 3.1, tr.2); dòng "Điểm cộng" → điều 2.1 (tr.1); dòng do giảng viên tự điền (lớp 2 sau điền "làm tròn") → "do giảng viên đặt" |
| TC-P6-06-25 | AC6 | như trên | Giải trình sv20 | có dòng lý do liệt: "Điểm thi cuối kỳ 2,75 dưới 3,0 → điểm F" (kèm điều 4.1); HP hiển thị **5,0** (không 0) |
| TC-P6-06-26 | AC6 | API | `j $Q/gradebook/students/$SVB_ID/explain -H "$TCH" \| jq -r '.lines[].value'` so với cột `explanation` của `seed/expected_final_grades.csv` dòng `sv.kha` | từng giá trị khớp (QC dùng `p6-hand-calc.py B`) |
| TC-P6-06-27 | AC6 | repo | `playwright test gradebook.spec.ts -g 'explain student B'` | xanh |
| TC-P6-06-28 | AC7 | `/gradebook` | Sửa điểm sv04 hai lần (6,25 → 7 → 6,25); menu dòng → `Lịch sử` | danh sách ≤ 50 mục mới nhất, mỗi mục: thời điểm, người sửa, thành phần, `cũ → mới`; có phân trang con trỏ (`?cursor=`); **không có** nút "Lịch sử" luôn hiện ngoài menu dòng |
| TC-P6-06-29 | AC7 | repo | `GO ./internal/grade "TestEntryHistoryFromAudit" -tags integration` | PASS |
| TC-P6-06-30 | AC8 | `LOAD1000` | `j "$Q_LOAD/gradebook?limit=100" -H "$TCH" -w '%{time_total}'` 20 lần | trang đầu p95 ≤ 500 ms; sắp theo `vn_fold(full_name), id` (QC kiểm tên có dấu: "Đặng" đứng sau "Dương"/theo quy ước `vn_fold`); `limit=101` → `422` hoặc kẹp 100 (ghi nhận; không trả > 100) |
| TC-P6-06-31 | AC8 | `LOAD1000` ở UI | Mở `/gradebook`, cuộn xuống cuối; đo số `tr` trong DOM | DOM ≤ **60** hàng ở mọi lúc; cuộn tải trang kế (`cursor`); 60 fps không giật (đo `performance` long task < 50 ms khi cuộn) |
| TC-P6-06-32 | AC8 | như trên | Gõ dở `7,` ở dòng 5, cuộn tới dòng 900, cuộn về | chữ `7,` **còn** ở ô (ô đang sửa ở kho ngoài DOM) |
| TC-P6-06-33 | AC8 | như trên | `Tab` / `Enter` điều hướng qua cuối cửa sổ ảo hoá | focus theo sang hàng/ô kế, cuộn theo, không mất focus; cột tên dính khi cuộn ngang |
| TC-P6-06-34 | AC8 | lớp 100 sinh viên | `EXPLAIN (ANALYZE) <truy vấn trang đầu>` — QC lấy truy vấn từ `pg_stat_statements` hoặc bật `log_min_duration_statement=0` trong lúc gọi API; chạy `EXPLAIN` cho từng truy vấn chạm `grade_entries`, `participation_events`, `attendance_records` | không `Seq Scan` trên 3 bảng; mỗi trang đúng **một** lượt truy vấn nguồn (đếm số câu SQL trong log khi gọi `limit=100`: không tỉ lệ với 100; so 10 dòng vs 100 dòng → số câu như nhau) |
| TC-P6-06-35 | AC8 | repo | `GO ./internal/grade "TestGradebookPageP95|TestGradebookNoNPlusOne|TestGradebookExplainNoSeqScan" -tags integration`; `playwright test gradebook.spec.ts -g 'virtualized 1000\|type then scroll keeps text'` | PASS; xanh |
| TC-P6-06-36 | AC8 | repo | `grep -rn 'DataTableVirtual' frontend/src/shared \| head`; `grep -rn 'cellNav' frontend/src/shared` | `cellNav` nằm trong `shared/` (mở rộng, không dựng bảng riêng ở `features/gradebook`: `grep -rn '<table' frontend/src/features/gradebook` rỗng) |
| TC-P6-06-37 | AC9 (a) | lớp 2 `NO_POLICY` | Mở `/gradebook` lớp 2 | EmptyState "Chưa có công thức điểm." + nút `Thiết lập công thức` (TEACHER) kèm banner; **không** cột điểm, không ô sửa được |
| TC-P6-06-38 | AC9 (b) | lớp có công thức nhưng 0 sinh viên (QC dựng lớp 3 trống + xác nhận công thức) | mở sổ | EmptyState "Lớp chưa có sinh viên." |
| TC-P6-06-39 | AC9 (c) | `/gradebook` | `setOffline(true)`; sửa 2 ô; `setOffline(false)` | ô xếp hàng, nhãn tại ô; tự gửi lại khi có mạng; DB nhận cả hai; không mất |
| TC-P6-06-40 | AC9 (d) | sổ `LOCKED` (sau tc-US-P6-08) | `PUT` ô; xem UI | `409 GRADEBOOK_LOCKED`; ô khoá; đầu bảng "Đã chốt v{n}"; `Mở khoá` ở menu |
| TC-P6-06-41 | AC9 | `/gradebook` | chặn `…/gradebook` trả 500 / chậm 3 s | skeleton khi tải; lỗi: nguyên nhân + `Thử lại` (`PageState`) |
| TC-P6-06-42 | AC9 | repo | `GO ./internal/grade "TestPutEntryLocked" -tags integration`; `playwright test gradebook.spec.ts -g 'gradebook states\|offline cell'` | PASS; xanh |
| TC-P6-06-43 | AC11 | lớp 1 | Một `PUT` thành công; `$PSQL "select entity, action, before, after, actor_id from audit_log where entity='grade_entries' order by created_at desc limit 1"` | **một** dòng `entity='grade_entries' action='set'`, `before` cũ / `after` mới, người sửa đúng; **cùng giao dịch** |
| TC-P6-06-44 | AC11 | stack riêng; làm audit lỗi (đổi tên bảng `audit_log` tạm thời trong DB của QC hoặc `REVOKE INSERT`) | `PUT` | **fail-closed**: `5xx`/`503`, điểm **không** ghi (`select score …` không đổi); khôi phục bảng sau |
| TC-P6-06-45 | AC11 | repo | `grep -rn "mock/gradebook" frontend/src` ; `GO ./internal/grade "TestPutEntryAuditFailClosed|TestPutEntryAuditRow" -tags integration` | rỗng; PASS |
| TC-P6-06-46 | AC11 | TEACHER mở hồ sơ B `/students/<B>` tab "Điểm" | xem | cùng `GradeCalculation` thật (số khớp TC-23); lớp 2: EmptyState "Lớp chưa có công thức điểm đã xác nhận." |
| TC-P6-06-47 | AC12 | `/gradebook` lớp 1; cột giữa kỳ `MANUAL`/`IMPORT` | Menu cột → `Công bố cho sinh viên` | `POST …/grade-items/{itemId}/publish {published:true}`; tiêu đề cột hiện chữ nhỏ "Đã công bố"; DB `published_at` có giá trị; `audit_log` có dòng; gọi lại cùng thân → idempotent (200, không audit thứ hai hoặc audit nhưng không đổi `published_at`) |
| TC-P6-06-48 | AC12 | như trên | `Gỡ công bố` | "Chưa công bố"; `published_at` NULL; audit |
| TC-P6-06-49 | AC12 | cột `EXAM` | menu cột | **không** có mục công bố; `curl publish` → `409 ITEM_NOT_PUBLISHABLE` |
| TC-P6-06-50 | AC12 | cột đã công bố | nhập điểm mới cho sinh viên Y; `sv Y` mở `/me` | điểm mới **tự** hiện cho Y (không cần thao tác thêm) |
| TC-P6-06-51 | AC12 | sổ `LOCKED`; TA | `publish` → `409 GRADEBOOK_LOCKED`; TA không thấy mục menu; `code -X POST -H "$TA_" …/publish` → `403` | đúng |
| TC-P6-06-52 | AC12 | repo | `GO ./internal/grade "TestPublishItemToggle|TestPublishItemAudit|TestPublishExamItemNotToggleable|TestPublishLockedRejected" -tags integration`; `playwright test gradebook.spec.ts -g 'publish column'` | PASS; xanh |
| TC-P6-06-53 | AC13 | công thức lớp (QC dựng lớp thử/ DB riêng) có `PENALTY.applicable=false` mà sinh viên có sự kiện `PENALTY` −0,5 | Mở `/gradebook`; `GET …/gradebook` trang đầu | **một** `InlineNotice` mỏng không chặn "Có điểm trừ ghi tay không được tính theo công thức"; mở xem danh sách sinh viên liên quan; `warnings[]` có `PENALTY_NOT_IN_SCHEME`; lớp 1 seed (không `PENALTY`) → **không** hiện |
| TC-P6-06-54 | AC13 | như trên với `BONUS.applicable=false` | mở sổ | cảnh báo "Có điểm cộng ghi tay không được tính theo công thức" + `BONUS_NOT_IN_SCHEME`; P = 0 / B = 0 trong giải trình |
| TC-P6-06-55 | AC13 | repo | `GO ./internal/grade "TestGradebookWarningsPenaltyNotInScheme" -tags integration`; `playwright test gradebook.spec.ts -g 'penalty not in scheme notice'` | PASS; xanh |

## Nhánh lỗi (SRS mục 3)
Đã phủ: điểm ngoài thang (TC-09), chính xác (TC-10, 11), ô EXAM (TC-13), `VERSION_CONFLICT` (TC-18), chưa công thức (TC-37), mất mạng (TC-39), đã chốt (TC-40).
| TC-id | Tình huống | Bước | Kết quả mong đợi |
| --- | --- | --- | --- |
| TC-P6-06-56 | `PUT` thiếu `version` / `version` kiểu chữ | `curl -X PUT $Q/grade-entries -H "$TCH" -d '{"item_id":"…","student_id":"…","score":"5"}'` | `422 VALIDATION_FAILED`; không ghi |
| TC-P6-06-57 | `PUT` cho sinh viên không thuộc lớp / đã rời lớp | `student_id` của sinh viên lớp 2 vào `$Q` | `404`/`422` (FK phức hợp); không ghi |
| TC-P6-06-58 | `PUT` score là số JSON (`5`) thay vì chuỗi | `-d '{…,"score":5,…}'` | chấp nhận hoặc `422` nhất quán; **không** lưu sai (QC thử `5.005`, `0.1+0.2` biểu diễn: gửi `"score":8.333` → `SCORE_PRECISION`) |
| TC-P6-06-59 | Đua nhập: 2 lần `PUT` cùng `version` song song (20 lượt) | script `xargs -P` | đúng một thắng, còn lại `409 VERSION_CONFLICT`; `audit_log` đúng số lần thành công |

## Phân quyền (AC10)
| TC-id | Tình huống | Bước | Kết quả mong đợi |
| --- | --- | --- | --- |
| TC-P6-06-60 | TA mở `/gradebook` lớp 1 | UI | đọc được bảng, mở `Giải trình`, `Lịch sử`; ô **không** sửa được (hiển thị chữ, không `input` / không con trỏ nhập); **không** `Chốt điểm`, `Nhập từ XLSX`, `Xuất XLSX`, `Mở khoá`, `Công bố cho sinh viên` |
| TC-P6-06-61 | TA / STUDENT / ADMIN gọi `PUT` | `for T in "$TA_" "$SVA" "$ADM"; do code -X PUT -H "$T" -H 'Content-Type: application/json' -d '{}' $Q/grade-entries; done` | `403 403 403` |
| TC-P6-06-62 | sinh viên / lớp khác | giảng viên lớp khác `GET $Q/gradebook`; STUDENT `GET $Q/gradebook` | `403`; `403` (#11) |
| TC-P6-06-63 | `GO` + UI | `GO ./internal/grade "TestGradebookRBAC" -tags integration`; `playwright test gradebook.spec.ts -g 'gradebook rbac'`; `bash docs/sprints/8/qc/scripts/p6-rbac.sh` | PASS; xanh; `RBAC: PASS` |

## Script chạy được
`scripts/p6-env.sh`, `p6-rbac.sh`, `p6-go-tests.sh`, `p6-hand-calc.py`; QC viết ở pha 2: `p6-load1000.sql`, `p6-put-race.sh`, `p6-grid-ui.mjs` (`playwright` Enter/Tab/virtualized/offline).

## Câu hỏi cho BA / PM (không tự đoán)
- Q-06-a (TC-P6-06-12): xoá ô = "xoá điểm" — `grade_entries` bị `DELETE` hay `score` NULL (`score numeric(5,2)` không NOT NULL?). QC chấp nhận cả hai miễn trạng thái sinh viên → "Thiếu điểm" và có audit; nếu cần một dạng, BA ghi vào SRS.
- Q-06-b (TC-P6-06-30): `limit > 100` trả 422 hay kẹp 100. Luật 13 nói `limit ≤ 100`; QC chấp nhận cả hai.

## Lịch sử sửa TC
- 2026-10-31 — **proposal #11 ACCEPTED** (spec đổi: ngoài lớp → `403`, id không phải uuid → `404`): TC-P6-06-62 `404; 403` → `403; 403`. TC-P6-06-57 giữ `404`/`422` (sinh viên không thuộc lớp trong thân yêu cầu = tài nguyên, không qua guard).

# QC report — US-P3-02 (`internal/privacy`)  · Kết luận: PASS (AC3 đóng 2026-10-11: `TestRosterInvalidateNotBlockedByLongJob` chạy xanh)

Handoff: `docs/sprints/6/handoff/dev-US-P3-02.md` (commit `db7c187`). Bộ TC: `tc-US-P3-02.md` (61 TC, không sửa).
**Giới hạn bề mặt:** story chỉ giao gói Go (không handler, không service chat). 29 TC viết theo API (`precheck`, `/chat/sessions`, SSE, provider giả) **chưa có bề mặt** → QC kiểm cùng nội dung ở **tầng gói** bằng test hộp đen mới `backend-go/internal/privacy/qc_p302_test.go` (`package privacy_test`, chỉ gọi hàm export, ca tấn công do QC soạn từ US/SRS, Redis dùng stack dev cổng 6380, khoá thử xoá sau). Ghi rõ từng TC "mức gói" vs "chưa kiểm được".
Không UI → không `playwright-cli`.

**Cập nhật theo PM (Q1, Q3):** TC thiếu bề mặt chuyển sang story giao bề mặt; Q3 (Redis nối lạnh) = D2 (ngưỡng 30 ms, dev xử lý, QC kiểm lại ở lượt sau). Lint `qc_p302_test.go` đã sạch.

| TC chuyển | Đích |
| --- | --- |
| 49, 50 (Redis hỏng / che lỗi qua `PRIVACY_MASK_TIMEOUT_MS`, `llm_audit MASK_FAILED`) | `report-US-P3-03` (hook `internal/llm` + route `_test/llm/payloads`, #9) |
| 19–22, 24 (roster qua API duyệt/mời + `precheck`), 58–60 (`precheck` phân quyền) | `report-US-P3-06` |
| 23, 25 (`TestRosterInvalidateNotBlockedByLongJob`, vô hiệu ≤ 5 s khi hàng AI bận) | `report-US-P3-06` — **còn FAIL tới khi dev thêm test** |
| 30–34 (payload), 36 (log), 43, 47, 52, 55 (p95 API), 56 (API), 61 | `report-US-P3-05` / `report-US-P3-03` (quét payload provider giả) |

**Kết luận story:** các TC thuộc bề mặt gói `internal/privacy` (01–18, 26–29, 35, 37–42, 44–46, 48, 51, 53–57 và TC-08) PASS; riêng AC3 còn treo vì thiếu `TestRosterInvalidateNotBlockedByLongJob` (dev đã ghi nợ, hạn US-P3-06). Theo Q1 story đóng khi phần còn lại PASS → đề nghị PM coi **AC3 là điều kiện chuyển tiếp ở P3-06**.

## Cổng đã chạy
| Lệnh | Kết quả |
| --- | --- |
| `go test -count=1 -race ./internal/privacy ./cmd/worker` | PASS — 58 passed, 2 gói |
| `go test -run 'TestUnmaskStream$' -v` | PASS — "số tổ hợp đã chạy: 72470" (≥ 50.000) |
| `go test -bench BenchmarkMask4k -benchtime=200x` | PASS — 1.339.720 ns/op ≈ 1,34 ms (≤ 5 ms) |
| `go test -run TestDetectLinearTime` + 3 họ test Roster / Redact / Scanner / Mask / Mapping (25 test) | PASS, tên test khớp TC-10/18/25(trừ NotBlocked)/29/48/51/54 |
| `go vet ./internal/privacy` | PASS |
| `QC_REDIS=redis://localhost:6380 go test -run TestQC -v` (QC, 8 test) | PASS 8/8 (xem dưới; lượt đầu TTL = −2 vì Redis nối lạnh >15 ms → fallback bộ nhớ; 5 lượt sau ổn định, xem Ghi chú 1) |

## TC
| TC | Kết quả | Chứng cứ |
| --- | --- | --- |
| 01–06 | PASS (mức gói) | 14 mẫu MSSV/email/SĐT/CCCD bị bắt đúng loại (`TestQCPositive`) |
| 07 | PASS (mức gói) | 8 mẫu âm số (cổng, năm, IP, CVE, hex, RSA 2048…) không bị bắt |
| 08 | PASS (mức gói) | `TestQCNoDoubleCount`: không khoảng nào chồng PHONE/CCCD (13 chữ số liền không bị bắt: ngoài quy tắc CCCD 12 số) |
| 09 | PASS (mức gói) | `0241234567` không bị bắt (theo cách đọc Q-QC-P3-02-1) |
| 10 | PASS | dev khai 22 dương / 21 âm; tên test chạy ok |
| 11–16 | PASS (mức gói) | có dấu / không dấu / HOA / đảo / rút bắt đúng; 7 mẫu âm tên (`Khải`, `An`, `anh…`, `Hoa Kỳ`, `MIT`, `Mai…`) không bị bắt |
| 17 | PASS (mức gói) | `Nguyễn Văn An` bị bắt, `Nguyễn Văn Anh` không |
| 18 | PASS | |
| 19–23 | KHÔNG KIỂM ĐƯỢC | cần API duyệt / mời thành viên + `precheck`. TC-23: không có test `…NotBlockedByLongJob` → **FAIL** |
| 20 | PASS (mức gói, một phần) | `TestRosterInvalidatedOnMemberChange` ok; TTL khoá roster chưa đo bằng `redis-cli` |
| 24 | PASS (gián tiếp) | `TestRosterExcludesStaff` ok |
| 25 | **FAIL** | `TestRosterScopedToCourse`, `TestRosterExcludesStaff`, `TestRosterInvalidatedOnMemberChange` ok; `TestRosterInvalidateNotBlockedByLongJob` **không tồn tại** |
| 26–28 | PASS (mức gói) | `"[đã ẩn] [đã ẩn] [đã ẩn] [đã ẩn]"`; idempotent; chuỗi sạch (tiếng Việt, emoji, xuống dòng, `[[ -f "$f" ]]`) giữ đúng từng byte |
| 29 | PASS | |
| 30–34 | PASS (mức gói) | `[[SV_1]]` ổn định qua 3 biến thể tên (có dấu / không dấu / HOA); đủ `[[SV_1]] [[SV_2]] [[MSSV_1]] [[EMAIL_1]] [[SDT_1]] [[CCCD_1]]`; tin sạch giữ nguyên, `n=0`; chủ phiên cũng bị che. Quét payload provider giả thật: KHÔNG KIỂM ĐƯỢC (chờ US-P3-03) |
| 35 | PASS | HASH có `p:[[SV_1]]`, `r:<sha256>`, `n:SV`; TTL = 24h0m0s |
| 36 | KHÔNG KIỂM ĐƯỢC | cần chạy chat + log; `TestMaskingNeverLogged` của dev ok |
| 37 | PASS | `Session ""`: số khoá `ep:mask:*` trước/sau bằng nhau |
| 38–39 | PASS (mức gói) | `"bui thanh khai"` khôi phục về bản gõ lần đầu; `[[SV_1]]`, `[[ SV_1 ]]`, `[[sv_1]]` đều khôi phục |
| 40 | PASS | 72.470 tổ hợp |
| 41 | PASS (mức gói) | QC cắt chuỗi có placeholder, `[`, `[[`, `[[x]]`, emoji ở **mọi** vị trí (cắt 2, cắt 3, từng rune: 3.003 tổ hợp): ghép = `Unmask(toàn)`, 0 khung chứa nửa placeholder |
| 42 | PASS | `` `[[ -f "$f" ]]` `` và `a[[i]]` giữ nguyên |
| 43 | KHÔNG KIỂM ĐƯỢC | cần provider giả đo thời gian |
| 44 | PASS (mức gói) | `[[SV_9]]` → "bạn"; dòng log `placeholder sót count=1` không chứa nội dung |
| 45 | PASS | xoá `ep:mask:{sid}` giữa chừng → "bạn" |
| 46 | PASS | `"… gửi cho [[MSSV_"` + `Flush` → `"… gửi cho bạn"` |
| 47 | KHÔNG KIỂM ĐƯỢC | cần SSE / DB |
| 48 | PASS | 4 test Scanner ok |
| 49–50 | KHÔNG KIỂM ĐƯỢC | chưa có công tắc thử / provider giả / `llm_audit`; `TestMaskRedisDownFallsBackInMemory`, `TestMaskFailsClosed` của dev ok |
| 51 | PASS | |
| 52 | KHÔNG KIỂM ĐƯỢC | cần phiên chat; `TestRosterScopedToCourse` của dev ok |
| 53–54 | PASS | `[[SV_1]]` của phiên khác → "bạn" |
| 55 | PASS (một phần) | 1,34 ms/op; p95 `precheck` API: KHÔNG KIỂM ĐƯỢC |
| 56 | PASS (mức gói) | `a1…`, `(((…` ở 1.000 / 4.000 / 8.000 / 100.000 ký tự: 0,03–10,9 ms, tăng tuyến tính; API: KHÔNG KIỂM ĐƯỢC |
| 57 | PASS | PII ở ký tự ~25.000 vẫn bị bắt |
| 58–61 | KHÔNG KIỂM ĐƯỢC | endpoint `precheck` và `/chat/sessions` chưa có |

Tổng: PASS 38 (trong đó 23 chỉ mức gói) · FAIL 2 (23, 25) · KHÔNG KIỂM ĐƯỢC 21 (tính FAIL).

## AC
| AC | Kết quả |
| --- | --- |
| AC1 regex | PASS (mức gói) |
| AC2 tên roster | PASS (mức gói) |
| AC3 roster theo lớp, vô hiệu ≤ 5 s, không bị việc AI chặn | **FAIL** — thiếu test + TC API không chạy được |
| AC4 `Redact` | PASS |
| AC5 placeholder ổn định | PASS (mức gói) |
| AC6 chủ phiên cũng bị che | PASS (mức gói) |
| AC7 Redis HASH, TTL, không log | PASS (HASH/TTL/không-phiên); không-log CHỜ P3-03 |
| AC8–AC9 khôi phục, cắt ở mọi vị trí | PASS |
| AC10 giữ ≤ 32 rune | KHÔNG KIỂM ĐƯỢC (chưa đo ở mức SSE; test dev ok) |
| AC11 placeholder sót / hết hạn / mở dở | PASS (mức gói) |
| AC12 Redis hỏng / che lỗi | KHÔNG KIỂM ĐƯỢC ở mức hệ thống |
| AC13 không chéo lớp / phiên | PASS (phiên); lớp chỉ test dev |
| AC14 hiệu năng / ReDoS | PASS (mức gói) |

## Lỗi / ghi chú
- **BUG-1 (Trung bình)** — AC3: không có test chứng minh vô hiệu roster không bị chặn bởi việc AI dài (handoff Nợ 1). Vì hàng việc AI dài chưa tồn tại, chỉ bổ sung được ở US-P3-06; TC-23 vẫn FAIL tới lúc đó.
- Ghi chú 1 (Thấp): lượt `Mask` đầu tiên qua kết nối Redis **lạnh** có thể vượt ngưỡng 15 ms và rơi sang bộ nhớ (QC gặp 1/6 lượt: TTL −2, 0 khoá). Đúng thiết kế fallback, nhưng ánh xạ của lượt đó không bền → lượt sau `Unmask` ra "bạn". Đề nghị dev cân nhắc khởi động ấm kết nối (ghi ở `proposals.md`).
- Ghi chú 2: câu hỏi mở Q-QC-P3-02-1…5 còn chờ BA.

## Kiểm phản mẫu / luật / phân quyền
Không `fetch`, không thư viện mới, không `float64`; không log PII (dòng warn chỉ có `count`). Phân quyền (TC-58–61) chờ endpoint.

## Đề nghị
FAIL tới khi: (a) bổ sung `TestRosterInvalidateNotBlockedByLongJob` (US-P3-06) ; (b) PM chuyển TC cần bề mặt API (19–23, 36, 43, 47, 49–52, 58–61) sang report của US-P3-03/05/06 qua `proposals.md`.


## Cập nhật AC3 (2026-10-11, sau fix P3-06 `23c1587`)
`go test -count=1 -race ./cmd/worker -run 'TestRosterInvalidateNotBlockedByLongJob|TestRosterInvalidateRegistered'` → **PASS** (0,45 s; 0,70 s). TC-23 / TC-25 chuyển FAIL → **PASS** (mức test Go); đo trên stack với thành viên thật chưa chạy. AC3 đóng; story đóng khi các TC chuyển (Q1) PASS ở P3-03/05/06.

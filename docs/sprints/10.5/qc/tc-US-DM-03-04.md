# QC test case — US-DM-03 (cổng Threads), US-DM-04 (cổng cờ chèn lệnh)
Nguồn: US.md + SRS §3.2, 3.3, 4.3; Q-D-3, 7, 8, 10; D60; P-11-9. Hộp đen; quy ước: `tc-00-chung.md`. **Provider giả, không Gemini thật.** `Structured` = LLM qua 9router (hoặc `fake` chat); mỗi TC ghi rõ.
Cách dựng ca: đầu vào có trong bản phát lại với `P(có)` đã biết (Q-S105-2). Vùng lưỡng lự của `AddOnly` là 0,05 < P < 0,5; **đầu vào ngoài bản phát lại trả P=0,5 theo SRS 4.6 thì với `AddOnly` rơi vào «có» (≥ 0,5), không phải lưỡng lự** — mâu thuẫn SRS cần BA/dev làm rõ (Q-S105-3); TC dùng đầu vào có trong bản phát lại với P = 0,2 cho ca lưỡng lự. Bật cổng: ADMIN `PUT` cổng + lớp 1 (hoặc `SEED_DECISION_DEMO=1`).

## US-DM-03 Cổng `THREADS_PRECHECK`
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-D03-01 | AC1 | Cổng bật lớp 1; bài chứa MSSV (luật bắt) | `sv.kha` `POST /courses/$C1/threads` | Bị chặn bởi luật; **0 dòng** `llm_audit DECIDE` (không gọi decision khi luật đã chặn) | API+DB |
| TC-D03-02 | AC1, AC2 | Bài sạch với luật nhưng riêng tư tinh vi (replay P(có)=0,9) | Đăng | Bị chặn, `cascade_path=FAST`, đúng 1 dòng `DECIDE`; hộp thoại hai lối | API+E2E |
| TC-D03-03 | AC2 | Bài sạch (replay P(có)=0,01 ≤ ε) | Đăng | Lưu được (đăng thành công); `FAST`; **0** lời gọi `Structured` (`llm_audit` task `CLASSIFY`/`UTILITY` không thêm) | API+DB |
| TC-D03-04 | AC2 | Bài P(có)=0,2 (vùng giữa) | Đăng | Gọi `Structured` qua 9router (1 dòng task gốc + 1 dòng `DECIDE`); `cascade_path=FALLBACK`; kết quả theo `Structured` (cả hai nhánh: cho qua / chặn) | API+DB |
| TC-D03-05 | AC2 | `FAKE_MODE=error` | Đăng bài sạch và bài riêng tư | Thẳng `Structured` (`FALLBACK`); bài sạch lưu, bài riêng tư chặn; không 5xx | API |
| TC-D03-06 | AC2 | `FAKE_MODE=error` + `Structured` lỗi (9router tắt / chặn host) | Đăng bài sạch | **Chặn** (`FAIL_CLOSED`); người dùng thấy hộp thoại hai lối; không 5xx; không lưu bài | API+E2E |
| TC-D03-07 | AC2 | `Structured` chậm 10 s (`THREADS_STRUCTURED_TIMEOUT=6s`) | Đăng; đo thời gian | Chờ ~6 s rồi chặn (`FAIL_CLOSED`); không treo quá 6 s + phụ | API |
| TC-D03-08 | AC3 | Mẫu `dev` của E1 (nơi: Q-S105-4) | `go test ./internal/privacy -run TestPrecheckAddOnly -v`; QC chạy lại E1 `dev` với cổng bật / tắt | Không mẫu dương luật đã bắt mà nay lọt; mọi mẫu luật chặn vẫn chặn dù decision «không» (`always_no`) | GO+API |
| TC-D03-09 | AC4 | Bài chỉ gồm placeholder `[[MSSV_1]]` / `[[SV_1]]` | Đăng | Luật chặn trước decision (0 `DECIDE`); `go test … -run TestPlaceholderOnlyBlockedByRules` PASS | GO+API |
| TC-D03-10 | AC5 | Cổng bật | `POST …/threads/precheck` (lúc gõ) 20 lần, nhiều nội dung | **0** lời gọi `DECIDE`; trả lời như trước (luật + embedding); giao diện không đổi | API+DB |
| TC-D03-11 | AC1 | – | Sửa bài (`PATCH`) thành nội dung riêng tư tinh vi; bình luận | Cũng qua cổng (đăng / sửa / bình luận); lúc `precheck` thì không | API |
| TC-D03-12 | AC6 | Bài bị chặn (4 nguyên nhân: luật, decision, `Structured`, fail-closed) | Quan sát giao diện | Hộp thoại hai lối «Chuyển sang chat riêng» / «Ẩn thông tin rồi đăng», **cùng lời** ở cả 4; không từ kỹ thuật / nguyên nhân (`grep` DOM: không «decision», «cascade», «chặn bởi»); chữ đã gõ còn (`useAutosaveDraft`) | E2E |
| TC-D03-13 | AC6 | Lời gọi chậm (fake p50 1.500 ms) | Bấm đăng | Sau 800 ms hiện «Đang kiểm tra»; xong → kết quả; nút không cho bấm đôi (không đăng trùng) | E2E |
| TC-D03-14 | AC7 | Bài khủng hoảng (mẫu có trong luật) | Đăng, cổng bật; đếm `llm_audit` | `422 CRISIS_SUPPORT` như hiện hành; **0** `DECIDE`, 0 lời gọi `Structured`, 0 `llm_audit`; `TestCrisisNeverReachesDecide` (Threads) PASS | API+GO |
| TC-D03-15 | AC8 | Bản phát lại có nhãn tập `dev` E1 | Chạy cổng lên toàn bộ `dev` (script QC), tính (i)–(iv) | (i) 0 dương luật-bắt nay lọt; (ii) số dương mới bắt ≥ luật một mình; (iii) chặn nhầm trên âm ≤ 0,05; (iv) rò rỉ payload = 0 (`grep` thân tới provider giả); tập `test` **không** chạy ở pha này (chỉ lần chốt, 1 lần) | API |
| TC-D03-16 | AC9 | Đăng bài có PII | Bắt thân tới provider (máy chủ giả) + `pii_events` | Payload đã che; `llm_audit` không nội dung; `pii_events` chỉ đếm (không cột văn bản mới) | API+DB |
| TC-D03-17 | AC10 | Cổng bật chỉ lớp 1 | `sv.kha` (lớp 1) và SV lớp 2 đăng cùng nội dung riêng tư tinh vi | Lớp 1 qua cổng (`DECIDE` có); lớp 2 **không** (đường cũ) | API+DB |
| TC-D03-18 | AC10 | – | Teacher, TA lớp 1 đăng cùng bài | Cũng qua cổng như SV (quy tắc tường lửa hiện hành) | API |
| TC-D03-19 | AC11 | `GEMINI_API_KEY` trống; rồi hết hạn mức (`FAKE_MODE=error` + `budget_blocked`); rồi mạng lỗi | Đăng bài sạch / riêng tư | Đúng AC2; không 5xx; mất cả hai nhà cung cấp → chặn + hộp thoại hai lối | API |
| TC-D03-20 | AC2 | Hai bài cùng lúc (song song 10 request) khi decision chậm | Đăng | Không deadlock; mỗi request một kết quả đúng; không đăng trùng; hạn mỗi request riêng | API |

## US-DM-04 Cổng `INJECTION_FLAG`
Cờ = regex OR `Structured` OR decision(P≥0,5). Chấm bài tập P7 ở lớp 1 (rubric có sẵn). Fixture: `docs/specs/FEAT-assignments/fixtures/submissions/s1…s7` + `benchmarks/real-llm/data/injection/v1…v5`.
| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-D04-01 | AC1 | Cổng bật lớp 1; mỗi bài chèn lệnh (6 biến thể) | Nộp rồi chấm; đọc cờ | Cờ khi có **ít nhất một** tầng true; tầng vắng (lỗi / tắt) thì cờ theo tầng còn lại; QC dựng 7 tổ hợp (regex / Structured / decision) bằng `fake` + bản phát lại và đối chiếu bảng OR | API |
| TC-D04-02 | AC1, AC7 | `FAKE_MODE=error` | Chấm 6 biến thể | Việc chấm **hoàn tất**; cờ theo regex OR `Structured`; điểm không đổi; hàng chờ không treo | API |
| TC-D04-03 | AC2 | Cùng bài, cổng bật / tắt | So `total` + điểm từng tiêu chí + chữ đầu vào chấm | **Điểm y hệt** bật / tắt (cờ không đổi điểm); câu chèn **vẫn có** trong đầu vào chấm; việc chấm không bị chặn | API+DB |
| TC-D04-04 | AC2 | `go test ./internal/grading -run 'TestInjectionOR\|TestInjectionScoreUnchanged' -v` | – | PASS | GO |
| TC-D04-05 | AC3 | `s4-trong` (bài trống); bài 19 từ; bài đúng 20 từ có chèn lệnh | Chấm; đếm `llm_audit` | Trống / 19 từ: `EMPTY_SUBMISSION` (trống) hoặc không gọi **decision và `Structured` gắn cờ** (0 dòng `DECIDE`); không `INJECTION_SUSPECTED` từ hai tầng đó; 20 từ: có gọi (biên: Q-S105-5) | API+DB |
| TC-D04-06 | AC3 | – | `go test … -run TestInjectionEmptySkipped -v` | PASS | GO |
| TC-D04-07 | AC4 | Bài 40.000 ký tự: chèn lệnh ở giữa (vị trí 20.000, **ngoài** 8.000 đầu + 8.000 cuối) và ở đầu | Chấm; bắt thân tới provider giả | Thân gửi decision ≤ 16.000 ký tự = 8.000 đầu + 8.000 cuối (phần giữa cắt, không lưu ở đâu — kiểm DB + log); regex vẫn quét toàn văn → bài chèn lệnh ở giữa vẫn có cờ (regex) | API+DB |
| TC-D04-08 | AC4 | Bài có dòng tiêu đề danh tính (tên, MSSV) | Bắt thân | Đã bỏ dòng danh tính; placeholder; không tên / MSSV | API |
| TC-D04-09 | AC5 | Bản phát lại có nhãn; cổng bật | Chấm 6 biến thể + 7 đối chứng (`s1`…`s7` không chèn lệnh gồm `s4-trong`; Q-S105-6 về con số 7) | OR ba tầng bắt **≥ 6/6**; **0 cờ** trên 7 đối chứng; regex một mình 1/6 (mốc, tắt hai tầng kia) | API |
| TC-D04-10 | AC5 | `go test ./internal/grading -run TestInjectionSet6of6 -v` | – | PASS; log từng bài | GO |
| TC-D04-11 | AC5 | **Chờ phép PM** | Lượt thật ≤ 300 lời gọi | Ghi «KHÔNG KIỂM ĐƯỢC (chờ phép PM)» — không tính FAIL; không chạy | – |
| TC-D04-12 | AC6 | Cổng bật | `llm_audit` lời gọi `DECIDE` của chấm | Làn `BATCH`; `timeout_ms=3000`; chạy song song với lời gọi chấm (thời gian chấm tổng ≈ max, không cộng) | DB |
| TC-D04-13 | AC6 | `FAKE_P50_MS=5000` | Chấm | Decision hết hạn ở ~3 s; chấm không bị giữ quá hạn đó; kết quả hoàn tất | API |
| TC-D04-14 | AC7 | Cầu dao mở / `budget_blocked` | Chấm | `FALLBACK` (tầng vắng); chấm xong; `llm_audit` ghi lỗi | API+DB |
| TC-D04-15 | AC8 | Bài có PII | Bắt thân | Đã che; `llm_audit` không nội dung | API+DB |
| TC-D04-16 | AC8 | Bài bị cờ | `sv.kha` (chủ bài): bài / kết quả / bản công bố; `ta@`, `teacher@` lớp 1: màn duyệt | SV **không** thấy cờ (`grep` `INJECTION_SUSPECTED`, «Nghi chèn lệnh» = 0 mọi phản hồi, cả sau công bố); cờ **không** vào bản công bố; TA / GV lớp thấy ở màn duyệt | API+E2E |
| TC-D04-17 | AC8 | – | GV / TA lớp 2; SV lớp khác | `404` / `403` | API |
| TC-D04-18 | AC9 | Repo | `grep -rn "Decide(\|Cascade(" backend-go/internal --include=*.go` (không test) + `go test … -run TestDecideCallersAllowlist -v` | Ở `internal/grading`, chỉ từ `flagInjectionDecision`; không từ tính điểm / rubric / tổng / làm tròn; test PASS (và thật sự đỏ khi thêm lời gọi — QC thêm lời gọi thử vào bản sao tạm, test đỏ, xoá) | GO+DIFF |
| TC-D04-19 | AC1 | Cổng chèn lệnh tắt ở lớp (bật ở lớp khác) | Chấm bài chèn lệnh | Cờ chỉ regex OR `Structured` (P-11-9 vẫn chạy); 0 `DECIDE` | API+DB |
| TC-D04-20 | AC1 | Lớp đã bật | Chấm lại (round 2) / chấm nhiều bài song song | Không gọi decision hai lần cho cùng bài-vòng (idempotent); không rò cờ giữa bài | API+DB |

## Nhánh lỗi
TC-D03-05…07, 19, 20; TC-D04-02, 05, 13, 14.

## Phân quyền
TC-D03-17, 18; TC-D04-16, 17.

## Lịch sử sửa TC
(chưa có)

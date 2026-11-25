# Trạng thái lỗi (qc2, 25/11/2026)

**Cách đếm.** Một dòng cho mỗi mã `DEF-*` có mục riêng trong `defects-<sprint>.md` (tiêu đề `###`/`##`, dòng bảng hoặc dòng `- **DEF-…**`) và `defects-opencode-go.md`; mã chỉ được nhắc tới trong mục khác không tính. Tổng **212** mã (132 là số tiêu đề `###`; còn lại là dòng bảng ở defects-9/10/10.5/10.6, tiêu đề `##` ở defects-4/10.5, dòng `- **DEF-…**` ở defects-4). `defects.md` (16/11, 72 ca) và bản `defects-status.md` cũ (19/11, 111 mã) là số của ngày đó, thay bằng tệp này.

**Trạng thái cuối** lấy theo thứ tự: (1) `~/.config/edupilot-team/fixes.tsv` (đóng / rút, đã bung dải mã như `R10S-04..07`); (2) nếu fixes.tsv không có mã đó: mọi TC của DEF có dòng cuối là P trong `smoke-*.md` (đếm bằng `env/qc2-count.py`) hoặc TC lỗi thời đã xoá có nguồn; (3) sha lấy từ fixes.tsv → `env/def-sha.tsv` → chữ trong mục DEF. Mục DEF viết lúc mới phát hiện (ví dụ «Trạng thái: MỞ» ở DEF-R105S-06) có thể cũ hơn: tệp này thắng.

## Tổng

| Trạng thái | Số mã |
|---|---|
| ĐÓNG | 204 |
| ĐÃ SỬA, CHỜ XÁC NHẬN | 0 |
| RÚT (không phải lỗi sản phẩm) | 8 |
| MỞ | 0 |
| Nợ PR | 0 |
| **Tổng** | **212** |

DEF-R10.8-05 (Back native Android) đã ĐÓNG 26/11: qc2 chạy lại AC7(a) trên ep-api36 — khung quét đóng khi Back, app không thoát. Nợ PR: không còn; DEF-R6-01 (vai DB siêu quyền) từng là nợ PR, đã sửa ở 3f332fb3 + ea0cf217 (migration 00030) và TC-P3-01-49 chạy lại P.

### Cách đóng

| Cách | Số mã |
|---|---|
| sửa mã/tài liệu (có sha) | 167 |
| chạy lại đạt (không ghi sha) | 25 |
| spec/TC đổi (P-xx) | 11 |
| không phải lỗi sản phẩm | 8 |

### Theo mức

| Mức | ĐÓNG | CHỜ XÁC NHẬN | RÚT | MỞ | Tổng |
|---|---|---|---|---|---|
| Cao | 10 | 0 | 0 | 0 | 10 |
| Trung bình | 43 | 0 | 0 | 0 | 43 |
| Thấp | 150 | 0 | 5 | 0 | 155 |
| không ghi | 0 | 0 | 3 | 0 | 3 |

### Theo «lọt sprint gốc / do sprint sau»
Lấy từ trường «Hồi quy do sprint sau?» hoặc chữ «Không phải hồi quy» / «hồi quy do sprint 11» trong mục DEF; mục không ghi thì xếp «không ghi» (dòng bảng của defects-9/10/10.6 chỉ có mức, TC và mô tả).

| Nhóm | ĐÓNG | CHỜ XÁC NHẬN | RÚT | MỞ | Tổng |
|---|---|---|---|---|---|
| lọt sprint gốc | 46 | 0 | 1 | 0 | 47 |
| do sprint sau | 24 | 0 | 0 | 0 | 24 |
| không rõ | 22 | 0 | 0 | 0 | 22 |
| không ghi | 110 | 1 | 7 | 0 | 118 |

### Theo tệp sprint

| Tệp | ĐÓNG | CHỜ XÁC NHẬN | RÚT | MỞ | Tổng |
|---|---|---|---|---|---|
| 1 | 1 | 0 | 0 | 0 | 1 |
| 1.5 | 2 | 0 | 0 | 0 | 2 |
| 2 | 4 | 0 | 0 | 0 | 4 |
| 3 | 7 | 0 | 1 | 0 | 8 |
| 4 | 12 | 0 | 5 | 0 | 17 |
| 5 | 23 | 0 | 0 | 0 | 23 |
| 5.5 | 1 | 0 | 0 | 0 | 1 |
| 6 | 23 | 0 | 0 | 0 | 23 |
| 7 | 12 | 0 | 0 | 0 | 12 |
| 8 | 7 | 0 | 0 | 0 | 7 |
| 9 | 10 | 0 | 1 | 0 | 11 |
| 9.1 | 6 | 0 | 0 | 0 | 6 |
| 9.5 | 26 | 0 | 0 | 0 | 26 |
| 10 | 24 | 0 | 0 | 0 | 24 |
| 10.5 | 7 | 0 | 1 | 0 | 8 |
| 10.6 | 27 | 0 | 0 | 0 | 27 |
| 10.7 | 4 | 0 | 0 | 0 | 4 |
| 10.8 | 5 | 1 | 0 | 0 | 6 |
| opencode-go | 1 | 0 | 0 | 0 | 1 |

## Từng mã

| Sprint | DEF | Mức | TC (dòng cuối) | Trạng thái | Cách đóng / sha | Gốc↔sau | Tiêu đề |
|---|---|---|---|---|---|---|---|
| 1 | DEF-R1-01 | Thấp | P0-01/TC-07 | ĐÓNG | sửa 752ce1f9 | do sprint sau | DEMO_SCRIPT nêu spec E2E không có trong FLOWS.md |
| 1.5 | DEF-R1.5-01 | Trung bình | TC-00-57, TC-00-59 | ĐÓNG | chạy lại đạt | do sprint sau | Đã đăng nhập mà mở đường dẫn không có thật bị đưa về /login thay vì màn «Không tìm thấy trang» |
| 1.5 | DEF-R1.5-02 | Thấp | TC-00-36 | ĐÓNG | chạy lại đạt | không rõ | TA: nút hồ sơ ở topbar tràn ngang ở 375 và 390 px |
| 2 | DEF-R2-01 | Trung bình | TC-PG06-01, P0-02/TC-21 | ĐÓNG | sửa 816498c6 | do sprint sau | openapi.yaml không qua redocly lint (1 lỗi struct) |
| 2 | DEF-R2-02 | Cao | TC-PG06-10, TC-PG06-11, TC-PG06-28=LT | ĐÓNG | chạy lại đạt | do sprint sau | go test ./internal/contract đỏ: TestStatusParity / TestSpec_EveryDocumentedStatusExercised |
| 2 | DEF-R2-03 | Trung bình | P0-02/TC-20, P0-03/TC-15 | ĐÓNG | sửa c272d3d3 | do sprint sau | go test ./internal/store đỏ: TestLLMSchema (6 bảng llm_% ≠ 9) — ĐÃ SỬA c272d3d3 |
| 2 | DEF-R2-04 | Trung bình | P0-02/TC-20, P0-03/TC-15 | ĐÓNG | sửa 562f5a45 | do sprint sau | go test ./internal/contract KHÔNG có -tags testroutes đỏ cố định |
| 3 | DEF-R3-01 | Thấp | TC-P105-44 | ĐÓNG | sửa ffb46656 | không rõ | Bảng «Mức dùng theo tác vụ» hiện chữ thô «EMBEDDING» |
| 3 | DEF-R3-02 | Thấp | TC-PU01-01=LT | ĐÓNG | sửa d6de360f | do sprint sau | docs/design/DESIGN_TOKENS.css lệch frontend/src/shared/styles/tokens.css ở 2 token |
| 3 | DEF-R3-03 | Thấp | TC-PU01-05, TC-GATEPU-01 | ĐÓNG | chạy lại đạt | do sprint sau | pnpm lint có 5 cảnh báo (nền sprint 3 = 0) |
| 3 | DEF-R3-04 | Thấp | TC-P101-24 | ĐÓNG | chạy lại đạt | lọt sprint gốc | DecryptKey( được gọi ngoài internal/llm/ và internal/llmconfig/ |
| 3 | DEF-R3-05 | Trung bình | TC-P102-14, TC-P103-19 | ĐÓNG | sửa c561c25b | không rõ | Làn BATCH bị gọi lại/chờ bất thường trên ep-r (TC-P102-14, TC-P103-19, TC-P103-20) |
| 3 | DEF-R3-SMK-01 |  | TC-P103-08 | RÚT | không phải lỗi sản phẩm (lỗi đo / đã loại) | lọt sprint gốc | RÚT LẠI (lỗi đo của QC, KHÔNG phải lỗi sản phẩm): «bucket RPM cho qua 50–65 lời gọi / 10 s» |
| 3 | DEF-R3-SMK-02 | Thấp | TC-PU05-35 | ĐÓNG | chạy lại đạt | lọt sprint gốc | Nhánh ci/ui-drift chưa được xoá (TC-PU05-35) |
| 3 | DEF-R3-SMK-03 | Trung bình | TC-PU05-40, TC-GATEPU-15 | ĐÓNG | sửa 07a6b69c | không ghi | Sinh viên thấy «Chi tiết kỹ thuật … trace_id» ở màn lỗi /me và /threads (TC-PU05-40, TC-GATEPU-… |
| 4 | DEF-Q2-DEV-01 | Thấp | — | ĐÓNG | sửa d9f244a5 | không ghi | pnpm dev với EP_PORT_OFFSET: seed không nhận cổng mock-imap / mock-graph (Thấp) |
| 4 | DEF-Q2-S4-01 | Thấp | TC-P212-14 | ĐÓNG | sửa f473c2ff | không ghi | Seed tự chạy trên DB trống mất 752 s, spec AC7 (US-P2-12) đòi ≤ 3 phút (TC-P212-14) |
| 4 | DEF-Q2-S4-02 | Thấp | TC-GATEP2-28 | RÚT | không phải lỗi sản phẩm: TC lệch spec hiện hành, sửa TC P-11-103 (9cac4c20), TC chạy lại P | không ghi | TC-GATEP2-28 (Thấp, không phải lỗi sản phẩm): đếm nav thật ở ep-r (21/11 @c12be038, 9 vai/tài k… |
| 4 | DEF-Q2-S4-03 | Thấp | TC-P202-44 | RÚT | không phải lỗi sản phẩm: TC lệch spec hiện hành, sửa TC P-11-103 (9cac4c20), TC chạy lại P | không ghi | TC-P202-44 (Thấp, không phải lỗi sản phẩm): smkt3-nav (21/11): phần hành vi đạt (chưa đăng nhập… |
| 4 | DEF-Q2-S4-04 | Thấp | TC-P207-14 | RÚT | không phải lỗi sản phẩm: TC lệch spec hiện hành, sửa TC P-11-103 (9cac4c20), TC chạy lại P | không ghi | TC-P207-14 (Thấp, không phải lỗi sản phẩm): gieo route qc-probe không guard → TestAllCourseRout… |
| 4 | DEF-Q2-S4-05 | Thấp | TC-P211-30 | RÚT | không phải lỗi sản phẩm: TC lệch spec hiện hành, sửa TC P-11-103 (9cac4c20), TC chạy lại P | không ghi | TC-P211-30 (Thấp, không phải lỗi sản phẩm): đo trên gateway phụ (2626 user/475 lớp): truy vấn/y… |
| 4 | DEF-Q2-S4-06 | Thấp | TC-P212-18 | RÚT | không phải lỗi sản phẩm: TC lệch spec hiện hành, sửa TC P-11-103 (9cac4c20), TC chạy lại P | không ghi | TC-P212-18 (Thấp, không phải lỗi sản phẩm): mail_outbox sau seed: 1 DEAD (seed_observability_fa… |
| 4 | DEF-R4-01 | Trung bình | TC-P208-02 | ĐÓNG | sửa migration-00026 | lọt sprint gốc | Mã lớp phân biệt hoa/thường: tạo được hai lớp SK20D91 và sk20d91 |
| 4 | DEF-R4-02 | Thấp | TC-P210-03 | ĐÓNG | sửa b6d20a67 | không rõ | Import roster nhận tệp .xlsx thực chất là văn bản (TC-P210-03) |
| 4 | DEF-R4-03 | Thấp | TC-P202-46 | ĐÓNG | spec/TC đổi P-11-72 | lọt sprint gốc | Thấp — TC-P202-46 (sprint gốc 4, US-P2-02 AC15) |
| 4 | DEF-R4-04 | Thấp | TC-P206-28 | ĐÓNG | sửa 3121bda8 | lọt sprint gốc | Thấp — TC-P206-28 (sprint gốc 4, US-P2-06 AC8) |
| 4 | DEF-R4-05 | Trung bình | TC-P201-11 | ĐÓNG | sửa f5f8a8be | lọt sprint gốc | Trung bình — TC-P201-11 (sprint gốc 4, US-P2-01 AC3; vi phạm US-P2-04 AC13) |
| 4 | DEF-R4-06 | Thấp | TC-P202-47 | ĐÓNG | spec/TC đổi P-11-73 | không ghi | Thấp — TC-P202-47 (sprint gốc 4, US-P2-02 AC15) |
| 4 | DEF-R4-07 | Thấp | TC-P201-19 | ĐÓNG | spec/TC đổi P-11-74 | lọt sprint gốc | Thấp — TC-P201-19 (sprint gốc 4, US-P2-01 AC6) |
| 4 | DEF-R4-08 | Thấp | TC-GATEP2-09=LT, TC-P212-14 | ĐÓNG | sửa 69b06337+3b82e1cd | lọt sprint gốc | Thấp — TC-GATEP2-09 / TC-P212-14 (sprint gốc 4, US-P2-12 AC7) |
| 4 | DEF-R4-09 | Thấp | TC-P211-44, TC-P212-27 | ĐÓNG | sửa 9754c523 | do sprint sau | Thấp — TC-P211-44 / TC-P212-27 (sprint gốc 4, US-P2-11 / US-P2-12; hồi quy do sprint 11) |
| 4 | DEF-R4-10 | Thấp | TC-P208-34 | ĐÓNG | sửa d1f278f4 | không ghi | Thấp — TC-P208-34 (sprint gốc 4, US-P2-08 AC11; biến thể «trang đã mở») |
| 5 | DEF-R5-01 | Thấp | TC-PE03-52 | ĐÓNG | sửa 6a78c48f+1c0700e2 | lọt sprint gốc | Nút đầu trang AppShell: axe label-content-name-mismatch (chỉ khi bật thẻ wcag21a) |
| 5 | DEF-R5-02 | Thấp | TC-PE06-08 | ĐÓNG | sửa d1374b09 | không rõ | Vùng cuộn ở màn làm bài code không focus được bằng bàn phím |
| 5 | DEF-R5-03 | Trung bình | TC-PE03-49 | ĐÓNG | sửa 46bf5626 | lọt sprint gốc | Ngân hàng câu hỏi: nháp tự lưu nhưng KHÔNG khôi phục khi mở lại (mất chữ gõ dở) |
| 5 | DEF-R5-04 | Trung bình | TC-PE03-51 | ĐÓNG | sửa 4cd9c07e | không rõ | Ngân hàng câu hỏi: Enter trên hàng mở Drawer rồi Drawer đóng ngay (bàn phím không dùng được) |
| 5 | DEF-R5-05 | Thấp | TC-PE01-39 | ĐÓNG | spec/TC đổi P-11-50 | do sprint sau | go test ./internal/contract/... không tag testroutes đỏ (lệnh trong TC tổng và scripts/gate-pe.… |
| 5 | DEF-R5-06 | Thấp | TC-PE06-27 | ĐÓNG | sửa 0e52f1e5 | lọt sprint gốc | Lần nộp code: nhãn «Đã nhận, đang xếp hàng» (QUEUED) không bao giờ hiện |
| 5 | DEF-R5-07 | Thấp | TC-PE07-18 | ĐÓNG | sửa d7fecdc2 | không ghi | Tín hiệu liêm chính cuối cùng bị mất khi đóng tab (không gửi lúc pagehide) |
| 5 | DEF-R5-08 | Thấp | TC-PE08-31 | ĐÓNG | sửa 473472ec | lọt sprint gốc | Màn kết quả SV thiếu dòng tóm tắt và từng câu không gập mặc định |
| 5 | DEF-R5-09 | Thấp | TC-PE08-38 | ĐÓNG | sửa 247bf4e4+42c1ed09 | không ghi | Giao diện Staff /exams/[id]/results lệch US AC10 ở 3 điểm |
| 5 | DEF-R5-10 | Thấp | TC-PE03-33 | ĐÓNG | sửa dd160463 | lọt sprint gốc | Sửa test sau đóng: bài cũ không hiện «đã chấm theo bộ test cũ» |
| 5 | DEF-R5-11 | Thấp | TC-PE08-04 | ĐÓNG | sửa 3f748ec2 | không rõ | Chấm trắc nghiệm đồng bộ không phát sự kiện SSE exam.attempt |
| 5 | DEF-R5-12 | Trung bình | TC-PU06-17, TC-UI7-06 | ĐÓNG | chạy lại đạt | không ghi | pnpm lint ở frontend rc=1 (TC-PU06-17, TC-UI7-06) |
| 5 | DEF-R5-13 | Trung bình | TC-PE04-50 | ĐÓNG | sửa 9754c523 | do sprint sau | ; cổng vẫn đỏ vì DEF-R5-18] golangci-lint run còn 1 vấn đề (gochecknoglobals) — cổng tổng đỏ (T… |
| 5 | DEF-R5-14 | Thấp | — | ĐÓNG | sửa d27b87b0 | lọt sprint gốc | Seed PE không có lượt nộp code TLE (TC-PE09-07) |
| 5 | DEF-R5-15 | Thấp | TC-PE09-12 | ĐÓNG | sửa d0d548e0 | không rõ | Seed lần hai không in «Seed xong (không đổi)» (TC-PE09-12) |
| 5 | DEF-R5-16 | Thấp | TC-PE09-13 | ĐÓNG | sửa f473c2ff | do sprint sau | ; đo lại qc2seedt2 @6b3678c4: seed 281 s ≤ 360 s; TC-PE09-13 P] Seed trên DB trống mất 652 s, t… |
| 5 | DEF-R5-17 | Thấp | TC-PE04-50, TC-PU06-18 | ĐÓNG | sửa 58448e0a | không rõ | AUDIT cut ≠ 0 ở bảng ảo hoá DataTable 390 px: TA/GV /students, /gradebook (TC-PE04-50…, TC-PU06… |
| 5 | DEF-R5-18 | Trung bình | TC-PE04-50 | ĐÓNG | sửa 39887d6e | do sprint sau | golangci-lint run đỏ lại: cmd/regress/main.go:213 flipRight biến toàn cục (cổng tổng TC-PE04-50… |
| 5 | DEF-R5-19 | Thấp | TC-PU06-11 | ĐÓNG | sửa e345ef3a | do sprint sau | /chat Sinh viên: CLS thật 0,16 > 0,1 (cột danh sách phiên biến mất làm cả cột chi tiết dịch) (T… |
| 5 | DEF-R5-20 | Thấp | TC-PE09-12 | ĐÓNG | sửa 581dbccd | không rõ | »] [chờ chạy lại 581dbccd (cần cửa sổ SEED)] Seed lần hai vẫn không in «Seed xong (không đổi)»:… |
| 5 | DEF-R5-21 | Thấp | TC-PE09-25 | ĐÓNG | sửa a0d98671 | không rõ | Kết quả sinh viên /exams/[id]/take: bảng «Kết quả test mẫu» làm trang tràn ngang 287 px ở 375, … |
| 5 | DEF-R5-22 | Thấp | TC-PE09-20 | ĐÓNG | sửa d0dcf4fe | lọt sprint gốc | exam.spec.ts chưa phủ ý (9) chat-gate và (11) rò đáp án quét mạng của AC7 (TC-PE09-20) |
| 5 | DEF-R5-23 | Thấp | TC-PE09-25 | ĐÓNG | sửa 612e8a23 | do sprint sau | «Mã của bạn» (<pre>) cuộn được nhưng không focus được bằng bàn phím (axe serious scrollable-reg… |
| 5.5 | DEF-R5.5-01 | Trung bình | TC-UI7-06 | ĐÓNG | chạy lại đạt | không ghi | pnpm lint frontend rc=1 (TC-UI7-06) |
| 6 | DEF-R6-01 | Thấp | TC-P3-01-49 | ĐÓNG | sửa 3f332fb3,ea0cf217 | lọt sprint gốc | — PM 2026-11-21: không giao sửa lúc này] Vai DB của ứng dụng là siêu quyền |
| 6 | DEF-R6-02 | Cao | TC-P3-02-21 | ĐÓNG | sửa afbd433e | không rõ | Bộ nhớ đệm roster PII không bị vô hiệu khi sinh viên vào lớp (duyệt hoặc vào bằng mã) |
| 6 | DEF-R6-C1 | Cao | TC-P8-03-11, TC-P8-03-11b=LT | ĐÓNG | sửa eb8f16a0 | lọt sprint gốc | Sửa/xoá sự kiện từ giao diện lịch trả 404, DB không đổi |
| 6 | DEF-R6-C-01 | Thấp | TC-P3-06-42 | ĐÓNG | sửa a081590a | lọt sprint gốc | Dòng khoá ở /threads dùng chữ của chat thay vì "Đăng bài tạm khóa…" |
| 6 | DEF-R6-D1 | Trung bình | TC-P3-05-42, TC-P8-02-20 | ĐÓNG | sửa ec157a06 | không rõ | Trích dẫn cũ vẫn hiện tên tài liệu đã bị xoá, không có «Nguồn đã gỡ» |
| 6 | DEF-R6-D2 | Cao | TC-P3-05-42, TC-P8-02-20 | ĐÓNG | sửa 210f72d6 | do sprint sau | GET /chat/sessions/{id}/messages 500 với mọi phiên có trích dẫn (hồi quy do ec157a06, DEF-R6-D1) |
| 6 | DEF-R6-P1 | Thấp | TC-P8-01-18 | ĐÓNG | spec/TC đổi P-11-79 | không ghi | Bộ chia đoạn không gộp mẩu < 200 rune qua ranh giới trang |
| 6 | DEF-R6-P2 | Trung bình | TC-P8-01-37 | ĐÓNG | sửa 506ae3a0 | không ghi | Chia sẻ tài liệu không tăng ep:rag:ver của lớp đích |
| 6 | DEF-R6-P3 | Thấp | TC-P8-02-24 | ĐÓNG | sửa 32022e99 | không ghi | snippet của /library?q= không chứa từ khoá |
| 6 | DEF-R6-P4 | Trung bình | TC-P8-02-33 | ĐÓNG | sửa 545bbdd4 | không ghi | search_library không được nối vào agent Threads |
| 6 | DEF-R6-T1 | Thấp | TC-P8-03-51 | ĐÓNG | chạy lại đạt | không ghi | Seed bước 11 (tài liệu docling thật) vượt 4 phút trên DB trống |
| 6 | DEF-R6-U1 | Thấp | TC-P3-05-35 | ĐÓNG | sửa 812ab974 | không rõ | Chữ «Tìm hiểu» của chat riêng khác SRS |
| 6 | DEF-R6-U2 | Thấp | TC-P3-06-09 | ĐÓNG | sửa 812ab974 | không ghi | Tiêu đề hộp thoại PII ở Threads khác SRS |
| 6 | DEF-R6-U3 | Thấp | TC-P3-06-34 | ĐÓNG | sửa 812ab974 | không ghi | Nút đăng lại khi mất mạng ở Threads khác SRS |
| 6 | DEF-R6-U4 | Thấp | TC-P3-06-37 | ĐÓNG | spec/TC đổi P-11-80 | không ghi | Trạng thái rỗng /threads thiếu câu mời |
| 6 | DEF-R6-U5 | Thấp | TC-P3-02-50 | ĐÓNG | sửa 812ab974 | không ghi | Chữ lỗi MASK_FAILED của chat khác SRS |
| 6 | DEF-R6-U6 | Thấp | TC-P3-05-26 | ĐÓNG | sửa 812ab974 | không ghi | Chữ CHAT_UNAVAILABLE ở UI khác SRS |
| 6 | DEF-R6-U7 | Thấp | TC-P8-02-09 | ĐÓNG | sửa 1f9738d4 | không ghi | Hàng ANSWER_KEY ở /documents không ghi hai dòng cố định |
| 6 | DEF-R6-U8 | Thấp | TC-P8-02-07 | ĐÓNG | sửa 1f9738d4 | không ghi | Chữ lỗi chọn tệp sai loại/quá cỡ khác SRS |
| 6 | DEF-R6-U9 | Thấp | TC-P8-02-22 | ĐÓNG | sửa 5fbb9c8f | không ghi | ) Thư viện của SV thiếu bộ lọc «Chủ đề» |
| 6 | DEF-R6-U10 | Thấp | TC-P8-02-19 | ĐÓNG | sửa 6e362a9b | không ghi | ) Hộp xác nhận xoá tài liệu: «{m} lớp» bằng 0 với tài liệu chưa chia sẻ |
| 6 | DEF-R6-U11 | Thấp | TC-P8-01-17 | ĐÓNG | sửa a79f5c69 | không ghi | Danh sách /documents không tự cập nhật khi tệp được tải lên từ nơi khác |
| 6 | DEF-R6-U12 | Thấp | TC-P8-01-43 | ĐÓNG | spec/TC đổi P-11-94 | không ghi | — không phải lỗi mã; TC-P8-01-43 chấm lại P ngày 2026-11-22] Docling tắt: tài liệu ra FAILED sa… |
| 7 | DEF-R7-01 | Trung bình | TC-20 | ĐÓNG | sửa 842f431c | do sprint sau | CI chạy đủ job vượt ngưỡng cứng 10 phút của US-CI-02 |
| 7 | DEF-R7-02 | Trung bình | TC-P4-02-14 | ĐÓNG | sửa e4b82a05 | lọt sprint gốc | Chuông gộp TICKET_NEW không báo ticket mới sau khi hàng chờ về 0 |
| 7 | DEF-R7-04 | Thấp | TC-P4-03-36 | ĐÓNG | sửa 24036aae | không ghi | Ctrl+↵ gửi trả lời không sang phiếu kế khi bộ lọc là «Đang chờ» |
| 7 | DEF-R7-05 | Thấp | TC-P4-03-33 | ĐÓNG | sửa 24036aae | lọt sprint gốc | Ô trả lời của hộp thư không «dính đáy» ở 375 px |
| 7 | DEF-R7-06 | Thấp | TC-E2 | ĐÓNG | sửa 7062b009 | không ghi | Timeout của job CI không đạt «≥ 2× thời gian thực tế»; changes và judge-attacks không có timeou… |
| 7 | DEF-R7-07 | Thấp | TC-E4 | ĐÓNG | sửa 7062b009 | do sprint sau | Shard Playwright rỗng chạy lại CẢ BỘ test thay vì 0 test |
| 7 | DEF-R7-08 | Trung bình | TC-P5-02-33 | ĐÓNG | sửa 131a5868 | không ghi | Nút «Lưu điểm danh» bị khoá khi mất mạng, không xếp hàng complete:true |
| 7 | DEF-R7-09 | Trung bình | TC-P5-03-29 | ĐÓNG | chạy lại đạt | không ghi | Nhịp heartbeat khi Redis dừng mất ~8 giây (SRS: ≤ 1 s) |
| 7 | DEF-R7-10 | Thấp | TC-P5-01-26 | ĐÓNG | sửa 01b3f6d9 | không ghi | «Xem trước» tạo lịch buổi học gửi Idempotency-Key |
| 7 | DEF-R7-11 | Thấp | TC-P4-03-38 | ĐÓNG | sửa debd2d05 | không ghi | Trạng thái rỗng của Hộp thư hỗ trợ không đúng chữ / thiếu nút «Xem tất cả» |
| 7 | DEF-R7-12 | Thấp | TC-P4-06-47 | ĐÓNG | sửa 1995e59b | không ghi | Menu bài AI của giảng viên ghi «Loại», không phải «Loại khỏi tri thức» |
| 7 | DEF-R7-13 | Thấp | TC-P5-06-18 | ĐÓNG | spec/TC đổi P-11-82 | không ghi | Trạng thái rỗng khối Chuyên cần của /me thiếu dấu chấm cuối |
| 8 | DEF-R8-A1 | Thấp | TC-P6-03-36, TC-P6-03-19 | ĐÓNG | sửa 6c75bd54 | không ghi | Trích công thức: đầu ra không phải JSON bị thử lại ×3 và báo «AI đang không dùng được» thay vì … |
| 8 | DEF-R8-A2 | Thấp | TC-P6-04-03, TC-P6-07-47 | ĐÓNG | sửa ec7f29da | không ghi | Đóng Drawer / Dialog làm focus rơi về <body> thay vì nút mở (TC-P6-04-03, TC-P6-07-47) |
| 8 | DEF-R8-A3 | Trung bình | TC-P6-03-19 | ĐÓNG | sửa 7d2de170 | không ghi | Trích công thức: trọng số ngoài [0;100] bị đổi thầm thành 0, không có mục PARAM_INVALID (TC-P6-… |
| 8 | DEF-R8-A4 | Thấp | TC-P6-05-37 | ĐÓNG | sửa 22387b06 | không ghi | Chat riêng: «Cô tính điểm môn này thế nào?» không được nhận là GRADE_FORMULA (TC-P6-05-37) |
| 8 | DEF-R8-A5 | Thấp | TC-P6-05-42 | ĐÓNG | sửa 2479feb9 | không ghi | Xoá tài liệu nguồn của công thức đã xác nhận: UI không hiện «Tài liệu nguồn đã gỡ» (TC-P6-05-42) |
| 8 | DEF-R8-s8core-01 | Trung bình | TC-P6-01-20, TC-P6-02-25 | ĐÓNG | sửa d3ac89fc | không ghi | Giải trình không có dòng «không dự → 0 theo quy chế» |
| 8 | DEF-R8-s8core-02 | Thấp | TC-P6-02-15 | ĐÓNG | sửa 1dea359f | không ghi | float64 chuyển điểm thi trong seed tool |
| 9 | DEF-R9-01 | Thấp | TC-P7-01-G06, TC-P7-02-G10, TC-P7-03-G03 +3 | ĐÓNG | sửa afbc1fb0 | lọt sprint gốc | Tên test ở dòng Kiểm: của US không có trong repo (hoặc ở gói khác) |
| 9 | DEF-R9-02 |  | — | RÚT | không phải lỗi sản phẩm (lỗi đo / đã loại) | không ghi | ĐÃ LOẠI: go test ./internal/contract/... đỏ lúc tải cao |
| 9 | DEF-R9S-01 | Thấp | TC-P7-09-M6 | ĐÓNG | sửa 80d24cd7 | không ghi | Đột biến bỏ AND status='REVIEWED' ở UPDATE công bố (store/review.sql.go PublishMark) sống: toàn… |
| 9 | DEF-R9S-02 | Thấp | TC-P7-06-15, TC-P7-07-06 | ĐÓNG | sửa 6abb09af | không ghi | Hàng chờ /grading bài FAILED chỉ hiện «Chấm lỗi», không nêu nguyên nhân bằng chữ; chữ «Hết ngân… |
| 9 | DEF-R9S-03 | Thấp | TC-P7-07-M4 | ĐÓNG | sửa 186bfe7b | không ghi | Đột biến XADD trước COMMIT (Signal gọi trong enqueue trước tx.Commit) sống: nhóm test TestGrade… |
| 9 | DEF-R9S-04 | Thấp | TC-P7-07-01 | ĐÓNG | sửa 7ac1a91f | không ghi | POST …/assignments/{aid}/grade-all khi Redis dừng trả 503 SERVICE_UNAVAILABLE và không tạo dòng… |
| 9 | DEF-R9S-05 | Thấp | TC-P7-08-04 | ĐÓNG | sửa 6984efab | không ghi | Màn duyệt /grading/{id} 1440 px: cột trái 576 px / cột phải 576 px (49 % / 49 %), TC yêu cầu 52… |
| 9 | DEF-R9S-06 | Thấp | TC-P7-07-13 | ĐÓNG | sửa ed1f24c5 | không ghi | Không có /metrics; grading_jobs_total, grading_dead_total, grading_duration_seconds, grading_qu… |
| 9 | DEF-R9S-07 | Thấp | TC-P7-08-13 | ĐÓNG | sửa 5aad539e | không ghi | /grading của lớp chưa có bài nộp: trạng thái rỗng là «Không có bài chưa duyệt» (theo bộ lọc mặc… |
| 9 | DEF-R9S-08 | Thấp | TC-P7-02-02 | ĐÓNG | sửa 95f3f025 | không ghi | Lớp mới chưa có bài tập và chưa có việc chờ: /grading vẫn mở tab «Hàng chờ», TC yêu cầu mặc địn… |
| 9 | DEF-R9S-09 | Thấp | TC-P7-03-02 | ĐÓNG | sửa 843a6f4a | không ghi | Trạng thái bài nộp phía sinh viên lệch chữ TC: sau nộp hiện «Đã nhận bài lúc …» thay vì «Đã nộp… |
| 9.1 | DEF-R9.1-01 | Thấp | TC-UX04-3, TC-UX08-5 | ĐÓNG | sửa 7322ac62 | do sprint sau | luật :hover ngoài @media (hover: hover) and (pointer: fine) (TC-UX04-3, TC-UX08-5) |
| 9.1 | DEF-R9.1-02 | Thấp | TC-UX07-8 | ĐÓNG | sửa 25cd3352 | do sprint sau | chuỗi «Kiểm duyệt Threads» hiển thị (TC-UX07-8) |
| 9.1 | DEF-R9.1-03 | Thấp | TC-UX08-50 | ĐÓNG | sửa 517e73ab | do sprint sau | axe color-contrast trên / ngay sau tải (TC-UX08-50) |
| 9.1 | DEF-R9.1-06 | Thấp | TC-UX12-2 | ĐÓNG | sửa 6d127b88 | lọt sprint gốc | chấm đã chọn (Muộn / Vắng phép / Vắng) xám khi chuột đang rê lên ô (TC-UX12-2) |
| 9.1 | DEF-R9.1-07 | Thấp | TC-UX16-14 | ĐÓNG | spec/TC đổi P-11-75 | không ghi | nháy lưu ô sổ điểm khi reduce: giữ ≈ 160 ms thay vì ≥ 300 ms (TC-UX16-14; AC mâu thuẫn) |
| 9.1 | DEF-R9.1-08 | Thấp | TC-UX09-16 | ĐÓNG | sửa e0ac8dc9 | không ghi | scripts/landing-video/record.mjs không chạy được ở HEAD (TC-UX09-16, Thấp) |
| 9.5 | DEF-R9.5-01 | Thấp | TC-N01-17 | ĐÓNG | sửa 6ec47f79 | lọt sprint gốc | refuseDemo bỏ lọt [::1] và host.docker.internal |
| 9.5 | DEF-R9.5-02 | Thấp | TC-N01-18 | ĐÓNG | sửa b7d1e713 | không rõ | canary-scan.sh không xoá nhà cung cấp canary |
| 9.5 | DEF-R9.5-03 | Thấp | TC-N04-12 | ĐÓNG | chạy lại đạt | lọt sprint gốc | load-a/c không có mẫu docker stats / ghi «không OOM, CPU < 95 %» |
| 9.5 | DEF-R9.5-04 | Thấp | TC-N05-03=LT | ĐÓNG | chạy lại đạt | lọt sprint gốc | Không có 00018_indexes_tuning.sql, đề xuất P-95-8 (dev) chưa có quyết định PM |
| 9.5 | DEF-R9.5-05 | Trung bình | TC-N10-10 | ĐÓNG | chạy lại đạt | lọt sprint gốc | 9.5 sửa test có sẵn |
| 9.5 | DEF-R9.5-06 | Thấp | TC-N10-06 | ĐÓNG | sửa 84e29eec | lọt sprint gốc | PROGRESS.md thiếu dòng Nợ cho TL-2c/d/e/f |
| 9.5 | DEF-R9.5-07 | Thấp | TC-N09-11, TC-N04-15 | ĐÓNG | chạy lại đạt | lọt sprint gốc | Báo cáo cuối không nêu hạn chế Mac ở từng mục |
| 9.5 | DEF-R9.5-08 | Thấp | TC-N07-10 | ĐÓNG | chạy lại đạt | lọt sprint gốc | Không ghi lần đo bị nhiễu |
| 9.5 | DEF-R9.5-09 | Trung bình | TC-N08-01 | ĐÓNG | chạy lại đạt | lọt sprint gốc | meta.config của tệp real-llm thiếu chuỗi model và RPM |
| 9.5 | DEF-R9.5-10 | Trung bình | TC-N08-06 | ĐÓNG | chạy lại đạt | lọt sprint gốc | R3 (mask) thiếu tỷ lệ placeholder và số lần mô hình đổi placeholder |
| 9.5 | DEF-R9.5-11 | Trung bình | TC-N08-11 | ĐÓNG | sửa 3600257a+e6f89f6c | lọt sprint gốc | R5 không đủ mẫu / AI Threads không có đầu ra |
| 9.5 | DEF-R9.5-12 | Cao | TC-N08-16, TC-N08-17 | ĐÓNG | sửa 965ca248 | lọt sprint gốc | real_llm.py không có --resume, không dừng sớm khi hết hạn mức |
| 9.5 | DEF-R9.5-13 | Thấp | TC-N09-01 | ĐÓNG | chạy lại đạt | lọt sprint gốc | Báo cáo research thiếu mục NFR-02/03 và «6 phần» |
| 9.5 | DEF-R9.5-14 | Cao | TC-N09-04, TC-N09-05, TC-N09-13 | ĐÓNG | sửa 4bfe444c | lọt sprint gốc | Báo cáo research lệch tệp nguồn sau khi đo lại qua 9router |
| 9.5 | DEF-R9.5-15 | Thấp | TC-N09-07 | ĐÓNG | chạy lại đạt | lọt sprint gốc | «Câu cần vá» không có tệp:dòng |
| 9.5 | DEF-R9.5-16 | Thấp | TC-N01-12 | ĐÓNG | sửa 6ec47f79 | lọt sprint gốc | check-meta.mjs không kiểm giá trị demo_stack |
| 9.5 | DEF-R9.5-17 | Trung bình | TC-N02-02, TC-N02-03 | ĐÓNG | sửa 6ec47f79 | không rõ | Seed không chạy lại được sau khi dữ liệu bị sửa; check-*-seed đếm tuyệt đối |
| 9.5 | DEF-R9.5-18 | Thấp | TC-N02-06 | ĐÓNG | chạy lại đạt | không rõ | Hôm nay của Admin trống ngay sau seed |
| 9.5 | DEF-R9.5-19 | Thấp | TC-N02-09, TC-N02-11 | ĐÓNG | sửa b7d1e713 | lọt sprint gốc | Seed không từ chối khi thiếu hoặc rỗng SEED_DEFAULT_PASSWORD |
| 9.5 | DEF-R9.5-20 | Trung bình | TC-N06-04 | ĐÓNG | sửa 98d4373d | lọt sprint gốc | Redis tắt: chat treo 39,5 s (TL-2d còn mở) |
| 9.5 | DEF-R9.5-21 | Trung bình | TC-N08-17 | ĐÓNG | sửa 0578695c | lọt sprint gốc | real_llm.py chết bằng Traceback khi server đóng kết nối; --resume chỉ ở mức lệnh |
| 9.5 | DEF-R9.5-22 | Thấp | TC-N01-03 | ĐÓNG | sửa 239df7fc | không ghi | README «Đo trên Mac» thiếu --email E --name N ở gateway admin create (TC-N01-03, Thấp) |
| 9.5 | DEF-R9.5-23 | Thấp | TC-N05-02 | ĐÓNG | chạy lại đạt | không ghi | 10 truy vấn «nặng nhất» của dev không khớp top theo total_exec_time (TC-N05-02, Thấp) |
| 9.5 | DEF-R9.5-24 | Thấp | TC-N09-04 | ĐÓNG | sửa 84fe3187 | lọt sprint gốc | Báo cáo ghi lượt 235/1.000 là «Gemini trực tiếp» nhưng meta.config là nhà cung cấp giả trễ 20 s… |
| 9.5 | DEF-R9.5-25 | Thấp | TC-N07-02 | ĐÓNG | sửa c17e5c28 | lọt sprint gốc | Bảng RAM nghỉ của dev không ghi quy mô dữ liệu; TC-N07-02 đo ở T0 lệch −51 % (Thấp) |
| 9.5 | DEF-R9.5-26 | Thấp | TC-N02-02, TC-N02-03 | ĐÓNG | sửa 9c1336ce,674f9779,6cb27001 | không rõ | seed.mjs bước 3 không chạy lại được khi DB có hơn 100 lớp (TC-N02-02/03 chạy lại, Thấp) |
| 10 | DEF-R10-01 | Trung bình | TC-12-14 | ĐÓNG | sửa 3e0f383d | không rõ | go test ./internal/contract không xanh (TC-12-14) |
| 10 | DEF-R10S-01 | Trung bình | TC-05-04 | ĐÓNG | sửa 186bfe7b | không ghi | Ma trận ô thiếu: BuildMatrix (matrix.go) chỉ ghi notes khi cả ngân hàng hết câu; bù từ chủ đề k… |
| 10 | DEF-R10S-02 | Thấp | TC-10-16 | ĐÓNG | sửa d8152695 | không ghi | AC8: question_id lớp khác trong PUT answers phải 422 INVALID_QUESTION; thực tế option của câu l… |
| 10 | DEF-R10S-03 | Thấp | TC-12-16 | ĐÓNG | sửa 7a37b68c | không ghi | docs/PROGRESS.md: F12 ☐, P9 «Dev xong, chờ QC» chưa cập nhật (PM). |
| 10 | DEF-R10S-04 | Thấp | TC-06-07 | ĐÓNG | sửa 186bfe7b | không ghi | Lượt TOPIC ở ≥1024 px không có «Danh sách câu» (chỉ MockRun dựng aside). AC3 nói danh sách số c… |
| 10 | DEF-R10S-05 | Trung bình | TC-06-10 | ĐÓNG | sửa 186bfe7b | không ghi | TOPIC mất mạng khi «Kiểm tra»: thông báo «Không kết nối được tới máy chủ. Chữ bạn đã nhập vẫn đ… |
| 10 | DEF-R10S-06 | Thấp | TC-06-21 | ĐÓNG | sửa 186bfe7b | không ghi | Tải lại giữa lượt TOPIC: về đúng câu 6 nhưng lựa chọn chưa gửi ở câu 6 mất (nút Kiểm tra tắt) —… |
| 10 | DEF-R10S-07 | Trung bình | TC-06-24 | ĐÓNG | sửa 186bfe7b | không ghi | 375 px lượt TOPIC: không tràn, nút ≥44 px, nhưng thanh «Kiểm tra / Câu tiếp theo» không sticky/… |
| 10 | DEF-R10S-08 | Thấp | TC-07-09 | ĐÓNG | spec/TC đổi P-11-62 | không ghi | Hàng chờ ngoại tuyến MOCK lưu ở localStorage (ep:draft:…:practice:…), IndexedDB 0 mục; US AC4 g… |
| 10 | DEF-R10S-09 | Thấp | TC-11-13 | ĐÓNG | sửa 55664e51 | không ghi | US-P9-11 AC8 / Q-P9-7: tệp Forms gốc lưu tối đa 90 ngày — không có hạn: bucket MinIO không có l… |
| 10 | DEF-R10S-10 | Thấp | TC-07-21 | ĐÓNG | sửa 4cbc7dcd | không ghi | Mất mạng khi bấm Nộp bài (thi thử): chữ «Chưa gửi được. Bài vẫn giữ trên máy bạn.» thay vì «Chư… |
| 10 | DEF-R10S-11 | Thấp | TC-09-19 | ĐÓNG | sửa 4713d46e | không ghi | /practice/history SV chưa có lượt: thiếu nút «Luyện ngay» (US-P9-09 AC; TC-09-19). |
| 10 | DEF-R10S-12 | Thấp | TC-02-14, TC-03-08 | ĐÓNG | sửa e42d8b49 | không ghi | Chưa có bằng chứng nào cho «OVERLOADED → retry đúng 3 lần → dead-letter → «Chưa trích được. Thử… |
| 10 | DEF-R10S-13 | Thấp | TC-10-09, TC-12-10 | ĐÓNG | sửa 44991583 | không ghi | quiz-integrity.spec.ts và practice.spec.ts chạy bằng mock (route.fulfill), không phủ các bước T… |
| 10 | DEF-R10S-14 | Cao | TC-12-11 | ĐÓNG | sửa 86aa1066 | không ghi | Thi thử (MOCK) 20 câu: tải lại trang giữa chừng (ở câu 8) làm MẤT 7/12 đáp án chọn sau đó (prac… |
| 10 | DEF-R10S-15 | Trung bình | TC-02-20 | ĐÓNG | sửa 40462c50 | không ghi | extract tài liệu > 60 trang: mã xếp việc (202) rồi cắt 60 trang + skipped PAGE_LIMIT; SRS FR-28… |
| 10 | DEF-R10S-16 | Thấp | TC-03-09 | ĐÓNG | sửa 424b1f77 | không ghi | Dedupe câu sinh dùng quiz.NormalizeShort (NFC, gộp khoảng trắng, chữ thường, bỏ dấu câu cuối) —… |
| 10 | DEF-R10S-17 | Thấp | TC-03-05 | ĐÓNG | sửa 161cf550 | không ghi | Câu sinh từ tài liệu visible_to_students=false, SV sau khi trả lời thấy «Nguồn: trang 1» (không… |
| 10 | DEF-R10S-18 | Thấp | TC-12-13 | ĐÓNG | sửa df07acca | không ghi | /questions (GV) tiêu đề «N câu chờ duyệt» đếm theo hàng đã tải (QuestionBank.tsx:124 rows.filte… |
| 10 | DEF-R10S-19 | Thấp | TC-07-25 | ĐÓNG | sửa 10749849 | không ghi | benchmarks/load/practice-mock.js gửi {answer:{text:'x'}} cho câu không có phương án, gồm cả TRU… |
| 10 | DEF-R10S-20 | Cao | TC-12-17 | ĐÓNG | sửa 154fc59b | không ghi | Hiệu năng tải: LCP (Lighthouse devtools) /practice 4.086 ms và /practice/[id] (lượt TOPIC đang … |
| 10 | DEF-R10S-21 | Thấp | TC-02-18 | ĐÓNG | chạy lại đạt | không ghi | Chat riêng 1 QPS: TTFT p95 nền 21,6–44,6 ms (3 lượt) → 197,4 ms khi một việc extract đề 60 tran… |
| 10 | DEF-R10S-22 | Thấp | TC-09-05, TC-12-15 | ĐÓNG | sửa 21b9b87b | không ghi | /practice/history: khối chủ đề yếu hiện tiêu đề «Chủ đề bạn hay sai» (PracticeHistory.tsx:33), … |
| 10 | DEF-R10S-23 | Cao | — | ĐÓNG | sửa 28d417ad | không ghi | docker compose build frontend @21b9b87b (và mọi sha từ deac438b) LỖI: pnpm -C frontend build → … |
| 10.5 | DEF-R10.5-01 | Thấp | TC-D07-02 | ĐÓNG | chạy lại đạt | do sprint sau | Bảng «Quyết định nhanh» ở /observability nằm trên bảng «7 ngày qua» |
| 10.5 | DEF-R105S-01 | Trung bình | TC-D08-06 | ĐÓNG | sửa ca0aa7df | không ghi | cascade.spec.ts không phủ các nhánh (a)–(f) của US-DM-08 AC3 |
| 10.5 | DEF-R105S-02 | Thấp | TC-D08-14 | ĐÓNG | sửa af07726d | không ghi | benchmarks/decision/run.py không có đường «lượt thật» (cột «thật», cờ xác nhận, trần 300) |
| 10.5 | DEF-R105S-03 | Trung bình | TC-D02-10, TC-D01-25 | ĐÓNG | sửa f26b15db | không ghi | Provider «Quyết định» tắt nhưng cổng chat / chèn lệnh vẫn gọi decision |
| 10.5 | DEF-R105S-04 | Thấp | TC-D04-07, TC-D09-05 | ĐÓNG | sửa 1fdcda73 | không ghi | Thân gửi decision của cổng INJECTION_FLAG dài 16.001 ký tự (> 16.000) |
| 10.5 | DEF-R105S-05 |  | — | RÚT | không phải lỗi sản phẩm (môi trường QC); chốt chặn 49ebd12a | không ghi | RÚT LẠI (lỗi môi trường của QC, không phải lỗi mã) |
| 10.5 | DEF-R105S-06 | Thấp | TC-D06-22 | ĐÓNG | sửa 9764a9d6 | không ghi | /settings/llm: 8 chữ đỏ cùng lúc, liên kết «Cài đặt nâng cao» đỏ ở bảng tác vụ nhưng xám ở khối… |
| 10.5 | DEF-R105S-07 | Thấp | TC-D08-13 | ĐÓNG | sửa 9764a9d6 | không ghi | benchmarks/decision/run.py (dòng ~350 và mục «Giới hạn») luôn in «provider decision GIẢ (p50 42… |
| 10.6 | DEF-R10.6-01 | Thấp | TC-G03-24, TC-G05-24 | ĐÓNG | chạy lại đạt | lọt sprint gốc | reflect / regrade-stale qua lớp có cờ tắt trả 409 GUIDANCE_DISABLED trước 404 |
| 10.6 | DEF-R106S-01 | Thấp | TC-G01-17 | ĐÓNG | sửa a4f0a8b3 | không ghi | 409 khi lưu công tắc nhưng hàng không tải lại (tab 2 vẫn «Tắt» khi DB bật) |
| 10.6 | DEF-R106S-02 | Trung bình | TC-G03-14 | ĐÓNG | sửa ab89d82b | không ghi | «Rút lưu ý» FAILED, bấm lại trong 1 giờ trả lại đúng job FAILED dù Idempotency-Key khác (khoá R… |
| 10.6 | DEF-R106S-03 | Thấp | TC-G03-03 | ĐÓNG | sửa ff09501f | không ghi | reflect với bài tập QUIZ trả 404 ASSIGNMENT_NOT_FOUND |
| 10.6 | DEF-R106S-04 | Thấp | TC-G04-01 | ĐÓNG | sửa 87073119 | không ghi | Khối «Lưu ý chấm» thiếu «M chờ duyệt» |
| 10.6 | DEF-R106S-05 | Thấp | TC-G04-02 | ĐÓNG | sửa c3632cdc | không ghi | «Đang dùng · vN» thay «Đã áp dụng», «bài đổi điểm» thay «lần sửa», mục «Đã nghỉ» không hiện |
| 10.6 | DEF-R106S-06 | Thấp | TC-G04-15 | ĐÓNG | sửa 81e58ce6 | không ghi | Không EmptyState; GET 500 làm khối biến mất không «Thử lại» |
| 10.6 | DEF-R106S-07 | Thấp | TC-G04-02 | ĐÓNG | sửa 1595bfa1 | không ghi | Chữ chiều lưu ý: UI «Chấm chặt hơn / Chấm rộng hơn / Làm rõ tiêu chí» (guidanceApi.ts:85) ≠ US-… |
| 10.6 | DEF-R106S-08 | Thấp | TC-G05-17 | ĐÓNG | sửa 18a81f40 | không ghi | Công bố lô có bài miễn: hộp xác nhận chỉ hiện «8 bài đã duyệt chấm theo bản lưu ý cũ, vẫn công … |
| 10.6 | DEF-R106S-09 | Thấp | TC-G05-12 | ĐÓNG | sửa 18a81f40 | không ghi | «Chấm lại» hàng loạt: bấm → khối «N bài chấm theo bản cũ · Chấm lại» biến mất ngay (đếm stale l… |
| 10.6 | DEF-R106S-10 | Thấp | TC-G06-06 | ĐÓNG | sửa d782121d | không ghi | /grading/[submissionId] kết quả v1: một nút «bản v1 · 3 lưu ý» trên thanh công cụ (GuidanceMeta… |
| 10.6 | DEF-R106S-11 | Thấp | TC-G06-11 | ĐÓNG | sửa 18a81f40 | không ghi | Chặn GET …/guidance/stats trả 500: khối «Lưu ý chấm» vẫn hiện «Bản v1 (đang dùng) · chưa đủ bài… |
| 10.6 | DEF-R106S-12 | Thấp | TC-G07-07 | ĐÓNG | sửa 18a81f40 | không ghi | cmd/regress run chỉ nhận --course, báo cáo không có «số tốt lên» (US-GD-07 AC3: --assignment <i… |
| 10.6 | DEF-R106S-13 | Thấp | TC-G07-08 | ĐÓNG | sửa 18a81f40 | không ghi | regress run không kiểm ADMIN / môi trường đo (chỉ export có cổng REGRESSION_EXPORT_ALLOWED) |
| 10.6 | DEF-R106S-14 | Thấp | TC-G07-09 | ĐÓNG | sửa 18a81f40 | không ghi | regressed() chỉ xét MAE và \ |
| 10.6 | DEF-R106S-15 | Thấp | TC-G07-10 | ĐÓNG | sửa 18a81f40 | không ghi | Thiếu checklist PR ghi lệnh regress run (guide có, CI đúng không chạy) |
| 10.6 | DEF-R106S-16 | Thấp | TC-G08-07 | ĐÓNG | sửa 5c160107 | không ghi | Gợi ý khi soạn thread: debounce thực ≈ 800 ms (usePostGate.tsx, precheck 783–801 ms rồi similar… |
| 10.6 | DEF-R106S-17 | Trung bình | TC-G09-09 | ĐÓNG | sửa 5c160107 | không ghi | GET /courses/{id}/questions/gen-stats?job_id=… không bao giờ trả hints (US-GD-09 AC6): Practice… |
| 10.6 | DEF-R106S-18 | Thấp | TC-P11-07 | ĐÓNG | sửa 37084f11 | không ghi | privacy.Detect 1.000 tên roster × 5.000 ký tự: bản trước P-106-11 (4b664dcb) 3,2–3,5 ms/lượt; H… |
| 10.6 | DEF-R106S-19 | Trung bình | TC-P12-25 | ĐÓNG | sửa 058f25bb | không ghi | Sau 2733c9b3 (R106S-17): go test -tags testroutes ./internal/contract/... ĐỎ ở 5c160107: TestCh… |
| 10.6 | DEF-R106S-20 | Thấp | TC-UX-d | ĐÓNG | sửa 37084f11 | không ghi | /grading?tab=queue: bộ lọc «Cần xem kỹ» tự bật lại mỗi lần vào hàng chờ: bật → tắt → rời sang t… |
| 10.6 | DEF-R106S-21 | Thấp | TC-UX-o | ĐÓNG | sửa 37084f11 | không ghi | Hàng chờ không tự cập nhật khi người khác đổi: A giữ /grading?tab=queue&assignment=…&status=DRA… |
| 10.6 | DEF-R106S-22 | Thấp | TC-UX-j, TC-UX-17 | ĐÓNG | sửa 37084f11 | không ghi | Tab Bài tập ở 375 px: hàng bài tập chỉ có liên kết tiêu đề cao 20 px (99×20); chạm ngoài chữ (1… |
| 10.6 | DEF-R106S-23 | Thấp | TC-G10-04 | ĐÓNG | spec/TC đổi P-11-91 | không ghi | US-GD-10 AC2 (và TC) ghi lệnh go test ./internal/assignment ./internal/guidance -run 'TestPubli… |
| 10.6 | DEF-R106S-24 | Thấp | TC-G10-05 | ĐÓNG | chạy lại đạt | không ghi | grading-guidance.spec.ts (và thread-suggest / question-hints) chạy trên gateway GIẢ (page.route… |
| 10.6 | DEF-R106S-25 | Thấp | TC-G10-14 | ĐÓNG | sửa 2fcf3827 | không ghi | docs/guides/grading-guidance.md không có mục «chi phí» (số lời gọi LLM mỗi lần Rút lưu ý / mỗi … |
| 10.6 | DEF-R106S-26 | Thấp | TC-G07-12 | ĐÓNG | sửa c7729a1b | không ghi | cmd/regress export ghi audit_log action regress.export after {"cases": N} nhưng actor_id NULL, … |
| 10.7 | DEF-R10.7-01 | Trung bình | — | ĐÓNG | sửa 13c14bc9 | do sprint sau | Trang Quan sát quyết định lệch khung (teacher: meta lặp tên lớp, lớp rỗng tự dựng; admin: nhịp … |
| 10.7 | DEF-R10.7-02 | Thấp | — | ĐÓNG | sửa cc182747 | do sprint sau | Sổ điểm: đầu trang cao hơn 3,4 px và nút «Chốt điểm» mất khi đang tải / lỗi |
| 10.7 | DEF-R10.7-03 | Thấp | — | ĐÓNG | sửa 82fe7737 | không rõ | Luyện tập sinh viên: tab «Theo chủ đề / Thi thử» biến mất khi đang tải và khi lỗi (375) |
| 10.7 | DEF-R10.7-04 | Thấp | — | ĐÓNG | sửa 9c346b86 | không rõ | /settings/llm: «Máy chủ riêng (tương thích OpenAI)» lặp 7 lần |
| 10.8 | DEF-R10.8-01 | Trung bình | TC-AS08-25, TC-AS09-36, TC-AS09-39 | ĐÓNG | sửa 1c0700e2+ec1657c2 | không rõ | label-content-name-mismatch (axe, serious) ở ba nút thanh trên của khung ứng dụng |
| 10.8 | DEF-R10.8-02 | Thấp | TC-AS14-11, TC-AS13-05 | ĐÓNG | sửa b8c43df8 | lọt sprint gốc | bảng truy vết SRS §11 trỏ tới 13 tên test không tồn tại |
| 10.8 | DEF-R10.8-03 | Thấp | TC-AS08-20 | ĐÓNG | sửa b3c039c0 | lọt sprint gốc | «Đóng» khi mất mạng thử lại bằng Idempotency-Key mới |
| 10.8 | DEF-R10.8-04 | Thấp | TC-AS11-07 | ĐÓNG | sửa b3c039c0 | lọt sprint gốc | nút «Quét mã điểm danh» hiện cả khi qr_code không được hỗ trợ |
| 10.8 | DEF-R10.8-05 | Trung bình | TC-AS11-17=F | ĐÓNG | sửa dea5f197 (qc2 xác nhận 26/11: AC7(a) Back đóng khung quét, ở lại app) | không ghi | Back khi đang quét QR đưa cả app Android ra launcher thay vì đóng khung quét |
| 10.8 | DEF-R10.8-06 | Cao | TC-AS11-17=F | ĐÓNG | sửa e8b2162d | không ghi | gateway không nhận APP_PUBLIC_URL: qr_url / join_url / feed lịch dựng từ https://localhost |
| 10.8 | DEF-R10.8-07 | Thấp | TC-MOB-CL-4=F | ĐÓNG | sửa f6113582 (qc2 xác nhận: dock không còn bị nav đè) | không rõ | cỡ chữ hệ thống 2,0× làm dock «Điểm danh» bị thanh điều hướng đè trên /checkin |
| opencode-go | DEF-OC-SI-01 | Thấp | TC-OC-70 | ĐÓNG | sửa 55664e51 | không ghi | log structured_invalid còn chép một phần nội dung đầu ra của mô hình |

TC ghi riêng mã: mặc định dòng cuối là P; `=LT` là TC lỗi thời đã xoá có nguồn (không còn dòng trong smoke).

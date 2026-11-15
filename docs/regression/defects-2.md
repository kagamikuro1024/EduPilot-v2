### DEF-R2-01 — `openapi.yaml` không qua `redocly lint` (1 lỗi struct)
- Mức: Trung bình
- Sprint gốc: 2 (US-PG-06; cũng sprint 1 US-P0-02/TC-21) · TC gốc: TC-PG06-01, P0-02/TC-21 · Hồi quy do sprint sau?: CÓ (sprint 11 đợt 2, commit 0d38d7a5 «US-P10-06 API thống kê điểm thi»)
- Môi trường: be0c0d96, không cần stack
- Bước tái hiện: 1. `cd` gốc repo. 2. `pnpm exec redocly lint backend-go/api/openapi.yaml` → rc 1, «10370:7 error struct Expected type `ParameterList` (array) but got `null`».
- Kỳ vọng: 0 lỗi lint · Thực tế: 1 lỗi + 21 cảnh báo; thao tác `compareExamAnalytics` (GET /api/v1/analytics/exams/compare) khai báo `parameters:` rỗng
- Bằng chứng: evidence-1/run-02-tc21.log · Nghi ngờ mã: backend-go/api/openapi.yaml:10370 (xoá khoá `parameters:` hoặc điền danh sách)
- Chạy lại lần 2: ổn định
- **TRẠNG THÁI: ĐÃ SỬA (816498c6)** — rerun 2026-11-16: `pnpm exec redocly lint backend-go/api/openapi.yaml` rc 0, 0 lỗi (21 cảnh báo); TC-PG06-01 và P0-02/TC-21 đạt.

### DEF-R2-02 — `go test ./internal/contract` đỏ: TestStatusParity / TestSpec_EveryDocumentedStatusExercised
- Mức: Cao (hợp đồng API↔spec; có một 500 INTERNAL thay vì 404)
- Sprint gốc: 2 (US-PG-06) · TC gốc: TC-PG06-10, TC-PG06-11 (TC-PG06-28 lỗi thời cách chọn test) · Hồi quy do sprint sau?: CÓ (scenarios_exam_test.go thuộc sprint 9.x; spec mở rộng 298 route)
- Môi trường: be0c0d96, testcontainers (Colima), máy tải cao (load ~20)
- Bước tái hiện: 1. `cd backend-go && go test -race -count=1 -tags testroutes -v -run TestStatusParity ./internal/contract`. Lượt đầy đủ: `scenarios_exam_test.go:606: GET …/exams/{eid}/attempts/{aid}/code/{pid}/submissions → 500 (cần 404): {"code":"INTERNAL"…}`; lượt chạy riêng: `scenarios_test.go:441: POST /api/v1/admin/llm/providers → 504 (cần 201) DEADLINE_EXCEEDED` (nghi tải máy). `TestSpec_EveryDocumentedStatusExercised`: nhiều trạng thái khai báo (DELETE /admin/llm/providers/{id}: 204/401/403/404/409; DELETE /chat/sessions/{sid}…) «không test nào sinh ra».
- Kỳ vọng: TestStatusParity, TestSpec_EveryDocumentedStatusExercised PASS · Thực tế: FAIL cả hai lần (chạy riêng và cả bộ)
- Bằng chứng: /tmp/qc2-A-statusparity.log; evidence-2/run-logs/06.log · Nghi ngờ mã: backend-go/internal/contract/scenarios_exam_test.go:606 (500 thay vì 404 ở submissions mã lập trình), contract/exempt.go (thiếu miễn trừ cho route mới)
- Chạy lại lần 2: trượt (2/2)
- **Phân tích SmkJ (23:40, chạy riêng `TestStatusParity` với env CI `TESTCONTAINERS_RYUK_DISABLED=true EP_SKIP_TIMING=1 -race -tags testroutes -count=1`, load máy 18–20, VM Colima load 43–62):** lượt 23:33 trượt ở `scenarios_test.go:501 PUT /admin/llm/routes → 504 DEADLINE_EXCEEDED` (không tới được dòng 606; log gateway của rig bị `io.Discard` ở `rig_test.go`, không có dòng log/stack để trích). Mỗi lượt đỏ ở MỘT yêu cầu khác nhau (441 → 501 → 606; DEF-R10-01 cũng khác endpoint mỗi lần) ⇒ không phải một lỗi xác định. Nguyên nhân gốc đọc từ mã: `rig_test.go:66 REQUEST_TIMEOUT=1s` + Postgres/Redis/MinIO testcontainers DÙNG CHUNG toàn máy (tên cố định, Reuse) với mọi agent đang chạy `go test` ⇒ một câu SQL > 1 s là hết hạn. Hết hạn chưa ghi gì → `timeoutMiddleware` trả 504 (đúng spec); NHƯNG 27 chỗ `apierr.Internal` ở các handler (vd. `examhttp/questions.go:99`, mặc định của `h.fail`) ghi 500 INTERNAL cho mọi lỗi lạ, kể cả `context deadline exceeded` từ DB, TRƯỚC khi middleware kịp xét (`!sw.wrote` sai) ⇒ cùng nguyên nhân cho ra 500 (606, DEF-R10-01: `POST …/report`, `/auth/verify-email`) hoặc 504 (441, 501). Kết luận: **do tải** ở mức kiểm thử (rig 1 s); kèm lỗi nhẹ thật của sản phẩm: lỗi DB do hết hạn bị gói thành 500 thay vì 504 (SRS 3.4) ở handler không dùng ánh xạ `DeadlineExceeded` (chỉ `llmhttp/verify.go`, `observabilityhttp`, `todayhttp` có). Chưa có lượt xanh trên máy rảnh (tải không xuống dưới 8) — lượt kế tiếp: xem mục «lượt chạy» ở smoke-2.md.
- **Cập nhật SmkJ 23:46:** lượt 2 cùng lệnh, cùng mã, load 14–17: `--- PASS: TestStatusParity (110.74s)`, `ok … internal/contract 113s` (log `/tmp/qc2-j-sp.log`). Lượt 1 (load 18–20) đỏ 504 ở dòng 501 ⇒ cùng mã vừa đỏ vừa xanh ⇒ **do tải (REQUEST_TIMEOUT=1s của rig), không phải lỗi xác định**. Riêng dòng 606/500 chưa bắt lại được (log rig bị nuốt) — [INFERENCE] cùng cơ chế 500-thay-504 ở trên. Mức đề xuất hạ: Thấp (test chập chờn theo tải) + Thấp (ánh xạ hết hạn→500 ở handler).
- **TRẠNG THÁI: ĐÓNG — do tải.** Lượt 3 (23:57, load 12–13) `TestStatusParity` PASS 46 s; Main chạy `go test -count=1 -tags testroutes ./internal/contract/...` lúc load ≈ 6 trên bản dbf48ed0: ok 37 s (`evidence-2/logs-contract-rerun2.log`). Tổng riêng `TestStatusParity`: 1 đỏ (504, load ≈ 19) / 2 xanh. TC-PG06-10, -11 = P. Còn lại (Thấp, không chặn): lỗi hết hạn DB bị handler ghi 500 INTERNAL thay vì 504 (xem phân tích trên).

### DEF-R2-03 — `go test ./internal/store` đỏ: TestLLMSchema (6 bảng `llm_%` ≠ 9) — ĐÃ SỬA c272d3d3
- Mức: Trung bình (cổng `go test` đỏ cố định; test cũ không theo migration mới)
- Sprint gốc: 2 (GATE-PG-03/04; sprint 1 P0-02/TC-20, P0-03/TC-15 cũng gọi `go test ./...`) · Hồi quy do sprint sau?: CÓ (sprint 11 P-11-44, commit eb585556: migration 00027 thêm `llm_audit_hourly`, `llm_audit_hourly_lat`, `llm_audit_rollup_state`)
- Môi trường: f3d0740e-trước-sửa, `go test -race -tags testroutes -count=1 ./internal/store`, load ≈ 8
- Bước tái hiện: 1. `cd backend-go && go test -race -tags testroutes -count=1 -run TestLLMSchema ./internal/store` → `llm_test.go:61: expected: 6, actual: 9`.
- Kỳ vọng: gói store xanh · Thực tế: FAIL 3/3 lượt (ổn định, không do tải)
- Bằng chứng: /tmp/qc2-j-gate-testroutes/internal_store.json (lượt đỏ) · Nghi mã: backend-go/internal/store/llm_test.go:61
- **Rerun sau c272d3d3 (2026-11-16 00:5x, bản f3d0740e): `TestLLMSchema` PASS; cả gói `./internal/store` ok (49 s).**

### DEF-R2-04 — `go test ./internal/contract` KHÔNG có `-tags testroutes` đỏ cố định
- Mức: Trung bình (cổng PG.md ghi «contract không tag» và `go test -race ./...` thuần; Makefile/CI chỉ chạy bản có tag nên không ai thấy)
- Sprint gốc: 2 (GATE-PG-03/06; sprint 1 P0-02/TC-20, P0-03/TC-15) · Hồi quy do sprint sau?: CÓ (sprint 7: `scenarios_assignment_test.go:91` gọi `POST /api/v1/_test/llm/fake` nhưng tệp không có build tag; bản mặc định không có route thử → 404)
- Môi trường: f3d0740e, máy load ≈ 6–9, lặp 3 lượt trong gate từng gói + 1 lượt chạy tay
- Bước tái hiện: 1. `cd backend-go && go test -count=1 ./internal/contract` → FAIL: `TestChatThreadsContract` («POST /api/v1/_test/llm/fake → 404 (cần 200)»), `TestContract_ResponsesMatchSpec` («chỉ có 0 quan sát»), `TestSpec_EveryDocumentedStatusExercised` (hàng trăm trạng thái «không test nào sinh ra»). 2. Cùng lệnh thêm `-tags testroutes` → ok.
- Kỳ vọng: gói contract xanh ở cả hai bản dựng (hoặc các kịch bản dùng route thử có `//go:build testroutes`) · Thực tế: đỏ ở bản mặc định
- Bằng chứng: /tmp/qc2-j-gate-notag/internal_contract.json · Nghi mã: backend-go/internal/contract/scenarios_assignment_test.go:91 (và mọi kịch bản dùng route `_test`), contract_test.go:114/174. Đề xuất: PM quyết — thêm `//go:build testroutes` cho các tệp kịch bản (khi đó cổng «không tag» xanh) hoặc xác nhận `-tags testroutes` là cách chạy chuẩn (khi đó GATE-03/06, P0-02/TC-20, P0-03/TC-15 chuyển LT).

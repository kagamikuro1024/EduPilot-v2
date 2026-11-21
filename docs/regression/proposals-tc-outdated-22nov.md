# Đề xuất PM: TC cũ lệch sản phẩm (sprint 1, 1.5, 2) — qc2 22/11

Mỗi dòng là một TC đã chạy lại trên HEAD aba738ef và cho kết quả F vì TC viết theo trạng thái cuối sprint đó, sản phẩm đã đổi hợp lệ ở sprint sau. Đề xuất: PM chấp nhận proposal «sửa TC theo kỳ vọng mới» (qc2 sửa TC + chạy lại → P) hoặc «LT» có nguồn quyết định. Chưa có proposal nào được chấp nhận ⇒ các ca này giữ F.
Tổng: 59 dòng.

## Sprint 1
| TC | Lệch |
|---|---|
| P0-01/TC-02 | : 1 kiểm FAIL — 7 bảng, mỗi bước một bảng đúng 6 tiêu đề cột — got [8] want [7] (log evidence-1/run-22nov/; TC viết cho sprint 1, sản phẩm đã đổi hợp lệ — chờ PM quyết sửa TC/LT (proposal)) |
| P0-01/TC-03 | : 2 kiểm FAIL — tổng khớp mốc 'Hết kịch bản' 13:45 (825 s trừ mở đầu = 13:45 tính cả 0:30) — got [735] want [825]; không thấy dòng 'Hết kịch bản' (log evidence-1/run-22nov/; TC viết cho sprint 1, sản phẩm đã đổi hợp lệ — chờ PM quyết sửa TC/LT (proposal)) |
| P0-01/TC-04 | : 3 kiểm FAIL — route bịa (lệnh AC4); route chỉ xuất hiện ngoài §14/§7 (xem tay) (log evidence-1/run-22nov/; TC viết cho sprint 1, sản phẩm đã đổi hợp lệ — chờ PM quyết sửa TC/LT (proposal)) |
| P0-01/TC-06 | : 2 kiểm FAIL — lệnh AC6; khoá không nằm trong mục 5 (log evidence-1/run-22nov/; TC viết cho sprint 1, sản phẩm đã đổi hợp lệ — chờ PM quyết sửa TC/LT (proposal)) |
| P0-01/TC-13 | : 1 kiểm FAIL — thiếu dòng phiên bản (log evidence-1/run-22nov/; TC viết cho sprint 1, sản phẩm đã đổi hợp lệ — chờ PM quyết sửa TC/LT (proposal)) |
| P0-02/TC-01 | : 1 kiểm FAIL — gốc repo — got [] want [.claude .dockerignore .env.example .gitattributes .github .gitignore AGENTS. (log evidence-1/run-22nov/; TC viết cho sprint 1, sản phẩm đã đổi hợp lệ — chờ PM quyết sửa TC/LT (proposal)) |
| P0-02/TC-05 | : 3 kiểm FAIL — tokens.css + 3 brand — got [6] want [4]; brand không đổi so với base (log evidence-1/run-22nov/; TC viết cho sprint 1, sản phẩm đã đổi hợp lệ — chờ PM quyết sửa TC/LT (proposal)) |
| P0-02/TC-07 | : 1 kiểm FAIL — legacy/ có file đổi nội dung hoặc mới (log evidence-1/run-22nov/; TC viết cho sprint 1, sản phẩm đã đổi hợp lệ — chờ PM quyết sửa TC/LT (proposal)) |
| P0-02/TC-08 | : 2 kiểm FAIL — commit dời lẫn mã mới: 78bfeb6a0c9fb86adb726c65fed87b5b628c0b69; mã mới (#28) trước/cùng commit dời (#85) (log evidence-1/run-22nov/; TC viết cho sprint 1, sản phẩm đã đổi hợp lệ — chờ PM quyết sửa TC/LT (proposal)) |
| P0-02/TC-09 | : 5 kiểm FAIL — đúng 6 service — got [caddy frontend gateway judge mailpit migrate minio mock-graph mock-imap pgboun; mọi service có healthcheck — got [1] want [0] (log evidence-1/run-22nov/; TC viết cho sprint 1, sản phẩm đã đổi hợp lệ — chờ PM quyết sửa TC/L |
| P0-02/TC-19 | : 1 kiểm FAIL — không có dep ngoài phạm vi (tailwind/zustand/tanstack) — got [1] want [0] (log evidence-1/run-22nov/; TC viết cho sprint 1, sản phẩm đã đổi hợp lệ — chờ PM quyết sửa TC/LT (proposal)) |
| P0-02/TC-27 | : 2 kiểm FAIL — đủ env, DB/Redis không tồn tại: /healthz vẫn ok (chỉ kiểm sống) — got [] want [{"status":"ok"}]; SIGTERM: thoát mã 0 (FR-5 v2) — got [1] want [0] (log evidence-1/run-22nov/; TC viết cho sprint 1, sản phẩm đã đổi hợp lệ — chờ PM quyết sửa TC/LT  |
| P0-02/TC-31 | : 1 kiểm FAIL — frontend/ backend-go/ compose, dev.mjs không nhắc legacy — got [4] want [0] (log evidence-1/run-22nov/; TC viết cho sprint 1, sản phẩm đã đổi hợp lệ — chờ PM quyết sửa TC/LT (proposal)) |
| P0-02/TC-34 | : 2 kiểm FAIL — có biến ngoài FR-10; dùng lại secret cũ (log evidence-1/run-22nov/; TC viết cho sprint 1, sản phẩm đã đổi hợp lệ — chờ PM quyết sửa TC/LT (proposal)) |
| P0-02/TC-36 | : 1 kiểm FAIL — nghi secret (log evidence-1/run-22nov/; TC viết cho sprint 1, sản phẩm đã đổi hợp lệ — chờ PM quyết sửa TC/LT (proposal)) |
| P0-02/TC-37 | : 1 kiểm FAIL — gateway không ghi đĩa (luật 10) — got [21] want [0] (log evidence-1/run-22nov/; TC viết cho sprint 1, sản phẩm đã đổi hợp lệ — chờ PM quyết sửa TC/LT (proposal)) |
| P0-02/TC-38 | : 2 kiểm FAIL — còn benchmarks ở gốc; không file Python ngoài legacy/docs — got [23] want [0] (log evidence-1/run-22nov/; TC viết cho sprint 1, sản phẩm đã đổi hợp lệ — chờ PM quyết sửa TC/LT (proposal)) |
| P0-03/TC-01 | : 1 kiểm FAIL — không có run nào trên đúng HEAD 78bfeb6a0c9fb86adb726c65fed87b5b628c0b69 (đã push chưa? có commit mớ (log evidence-1/run-22nov/; TC viết cho sprint 1, sản phẩm đã đổi hợp lệ — chờ PM quyết sửa TC/LT (proposal)) |
| P0-03/TC-02…P0-03/TC-03 | : 2 kiểm FAIL — đúng 2 job — got [1] want [2]; Go + Frontend đều success — got [,] want [Frontend success,Go success,] (log evidence-1/run-22nov/; TC viết cho sprint 1, sản phẩm đã đổi hợp lệ — chờ PM quyết sửa TC/LT (proposal)) ; : 8 kiểm FAIL — chỉ 0 bước kh |
| P0-03/TC-04 | : 8 kiểm FAIL — push không lọc nhánh/đường dẫn (mọi nhánh) — got [2] want [0]; không Node 24 (log evidence-1/run-22nov/; TC viết cho sprint 1, sản phẩm đã đổi hợp lệ — chờ PM quyết sửa TC/LT (proposal)) |
| P0-03/TC-06 | : 3 kiểm FAIL — tải log run — got [1] want [0]; log không có 'golangci-lint run' (log evidence-1/run-22nov/; TC viết cho sprint 1, sản phẩm đã đổi hợp lệ — chờ PM quyết sửa TC/LT (proposal)) |
| P0-03/TC-07 | : 1 kiểm FAIL — nhánh ci/red-check đã xoá (TC-12 đạt: ls-remote rỗng) nên không còn run đỏ để chấm (thiếu RUN_RED_GO/RUN_RED_UI) (log evidence-1/run-22nov/; TC viết cho sprint 1, sản phẩm đã đổi hợp lệ — chờ PM quyết sửa TC/LT (proposal)) |
| P0-03/TC-08…P0-03/TC-09 | : 1 kiểm FAIL — nhánh ci/red-check đã xoá (TC-12 đạt: ls-remote rỗng) nên không còn run đỏ để chấm (thiếu RUN_RED_GO/RUN_RED_UI) (log evidence-1/run-22nov/; TC viết cho sprint 1, sản phẩm đã đổi hợp lệ — chờ PM quyết sửa TC/LT (proposal)) ; : 1 kiểm FAIL — nhá |
| P0-03/TC-10 | : 1 kiểm FAIL — nhánh ci/red-check đã xoá (TC-12 đạt: ls-remote rỗng) nên không còn run đỏ để chấm (thiếu RUN_RED_GO/RUN_RED_UI) (log evidence-1/run-22nov/; TC viết cho sprint 1, sản phẩm đã đổi hợp lệ — chờ PM quyết sửa TC/LT (proposal)) |
| P0-03/TC-11 | : 1 kiểm FAIL — nhánh ci/red-check đã xoá (TC-12 đạt: ls-remote rỗng) nên không còn run đỏ để chấm (thiếu RUN_RED_GO/RUN_RED_UI) (log evidence-1/run-22nov/; TC viết cho sprint 1, sản phẩm đã đổi hợp lệ — chờ PM quyết sửa TC/LT (proposal)) |
| P0-03/TC-16 | : 1 kiểm FAIL — không còn commit chưa push (AC1 cần HEAD đã push) — got [2213] want [0] (log evidence-1/run-22nov/; TC viết cho sprint 1, sản phẩm đã đổi hợp lệ — chờ PM quyết sửa TC/LT (proposal)) |

## Sprint 1.5
| TC | Lệch |
|---|---|
| TC-00-10 | : package.json dependencies = @tanstack/react-query, @tanstack/react-virtual, lucide-react, next, react, react-dom, uqr (TC: không thêm phụ thuộc ngoài lucide-react); màu cứng #CE1628 trong src/shared/styles/brand.ts ngoài tokens.css; sessionStorage dùng ở app |
| TC-00-60 | : grep «N ngày/giờ/phút trước» ngoài mock/derive.ts = 0 (đạt) nhưng «hôm qua 16:40» không còn ở Admin / và /admin/courses (grep src = 0; dữ liệu mock đã thay bằng API thật) — TC viết cho prototype mock; chờ PM quyết sửa TC/LT (proposal) |
| TC-03-23 | : chuỗi «Chỉ giảng viên công bố điểm» không còn trong frontend/src (grep = 0; TA chỉ đọc đơn khiếu nại: AppealPanel «Chỉ TEACHER quyết; TA chỉ đọc»); «Xác nhận công thức» chỉ có ở GradeScheme.tsx:248,270 (TA không thấy theo phân quyền, kiểm ở S8) — TC lệch câu |

## Sprint 2
| TC | Lệch |
|---|---|
| TC-PG01-11 | PG01-11: biến JWT_EXPIRATION không còn (đổi thành ACCESS_TOKEN_TTL, sprint 3) [ms khi JWT_EXPIRATION sai: thực tế [2457] phải ≤ 1000] (log evidence-2/run-logs-22nov/01-n.log) |
| TC-PG01-16 | PG01-16: APP_ENV=production nay kiểm khoá/CORS bắt buộc nên thoát <100 ms (sprint sau) [APP_ENV=production: đi tới bước chờ phụ thuộc (ms): thực tế [48] phải ≥ 1500] (log evidence-2/run-logs-22nov/01-n.log) |
| TC-PG01-34 | PG01-34: Redis ep-r dùng chung 7 nhóm khoá (ep:auth:*, ep:llm:* … thêm sau sprint 2) [số khoá Redis ngoài quy ước: thực tế [4] ≠ mong đợi [0]] (log evidence-2/run-logs-22nov/01-n.log) |
| TC-PG01-68 | PG01-68: Makefile test nay có 2 lần RYUK_DISABLED=true [số lần RYUK_DISABLED=true: thực tế [2] ≠ mong đợi [1]] (log evidence-2/run-logs-22nov/01-n.log) |
| TC-PG01-85 | PG01-85: Makefile lint nay gọi golangci-lint 3 lần (không tag, testroutes, +1) [số lần gọi golangci-lint: thực tế [3] ≠ mong đợi [2]] (log evidence-2/run-logs-22nov/01-n.log) |
| TC-PG02-01…TC-PG02-02 | PG02-01: schema 00001→00029: nay 60+ bảng (sprint 3…11); TC đếm theo 00001 [danh sách bảng public: thực tế [assignments attendance_checkins attendance_records audit_log auth_sessions auth_tokens calendar_events chat_]; PG02-02: enum nay nhiều hơn (sprint 3…11) |
| TC-PG02-19 | PG02-19: index 308 (TC đòi 17 theo 00001) [số index: thực tế [308] ≠ mong đợi [17]] (log evidence-2/run-logs-22nov/02-n.log) |
| TC-PG02-21 | PG02-21: 66 cột id uuidv7() (TC đòi 5) [cột id mặc định uuidv7(): thực tế [66] ≠ mong đợi [5]] (log evidence-2/run-logs-22nov/02-n.log) |
| TC-PG02-41 | trên Postgres tạm (evidence-2/scripts-j/pg02-j.sh, log run-logs-22nov/02-j.log): migrate up lần hai rc=0 và goose_db_version không thêm dòng (đạt) nhưng «vẫn đúng 5 bảng + goose = 6» thực tế 83 bảng public (schema 00001→00029) — TC lệch sau sprint 3…11, chờ PM |
| TC-PG02-42 | PG02-42: goose version mới nhất 29 (TC đòi 1) [goose_db_version.version_id mới nhất: thực tế [29] ≠ mong đợi [1]] (log evidence-2/run-logs-22nov/02-n.log) |
| TC-PG02-48 | PG02-48: float64 trong mã sqlc sinh từ cột double precision (sprint sau; spec v1.5 miễn _test.go) [không float64 trong mã sản xuất internal/store (spec v1.5, #12: _test.go miễn): thực tế [19] ≠ mong đợi [0]] (log evidence-2/run-logs-22nov/02-n.log) |
| TC-PG02-49 | PG02-49: 1 dòng «OFFSET» nằm trong chú thích «không OFFSET» (observability.sql:2), không có OFFSET trong câu SQL — grep quá chặt [OFFSET trong internal/store/queries: thực tế [1] ≠ mong đợi [0]] (log evidence-2/run-logs-22nov/02-n.log) |
| TC-PG02-70 | PG02-70: cùng lý do PG02-01/19/21 (schema tăng) [bảng public sau khi loại _test_%: thực tế [assignments attendance_checkins attendance_records audit_log auth_sessions auth_tokens calendar_e] (log evidence-2/run-logs-22nov/02-n.log) |
| TC-PG03-51 | PG03-51: 11 dòng OFFSET trong mã/chú thích ngoài _test.go (sprint sau thêm truy vấn OFFSET ở bảng nhỏ hoặc chú thích) [số dòng có OFFSET (ngoài _test.go): thực tế [11] ≠ mong đợi [0]] (log evidence-2/run-logs-22nov/03-n.log) |
| TC-PG03-98 | PG03-98: APP_ENV=production: log lỗi nêu biến production khác (kiểm tra production sprint sau) [log nêu APP_ENV: thực tế [0] phải ≥ 1] (log evidence-2/run-logs-22nov/03-n.log) |
| TC-PG04-44 | PG04-44: openapi nay có /auth/ /me/ /admin/ (sprint 3…11), TC đòi 0 [grep -c '/auth/' backend-go/api/openapi.yaml: thực tế [11] ≠ mong đợi [0]] (log evidence-2/run-logs-22nov/04-n.log) |
| TC-PG04-45 | PG04-45: route auth/me/admin nay tồn tại (405/415/200), TC đòi 404 theo sprint 2 [GET /api/v1/auth/login (chế độ test) · status: thực tế [405] ≠ mong đợi [404]] (log evidence-2/run-logs-22nov/04-n.log) |
| TC-PG04-46 | PG04-46: route auth/me/admin nay tồn tại (401/405/415), TC đòi 404 theo sprint 2 [không vào được chế độ default] (log evidence-2/run-logs-22nov/04-n.log) |
| TC-PG06-04…TC-PG06-07 | PG06-04: openapi.yaml 299 đường dẫn (TC đòi 5) [grep -cE '^  /' api/openapi.yaml: thực tế [299] ≠ mong đợi [5]]; PG06-05: 2 chữ «_test» nằm trong mô tả (sprint sau) [grep -c '_test' api/openapi.yaml: thực tế [2] ≠ mong đợi [0]]; PG06-06: openapi.test.yaml 19 đ |
| TC-PG06-24 | PG06-24:  (log evidence-2/run-logs-22nov/06-n.log) |
| TC-PG06-28 | PG06-28:  (log evidence-2/run-logs-22nov/06-n.log) |
| TC-PG07-01…TC-PG07-02 | PG07-01: compose nay 13 service (thêm judge, mock-graph, mock-imap…) [danh sách service của docker-compose.local.yml: thực tế [caddy frontend gateway judge mailpit migrate minio mock-graph mock-imap pgbouncer p]; PG07-02: compose nay có docling (11 chuỗi) [chu |
| TC-PG07-22 | PG07-22:  (log evidence-2/run-logs-22nov/07-n.log) |
| TC-PG07-35 | PG07-35: 4 chỗ ghi đĩa: internal/llm/fake/fake.go (ghi bản ghi replay của provider giả, chỉ dùng test/demo) và cmd/regress/main.go (CLI offline) — không nằm trên đường request [số chỗ ghi đĩa trong mã sản xuất: thực tế [4] ≠ mong đợi [0]] (log evidence-2/run-l |
| TC-PG07-37 | PG07-37: mỗi container có 2 thay đổi FS: A /seed, A /seed/demo (thư mục mount seed của ep-r) [số thay đổi hệ thống file của 325f91ee45ade8bf9413ae65629e6695dfe9051dd6f40ccb0885db34462815a0: thực tế [2] ≠ mong đợi [0]] (log evidence-2/run-logs-22nov/07-n.log) |
| TC-PG07-38…TC-PG07-41 | PG07-38: gh không có run CI cho sha 06f164c7 (commit cũ của sprint 2 đã bị thay bằng rebase/merge) [headSha của run mới nhất: thực tế [KHÔNG CÓ] ≠ mong đợi [06f164c780e58475cbdcc3dd74228ff32635a344]]; PG07-39: không tìm được run CI của sprint 2 [tìm được run i |
| TC-PG07-43…TC-PG07-44 | PG07-43: nhánh ci/sqlc-drift đã xoá (TC ghi: xoá sớm = FAIL) [không tìm thấy run ci.yml của nhánh ci/sqlc-drift (chạy lại với RUN_ID=<id> lấy từ handoff; nhánh bị xoá sớm hơn lúc QC chấm = FAIL)]; PG07-44: không có run ci/sqlc-drift để đối chiếu [không có run  |
| TC-PG07-56 | PG07-56: compose nay công bố thêm cổng mock-graph, mock-imap; trên ep-r caddy ánh xạ 780/1143, frontend không công bố 3000 (ep-r dùng cổng riêng) [tập service có cổng công bố: thực tế [caddy frontend mailpit minio mock-graph mock-imap postgres redis] ≠ mong đợ |
| TC-PG07-58 | PG07-58: .env.example có SEED_DEFAULT_PASSWORD không chứa «-dev» (mật khẩu seed theo thiết kế) [số giá trị bí mật của .env.example KHÔNG chứa '-dev': thực tế [1] ≠ mong đợi [0]] (log evidence-2/run-logs-22nov/07-n.log) |
| TC-PG07-66 | PG07-66: compose có override nay 13 service (thêm judge, mock-graph, mock-imap) [danh sách service khi có override: thực tế [caddy frontend gateway judge mailpit migrate minio mock-graph mock-imap pgbouncer postgres redis] (log evidence-2/run-logs-22nov/07-n.l |

# QC test case — US-P10-12 (bảo mật, README, compose prod, visual QA, 10 câu nghiệm thu, demo 15 phút)
Nguồn: US.md + SRS FR-22; `docs/DEMO_SCRIPT.md`; AGENTS.md (cấm secret / PII trong log). Hộp đen. TC ID: `TC-12-nn` (file này; không trùng `TC-12-nn` của sprint 10 vì khác thư mục).

| TC | AC | Tiền điều kiện | Bước / lệnh | Kết quả mong đợi | Loại |
| --- | --- | --- | --- | --- | --- |
| TC-12-01 | AC1 | Repo, mọi commit của nhánh gốc..HEAD | `git log -p <gốc>..HEAD \| grep -E` mẫu khoá (`AIza[0-9A-Za-z_-]{30,}`, `sk-[A-Za-z0-9]{20,}`, `-----BEGIN .*PRIVATE`, `eyJ[A-Za-z0-9_-]{20,}\.`, `postgres://[^ ]*:[^@ ]+@`); `git ls-files \| grep -E '^\.env'` | 0 khoá thật (ví dụ `.env.example` với giá trị giả chấp nhận, ghi rõ); không `.env*` nào ngoài `.env.example` được track | TAY |
| TC-12-02 | AC1 | Stack | Chạy chat riêng + chấm 1 bài + `generate` insight; `docker compose logs --no-color gateway worker` rồi `grep -Ei` tên / MSSV / email / SĐT của SV seed | 0 khớp trong log | TAY |
| TC-12-03 | AC2 | Cấu hình CORS | `curl -sk -H 'Origin: https://evil.example' -X OPTIONS $B/auth/login -i`; rồi origin hợp lệ | Origin lạ: không có `Access-Control-Allow-Origin` (hoặc không phản chiếu `*`/origin lạ); origin cấu hình: có | API |
| TC-12-04 | AC2 | Rate limit (Q-S11-15: ngưỡng) | Lặp `POST /auth/login` sai mật khẩu, `register`, `password/reset`, `join` (mã lớp) cho tới khi bị chặn; đo số lần | Mỗi nhóm bị `429` + `Retry-After` (số giây hợp lệ); đếm ở Redis (đổi pod / khởi lại gateway không xoá hạn mức); hết thời hạn → hoạt động lại | API |
| TC-12-05 | AC2 | – | Xác nhận kiểm thử | Có test Go cho rate limit và CORS (`go test` tên nêu ở handoff); xanh | GO |
| TC-12-06 | AC3 | Bảng nhóm endpoint × test trong handoff | Với mỗi nhóm endpoint lớp học: gọi bằng SV lớp khác, vai sai | SV lớp khác → `404`; vai sai → `403`; QC chọn ngẫu nhiên ≥ 10 endpoint ngoài bảng của dev và tự gọi | API |
| TC-12-07 | AC3 | – | `grep -rn "CourseAccessGuard" backend-go/internal/**/routes*.go`; liệt kê route `/courses/{id}/…` không qua guard (đối chiếu `openapi.yaml`) | Mọi route lớp học qua guard; ngoại lệ có lý do | DIFF |
| TC-12-08 | AC4 | – | Đọc `docs/sprints/11/qc/security-review.md` | Liệt kê mục đã rà, lỗi, mức, đã sửa / nợ; lỗi **Cao** đã sửa (QC kiểm lại từng lỗi); QC rà độc lập thêm: IDOR vài endpoint mới (`open`, `latest`, `exams/*`), mass-assignment, JWT, upload | TAY |
| TC-12-09 | AC5 | Worktree sạch, macOS Apple Silicon + Colima | Làm theo README từng bước (không đọc nơi khác); ghi thời gian | Dựng được stack bằng `pnpm dev`; đăng nhập demo; chạy được test cơ bản; có mục «Chạy trên máy», «Biến môi trường», «Dữ liệu demo mô phỏng (D44)», «Khắc phục sự cố ngắn»; bước nào sai → lỗi (ghi bước) | TAY |
| TC-12-10 | AC5 | – | Mọi lệnh / biến trong README | Tồn tại thật (`grep` trong repo); không đường dẫn tuyệt đối máy dev; không khoá thật | DIFF |
| TC-12-11 | AC6 | – | `docker compose -f docker-compose.prod.yml config`; dựng thử 1 lần (khi PM xếp RAM) | `config` hợp lệ; khởi động được tới `healthy`; không khoá cứng trong file (dùng biến); hướng dẫn triển khai ghi rõ VPS cũ đã thu hồi | TAY |
| TC-12-12 | AC7 | – | Red Thread Transition: đo thời lượng (Performance API) | < 300 ms | E2E |
| TC-12-13 | AC7 | `prefers-reduced-motion: reduce` | Cùng chuyển cảnh | Không chuyển động (không animation/transition kéo dài) | E2E |
| TC-12-14 | AC7 | – | Nếu trễ | Ghi «cắt» khi dev báo; không tính FAIL story (AC không chặn) | – |
| TC-12-15 | AC8 | 7 route đại diện (Q-S11-16; QC đề xuất: `/`, `/chat`, `/threads`, `/gradebook`, `/practice`, `/observability`, `/insights`) | Lượt A: chụp 1440 + 375 → ghi danh sách lỗi → dev sửa một mẻ → Lượt B chụp lại | Ảnh + danh sách ở `docs/sprints/11/qc/visual/`; **không có lượt thứ ba**; lỗi lượt A đã hết ở lượt B (hoặc ghi nợ) | TAY |
| TC-12-16 | AC8 | – | Mỗi ảnh | Đối chiếu `DESIGN.md` §22 (10 điều kiện) + `UX.md` 375 px (không tràn ngang, vùng chạm ≥ 44 px) | TAY |
| TC-12-17 | AC9 | – | `docs/sprints/11/qc/acceptance-10.md` | 10 câu ở `docs/design/AGENT_PROMPT.md` §«Final acceptance questions» (Q-S11-17, PM 2026-11-14) mỗi câu có trả lời + bằng chứng (lệnh / ảnh / test); QC chạy lại ≥ 3 bằng chứng ngẫu nhiên; câu không đạt ghi rõ | TAY |
| TC-12-18 | AC10 | Stack demo | Chạy `docs/DEMO_SCRIPT.md` trọn, bấm giờ (một người, không bỏ bước) | Tổng ≤ 15 phút; mọi «kết quả phải thấy» đúng; lệch → sửa kịch bản/seed, **không** sửa kết quả mong đợi cho khớp lỗi (`git diff DEMO_SCRIPT.md` kiểm) | TAY |
| TC-12-19 | AC11 | – | Bước 7 (F17) | Khớp US-P10-04/05: A, B đầu lớp 1; C lớp 2; chủ đề < 3 SV không câu mẫu; «Nhật ký chạy lại» có dòng mới (ngày, commit, thời lượng) | TAY |
| TC-12-20 | AC12 | – | Video | «chờ chủ dự án» — không kiểm, không FAIL | – |
| TC-12-21 | AC13 | – | `bash scripts/ui-antipatterns.sh` | Mã 0 | TAY |
| TC-12-22 | AC13 | – | Mở `/settings/llm`, `/settings/integrations`, khối bảo mật `/settings` | Không còn câu mô tả dưới tiêu đề / dòng giải thích thừa; tooltip chỉ cho cú pháp / lý do khoá / hệ quả; nhãn + điều khiển + giá trị; **không đổi message có trong SRS** mà không vá SRS (CLAUDE.md luật chữ UI) | TAY |
| TC-12-23 | AC13 | – | Chạy playwright spec của 3 màn đó | Xanh, không sửa assertion để qua (chữ bị xoá phải có proposal / vá spec) | E2E+DIFF |
| TC-12-24 | AC14 (P-11-9) | Repo | Đọc diff `backend-go/internal/grading/` (mã injection + luồng chấm) | `injection.go` (regex) còn nguyên làm lớp đầu; cờ cuối = regex **OR** LLM; gắn LLM dùng `llm.Structured(json_schema)` qua `internal/llm` (không import SDK provider); gộp vào lời gọi chấm sẵn có hoặc một lời gọi riêng; không vòng gọi mới trong vòng lặp tiêu chí | DIFF |
| TC-12-25 | AC14 | Bài tập có rubric, `fake` | Chấm 1 bài hợp lệ; đếm `llm_audit` (`task` = chấm, theo `trace_id`/job) trước và sau | Số lời gọi LLM mỗi bài: **bằng** trước P-11-9 nếu gộp; nếu riêng thì đúng +1 (làn `BATCH`, `task` có tên riêng); không +N theo số tiêu chí | DB |
| TC-12-26 | AC14 | Bài có chèn lệnh, `fake` trả cờ true; rồi `fake` trả cờ false và regex khớp; rồi cả hai false | `GET` kết quả chấm của từng bài | Cờ `INJECTION_SUSPECTED` khi (LLM true) hoặc (regex true) và chỉ khi đó (bảng OR 3 hàng + hàng cả hai true); không cờ khác bị thêm / mất | API |
| TC-12-27 | AC14 | Cùng bài với/không có chèn lệnh, cùng `fake` chấm | So `total` + điểm từng tiêu chí giữa bài có cờ và bản đối chứng (cùng nội dung bỏ câu chèn) | Điểm **không đổi** do cờ (chênh chỉ do nội dung); `go test ./internal/grading/...` xanh | API+GO |
| TC-12-28 | AC14 | Bài có chèn lệnh | `llm_audit.payload` / `llm_request_payloads` (mở bằng `open`): chữ gửi LLM | Câu chèn lệnh **vẫn có** trong đầu vào (không xoá, TLR-4); việc chấm hoàn tất, trạng thái bình thường (không bị chặn / `FAILED`) | DB+API |
| TC-12-29 | AC14 | – | Câu nghi được tô | Màn duyệt `/grading/<id>` tô câu nghi (nếu LLM trả vị trí) hoặc chỉ cờ; không lỗi khi LLM không trả vị trí | E2E |
| TC-12-30 | AC15 | LLM thật qua 9router (`ag/gemini-3.8-flash`, cấu hình `/settings/llm`); lớp có bài tập + rubric | Nộp lần lượt 6 bài: `benchmarks/real-llm/data/injection/v1…v5` và `docs/specs/FEAT-assignments/fixtures/submissions/s6-injection.md`; chấm; đọc cờ | **≥ 6/6** có `INJECTION_SUSPECTED`; ghi nhà cung cấp + model + ngày + từng bài (có / không); lặp 3 lần/bài (LLM không tất định), tính bắt ổn định | API (LLM thật) |
| TC-12-31 | AC15 | Hạ lớp LLM: tắt LLM gắn cờ (`fake` trả false) hoặc đọc kết quả regex riêng | Cùng 6 bài | Regex một mình bắt **1/6** (mốc so sánh); ghi vào báo cáo | API |
| TC-12-32 | AC15 | CI `fake` | Đọc test Go / CI của dev cho AC15 | Chỉ kiểm đường mã (OR, không đổi điểm), **không** khẳng định độ bắt với `fake` (không giả vờ 6/6); báo cáo có số LLM thật | DIFF |
| TC-12-33 | AC16 | – | Nộp bài rỗng, chỉ khoảng trắng / xuống dòng, bài dưới `GRADING_MIN_CHARS`; đếm `llm_audit` | Cờ `EMPTY_SUBMISSION`, 0 điểm; **0 lời gọi LLM** (cả chấm lẫn gắn cờ); **không** `INJECTION_SUSPECTED` (LLM thật: lặp 3 lần, cả Gemini nhầm bài trống ở PoC) | API+DB |
| TC-12-34 | AC16 | LLM thật | Chấm các bài hợp lệ còn lại của `FEAT-assignments` (`s1`, `s2`, `s3`, `s5`, `s7`: không chèn lệnh, không trống; spec nói «7 bài», thư mục có 6 bài ngoài `s6`, trong đó `s4` trống: Q-S11-22) | **0** bài có `INJECTION_SUSPECTED` (đối chứng, ghi số bài, lần lặp); bài bị gắn nhầm → lỗi (báo mức Trung bình) | API (LLM thật) |
| TC-12-35 | AC17 | `fake` trả `OVERLOADED`; rồi lỗi mạng; rồi JSON sai schema (thiếu trường cờ) | Chấm bài có chèn lệnh khớp regex, và bài không khớp | Việc chấm **hoàn tất** (không treo / `FAILED` / retry vô hạn); cờ chỉ theo regex (bài khớp regex có cờ, bài không khớp không); điểm không đổi; `llm_audit` ghi lỗi (status, không PII) | API+DB |
| TC-12-36 | AC17 | LLM chậm hơn hạn | Chấm | Có timeout (Q-S11-21: giá trị); không giữ lease việc chấm quá hạn | API |
| TC-12-37 | AC18 | Bài có tên / MSSV / email thật của SV seed trong nội dung | Chấm; `grep` tên / MSSV / email trong `llm_request_payloads.masked_prompt` của lời gọi gắn cờ (và `tools`) | 0 khớp; có placeholder; `llm_audit.pii_masked_count` > 0 | DB |
| TC-12-38 | AC18 | Bài đã chấm có cờ | `sv.kha` (chủ bài): `GET` bài / kết quả / màn của mình | **Không** thấy cờ `INJECTION_SUSPECTED` hay chữ «Nghi chèn lệnh» trong mọi phản hồi (`grep`); `ta@`, `teacher@` lớp thấy ở màn duyệt | API+E2E |
| TC-12-39 | AC18 | – | GV / TA lớp 2; SV khác; ADMIN gọi kết quả chấm bài lớp 1 | Lớp khác `404`; vai SV `403`/`404`; ADMIN theo quyền hiện hành của P7 (ghi, không đổi) | API |

## Nhánh lỗi
TC-12-04 (429), TC-12-09 (bước sai), TC-12-11, TC-12-33, TC-12-35, TC-12-36.

## Phân quyền
TC-12-06, 12-07, 12-08, TC-12-38, TC-12-39.

## Lịch sử sửa TC
- 2026-11-14: TC-12-17 ghi nguồn 10 câu theo trả lời PM Q-S11-17 (spec không đổi; chỉ điền chỗ trống).
- 2026-11-14: thêm TC-12-24…39 cho US-P10-12 AC14–18 (P-11-9, SRS v1.3, FR-24; BA `d03187aa`).

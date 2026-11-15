# QC sprint 11 (FEAT-p10-eval) — sổ lỗi đợt 1–3 (ep-qc, nhánh `sprint/11-p10-eval`)

Mức: **Cao** (luồng chính hỏng / mất / lộ dữ liệu), **TB** (nhánh lỗi, quyền, lệch AC), **Thấp** (giao diện, chữ, tài liệu).
Trạng thái: **ĐÃ SỬA** = dev/BA sửa, QC chạy lại PASS ở commit ghi kèm. **MỞ** = chưa sửa tại thời điểm viết. Bằng chứng từng ca: `scripts/results-p10.jsonl` (khoá = tên TC).

## A. Lỗi sản phẩm / tài liệu đã tìm và đã sửa (QC kiểm lại PASS)
| ID | Mức | TC | Lỗi | Sửa | QC chạy lại |
| --- | --- | --- | --- | --- | --- |
| D-11-01 | TB | TC-01-05b, 01-06b | `filter=problem` trả cả status không lỗi; `from>to` trả 200 thay vì 422 | dev | PASS |
| D-11-02 | TB | TC-04-04, 04-27 | `period_days=0` → 200 (phải 422); GV/SV lớp khác → 404 thay vì 403 | dev, P-11-35 | PASS |
| D-11-03 | Thấp | TC-02-13 | Drawer «Mã theo dõi» thiếu nút sao chép | `b354fd16` | PASS |
| D-11-04 | Thấp | TC-14-82 | Lọc «Người» ở `/admin/audit` bắt dán uuid | dev | PASS (AUD-3) |
| D-11-05 | Thấp | TC-05-21 | Mục gập `/insights` cao 30 px ở 375 px (cần ≥ 44) | `f25857bb` | PASS (INS-3) |
| D-11-06 | TB | @real F3 | Tin khủng hoảng không dấu («em dang muon chet…») vẫn sinh lời gọi EMBEDDING | P-11-34 | PASS 3/3 |
| D-11-07 | TB | TC-02-14 | Payload quá 30 ngày chưa bị job dọn vẫn mở được 200 (phải 410) | P-11-35 (3) | PASS |
| D-11-08 | **Cao** | TC-01-13 | 1 triệu hàng `llm_audit`: summary 7 ngày p95 8–11 s, 24 giờ 504 (cần ≤ 500 ms) | bảng cộng gộp theo giờ, migration 00027 (P-11-44) | PASS: 7d p95 32 ms, 24h 15 ms; đếm / lỗi / dự phòng / chi phí khớp tuyệt đối, p50/p95 lệch < 0,2 % |
| D-11-09 | **Cao** | @real F9 | SV nộp PDF trong app: «Chưa gửi được» (CSP `connect-src 'self'` + Caddyfile local không có `/edupilot/*`) — chỉ VPS chạy được | `54265ad6` | PASS (F9/F11 3/3) |
| D-11-10 | TB | U1 / `/assignments/[id]` | Script làm-mới-sớm dùng `NEXT_PUBLIC_API_URL` lúc chạy, bundle dùng giá trị lúc build → lệch thì tải thẳng `/assignments/<id>` văng `/login` | `36c960f5` | PASS (deep link vào bài tập chạy) |
| D-11-11 | TB | TC-04-04b | `/insights` không có bộ chọn kỳ (SRS: 7 / 14 / 30 ngày + «Toàn kỳ» = 120) | `5f38581d` | PASS (INS-4) |
| D-11-12 | TB | TC-04-15 | `llm_audit.pii_masked_count` của lời gọi INSIGHT luôn 0 dù đã che PII | `c5d406fe` | PASS (Δ = +3 đúng 3 email thêm) |
| D-11-13 | TB | UX «không mất chữ» | Nháp thread ghim chỉ lưu nội dung, tiêu đề mất khi tải lại | `51c14cd8` | PASS (INS-7) |
| D-11-14 | TB | TC-07-15 | `00022` < `00024` đã merge → `gateway migrate` trên DB main lỗi out-of-order | `08886ae6` (`WithAllowOutofOrder`) | PASS (DB tạm 0–21 + 24 → áp 22/25/26/27) |
| D-11-15 | Thấp | TC-07-07 | US.md AC3: lệnh `go test ./internal/observability …` → «no tests to run» | BA `8dcaff28` | PASS |
| D-11-16 | Thấp | TC-07-03 | `seed.mjs` chạy lại ngày khác → mã 1 (bước 16) và `study_time_daily` +2.580 hàng | `7545f033`, `09cb419c` | PASS lệch 0 / −1 ngày (số hàng không đổi) — xem D-11-19 cho −7 |

## B. Còn MỞ khi viết
| ID | Mức | TC | Lỗi | Ghi chú |
| --- | --- | --- | --- | --- |
| D-11-17 | TB | TC-04-11 | Embedding thật (gemini-embedding-001): 85 câu → 45 cụm, A (an ninh mạng) và B (quy chế) tản thành nhiều cụm 1–3 câu nên không đứng đầu; top5 = «Thông tin học phần», «Quy định chuyển và tính điểm», «Cơ chế bảo vệ của Firewall» | ngưỡng gom cụm quá chặt với embedding thật; dev nhận (quét ngoại tuyến bằng vector đã lưu) | **Lượt 22:06Z @708dc729 (UPGMA tau 0,52 + lô ≤50): CHƯA CHẤM LẠI được — khoá Gemini miễn phí hết hạn mức NGÀY (429 `EmbedContentRequestsPerDayPerProjectPerModel-FreeTier`, 1000 chuỗi/ngày/model; gọi thẳng chuỗi 50 → 429, 10 → 200). Chạy lại sau 07:00Z (đặt lại 00:00 PT). Môi trường, không phải lỗi mã.**
| D-11-18 | Thấp | TC-04-11 | Kỳ 120 ngày (≈190 câu) → một lô nhúng > 100 chuỗi → 429 khoá Gemini miễn phí → job `FAILED` (gọn, không 5xx) | nên chia lô ≤ 50 + giãn nhịp; chỉ lộ với khoá bậc miễn phí |
| D-11-19 | Thấp | TC-07-03c | Seed lại sau khi dịch `class_sessions` −7 / −8 ngày vẫn mã 1 ở bước 16 («AT_RISK 1, WATCH 2», cần «AT_RISK 1, WATCH 3») | dev nhận; lệch 0 và ±1 ngày đạt |
| D-11-20 | Thấp (ĐÃ SỬA `5852c9c2`, QC đọc spec PASS) | TC-07-09 | `insights.spec.ts` (mock) thiếu nhánh AC4: đổi lớp 2 / «C đứng đầu lớp 2», thread ghim hiện ở `/threads`, «LLM lỗi → tên tạm» | QC bù bằng `@real` INS-1..7 + `p10_insight*.py`; dev thêm 2 nhánh hoặc BA sửa AC4 |
| D-11-21 | Thấp | TC-07-13 (dev nhận) | Ảnh mốc `insights-1440` (11-15 20:21) và `observability-1440` (11-15 16:27) cũ hơn UI (`5f38581d`, `51c14cd8`, `13c14bc9`) | dev sinh lại từ artifact CI |
| D-11-22 | Thấp | TC-07-14 (PM làm khi đóng sprint) | `docs/PROGRESS.md` còn F16 ☐ / F17 ☐ | PM cập nhật khi đóng sprint; luồng đã đi trọn (xem summary) |
| D-11-23 | Thấp (ĐÓNG theo P-11-53: k1/k2 trong ±15 %) | TC-10-16 (AC11 tái lập) | E6 hai lượt lệch ngoài ±15 % đã ghi (Q-S11-13): RAM 22,0 %, khởi động worker 20,9 % (gateway 14,9 % đạt); E1 3 lượt giống hệt | chấm đúng sai số ghi, không nới; TL giải thích giá trị nhỏ (RAM 14–23 MiB, khởi động 163–795 ms); lượt r1/r2 giữ FAIL (TC-10-16a); lượt có kiểm soát k1/k2 (P-11-53, `419241e4`): RAM 0,3 %, khởi động gateway 2,8 %, worker 4,1 % → TC-10-16b PASS; ghi cờ `contended` |
| D-11-24 | Cao | TC-08-02 (US-P10-08 AC2) | LCP devtools trung vị ở ep-qc (load < 8, đã «Đã hiểu»): /threads 5.314, /gradebook 5.113, /me 5.284, /assignments/[id] 4.859 ms > 2.500 (4/8 route trượt; /, /chat, /inbox, /settings/llm 1.52–1.59 s đạt). Phần tử LCP là dữ liệu thật; khớp lời dev-wave3 «dữ liệu hiện sau ≈ 4,7–6 s dưới throttle này» — mức đạt trong handoff nhờ chữ khung nhỏ hơn h1 trên dữ liệu T1 | Mở 4 route trên ep-qc (sv.kha / teacher) | Dev: đo trên T1 hoặc tối ưu đường dữ liệu (SSR/prefetch); QC đo lại ở stack T1 theo TC-08-02 | Mở |

## B2. Kiểm từ bằng chứng của TL (không chạy lại, tốn lời gọi thật)
- DEF-R9.5-10 (`a8b3a8cc`, mask.json): 31 probe, 24 tới mô hình, khôi phục 35/35, sót 0, đổi/bỏ 0, `spell_leak` 0 → PASS. Ghi nhận: bộ dò đánh vần nay chỉ bắt chuỗi có gạch nối (chữ-chữ-chữ), không bắt chữ cách khoảng trắng; 7 probe P17–P23 bị chặn `OTHER_PERSON` trước mô hình (tính riêng).
- DEF-R9.5-11 (threads.json, crisis.json): Threads 5/5 AI trả lời (p50 9,1 s), khủng hoảng 4/4 → PASS.

## C. Ghi nhận không phải lỗi sản phẩm
- Môi trường đã tìm và xử lý: provider fake qua ENV báo AUTH khi đặt `FAKE_LLM_VALID_KEY` (seed bước 10) · job `observability.alert` do judge giữ khoá leader (hạ tick, xoá khoá) · migration 24 vào DB sau 25 (áp tay trước khi có D-11-14) · `Caddyfile.kc` của QC thiếu `/edupilot/*` (đồng bộ từ `deploy/caddy/Caddyfile`) · `/gateway migrate` cũ · Hôm nay: nhà cung cấp `staff` có ngân sách 150 ms, khi máy quá tải bị bỏ và kết quả thiếu được nhớ 60 s (không lỗi; spec tải lại).
- Quan sát (không lỗi): ticket «Nhờ giảng viên» khoá khi SV đã có 5 ticket mở; thread ghim tối đa 3/lớp (`PIN_LIMIT`); `useAutosaveDraft` chỉ giữ chữ người dùng gõ.

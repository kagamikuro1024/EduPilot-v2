# QC report — US-P3-07 (độ tin cậy + E1)  · Kết luận: FAIL (1 lỗi Thấp; TC-07/08/18/27 chưa kiểm được)

Handoff: `docs/sprints/6/handoff/dev-US-P3-07.md`. Bộ TC: `tc-US-P3-07.md` (27 TC).
**Môi trường:** gateway + worker + frontend build từ HEAD `5a9a741`+, DB riêng `qc_p801`, nhà cung cấp `fake` (nhúng băm → bước tương đồng không đóng góp, đúng như handoff ghi). `eval_pii.py` chạy thật trên stack này bằng token `sv.gioi`. Không đụng stack s55.

## Số liệu E1 (đo thật, 2 lượt, `docs/sprints/6/qc/e1/`)
| Lượt | Lệnh | Kết quả |
| --- | --- | --- |
| 1 | `python3 benchmarks/eval_pii.py --min-recall 0.95 --max-false-block 0.05` | `E1 PASS recall=0.960 false_block=0.000`, rc=0 |
| 2 | như trên | `E1 PASS recall=0.960 false_block=0.000`, rc=0; `diff` hai `counts` **rỗng** (TP 96, FN 4, FP 0, TN 100); `dataset_sha256` ghi trong báo cáo, khớp `shasum` tệp |
| S7 | `.by_stratum.S7` | `{tp:0, fn:4, recall:0.0}` — đúng 4 mẫu tên ngoài roster, ghi riêng; `e1.md` có mục "Giới hạn đã biết" |
Lưu ý trung thực: bộ dữ liệu và luật do cùng một người viết (dev) nên con số có thể lạc quan; QC soát độc lập 50 mẫu (dưới): 0 bất đồng.

## Cổng đã chạy
| Lệnh | Kết quả |
| --- | --- |
| `go test -race ./internal/agent -run 'TestConfidenceFormula|…NoExtraLLMCall|…ToolIntent|TestGroundZeroWhenNoClaim|TestLowConfidenceUsesCourseThreshold|TestNoContextCannedReply'` | PASS (6) |
| `go test -race -tags testroutes ./internal/chat ./internal/thread ./internal/contract -run 'TestStudentNeverSeesConfidence|TestStaffConfidenceThreadsOnly|TestStaffCannotReadPrivateConfidence|TestE1DatasetMeetsGate'` | PASS |
| `eval_pii.py --validate --strict-synthetic` | PASS — `OK 200 items; positives=100 negatives=100; dev=60 test=140; strata=S1=12,S2=8,S3=8,S4=6,S5=20,S6=38,S7=4,S8=4,N1=50,N2=15,N3=15,N4=10,N5=10` |

## TC
| TC | Kết quả | Chứng cứ |
| --- | --- | --- |
| 01 | PASS | 6 test xanh |
| 02 | PASS (một phần) | bài AI của Threads: Staff thấy `confidence` (1,000: fake echo → groundedness 1, cos=1 → retr 1); đối chiếu công thức tay chưa làm cho giá trị khác |
| 03 | PASS | chat COURSE_QA: đúng +1 lời gọi `fake` (sinh chữ); không lời gọi thêm |
| 04 | PASS (một phần) | `EXAM_SCHEDULE` NoData: `confidence`/`low_confidence` NULL/false (không "chưa chắc"); intent có dữ liệu → 1,000 chưa kiểm vì lịch chưa nối (P8-03) |
| 05 | KHÔNG KIỂM ĐƯỢC | với embedding giả `confidence` chỉ là 1,000 (có ngữ cảnh) hoặc 0,000 + `no_context` (không ngữ cảnh), không có giá trị giữa nên không lật được qua ngưỡng 0,50 ↔ 0,95; `TestLowConfidenceUsesCourseThreshold` xanh |
| 06 | PASS | SV đọc `GET /chat/sessions/{id}/messages` (5 phiên), `GET /threads`, SSE `done`: 0 khoá `confidence`/`retrieval_score`/`groundedness` (chỉ `low_confidence` bool) |
| 07, 08 | KHÔNG KIỂM ĐƯỢC | không ép được `low_confidence=true` (xem 05); e2e của dev `private-chat.spec.ts` có ca này |
| 09 | PASS | UI (teacher, chọn lớp 761987) `/threads/{id}`: "Độ tin cậy 0,72" (dấu phẩy) cạnh bản nháp AI |
| 10 | PASS | `GET /chat/sessions/{sid}/messages` bằng TEACHER / TA / ADMIN → `403` |
| 11 | PASS | câu ngoài phạm vi → "Mình chưa tìm thấy nội dung này trong tài liệu của lớp.", 0 lời gọi sinh, `confidence=0.000`, `low_confidence=t`, `no_context=t` |
| 12, 13 | PASS | `--validate` OK; đếm nhóm đúng SRS 9.3 (S1 12 … N5 10) |
| 14 | PASS | 0 `id` trùng; trường bắt buộc đủ |
| 15 | PASS | soát độc lập 50 mẫu (25 dương + 25 âm, `random.seed(7)`): 50/50 trùng nhãn `expect_block`; 3 mẫu biên (`S6-21` "Em thi cuối kỳ phòng nào?", `S6-28` "điểm thi của em thấp quá…", `S6-32` "Trường hợp của em có được thi lại không") tôi gán dương theo "câu cá nhân", đồng ý nhãn |
| 16 | PASS | xem số liệu E1 |
| 17 | PASS | `e1.json` có `by_pii_type`, `by_split`, `by_stratum`, `channel_confusion`, `metrics`, `counts`, `dev_misses`, `s7_leaked`, `gate`, `constants`, `dataset_sha256` |
| 18 | KHÔNG KIỂM ĐƯỢC | chưa nới ngưỡng để thử rc=1 |
| 19 | PASS | hai lượt: `counts` giống hệt; hash bộ dữ liệu ghi |
| 20 | PASS | `--api http://localhost:9` → rc=2, "LỖI: không gọi được API …", không in PASS |
| 21 | PASS (một phần) | token rác → rc=2 "chỉ chạy bằng token Sinh viên (token này có vai None)"; chưa thử token hết hạn thật (401) |
| 22 | PASS | trùng `id` / nhãn `maybe` / ô `{{SV99.mssv}}` không có trong roster → rc=2, nêu rõ lỗi |
| 23 | PASS | token Giảng viên → rc=2 "từ chối"; `--strict-synthetic` → OK |
| 24 | PASS | số `forum_threads / forum_posts / pii_events` = 5/6/4 trước và sau các lượt `eval_pii.py` (chỉ gọi `precheck`, chỉ đọc) |
| 25 | **FAIL (Thấp)** | chèn mẫu có tên `Trần Quang Vinh` (không thuộc roster seed, không thuộc `outside_roster_names.json`, không dùng ô `{{…}}`) → `--strict-synthetic` vẫn `OK 200 items` (rc=0); script chỉ kiểm ô điền, không phát hiện tên tự do |
| 26 | PASS | `e1.md` có "## Giới hạn đã biết" nêu S7; `by_stratum.S7` riêng |
| 27 | CHUYỂN → US-P3-08 | `scripts/gate-p3.sh` chưa tồn tại |

## AC
AC1–AC7, AC9–AC12: PASS (AC4/TC-07, 08 chưa kiểm được trên stack, test Go + e2e dev xanh); **AC8 PASS** (E1 PASS trên stack thật: recall 0,960, chặn nhầm 0,000); AC11 có một lệch (BUG-1).

## Lỗi
- **BUG-1 (Thấp)** — `--strict-synthetic` không bắt tên tự do ngoài roster (TC-25), nên "dữ liệu mô phỏng D44" chỉ được bảo đảm cho các ô điền. Đề xuất: so mọi cụm 2–4 từ viết hoa với roster + danh sách bịa, hoặc đổi AC cho khớp.
- Ghi chú: cache câu trả lời cho câu hỏi lặp trả `confidence` NULL (lần đọc từ cache); không ảnh hưởng SV.

## Đề nghị
FAIL chỉ vì BUG-1 (Thấp) và TC chưa kiểm được. PM có thể chấp nhận có điều kiện vì cổng E1 đạt. Chuyển TC-27 sang P3-08.

# DEV handoff — US-P3-07 (độ tin cậy + E1) — **giao đủ phần mã và bộ dữ liệu; chưa đo E1 trên stack**
Nhánh `sprint/6-p3-p8`. Commit `US-P3-07: …`.

## Đã làm
- `internal/agent/confidence.go`: `ResponseMetadata`, `Retrieval` (clamp((cos−0,25)/(0,65−0,25), 0, 1)), `Groundedness` (câu ≥ 4 từ nội dung bỏ stop-words, ≥ 40 % từ có trong hợp các đoạn; không câu nào đủ → 0), `Score` = 0,6·retr + 0,4·ground làm tròn 3 chữ số bằng `shopspring/decimal`, `Outcome.Metadata` (tool có dữ liệu → 1,000; `no_context` → luôn `low_confidence`; câu mẫu / chào hỏi → NULL, không "chưa chắc"). Không lời gọi LLM nào thêm.
- `internal/chat/gen.go`: tính metadata sau khi sinh xong, đọc `courses.escalation_threshold` mỗi tin (đổi có hiệu lực ở tin kế tiếp), ghi `confidence` + `low_confidence`, khung `done` mang `low_confidence` thật; câu "chưa chắc" **không** được ghi vào cache trả lời.
- `internal/thread/answer.go`: `confidence` của bài AI dùng cùng công thức (trước đó chỉ là cosine).
- Giao diện `/chat`: dưới câu trả lời `low_confidence` chỉ hiện "AI chưa đủ chắc chắn về câu này" (chữ nhạt, không biểu tượng, không nút, không gọi `/escalate`).
- **E1:** `benchmarks/pii/e1_dataset.jsonl` (200 mẫu, 100 / 100, dev 60 / test 140, nhóm S1…S8 / N1…N5 đúng SRS 9.3), `roster_seed.json` + `dump_roster.mjs` (roster lớp 1 đúng như `scripts/seed.mjs`), `outside_roster_names.json`, `LABELING.md`; `benchmarks/eval_pii.py` (`--validate`, `--strict-synthetic`, `--min-recall`, `--max-false-block`, `e1.json` + `e1.md` có mục "Giới hạn đã biết", mã thoát 0 / 1 / 2, từ chối token không phải Sinh viên, chờ khi chạm giới hạn 60 yêu cầu / phút).
- **Luật PII chỉnh trên tập `dev`** (`internal/privacy`): tên đảo tuỳ ý (mọi hoán vị của 3–4 âm tiết), MSSV gõ tách giữa (`2022 4786`, `2022.4786`; không từ khoá chỉ tính khi trùng MSSV của roster), từ khoá `msv` / `ms sv` / `mã sv`, câu cá nhân về lịch thi / điểm cộng / điều kiện áp vào mình, và lỗi thật tìm được: "tối thiểu" / "tới" bỏ dấu thành `toi` bị coi là đại từ "tôi" (chặn nhầm "đặc quyền tối thiểu").
- Test: `TestConfidenceFormula` (16 hàng), `TestGroundZeroWhenNoClaim`, `TestConfidenceToolIntent`, `TestLowConfidenceUsesCourseThreshold`, `TestNoContextCannedReply`, `TestConfidenceNoExtraLLMCall` (`internal/agent`); `TestStudentNeverSeesConfidence` ở `internal/chat`, `internal/thread`, `internal/contract` (hợp đồng quét mọi lời gọi chat / Threads bằng token sinh viên của bộ kịch bản, gồm SSE); `TestStaffConfidenceThreadsOnly` (`thread`), `TestStaffCannotReadPrivateConfidence` (`chat`); `TestE1Rules` (`privacy`); `TestE1DatasetMeetsGate` (`thread`: 200 mẫu qua tường lửa thật + roster seed, in lỗi chỉ của `dev`); e2e `private-chat.spec.ts -g 'low confidence sentence only'`.

## Kết quả E1 (tầng dịch vụ, `TestE1DatasetMeetsGate`)
Tổng TP 96 · FN 4 · FP 0 · TN 100 → **recall 0,96, chặn nhầm 0,00**; 4 FN đều là S7 (tên người ngoài roster, vùng NER đã cắt — D46) đúng như SRS. Tập `test`: TP 67 / FN 3 / FP 0 / TN 70. Lần đầu (trước khi chỉnh luật): recall 0,80, chặn nhầm 0,02.
**Giới hạn trung thực:** bộ dữ liệu và luật do cùng một người (Dev) viết nên kết quả sạch này có thể lạc quan; QC soát ≥ 50 mẫu độc lập ở `LABELING.md`, và phép đo qua `precheck` của stack thật (nhúng `fake`, tức bước tương đồng không đóng góp) là việc của `gate-p3.sh` (US-P3-08).

## File đổi
`backend-go/internal/{agent,chat,thread,privacy,contract}`, `internal/chat/gen.go`; `benchmarks/eval_pii.py`, `benchmarks/pii/*`; `frontend/src/features/chat/{RealChat.tsx,useChatRun.ts}`, `frontend/e2e/private-chat.spec.ts`; `docs/sprints/6/proposals.md` (#15).

## Lệnh QC
```bash
cd backend-go && export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true
go test -count=1 -race ./internal/agent -run 'TestConfidenceFormula|TestConfidenceNoExtraLLMCall|TestConfidenceToolIntent|TestGroundZeroWhenNoClaim|TestLowConfidenceUsesCourseThreshold|TestNoContextCannedReply' -v
go test -count=1 -race -tags testroutes ./internal/chat ./internal/thread ./internal/contract -run 'TestStudentNeverSeesConfidence|TestStaffConfidenceThreadsOnly|TestStaffCannotReadPrivateConfidence|TestE1DatasetMeetsGate' -v
python3 ../benchmarks/eval_pii.py --validate --strict-synthetic     # OK 200 items; positives=100 negatives=100; dev=60 test=140
python3 ../benchmarks/eval_pii.py --api http://localhost:9; echo rc=$?   # 2
# trên stack seed: SEED_DEFAULT_PASSWORD=… python3 benchmarks/eval_pii.py --min-recall 0.95 --max-false-block 0.05
```

## AC tự đánh giá
AC1 ✓ · AC2 ✓ (ngưỡng mặc định lệch spec: đề xuất #15) · AC3 ✓ · AC4 ✓ (e2e) · AC5 ✓ · AC6 ✓ · AC7 ✓ (`--validate`; QC soát độc lập chưa có) · AC8 ◐ (đạt ở tầng dịch vụ; `eval_pii.py` chưa chạy với stack seed → `e1.json` / `e1.md` thật chưa có) · AC9 ✓ (script không có tham số ngẫu nhiên; hai lần chạy với máy chủ giả cho `counts` giống hệt, đã `diff`) · AC10 ✓ (`--api :9` → 2, bộ dữ liệu sai → 2, token Giảng viên → 2) · AC11 ✓ · AC12 ✓ (báo cáo ghi riêng S7).

## Nợ / cần hỏi
1. `eval_pii.py` chạy thật trên stack seed + `e1.json` / `e1.md`: làm ở `gate-p3.sh` (US-P3-08, cần compose; chưa xin dừng stack s55).
2. Đề xuất #15: `escalation_threshold` mặc định 0,60 ở DB (migration `00003` đã merge) lệch 0,80 của spec.
3. `confidence` chỉ được ghi cho COURSE_QA có đoạn và tool có dữ liệu; câu mẫu / chào hỏi để NULL (không phải "chưa chắc").

## Vòng sửa 1 (PM) — BUG-1 `--strict-synthetic` (TC-25)
Trước: chỉ kiểm ô điền và trường `outside_names`, nên tên tự do viết thẳng vào `text` lọt. Sau: `eval_pii.py --strict-synthetic` quét mọi cụm ≥ 2 từ viết hoa chữ đầu (NFC, tách theo chữ cái) trong văn bản đã điền ô và từ chối cụm không thuộc roster seed + `outside_roster_names.json` + `benign_capitalized.json` (mới: Hà Nội, Hoa Kỳ, Đà Nẵng, Content Security Policy — các cụm hợp lệ của nhóm N); khớp không phân biệt dấu / thứ tự, bỏ từ đầu câu viết hoa. Đã chạy: bộ 200 mẫu → `OK`; thêm một mẫu "Lê Quốc Zeta" → `rc=2` kèm id mẫu và cụm; không `--strict-synthetic` thì không kiểm (như trước). Ghi ở `benchmarks/pii/LABELING.md`.

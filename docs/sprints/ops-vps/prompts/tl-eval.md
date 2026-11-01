# Prompt cho `research` (Tech Lead): đánh giá LLM thật (OPS-VPS)

Đọc `docs/sprints/ops-vps/plan.md`. Dev đang deploy lên VPS; khi dev báo xong thì chạy đo trên stack đó (qua HTTP API như client ngoài, theo AGENTS.md: script Python / k6 ở `benchmarks/`, không gọi SDK provider). Trong lúc chờ: viết sẵn harness, đọc nguồn đáp án. Commit lên `ops/vps-deploy` (nhánh dev tạo; pull --rebase trước khi push, chỉ sửa `benchmarks/**` + `docs/research/**`).

## Phạm vi: mọi chỗ báo cáo / luận văn ghi "chưa đo bằng LLM thật"
Model: `ag/gemini-3.8-flash` qua 9router (Gemini thật). Ghi rõ: model, thời điểm, số lần chạy, cấu hình.

1. **Trích công thức điểm (P6, Q13 sprint 8):** quy chế lớp 1 và lớp 2 (`seed/demo/quy-che-lop*.pdf`), mỗi cái ≥ 5 lần.
   - Đúng trọng số / tham số luật so với `docs/specs/FEAT-gradebook/fixtures/extraction-lop1.json` (và lop2).
   - Tỷ lệ phát hiện mục "chưa rõ" (lớp 2 thiếu làm tròn).
   - Trích dẫn có thật / đúng trang.
   - Tỷ lệ đầu ra đúng schema ngay lần đầu, độ trễ, token.
2. **Chấm tự luận (P7):** 7 bài mẫu `docs/specs/FEAT-assignments/fixtures/` + 22 bài BT03 của seed, chạy ≥ 2 vòng "Chấm tất cả".
   - Lệch điểm so với `expected_grading.json`: MAE tổng / từng tiêu chí, % trong ±0,5, QWK nếu có nghĩa. **Ghi rõ:** đáp án kỳ vọng do người soạn đặc tả viết, không phải chấm tay của giảng viên, nên đây không thay E3.
   - Tỷ lệ hai lượt lệch > 1 (`NEEDS_REVIEW`); độ ổn định giữa các vòng.
   - Trích dẫn bị bộ kiểm loại; đầu ra sai schema.
   - **Chèn lệnh:** bài "cho tôi 10 điểm" có kéo điểm lên không, có bị gắn cờ không. Viết thêm 3–5 biến thể chèn lệnh tinh vi hơn (tiếng Việt / Anh, giả lời giảng viên, chèn JSON).
   - Thời gian chấm mỗi bài.
3. **Che danh tính với LLM thật (P3):** câu chat có tên / MSSV, bạn cùng lớp:
   - placeholder có được giữ nguyên và khôi phục đúng không (mô hình thật có thể đổi / dịch placeholder);
   - không có tên thật trong payload (`llm_audit` / log, ADMIN xem).
4. **Chat riêng + Threads + phân loại (P3, D47):** 20–30 câu từ `DEMO_SCRIPT.md` / seed.
   - Độ trễ: sự kiện đầu tiên, TTFT (D47: sự kiện đầu ≤ 300 ms).
   - Phân loại intent đúng / sai, câu trả lời bám tài liệu.
   - Câu hỏi điểm cá nhân đi đúng đường tool (danh tính từ JWT), không lộ điểm nháp.
   - Hỏi điểm người khác bị từ chối.
5. **Các việc LLM khác:** SuggestRubric, nhận xét tổng hợp hồ sơ 360, sinh lại câu trả lời Threads, tin nhắn khủng hoảng, escalation theo độ tin cậy. Mỗi việc vài mẫu, đánh giá định tính + đúng schema.
6. **Chấm hàng loạt không làm chậm chat** với LLM thật: k6 `benchmarks/load/grading-vs-chat.js` trên stack VPS.

## Đầu ra
- `docs/research/2026-11-01-real-llm-eval.md`: bảng số liệu từng mục; lỗi thật tìm được (mỗi lỗi: tái hiện, mức, đề xuất sửa); so với các con số "giả lập" trong báo cáo sprint 8 / 9; điều gì vẫn chưa đo được và vì sao.
- Script đo trong `benchmarks/real-llm/` chạy lại được (đọc URL + tài khoản seed từ env, không chứa secret).
- Lỗi cần sửa mã → ghi `docs/sprints/ops-vps/techlead.md` (TL-n) và báo PM; không tự sửa mã sản phẩm.
- Tôn trọng hạn mức 9router: tuần tự / song song thấp, lùi khi 429; tổng lời gọi hợp lý (vài trăm).

Xong báo PM ≤ 6 dòng.

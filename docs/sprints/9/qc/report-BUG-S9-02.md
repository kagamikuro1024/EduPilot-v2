# QC — BUG-S9-02 (proposals #17): chú thích HTML nội bộ lọt ra trang sinh viên

Commit chấm: `01dacb32` (CI xanh). Stack: dựng từ DB trống (project `edupilot100`). Kết luận: **PASS**.

| # | Kiểm | Cách | Kết quả |
|---|---|---|---|
| 1 | Seed không nạp chú thích | So `createBody()` trước sửa (`01dacb32~1`) và sau sửa với cùng `create-bt03.json` | Trước: `description`, `reference_answer` còn `<!--`. Sau: không còn; `CANARY-BT03-ANSWER` vẫn còn. PASS |
| 2 | Dữ liệu thật trên stack | `check-assignments-seed.mjs bt03` | `OK 22/22 khớp ngay + 3 chưa khớp + 6 không nộp`. PASS |
| 3 | Không còn ở DB | `pg_dump --data-only` tìm `BA sở hữu` / `NỘI DUNG GỐC` | 0 dòng. `assignments`: 0 bài còn `<!--`/`D44`. PASS |
| 4 | Chấm không lệch | (dev chạy `grade` 22/22 sau sửa; QC không chạy lại, chỉ chạy `bt03`) | Ghi nhận, không tính vào kết luận |
| 5 | Bộ hiển thị: một dòng, nhiều dòng, giữa dòng, hai chú thích một dòng, `<!-->`, chưa đóng (bỏ tới hết), `-->` lẻ không đổi | Gọi thẳng `stripComments` / `parseMarkdown` | PASS (mọi kết quả đúng về nội dung; số dòng trống giữ nguyên, Markdown bỏ qua) |
| 6 | Khối mã giữ nguyên chữ; khối mã không đóng giữ nguyên; dòng ``` bên trong chú thích không đảo trạng thái | Gọi thẳng | PASS |
| 7 | Trang sinh viên BT03 ở 375 px | Playwright, `sv.kha`: không có `<!--`, `-->`, `BA sở hữu`, `NỘI DUNG GỐC`, `D44`, `seed dùng`; còn đủ Loại / Thang điểm / Hạn nộp / Đề bài; không cuộn ngang | PASS (ảnh `p710-student-form.png`) |
| 8 | Test dev | `markdown.spec.ts` + `assignment-lifecycle.spec.ts` + axe `assignment a11y` | 9 passed, `--retries=0` |
| 9 | Đột biến: bỏ `stripComments` khỏi `parseMarkdown` | Worktree sạch, build lại | 2 test đỏ (`markdown: chú thích HTML không hiện thành chữ`, `student page hides html comment`). Hoàn nguyên xong |

## Ghi nhận ngoài #17 (không tính FAIL)
- **Tiêu đề Markdown hiện nguyên dấu `#`:** trang sinh viên in `# Bài tập 03 — …` và `## Đề bài` thành chữ thường, trong khi khối đã có tiêu đề "Đề bài" (lặp hai lần). Bộ hiển thị chỉ hỗ trợ đoạn / nhấn / mã / danh sách (`markdown.spec.ts: cú pháp tối thiểu`). Đề xuất PM chọn một: seed bỏ dòng `# …` và `## Đề bài` đầu đề, hoặc bộ hiển thị hỗ trợ tiêu đề.
- **Ngoài phạm vi:** `<!-- formula-not-decoded -->` trong đoạn trích tài liệu thư viện (docling) là dữ liệu gốc, không phải chú thích nội bộ; bộ hiển thị mới sẽ ẩn nó khi hiển thị qua `parseMarkdown`.
- Giới hạn đã biết của dev: `<!--` trong dấu backtick một dòng bị coi là chú thích. Chưa gặp nhu cầu.
- Chưa đột biến phía seed (bỏ `stripHtmlComments`) trên stack mới; mục 1 đã thay bằng so sánh trực tiếp trước/sau.

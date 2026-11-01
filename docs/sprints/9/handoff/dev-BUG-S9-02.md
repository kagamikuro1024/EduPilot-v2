# Handoff — BUG-S9-02 (proposals #17): chú thích HTML nội bộ lọt ra trang sinh viên

## Nguyên nhân
Tệp BA `fixtures/de-bai-bt03.md` / `dap-an-chuan-bt03.md` mở đầu bằng khối `<!-- NỘI DUNG GỐC … BA sở hữu … D44 -->`; `create-bt03.json` (`description`, `reference_answer`) mang nguyên văn khối đó, seed nạp thẳng vào `assignments`, và bộ hiển thị Markdown (HTML thô = chữ) in nó ra trang `/assignments/[id]` của sinh viên.

## Đã sửa (hai lớp, theo #17)
- **Seed:** `scripts/assignment-seed-data.mjs` — `createBody()` bỏ mọi khối `<!-- … -->` (kể cả nhiều dòng) ở `description`, `reference_answer`, tên / hướng dẫn tiêu chí trước khi gửi API; `CANARY-BT03-ANSWER` nằm NGOÀI chú thích nên vẫn còn trong đáp án. Tệp của BA (`fixtures/`) không đổi.
- **Bộ hiển thị:** `frontend/src/shared/lib/markdown.ts` — `stripComments()` chạy đầu `parseMarkdown`: bỏ chú thích HTML (một dòng, nhiều dòng, giữa dòng, chưa đóng → bỏ tới hết) NGOÀI khối mã; trong khối ``` giữ nguyên chữ; HTML thô khác vẫn chỉ là chữ. Giới hạn đã biết: `<!--` nằm trong dấu backtick một dòng cũng bị coi là chú thích (chưa có nhu cầu).

## Test
- `markdown.spec.ts`: `markdown: chú thích HTML không hiện thành chữ …` (đúng chuỗi "NỘI DUNG GỐC … BA sở hữu … D44", nhiều dòng, giữa dòng, chưa đóng, khối mã giữ nguyên).
- `assignment-lifecycle.spec.ts`: `student page hides html comment` — đề bài có chú thích thì `main` của trang sinh viên không chứa chữ của chú thích, nội dung đề vẫn đủ.
- `check-assignments-seed.mjs bt03`: `description` / `reference_answer` của BT03 và BT04 không còn `<!--`, `BA sở hữu`, `NỘI DUNG GỐC`, `D44`; phản hồi `…/mine` của sinh viên không chứa chúng cũng không chứa `CANARY`.
- Chạy: `lint`, `tsc`, Playwright `markdown + assignment-lifecycle + exam` 82 passed (`--retries=0`); seed trên stack dựng lại từ DB trống: xem kết quả bên dưới.
- Stack dựng lại từ DB trống: `seed.mjs` rc 0; `check-assignments-seed.mjs bt03` → `OK 22/22 …` (kèm kiểm chú thích / canary); `grade` → 22/22 khớp `expected_grading.json` (đề bài bỏ chú thích không làm lệch bản ghi `fake_llm.json`); `demo5` → `B 7.50`.

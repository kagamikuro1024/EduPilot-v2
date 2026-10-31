# QC sprint 9 (P7 Bài tập) — quy ước chung cho `tc-US-P7-01…11.md`, `tc-US-S9-SEED.md`, `gate-P7.md`
Nguồn: `docs/specs/FEAT-assignments/{US,SRS,QUESTIONS,TL-REVIEW}.md` (APPROVED, US v1.1) + `fixtures/` (BA sở hữu: `expected_grading.json`, `expected_checks.json`, `seed-plan.csv`, `teams-plan.csv`, `rubric-bt03.json`, `submissions/s1…s7`), `docs/phases/P7.md` (cổng 7 lệnh), `docs/FLOWS.md` F9, F11, `docs/DEMO_SCRIPT.md` bước 5 → 6, `docs/sprints/9/proposals.md`, `AGENTS.md`. **Viết trước khi có code, không đọc code của dev** (v1, 2026-11-01).

## Cách đọc
- Mỗi TC có ba loại hàng: **hộp đen** (`TC-P7-nn-01…`: gọi API / giao diện / DB bằng lệnh QC tự viết, so với số của BA), **đột biến / độc lập** (`…-M…`: QC phá thử mã của dev ở bản sao `/tmp/qc-mut`, test phải đỏ), và **`-G<AC>`** (chạy đúng test Go / Playwright mà US ghi ở dòng `Kiểm:`; runner `scripts/p7-go-tests.sh` FAIL nếu test được nêu tên không tồn tại, bị SKIP, không in `--- PASS`, hoặc có DATA RACE).
- Mỗi AC ≥ 1 hàng; mỗi story có hàng nhánh lỗi và hàng phân quyền. "KHÔNG KIỂM ĐƯỢC" tính là FAIL; một AC FAIL → story FAIL.
- Tài khoản: A `sv.gioi@`, **B `sv.kha@` (mẫu S7)**, C `sv.nguyco@` (không nộp), `sv04…sv30` (`seed-plan.csv`), D `sv.khac@` (chưa vào lớp), TEACHER `teacher@`, TA `ta@`, ADMIN `admin@`, giảng viên lớp khác `teacher2@` (QC lấy tên thật từ `ARCHITECTURE.md` §9; lệch → ghi câu hỏi). Biến shell: `source docs/sprints/9/qc/scripts/p7-env.sh` (`GW C1 C2 Q Q2 TCH TA_ SVA SVB SVC SVD ADM j idem PSQL MP A3 A4 SUB_B RES_B`, đúng SRS 9.1).
- Canary: `CANARY-BT03-ANSWER` (đã nằm trong đáp án chuẩn), `CANARY-DRAFT-7K3` (QC đặt vào nhận xét nháp), `CANARY-AISCORE` (điểm nháp 7,77 đặt qua provider giả hoặc `PUT`). Quét: `bash scripts/canary-scan.sh` + `grep -rc` trên phản hồi / DOM do QC thu.
- Đáp án QC tự tính độc lập: `scripts/p7-hand-calc.py` (đọc `fixtures/expected_grading.json` + `seed-plan.csv`, tự tính lại Σ tiêu chí, khoản trừ muộn `0,5 × min(⌈trễ/24⌉, 2)`, điểm học phần v2 từ `seed/expected_final_grades_v2.csv` bằng phân số) — chạy ở pha 2 để chắc đáp án BA đúng trước khi dùng làm chuẩn.
- Không gọi LLM / Microsoft / thư thật (`LLM_PROVIDER=fake`). Đột biến chạy ở `/tmp/qc-mut` (git worktree) rồi xoá. `docker volume prune -f` sau mỗi lượt.
- Pha 2 chạy khi có handoff + CI xanh; kết quả vào `report-US-P7-nn.md`.

## Thứ tự chấm khi dev giao (theo plan): 01 → 06 → 02 → 03 → 04 → 07 → 08 → 09 → 10 → 05 → 11 → S9-SEED → `gate-P7.md`.

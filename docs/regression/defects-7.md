### DEF-R7-01 — CI chạy đủ job vượt ngưỡng cứng 10 phút của US-CI-02
- Mức: Trung bình
- Sprint gốc: 7 · TC gốc: TC-20 (US-CI-02) · Hồi quy do sprint sau?: CÓ (các sprint 9.x–11 thêm nhiều spec Playwright vào cùng 3 shard; `.github/workflows/ci.yml` vẫn `shard: [1,2,3]`)
- Môi trường: GitHub Actions, nhánh sprint/11-p10-eval @be0c0d96 (run PR 43712425198), `gh run view`
- Bước tái hiện: 1. `gh run view 43712425198 --json jobs,createdAt,updatedAt`. 2. wall = 12:09:46 → 12:25:44 = 15,97 phút; job «Frontend e2e (Playwright 1/3)» 15,4 phút, 2/3 6,5 phút, 3/3 2,7 phút; Lighthouse simulate 10,3 phút, Go rest-1 9,5 phút. Run push e0bf1fc9 (`43710464639`) cũng 16,7 phút.
- Kỳ vọng (PM 2026-10-13): lượt chạy đủ wall ≤ 10 phút và nhanh hơn mốc cũ 13 m 43 s; Thực tế: ~16 phút, chậm hơn mốc cũ.
- Bằng chứng: `/tmp/qc2e/ci-run.json`, `/tmp/qc2e/ci-log.txt` · Nghi ngờ: `.github/workflows/ci.yml:235-253` (3 shard không cân bằng theo thời gian), job Lighthouse simulate 10,3 phút
- Chạy lại lần 2: ổn định (hai run liên tiếp ≥ 16 phút)

# Smoke hồi quy sprint 5.5 — kết quả
Mã đã chạy: HEAD sprint/11-p10-eval (worktree qc2, bản dựng cổng `pnpm build:gate`) · Người chạy: qc2 (SmkC, SmkL) · Ngày: 2026-11-16
Tổng TC cũ: 58 · Đạt: 51 · Trượt: 0 · K (a/b/c): 0 · LT: 7 · Chưa chạy: 0   (P+F+K+LT+S = 58)

Cách chạy: server bản dựng cổng `next start -p 3310` + gateway giả `api-server.mjs` :3312; `pnpm exec playwright test --workers=1 --reporter=json` (không `@real`) → `/tmp/qc2-c5/pw55b.json`, bảng theo spec ở dưới (`scripts/finalize_55.py`); TC đối chiếu spec theo tên `-g` của lệnh TC (`scripts/make55.py`); lệnh tĩnh (grep/ls/ui-antipatterns) chạy trực tiếp.

## Tổng hợp theo US
| US | TC tổng | P | F | LT | Chưa chạy |
| --- | --- | --- | --- | --- | --- |
| US-UI-01 | 8 | 4 | 0 | 4 | 0 |
| US-UI-02 | 11 | 10 | 0 | 1 | 0 |
| US-UI-03 | 8 | 7 | 0 | 1 | 0 |
| US-UI-04 | 8 | 8 | 0 | 0 | 0 |
| US-UI-05 | 8 | 8 | 0 | 0 | 0 |
| US-UI-06 | 6 | 6 | 0 | 0 | 0 |
| US-UI-07 | 9 | 8 | 0 | 1 | 0 |

## Kết quả từng ca
| TC | KQ | Ghi chú |
| --- | --- | --- |
| TC-UI1-01 | LT | trang `/dev/panels` + `panels-variants.spec.ts` không còn (chỉ còn `src/app/dev/ui`): khung §23 của 10.7 thay panel nổi (P-107-*) |
| TC-UI1-02 | LT | trang `/dev/panels` + `panels-variants.spec.ts` không còn (chỉ còn `src/app/dev/ui`): khung §23 của 10.7 thay panel nổi (P-107-*) |
| TC-UI1-03 | LT | trang `/dev/panels` + `panels-variants.spec.ts` không còn (chỉ còn `src/app/dev/ui`): khung §23 của 10.7 thay panel nổi (P-107-*) |
| TC-UI1-04 | P | 18 ảnh `docs/sprints/5.5/handoff/ui-01/*.png` (ls / wc -l = 18) |
| TC-UI1-05 | P | D59 = 0c3110b7; gốc UI-01 = 2f6e1d08; 28 commit chạm tệp cấm trong 2f6e1d08..HEAD đều có D59 là tổ tiên (merge-base --is-ancestor): 28/28 ok, 0 SAI |
| TC-UI1-06 | P | `grep -c D59 docs/DECISIONS.md` = 1 (quyết định D59 có ghi) |
| TC-UI1-07 | LT | `docs/design/DESIGN.md` đã viết lại theo §23 (commit 8999086f, 9bba4cb7… của 10.7/10.8): bảng 4.1 dòng cũ→mới không còn đối chiếu được |
| TC-UI1-08 | P | ui-antipatterns.sh: 32 ✓ (TC gốc 22 phép; thêm phép do sprint sau), rc=0, --selftest 32/32 |
| TC-UI2-01 | P | `grep -rn ep-paper frontend/src frontend/e2e docs/design` = 0 dòng |
| TC-UI2-02 | P | `ep-radius-panel` chỉ ở tokens.css + shared/ui/Panel.module.css; `ep-elevation-1` cùng hai tệp |
| TC-UI2-03 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts 59/59 desktop pass |
| TC-UI2-04 | P | ui-antipatterns.sh: 32 ✓ (TC gốc 22 phép; thêm phép do sprint sau), rc=0, --selftest 32/32 (phép Panel lồng Panel ✓); panels.spec.ts pass (NEST = 0) |
| TC-UI2-05 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts 59/59 desktop pass |
| TC-UI2-06 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): contrast.spec.ts 4/4 pass |
| TC-UI2-07 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts 59/59 desktop pass |
| TC-UI2-08 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts 59/59 desktop pass |
| TC-UI2-09 | P | ui-antipatterns.sh: 32 ✓ (TC gốc 22 phép; thêm phép do sprint sau), rc=0, --selftest 32/32 |
| TC-UI2-10 | LT | tiêu chí +2 KB so sprint 5 thay bằng trần lhci script ≤ 256000: JS truyền (lhci sim 5 lượt) /chat 233.683, /threads 224.921, /inbox 222.989, /gradebook 248.449, /settings/llm 232.262 B, đều ≤ 256000; gradebook +13 KB so sprint 5 (9.x/10.x) |
| TC-UI2-11 | P | DESIGN.md §10.14 "Panel / PanelSection" có; `docs/sprints/5.5/handoff/dev-US-UI-02.md` có |
| TC-UI3-01 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): shell.spec.ts 22/23 pass (1 skip), lcp.spec.ts 8/8, contrast sidebar pass |
| TC-UI3-02 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): shell.spec.ts 22/23 pass (1 skip), lcp.spec.ts 8/8, contrast sidebar pass |
| TC-UI3-03 | LT | AC3 sửa ở 10.7 (f87bba13, P-107-1): LEFT h1 = 240 (work) / 428 (read) ở 1440, 16 ở 390, 12 ở 375. Đo phiên thật ep-r 6 route: 1440 → 428 (/ SV) và 240; 1024 → 148/96; 375 → 12; AUDIT ox 0, cut 0 (sau khi loại <option> — lỗi script 1.5) |
| TC-UI3-04 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): shell.spec.ts 22/23 pass (1 skip), lcp.spec.ts 8/8, contrast sidebar pass |
| TC-UI3-05 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts "auth shell" 6/6; account.spec.ts 30/30, class-join.spec.ts 45/46 (1 skip) |
| TC-UI3-06 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): shell.spec.ts 22/23 pass (1 skip), lcp.spec.ts 8/8, contrast sidebar pass |
| TC-UI3-07 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): shell.spec.ts 22/23 pass (1 skip), lcp.spec.ts 8/8, contrast sidebar pass |
| TC-UI3-08 | P | 12 ảnh ui-03 (ls = 12); commit 9b4f531d đổi đúng 6 ảnh mốc = 6 hàng handoff; RERUN visual.spec.ts trong image Playwright v1.63.0-noble 2 lần: 26 passed + 26 passed (14 → 26 ảnh do sprint sau), không --update-snapshots |
| TC-UI4-01 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts student home/routes/exams student/mock routes/mobile gutter pass; exam.spec.ts 40/40; account 30/30; class-join 45/46; ui-antipatterns.sh: 32 ✓ (TC gốc 22 phép; thêm phép do sprint sau), rc=0, --selftest 32/32 |
| TC-UI4-02 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts student home/routes/exams student/mock routes/mobile gutter pass; exam.spec.ts 40/40; account 30/30; class-join 45/46; ui-antipatterns.sh: 32 ✓ (TC gốc 22 phép; thêm phép do sprint sau), rc=0, --selftest 32/32 |
| TC-UI4-03 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts student home/routes/exams student/mock routes/mobile gutter pass; exam.spec.ts 40/40; account 30/30; class-join 45/46; ui-antipatterns.sh: 32 ✓ (TC gốc 22 phép; thêm phép do sprint sau), rc=0, --selftest 32/32 |
| TC-UI4-04 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts student home/routes/exams student/mock routes/mobile gutter pass; exam.spec.ts 40/40; account 30/30; class-join 45/46; ui-antipatterns.sh: 32 ✓ (TC gốc 22 phép; thêm phép do sprint sau), rc=0, --selftest 32/32 |
| TC-UI4-05 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts student home/routes/exams student/mock routes/mobile gutter pass; exam.spec.ts 40/40; account 30/30; class-join 45/46; ui-antipatterns.sh: 32 ✓ (TC gốc 22 phép; thêm phép do sprint sau), rc=0, --selftest 32/32 |
| TC-UI4-06 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts student home/routes/exams student/mock routes/mobile gutter pass; exam.spec.ts 40/40; account 30/30; class-join 45/46; ui-antipatterns.sh: 32 ✓ (TC gốc 22 phép; thêm phép do sprint sau), rc=0, --selftest 32/32 |
| TC-UI4-07 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts student home/routes/exams student/mock routes/mobile gutter pass; exam.spec.ts 40/40; account 30/30; class-join 45/46; ui-antipatterns.sh: 32 ✓ (TC gốc 22 phép; thêm phép do sprint sau), rc=0, --selftest 32/32 |
| TC-UI4-08 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts student home/routes/exams student/mock routes/mobile gutter pass; exam.spec.ts 40/40; account 30/30; class-join 45/46; ui-antipatterns.sh: 32 ✓ (TC gốc 22 phép; thêm phép do sprint sau), rc=0, --selftest 32/32 |
| TC-UI5-01 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts staff home/members/exam editor/questions/staff routes pass; exam.spec.ts 40/40; today 19/21 (2 skip); shell 22/23; ui-antipatterns.sh: 32 ✓ (TC gốc 22 phép; thêm phép do sprint sau), rc=0, --selftest 32/32 |
| TC-UI5-02 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts staff home/members/exam editor/questions/staff routes pass; exam.spec.ts 40/40; today 19/21 (2 skip); shell 22/23; ui-antipatterns.sh: 32 ✓ (TC gốc 22 phép; thêm phép do sprint sau), rc=0, --selftest 32/32 |
| TC-UI5-03 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts staff home/members/exam editor/questions/staff routes pass; exam.spec.ts 40/40; today 19/21 (2 skip); shell 22/23; ui-antipatterns.sh: 32 ✓ (TC gốc 22 phép; thêm phép do sprint sau), rc=0, --selftest 32/32 |
| TC-UI5-04 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts staff home/members/exam editor/questions/staff routes pass; exam.spec.ts 40/40; today 19/21 (2 skip); shell 22/23; ui-antipatterns.sh: 32 ✓ (TC gốc 22 phép; thêm phép do sprint sau), rc=0, --selftest 32/32 |
| TC-UI5-05 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts staff home/members/exam editor/questions/staff routes pass; exam.spec.ts 40/40; today 19/21 (2 skip); shell 22/23; ui-antipatterns.sh: 32 ✓ (TC gốc 22 phép; thêm phép do sprint sau), rc=0, --selftest 32/32 |
| TC-UI5-06 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts staff home/members/exam editor/questions/staff routes pass; exam.spec.ts 40/40; today 19/21 (2 skip); shell 22/23; ui-antipatterns.sh: 32 ✓ (TC gốc 22 phép; thêm phép do sprint sau), rc=0, --selftest 32/32 |
| TC-UI5-07 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts staff home/members/exam editor/questions/staff routes pass; exam.spec.ts 40/40; today 19/21 (2 skip); shell 22/23; ui-antipatterns.sh: 32 ✓ (TC gốc 22 phép; thêm phép do sprint sau), rc=0, --selftest 32/32 |
| TC-UI5-08 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts staff home/members/exam editor/questions/staff routes pass; exam.spec.ts 40/40; today 19/21 (2 skip); shell 22/23; ui-antipatterns.sh: 32 ✓ (TC gốc 22 phép; thêm phép do sprint sau), rc=0, --selftest 32/32 |
| TC-UI6-01 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts admin users/courses/settings llm/admin routes pass; a11y.spec.ts 14/14 (0 critical/serious); settings-llm 14/14; shell nav per role pass |
| TC-UI6-02 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts admin users/courses/settings llm/admin routes pass; a11y.spec.ts 14/14 (0 critical/serious); settings-llm 14/14; shell nav per role pass |
| TC-UI6-03 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts admin users/courses/settings llm/admin routes pass; a11y.spec.ts 14/14 (0 critical/serious); settings-llm 14/14; shell nav per role pass |
| TC-UI6-04 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts admin users/courses/settings llm/admin routes pass; a11y.spec.ts 14/14 (0 critical/serious); settings-llm 14/14; shell nav per role pass |
| TC-UI6-05 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts admin users/courses/settings llm/admin routes pass; a11y.spec.ts 14/14 (0 critical/serious); settings-llm 14/14; shell nav per role pass |
| TC-UI6-06 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts admin users/courses/settings llm/admin routes pass; a11y.spec.ts 14/14 (0 critical/serious); settings-llm 14/14; shell nav per role pass |
| TC-UI7-01 | P | `ls ui-07/before/*.png ui-07/after/*.png / wc -l` = 78 (39 + 39) |
| TC-UI7-02 | P | RERUN visual.spec.ts trong mcr.microsoft.com/playwright:v1.63.0-noble ×2, không --update-snapshots: 26 passed / 26 passed; git diff --name-only 2f6e1d08 c2fc55cb ảnh mốc = 14 tệp = danh sách ảnh dev-US-UI-02 (khớp 1:1) |
| TC-UI7-03 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): a11y.spec.ts 14/14, contrast.spec.ts 4/4 pass; axe-allow.json = [] (0 mục) |
| TC-UI7-04 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): panels.spec.ts 59/59; ui-antipatterns.sh: 32 ✓ (TC gốc 22 phép; thêm phép do sprint sau), rc=0, --selftest 32/32 |
| TC-UI7-05 | P | lhci autorun bản dựng cổng: lighthouserc.json 9 URL × 5 rc=0 (TBT 24–60 ms, JS ≤ 248 KB, CLS ≤ 0,001); devtools 8 URL × 5 rc=0 (LCP 1448–2334; /dev/ui 2965 warn); diff cấu hình 5.5: 0 dòng nới ngưỡng |
| TC-UI7-06 | P | chạy lại ep-r @aba738ef (ngày 2026-11-21, qc3): pnpm -C frontend lint rc=0 (0 lỗi, 1 cảnh báo ở frontend/dbg91b.mjs); 32 lỗi cũ ở frontend/e2e/qc2-* không còn vì spec QC đã chuyển sang docs/regression/specs-qc2 (DEF-R5.5-01 = DEF-R5-12 đóng). Log: evidence-5/lint-aba738ef.txt |
| TC-UI7-07 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): shell nav per role + panels wall/strong pass |
| TC-UI7-08 | P | spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile): shell nav per role + panels wall/strong pass |
| TC-UI7-09 | LT | handoff dev-US-UI-07.md có; PROGRESS dòng 24, 55 có; grep "sprint 5.5"/"UI-0[1-7]" = 0 và không có dòng "Chủ dự án đã xem… gật"; cổng duyệt bị thay bằng quyết định chủ dự án 14/11 «Nền trắng» (P-107-10, D59 sửa ở 10.7) |

## Trượt
TC-UI7-06: `pnpm -C frontend lint` rc=1 do 10 tệp QC đã commit `frontend/e2e/qc2-*` (mã sản phẩm sạch); chi tiết `defects-draft-perf.md`. 0 FAIL trong 388 ca spec mock.

## Ca K và lý do
K (a/b/c) = 0. Ca "Chưa chạy" cần Lighthouse (lhci) / image Playwright docker (ảnh mốc) / pnpm lint+build / LEFT_SRC phiên thật — thuộc việc môi trường, phải chạy nốt (không phải K).

## Kết quả spec mock (Playwright, bản dựng cổng, --workers=1)
| spec | kết quả |
| --- | --- |
| a11y.spec.ts | {'desktop:passed': 14, 'mobile:skipped': 14} |
| account.spec.ts | {'desktop:passed': 30, 'mobile:passed': 24, 'mobile:skipped': 6} |
| analytics-exam.spec.ts | {'desktop:passed': 9, 'mobile:passed': 10, 'desktop:skipped': 1} |
| class-join.spec.ts | {'desktop:passed': 45, 'mobile:passed': 32, 'mobile:skipped': 14, 'desktop:skipped': 1} |
| contrast.spec.ts | {'desktop:passed': 4, 'mobile:skipped': 4} |
| exam.spec.ts | {'desktop:passed': 40, 'mobile:passed': 37, 'mobile:skipped': 3} |
| lcp.spec.ts | {'desktop:passed': 8, 'mobile:skipped': 8} |
| panels.spec.ts | {'desktop:passed': 59, 'mobile:skipped': 59} |
| settings-llm.spec.ts | {'desktop:passed': 14, 'mobile:skipped': 14} |
| shell.spec.ts | {'desktop:passed': 22, 'mobile:skipped': 23, 'desktop:skipped': 1} |
| today.spec.ts | {'desktop:passed': 19, 'mobile:passed': 21, 'desktop:skipped': 2} |

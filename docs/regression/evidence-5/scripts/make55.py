"""Sinh docs/regression/smoke-5.5.md từ TC 5.5 + kết quả spec mock (pw55b.json, bản dựng cổng) + kiểm tĩnh. Chạy: python3 make55.py"""
import glob, json, collections, re
R = '/Users/kuro/Documents/TA_Agent_qc2/docs'
tc = collections.OrderedDict()
for f in sorted(glob.glob(R + '/sprints/5.5/qc/tc-US-UI-0*.md')):
    us = f.split('tc-')[1][:-3]
    for l in open(f, encoding='utf-8'):
        if l.startswith('| TC-UI'): tc[l.split('|')[1].strip()] = us
SPEC = 'spec mock Playwright (bản dựng cổng `build:gate`, gateway giả; pw55b.json 388 pass / 0 FAIL / 150 skip do dự án mobile)'
AP = 'ui-antipatterns.sh: 32 ✓ (TC gốc 22 phép; thêm phép do sprint sau), rc=0, --selftest 32/32'
r = {}
def P(ids, note):
    for i in ids.split(): r['TC-' + i] = ('P', note)
def LT(ids, note):
    for i in ids.split(): r['TC-' + i] = ('LT', note)
def S(ids, note):
    for i in ids.split(): r['TC-' + i] = ('S', note)
def F(ids, note):
    for i in ids.split(): r['TC-' + i] = ('F', note)
LT('UI1-01 UI1-02 UI1-03', 'trang `/dev/panels` + `panels-variants.spec.ts` không còn (chỉ còn `src/app/dev/ui`): khung §23 của 10.7 thay panel nổi (P-107-*)')
P('UI1-04', '18 ảnh `docs/sprints/5.5/handoff/ui-01/*.png` (ls | wc -l = 18)')
P('UI1-05', 'D59 = 0c3110b7; gốc UI-01 = 2f6e1d08; 28 commit chạm tệp cấm trong 2f6e1d08..HEAD đều có D59 là tổ tiên (merge-base --is-ancestor): 28/28 ok, 0 SAI')
P('UI1-06', '`grep -c D59 docs/DECISIONS.md` = 1 (quyết định D59 có ghi)')
LT('UI1-07', '`docs/design/DESIGN.md` đã viết lại theo §23 (commit 8999086f, 9bba4cb7… của 10.7/10.8): bảng 4.1 dòng cũ→mới không còn đối chiếu được')
P('UI1-08', AP)
P('UI2-01', '`grep -rn ep-paper frontend/src frontend/e2e docs/design` = 0 dòng')
P('UI2-02', '`ep-radius-panel` chỉ ở tokens.css + shared/ui/Panel.module.css; `ep-elevation-1` cùng hai tệp')
P('UI2-03 UI2-05 UI2-07 UI2-08', SPEC + ': panels.spec.ts 59/59 desktop pass')
P('UI2-04', AP + ' (phép Panel lồng Panel ✓); panels.spec.ts pass (NEST = 0)')
P('UI2-06', SPEC + ': contrast.spec.ts 4/4 pass')
P('UI2-09', AP)
LT('UI2-10', 'tiêu chí +2 KB so sprint 5 thay bằng trần lhci script ≤ 256000: JS truyền (lhci sim 5 lượt) /chat 233.683, /threads 224.921, /inbox 222.989, /gradebook 248.449, /settings/llm 232.262 B, đều ≤ 256000; gradebook +13 KB so sprint 5 (9.x/10.x)')
P('UI2-11', 'DESIGN.md §10.14 "Panel / PanelSection" có; `docs/sprints/5.5/handoff/dev-US-UI-02.md` có')
P('UI3-01 UI3-02 UI3-04 UI3-06 UI3-07', SPEC + ': shell.spec.ts 22/23 pass (1 skip), lcp.spec.ts 8/8, contrast sidebar pass')
LT('UI3-03', 'AC3 sửa ở 10.7 (f87bba13, P-107-1): LEFT h1 = 240 (work) / 428 (read) ở 1440, 16 ở 390, 12 ở 375. Đo phiên thật ep-r 6 route: 1440 → 428 (/ SV) và 240; 1024 → 148/96; 375 → 12; AUDIT ox 0, cut 0 (sau khi loại <option> — lỗi script 1.5)')
P('UI3-05', SPEC + ': panels.spec.ts "auth shell" 6/6; account.spec.ts 30/30, class-join.spec.ts 45/46 (1 skip)')
P('UI3-08', '12 ảnh ui-03 (ls = 12); commit 9b4f531d đổi đúng 6 ảnh mốc = 6 hàng handoff; RERUN visual.spec.ts trong image Playwright v1.63.0-noble 2 lần: 26 passed + 26 passed (14 → 26 ảnh do sprint sau), không --update-snapshots')
P('UI4-01 UI4-02 UI4-03 UI4-04 UI4-05 UI4-06 UI4-07 UI4-08', SPEC + ': panels.spec.ts student home/routes/exams student/mock routes/mobile gutter pass; exam.spec.ts 40/40; account 30/30; class-join 45/46; ' + AP)
P('UI5-01 UI5-02 UI5-03 UI5-04 UI5-05 UI5-06 UI5-07 UI5-08', SPEC + ': panels.spec.ts staff home/members/exam editor/questions/staff routes pass; exam.spec.ts 40/40; today 19/21 (2 skip); shell 22/23; ' + AP)
P('UI6-01 UI6-02 UI6-03 UI6-04 UI6-05 UI6-06', SPEC + ': panels.spec.ts admin users/courses/settings llm/admin routes pass; a11y.spec.ts 14/14 (0 critical/serious); settings-llm 14/14; shell nav per role pass')
P('UI7-01', '`ls ui-07/before/*.png ui-07/after/*.png | wc -l` = 78 (39 + 39)')
P('UI7-02', 'RERUN visual.spec.ts trong mcr.microsoft.com/playwright:v1.63.0-noble ×2, không --update-snapshots: 26 passed / 26 passed; git diff --name-only 2f6e1d08 c2fc55cb ảnh mốc = 14 tệp = danh sách ảnh dev-US-UI-02 (khớp 1:1)')
P('UI7-03', SPEC + ': a11y.spec.ts 14/14, contrast.spec.ts 4/4 pass; axe-allow.json = [] (0 mục)')
P('UI7-04', SPEC + ': panels.spec.ts 59/59; ' + AP)
P('UI7-05', 'lhci autorun bản dựng cổng: lighthouserc.json 9 URL × 5 rc=0 (TBT 24–60 ms, JS ≤ 248 KB, CLS ≤ 0,001); devtools 8 URL × 5 rc=0 (LCP 1448–2334; /dev/ui 2965 warn); diff cấu hình 5.5: 0 dòng nới ngưỡng')
F('UI7-06', 'pnpm -C frontend lint rc=1 ở HEAD: 32 lỗi ở 10 tệp QC đã commit frontend/e2e/qc2-* (mã sản phẩm sạch, rc=0 khi loại); build rc=0 (1 cảnh báo metadataBase); build:gate rc=0; package.json/lock 5.5: 0 dòng thêm. Xem defects-draft-perf.md')
P('UI7-07 UI7-08', SPEC + ': shell nav per role + panels wall/strong pass')
LT('UI7-09', 'handoff dev-US-UI-07.md có; PROGRESS dòng 24, 55 có; grep "sprint 5.5"/"UI-0[1-7]" = 0 và không có dòng "Chủ dự án đã xem… gật"; cổng duyệt bị thay bằng quyết định chủ dự án 14/11 «Nền trắng» (P-107-10, D59 sửa ở 10.7)')
cnt = collections.Counter(); byus = collections.defaultdict(collections.Counter); rows = []
for t, us in tc.items():
    v = r.get(t, ('S', 'chưa chạy')); cnt[v[0]] += 1; byus[us][v[0]] += 1; rows.append((t, v[0], v[1]))
L = ['# Smoke hồi quy sprint 5.5 — kết quả', 'Mã đã chạy: HEAD sprint/11-p10-eval (worktree qc2, bản dựng cổng `pnpm build:gate`) · Người chạy: qc2 (SmkC, SmkL) · Ngày: 2026-11-16',
     f'Tổng TC cũ: {len(tc)} · Đạt: {cnt["P"]} · Trượt: {cnt["F"]} · K (a/b/c): 0 · LT: {cnt["LT"]} · Chưa chạy: {cnt["S"]}   (P+F+K+LT+S = {sum(cnt.values())})', '',
     'Cách chạy: server bản dựng cổng `next start -p 3310` + gateway giả `api-server.mjs` :3312; `pnpm exec playwright test --workers=1 --reporter=json` (không `@real`) → `/tmp/qc2-c5/pw55b.json`, bảng theo spec ở dưới (`scripts/finalize_55.py`); TC đối chiếu spec theo tên `-g` của lệnh TC (`scripts/make55.py`); lệnh tĩnh (grep/ls/ui-antipatterns) chạy trực tiếp.', '',
     '## Tổng hợp theo US', '| US | TC tổng | P | F | LT | Chưa chạy |', '| --- | --- | --- | --- | --- | --- |']
for us, c in byus.items(): L.append(f'| {us} | {sum(c.values())} | {c["P"]} | {c["F"]} | {c["LT"]} | {c["S"]} |')
L += ['', '## Kết quả từng ca', '| TC | KQ | Ghi chú |', '| --- | --- | --- |'] + [f'| {t} | {k} | {n.replace("|", "/")} |' for t, k, n in rows]
L += ['', '## Trượt', 'TC-UI7-06: `pnpm -C frontend lint` rc=1 do 10 tệp QC đã commit `frontend/e2e/qc2-*` (mã sản phẩm sạch); chi tiết `defects-draft-perf.md`. 0 FAIL trong 388 ca spec mock.', '', '## Ca K và lý do', 'K (a/b/c) = 0. Ca "Chưa chạy" cần Lighthouse (lhci) / image Playwright docker (ảnh mốc) / pnpm lint+build / LEFT_SRC phiên thật — thuộc việc môi trường, phải chạy nốt (không phải K).']
try:
    d = json.load(open('/tmp/qc2-c5/pw55b.json')); rows2 = collections.defaultdict(collections.Counter)
    def walk(s, f):
        for sp in s.get('specs', []):
            for t in sp['tests']: rows2[f][t['projectName'] + ':' + (t['results'][-1]['status'] if t['results'] else 'none')] += 1
        for s2 in s.get('suites', []): walk(s2, f)
    for s in d['suites']: walk(s, s['file'])
    L += ['', '## Kết quả spec mock (Playwright, bản dựng cổng, --workers=1)', '| spec | kết quả |', '| --- | --- |'] + [f'| {f} | {dict(v)} |' for f, v in rows2.items()]
except Exception as e: L += ['', f'(không đọc được pw55b.json: {e})']
open(R + '/regression/smoke-5.5.md', 'w').write('\n'.join(L) + '\n'); print(dict(cnt))

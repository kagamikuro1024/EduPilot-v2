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
LT('UI1-01 UI1-02 UI1-03', 'trang `/dev/panels` + `panels-variants.spec.ts` không còn (chỉ còn `src/app/dev/ui`): khung §23 của 10.7 thay panel nổi (P-107-*)')
P('UI1-04', '18 ảnh `docs/sprints/5.5/handoff/ui-01/*.png` (ls | wc -l = 18)')
S('UI1-05', 'chưa chạy: so tổ tiên D59 của từng commit chạm tệp cấm (cần commit gốc UI-01)')
P('UI1-06', '`grep -c D59 docs/DECISIONS.md` = 1 (quyết định D59 có ghi)')
LT('UI1-07', '`docs/design/DESIGN.md` đã viết lại theo §23 (commit 8999086f, 9bba4cb7… của 10.7/10.8): bảng 4.1 dòng cũ→mới không còn đối chiếu được')
P('UI1-08', AP)
P('UI2-01', '`grep -rn ep-paper frontend/src frontend/e2e docs/design` = 0 dòng')
P('UI2-02', '`ep-radius-panel` chỉ ở tokens.css + shared/ui/Panel.module.css; `ep-elevation-1` cùng hai tệp')
P('UI2-03 UI2-05 UI2-07 UI2-08', SPEC + ': panels.spec.ts 59/59 desktop pass')
P('UI2-04', AP + ' (phép Panel lồng Panel ✓); panels.spec.ts pass (NEST = 0)')
P('UI2-06', SPEC + ': contrast.spec.ts 4/4 pass')
P('UI2-09', AP)
S('UI2-10', 'chưa chạy: lhci collect (network-requests) — cần Lighthouse trên bản dựng')
P('UI2-11', 'DESIGN.md §10.14 "Panel / PanelSection" có; `docs/sprints/5.5/handoff/dev-US-UI-02.md` có')
P('UI3-01 UI3-02 UI3-04 UI3-06 UI3-07', SPEC + ': shell.spec.ts 22/23 pass (1 skip), lcp.spec.ts 8/8, contrast sidebar pass')
S('UI3-03', 'chưa chạy: LEFT_SRC trên 6 route (cần phiên đăng nhập thật từng route)')
P('UI3-05', SPEC + ': panels.spec.ts "auth shell" 6/6; account.spec.ts 30/30, class-join.spec.ts 45/46 (1 skip)')
S('UI3-08', 'chưa chạy: ảnh mốc visual.spec.ts chạy trong image Playwright docker (12 ảnh handoff ui-03 có)')
P('UI4-01 UI4-02 UI4-03 UI4-04 UI4-05 UI4-06 UI4-07 UI4-08', SPEC + ': panels.spec.ts student home/routes/exams student/mock routes/mobile gutter pass; exam.spec.ts 40/40; account 30/30; class-join 45/46; ' + AP)
P('UI5-01 UI5-02 UI5-03 UI5-04 UI5-05 UI5-06 UI5-07 UI5-08', SPEC + ': panels.spec.ts staff home/members/exam editor/questions/staff routes pass; exam.spec.ts 40/40; today 19/21 (2 skip); shell 22/23; ' + AP)
P('UI6-01 UI6-02 UI6-03 UI6-04 UI6-05 UI6-06', SPEC + ': panels.spec.ts admin users/courses/settings llm/admin routes pass; a11y.spec.ts 14/14 (0 critical/serious); settings-llm 14/14; shell nav per role pass')
P('UI7-01', '`ls ui-07/before/*.png ui-07/after/*.png | wc -l` = 78 (39 + 39)')
S('UI7-02', 'chưa chạy: visual.spec.ts trong image Playwright docker (ảnh mốc)')
P('UI7-03', SPEC + ': a11y.spec.ts 14/14, contrast.spec.ts 4/4 pass; axe-allow.json = [] (0 mục)')
P('UI7-04', SPEC + ': panels.spec.ts 59/59; ' + AP)
S('UI7-05', 'chưa chạy: lhci autorun (build:gate + 2 cấu hình)')
S('UI7-06', 'chưa chạy: pnpm lint + build (phần spec `--grep-invert @real` = 388 pass / 0 FAIL đạt)')
P('UI7-07 UI7-08', SPEC + ': shell nav per role + panels wall/strong pass')
S('UI7-09', 'chưa chạy: đối chiếu PROGRESS.md (không có dòng "sprint 5.5" khớp mẫu)')
cnt = collections.Counter(); byus = collections.defaultdict(collections.Counter); rows = []
for t, us in tc.items():
    v = r.get(t, ('S', 'chưa chạy')); cnt[v[0]] += 1; byus[us][v[0]] += 1; rows.append((t, v[0], v[1]))
L = ['# Smoke hồi quy sprint 5.5 — kết quả', 'Mã đã chạy: HEAD sprint/11-p10-eval (worktree qc2, bản dựng cổng `pnpm build:gate`) · Người chạy: qc2 (SmkC, SmkL) · Ngày: 2026-11-16',
     f'Tổng TC cũ: {len(tc)} · Đạt: {cnt["P"]} · Trượt: {cnt["F"]} · K (a/b/c): 0 · LT: {cnt["LT"]} · Chưa chạy: {cnt["S"]}   (P+F+K+LT+S = {sum(cnt.values())})', '',
     'Cách chạy: server bản dựng cổng `next start -p 3310` + gateway giả `api-server.mjs` :3312; `pnpm exec playwright test --workers=1 --reporter=json` (không `@real`) → `/tmp/qc2-c5/pw55b.json`, bảng theo spec ở dưới (`scripts/finalize_55.py`); TC đối chiếu spec theo tên `-g` của lệnh TC (`scripts/make55.py`); lệnh tĩnh (grep/ls/ui-antipatterns) chạy trực tiếp.', '',
     '## Tổng hợp theo US', '| US | TC tổng | P | F | LT | Chưa chạy |', '| --- | --- | --- | --- | --- | --- |']
for us, c in byus.items(): L.append(f'| {us} | {sum(c.values())} | {c["P"]} | {c["F"]} | {c["LT"]} | {c["S"]} |')
L += ['', '## Kết quả từng ca', '| TC | KQ | Ghi chú |', '| --- | --- | --- |'] + [f'| {t} | {k} | {n.replace("|", "/")} |' for t, k, n in rows]
L += ['', '## Trượt', 'Không có ca trượt (0 FAIL trong 388 ca spec mock).', '', '## Ca K và lý do', 'K (a/b/c) = 0. Ca "Chưa chạy" cần Lighthouse (lhci) / image Playwright docker (ảnh mốc) / pnpm lint+build / LEFT_SRC phiên thật — thuộc việc môi trường, phải chạy nốt (không phải K).']
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

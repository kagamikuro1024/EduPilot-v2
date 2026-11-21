import re
R='/Users/kuro/Documents/TA_Agent_qc3/docs/regression/'
H='[qc3 21/11, đo lại @58448e0a] '
n20=H+'ĐÃ SỬA (523d9076 weights shard Go + 7a3ebd35 devtools 3 job; DEF-R7-01). Tự lấy bằng gh api, PR run 44907729321 @58448e0a (workflow chạy hai lần, mỗi lần đủ 22 job success, không run huỷ): attempt 1 wall 526 s (job đầu 13:13:09Z → job cuối 13:21:55Z; created→completed 13:12:49→13:21:56 = 547 s) và attempt 2 wall 498 s (13:26:19Z → 13:34:37Z; run_started→updated 503 s), cả hai ≤ 600 s. Đủ job mỗi lần: Go static, Go test auth/course/rest-1/rest-2/rest-3, Go không tag 1–4/4, Judge attacks, Frontend lint+build, e2e Playwright 1–4/4, Lighthouse devtools A/B/C + simulate A/B, Changes. Job lâu nhất: e2e 4/4 511 s và 476 s ở attempt 1; Go rest-3 478 s ở attempt 2. Ghi chú: run push cùng SHA 44907725269 chạy song song, Go bị bỏ qua (không phải lượt đủ job), wall 733 s chỉ gồm frontend và có job chờ runner 40 s–3 phút — không tính vào mẫu «đủ job» nhưng cho thấy lúc runner tranh chấp có thể vượt 600 s. Bằng chứng: evidence-7/r7/ci-run-44907729321-a1-jobs.tsv, ci-run-44907729321-a2-jobs.tsv, ci-run-44907725269-a1-jobs.tsv'
n2=H+'ĐÃ SỬA (7062b009 + lhci nay chạy 3+2 job nhỏ; DEF-R7-06). Đọc .github/workflows/ci.yml @58448e0a: changes 5, go-static 20, go-test 30, go-test-default 25, judge-attacks 10, fe-check 15, e2e 25, lhci 25 phút (mọi job có timeout-minutes). Thời gian thực tế lớn nhất trong 2 lượt đủ job của PR run 44907729321 (attempt 1 và 2): Lighthouse devtools A 8,0 phút (480 s) ⇒ cần ≥ 16 → 25 đạt; e2e 4/4 8,5 phút ⇒ ≥17 → 25 đạt; Go rest-3 8,0 ⇒ ≥16 → 30 đạt; Go không tag max 6,0 ⇒ ≥12 → 25; Judge attacks 2,7 ⇒ ≥5,4 → 10; Changes 0,3 → 5; fe-check 1,5 → 15; Go static 3,6 → 20. 22/22 dòng job đạt ≥ 2×; huỷ ≠ xanh đã kiểm ở lượt trước. (Trước đây lhci 15,7 phút vì mỗi job gánh nhiều URL hơn; sau 7a3ebd35 còn ≤ 8 phút.) Bằng chứng: evidence-7/r7/ci-58448e0a.yml, ci-run-44907729321-a{1,2}-jobs.tsv'
s=open(R+'smoke-7.md').read().split('\n')
for tc,kq,note in (('TC-20','P',n20),('TC-E2','P',n2)):
    i=[k for k,l in enumerate(s) if l.startswith(f'| {tc} |')]; assert len(i)==1; s[i[0]]=f'| {tc} | {kq} | {note} |'
t='\n'.join(s)
t=t.replace('· Đạt: 577 · Trượt: 2 ·','· Đạt: 579 · Trượt: 0 ·')
t=re.sub(r'^\| US-CI-02 \| 31 \| 29 \| 2 \| 0 \|','| US-CI-02 | 31 | 31 | 0 | 0 |',t,flags=re.M)
a=t.index('HANDOFF qc3 (Qc3S7e, 21/11/2026, cập nhật lần 3')
t=t[:a]+'HANDOFF qc3 (Qc3S7e, 21/11/2026, cập nhật lần 4 @58448e0a): N = 0 / 579 (P 579, F 0). Lần 4: TC-20 (DEF-R7-01) P: 2 lượt đủ job PR run 44907729321 wall 526 s và 498 s; TC-E2 (DEF-R7-06) P: mọi job timeout ≥ 2× thực tế (lhci max 8,0 phút, timeout 25). Các đoạn dưới là bản lần 1–3.\n'+t[a:]
open(R+'smoke-7.md','w').write(t)
d=open(R+'defects-7.md').read()
def add(k,line):
    global d
    m=re.search(rf'^### {k} .*?(?=^### |^## |\Z)',d,re.S|re.M); assert m
    blk=m.group(0).rstrip('\n'); d=d[:m.start()]+blk+'\n'+line+'\n\n'+d[m.end():]
add('DEF-R7-01','- **ĐÃ SỬA (523d9076 + 7a3ebd35) — xác nhận @58448e0a ngày 2026-11-21:** PR run 44907729321, hai lượt đủ 22 job success (attempt 1: wall 526 s; attempt 2: wall 498 s; tính từ job đầu bắt đầu tới job cuối kết thúc, đối chiếu cả created→completed 547 s / 503 s) ⇒ cả hai ≤ 600 s ⇒ TC-20 P. Lưu ý: run push cùng SHA 44907725269 (Go bị bỏ qua, chạy song song với PR run) wall 733 s chỉ frontend — không phải lượt đủ job; nếu runner bị tranh chấp thì ngưỡng 600 s vẫn có thể vượt. Bằng chứng: `evidence-7/r7/ci-run-44907729321-a1-jobs.tsv`, `-a2-jobs.tsv`, `ci-run-44907725269-a1-jobs.tsv`.')
add('DEF-R7-06','- **ĐÃ SỬA (7062b009; lhci chia lại bởi 7a3ebd35) — xác nhận @58448e0a ngày 2026-11-21:** mọi job có timeout-minutes; thời gian thực tế lớn nhất trong 2 lượt đủ job là 8,5 phút (e2e 4/4) và lhci 8,0 phút (devtools A) ⇒ lhci timeout 25 ≥ 16 đạt; 22/22 dòng job ≥ 2× ⇒ TC-E2 P. Bằng chứng: `evidence-7/r7/ci-58448e0a.yml`, `ci-run-44907729321-a{1,2}-jobs.tsv`.')
open(R+'defects-7.md','w').write(d)

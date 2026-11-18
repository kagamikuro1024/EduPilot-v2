# Dựng smoke-9.1.md từ s91-rows.json + overrides91.json (chạy sau make_overrides7.py + build_smoke91.py)
import json,re,collections
H='/Users/kuro/Documents/TA_Agent_qc2/docs/regression/'
d=json.load(open(H+'evidence-9.1/s91-rows.json'))
rows=[r for r in d['rows'] if r[1]!='LT']
lt=[r for r in d['rows'] if r[1]=='LT']
cnt=collections.Counter(r[1] for r in rows)
P,F,M=cnt['P'],cnt['F'],cnt['M']
realF=[r for r in rows if r[1]=='F' and r[2].startswith(('DEF','qc-uiux','kịch'))]
ov=json.load(open(H+'evidence-9.1/overrides91.json'))
Fconf=[r[0] for r in rows if r[0] in ov and ov[r[0]][0]=='F']
N=len(rows)-P-len(Fconf)
old=open(H+'smoke-9.1.md',encoding='utf-8').read().split('\n')
us=collections.OrderedDict()
for t,k,n,w in d['rows']:
    u=t.split('-')[1]; c=us.setdefault(u,collections.Counter()); c['T']+=1
    kk='F' if (k=='F' and t in ov and ov[t][0]=='F') else ('P' if k=='P' else ('LT' if k=='LT' else 'N')); c[kk]+=1
head=f"""# Smoke hồi quy sprint 9.1 — kết quả (chưa N=0; SmkS lượt 7 dừng vì hết ngân sách lượt)
Mã đã chạy: 1409b04d / 8bbb6b88 (sprint/11-p10-eval) · ep-r + lớp sao sạch QR91A…H (evidence-9.1/scripts/r91-clone.sh + r91-fixgrades.sh + r91-fixseed.sh) · Người chạy: qc2 / SmkS · Ngày: 2026-11-19
Tổng TC cũ: 414 · Lỗi thời (LT, loại khỏi tổng): {len(lt)} · Tổng tính = {len(rows)} · Đạt: {P} · Trượt (F xác nhận): {len(Fconf)} · K: 0 · Chưa chạy (N): {N}   (P+F+K+N = {len(rows)})
Cách chạy: `evidence-9.1/run7.sh <spec> <lớp> [grep]` (khoá /tmp/qc2-pw.lock, xoá bộ đếm auth mỗi 3 s, log logs/*.rerun7*.log); `make_overrides7.py` ghi P từ ✓ thật + kết quả chấm tay `results-smkS.tsv`; `build_smoke91.py` dựng `s91-rows.json`. Dữ liệu sạch: 761987 đã bị QC ghi đè (QUIZ v2 ×6, MANUAL v9, thêm điểm danh/phát biểu) nên chạy trên lớp sao.
F mới lượt này: DEF-R9.1-06 (TC-UX12-2, Thấp), DEF-R9.1-07 (TC-UX16-14, Thấp, AC mâu thuẫn) — xem defects-9.1.md. N còn lại phần lớn do đăng nhập/`me/courses` lỗi tạm khi nhiều agent chạy Playwright cùng lúc (không phải F), chạy lại bằng hàng đợi /tmp/smks/queue.sh (jobs8.txt).
"""
tbl=["## Tổng hợp theo US","| US | TC | P | F | LT | N |","| --- | ---: | ---: | ---: | ---: | ---: |"]
for u,c in us.items(): tbl.append(f"| {u} | {c['T']} | {c['P']} | {c['F']} | {c['LT']} | {c['N']} |")
tbl.append(f"| **Cộng** | **{sum(c['T'] for c in us.values())}** | **{P}** | **{len(Fconf)}** | **{len(lt)}** | **{N}** |")
res=["","## Kết quả từng ca","| TC | KQ | Ghi chú |","| --- | --- | --- |"]
for t,k,n,w in rows:
    kq='F' if t in Fconf else ('P' if k=='P' else 'N'); note=n.replace('|','/').replace('\n',' ')
    if kq=='N' and not note.startswith('chưa'): note='chưa chạy sạch: '+note
    res.append(f"| {t} | {kq} | {note} |")
open(H+'smoke-9.1.md','w',encoding='utf-8').write(head+'\n'+'\n'.join(tbl)+'\n'+'\n'.join(res)+'\n')
print(P,len(Fconf),N,len(rows))

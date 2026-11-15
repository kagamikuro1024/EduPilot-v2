import re,collections,os
ROOT='/Users/kuro/Documents/TA_Agent_qc2/docs/regression/'
HEAD='Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 (nhóm A) · Ngày: 2026-11-15'
def num(tc):  # sắp xếp tự nhiên
    return [int(x) if x.isdigit() else x for x in re.split(r'(\d+)',tc)]
def rows(items):
    """items: list of (tc, status, note, us). Gom dòng liên tiếp trùng (status,note) khi status không phải P/F."""
    out=[]; i=0
    while i<len(items):
        tc,s,n,us=items[i]
        if s in ('P','F'):
            out.append('| %s | %s | %s |'%(tc,s,n)); i+=1; continue
        j=i
        while j+1<len(items) and items[j+1][1]==s and items[j+1][2]==n and items[j+1][3]==us: j+=1
        a=items[i][0]; b=items[j][0]
        out.append('| %s | %s | %s |'%(a if i==j else a+'…'+b, s, n)); i=j+1
    return out
def smoke(sprint,title_note,items,how,usnames,defs,extra_sections=''):
    c=collections.Counter(s for _,s,_,_ in items)
    tot=len(items)
    byus=collections.OrderedDict()
    for tc,s,n,us in items: byus.setdefault(us,collections.Counter())[s]+=1
    L=['# Smoke hồi quy sprint %s — kết quả'%sprint,HEAD,'']
    kabc=c['Ka']+c['Kb']+c['Kc']
    L.append('Tổng TC cũ: %d · Đạt (P): %d · Trượt (F): %d · Không chạy K (a/b/c): %d · Lỗi thời (D): %d · Chưa chạy — cần PM quyết (X): %d · Chưa chạy — ngoài mẫu (N): %d   (P+F+K+D+X+N = %d)'%(tot,c['P'],c['F'],kabc,c['D'],c['X'],c['N'],c['P']+c['F']+kabc+c['D']+c['X']+c['N']))
    L.append('Quy ước: **K(a)** thiết bị không có · **K(b)** cần người thật · **K(c)** ngoài phạm vi đồ án · **D** TC lỗi thời (spec đã đổi hợp lệ, ghi mã đề xuất/quyết định) · **X** chưa chạy vì lệnh phá huỷ/dựng stack thứ hai bị cấm (chờ PM) · **N** chưa chạy: dạng tài liệu, ngoài mẫu smoke «1 chính + 1 lỗi / US».')
    L.append('Cách chạy:'); L+=how; L.append('')
    L.append('## Tổng hợp theo US'); L.append('| US | TC tổng | P | F | K(a/b/c) | D | X | N | Ghi chú |'); L.append('| --- | --- | --- | --- | --- | --- | --- | --- | --- |')
    for us,cc in byus.items():
        L.append('| %s | %d | %d | %d | %d | %d | %d | %d | %s |'%(us,sum(cc.values()),cc['P'],cc['F'],cc['Ka']+cc['Kb']+cc['Kc'],cc['D'],cc['X'],cc['N'],usnames.get(us,'')))
    L.append(''); L.append('## Kết quả từng ca'); L.append('| TC | KQ | Ghi chú |'); L.append('| --- | --- | --- |')
    cur=None
    for us in byus:
        L.append('| **%s** | | |'%us)
        L+=rows([x for x in items if x[3]==us])
    L.append(''); L.append('## Trượt: bước tái hiện')
    L+=defs if defs else ['Không có ca trượt.']
    if extra_sections: L.append(''); L.append(extra_sections)
    open(ROOT+'smoke-%s.md'%sprint,'w').write('\n'.join(L)+'\n')
    return c

from w7lib import *
import fx
fx.use('QC3S6C','QKTWQ3C'); Q=fx.QA
qs={'A':'Giải thích chi tiết cơ chế hoạt động của thuật toán RSA cho mình','B':'Mô tả quy trình xác thực hai yếu tố trong hệ thống ngân hàng','C':'Trình bày các bước cấu hình VPN site to site','D':'Em là Vũ Hoàng Giang, MSSV 20229001, giải thích về kiểm soát truy cập','G':'Cấu hình tường lửa mạng cho máy chủ','H':'Phân biệt tấn công DDoS và DoS'}
for k,q in qs.items():
    ask('sv.gioi',q,course=Q,title=('QC3 tài liệu xoá' if k=='H' else None))
json.dump({'Q':Q,'qs':qs,'docdel':fx.doc('QC3 tài liệu xoá')},open('/tmp/qc3e/ui/prep.json','w'))
print(open('/tmp/qc3e/ui/prep.json').read())

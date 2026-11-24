import sys,json,hashlib,urllib.request,ssl
sys.path.insert(0,'/tmp/qc3oc')
from oc import *
c=psql("select id from courses where class_code='QCOC1'"); aid=psql(f"select id from assignments where course_id='{c}' and submission_code='OC01'"); Q=f"/courses/{c}"
print('aid',aid, A('teacher','POST',f"{Q}/assignments/{aid}/publish",{})[0])
def put_file(u,name,data):
    sha=hashlib.sha256(data).hexdigest()
    s,pre=A(u,'POST',f"{Q}/assignments/{aid}/submit/presign",{"filename":name,"size_bytes":len(data),"sha256":sha})
    if s!=200: return s,pre
    req=urllib.request.Request(pre['url'],data=data,method=pre['method'],headers=pre['headers'])
    try: r=urllib.request.urlopen(req,context=ctx); s2=r.status
    except urllib.error.HTTPError as e: s2=e.code
    return A(u,'POST',f"{Q}/assignments/{aid}/submit",{"upload_id":pre['upload_id']}),s2
txt={'sv.gioi':"# Phân tích vụ tấn công SolarWinds\n\nTác nhân: nhóm APT29 (Cozy Bear) cấy mã độc SUNBURST vào bản cập nhật phần mềm Orion. Bề mặt tấn công là chuỗi cung ứng phần mềm: kẻ tấn công chèn mã vào quy trình build, bản cập nhật có chữ ký hợp lệ nên khách hàng tin tưởng cài đặt. Hậu quả: hàng nghìn tổ chức bị cài cửa hậu, kẻ tấn công di chuyển ngang và đánh cắp thư điện tử. Biện pháp: kiểm tra toàn vẹn build, phân đoạn mạng, giám sát lưu lượng DNS bất thường, nguyên tắc đặc quyền tối thiểu.\n",
 'sv.kha':"# Bài làm\n\nSolarWinds là một vụ tấn công mạng lớn. Kẻ xấu đã chèn mã độc vào phần mềm. Nhiều công ty bị ảnh hưởng. Nên cập nhật phần mềm thường xuyên.\n"}
for u,t in txt.items(): print(u,put_file(u,'bai-lam.md',t.encode()))
print(psql(f"select count(*) from submissions where assignment_id='{aid}'"))

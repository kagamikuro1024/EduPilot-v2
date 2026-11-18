import sys, json
sys.path.insert(0, '.'); from q4lib import *
S = json.load(open('/tmp/qc2-s4-ui.json')); PWD = S['pwd']
oth, ccO, _ = new_course('oth', subject='QCOTH'); ri, ccRI, codeRI = new_course('ri'); settings(ri)
em = f'qc2s4.unv.{uuid.uuid4().hex[:5]}@edupilot.local'; resetrl()
s, j, _ = call('POST', '/auth/register', None, {'email': em, 'password': PWD, 'full_name': 'QC2S4 Chưa xác minh'}); assert s == 202
S['OTH'] = {'id': oth, 'class_code': ccO}; S['RI'] = {'id': ri, 'class_code': ccRI, 'code': codeRI}; S['UNV'] = em
S['EMPTYSRC'] = S['O']   # lớp O không có tài liệu → nguồn "chưa có tài liệu"
json.dump(S, open('/tmp/qc2-s4-ui.json', 'w'), ensure_ascii=False, indent=1); print(S['OTH'], S['RI'], em)

import sys, json
sys.path.insert(0, '.'); from q4lib import *
T_ = L('teacher'); C1 = psql("select id from courses where class_code='761987'")
cid2, cc2, code2 = new_course('sh'); cid3, cc3, code3 = new_course('s3'); settings(cid2); settings(cid3)
src_docs = psql(f"select id||'|'||type||'|'||visible_to_students from documents where course_id='{C1}' and status='READY' and coalesce(category,'')<>'THREAD_KNOWLEDGE' order by id").splitlines()
n_know = psql(f"select count(*) from documents where course_id='{C1}' and status='READY' and category='THREAD_KNOWLEDGE'")
def fp(course_filter):
    return psql(f"select count(*)||'/'||coalesce(md5(string_agg(c.id::text||coalesce(c.embedding::text,''), ',' order by c.id)),'') from content_chunks c join documents d on d.id=c.document_id where d.course_id='{C1}' and d.status='READY'")
f0 = fp(0); n0 = psql(f"select count(*) from document_courses where course_id='{cid2}'")
s, j, _ = call('POST', f'/courses/{cid2}/share-from', T_, {'source_course_id': C1, 'what': ['documents']}, hdr={'Idempotency-Key': str(uuid.uuid4())})
dc = psql(f"select string_agg(d.type::text, ',' order by d.id) from document_courses x join documents d on d.id=x.document_id where x.course_id='{cid2}'")
n1 = psql(f"select count(*) from document_courses where course_id='{cid2}'"); f1 = fp(0)
ch_in = psql(f"select count(*) from content_chunks where '{cid2}'=any(course_ids)"); ch_miss = psql(f"select count(*) from content_chunks c join document_courses x on x.document_id=c.document_id where x.course_id='{cid2}' and not ('{cid2}'=any(c.course_ids))")
s2, j2, _ = call('POST', f'/courses/{cid2}/share-from', T_, {'source_course_id': C1, 'what': ['documents']}, hdr={'Idempotency-Key': str(uuid.uuid4())}); n2 = psql(f"select count(*) from document_courses where course_id='{cid2}'"); f2 = fp(0)
ready_nonpolicy = sum(1 for d in src_docs if '|COURSE_POLICY|' not in d)
print(s, j, dc, n0, n1, n2, ch_in, ch_miss, f0 == f1 == f2, ready_nonpolicy)
C('TC-P210-26', s == 200 and j.get('shared', {}).get('documents') == ready_nonpolicy and any(k.get('reason') for k in j.get('skipped', [])) and 'COURSE_POLICY' not in (dc or '') and n1 == str(ready_nonpolicy) and ch_miss == '0' and f0 == f1 and n2 == n1 and f2 == f1,
  f'share-from lớp 761987 (nguồn: {len(src_docs)} tài liệu thật READY, trong đó 1 COURSE_POLICY, và {n_know} tài liệu ảo THREAD_KNOWLEDGE không được chia sẻ theo AC11) → {s} shared {j.get("shared")}, skipped {json.dumps(j.get("skipped"), ensure_ascii=False)}; document_courses mới = {n1} (không có COURSE_POLICY: {dc}); chunk có course_ids chứa lớp đích đủ (thiếu {ch_miss}); số chunk + md5 embedding nguồn trước/sau {f0.split("/")[0]}/{f1.split("/")[0]} {"khớp" if f0 == f1 else "KHÁC"} (không nhúng lại); chia sẻ lại → {s2}, document_courses {n2}, md5 không đổi')
# ---------- P210-29: không rò ngược / sang lớp 3 ----------
tS = L('sv.gioi'); u_in = student_pool(30)[20]; tIn = L(u_in); resetjoin(); call('POST', '/courses/join', tIn, {'code': code2})
docs3 = call('GET', f'/courses/{cid3}/documents', T_)[1]; lib3 = call('GET', f'/courses/{cid3}/library', tIn)  # tIn không thuộc lớp 3
ch3 = psql(f"select count(*) from content_chunks where '{cid3}'=any(course_ids)")
docs3n = len(docs3.get('items', []))
sv_docs = call('GET', f'/courses/{cid2}/documents', tIn); sv_lib = call('GET', f'/courses/{cid2}/library', tIn)
ak = [t for t in psql(f"select title from documents where course_id='{C1}' and type='ANSWER_KEY'").splitlines()]
blob = json.dumps([sv_docs[1], sv_lib[1]], ensure_ascii=False)
hidden = psql(f"select title from documents where course_id='{C1}' and visible_to_students=false").splitlines()
leak = [t for t in ak + hidden if t and t in blob]
# SV chỉ lớp 1 → lớp 2
tC1 = L('sv.gioi'); r_c2 = call('GET', f'/courses/{cid2}/today', tC1)
one_doc = psql(f"select id from documents where course_id='{C1}' and type='LECTURE' and status='READY' limit 1")
w1 = call('PATCH', f'/courses/{cid2}/documents/{one_doc}', T_, {'title': 'QC2 sửa thử', 'version': 1}); w2 = call('DELETE', f'/courses/{cid2}/documents/{one_doc}', T_)
still = psql(f"select count(*) from documents where id='{one_doc}' and course_id='{C1}'"); title_ok = psql(f"select title from documents where id='{one_doc}'"); print('29dbg', docs3n, ch3, leak, r_c2[0], w1[0], w2[0], still, title_ok, sv_docs[0], sv_lib[0])
C('TC-P210-29', docs3n == 0 and ch3 == '0' and not leak and r_c2[0] == 403 and w1[0] == 409 and w1[1].get('code') == 'DOCUMENT_SHARED_READONLY' and w2[0] == 409 and w2[1].get('code') == 'DOCUMENT_SHARED_READONLY' and still == '1' and 'QC2 sửa thử' not in title_ok,
  f'lớp 3 (không chia sẻ): documents {docs3n}, content_chunks có course_ids chứa lớp 3 = {ch3}; SV lớp đích gọi documents/library → {sv_docs[0]}/{sv_lib[0]} không có tiêu đề ANSWER_KEY / visible_to_students=false ({len(ak)+len(hidden)} tiêu đề kiểm, rò = {leak}); SV chỉ lớp 1 vào lớp 2 → {r_c2[0]}; GV lớp đích PATCH/DELETE tài liệu nguồn qua đường lớp đích → {w1[0]}/{w2[0]} DOCUMENT_SHARED_READONLY «chỉ lớp gốc mới sửa được» (nguồn còn nguyên: {still}, tiêu đề «{title_ok[:30]}»); chia sẻ một chiều')
# ---------- P210-31: share-sources ----------
cid_other, _, _ = new_course('ot', subject='QCOTH'); cid_arch, _, _ = new_course('ar'); call('POST', f'/admin/courses/{cid_arch}/archive', L('admin'))
src = call('GET', f'/courses/{cid2}/share-sources', T_); items = src[1].get('items', src[1] if isinstance(src[1], list) else [])
ids = [i.get('course_id') or i.get('id') for i in items]
psql(f"insert into enrollments(course_id,user_id,role_in_course,status,joined_via) select '{cid2}',id,'TA','ACTIVE','ADMIN' from users where email='ta@edupilot.local' on conflict do nothing")
r_ta = call('GET', f'/courses/{cid2}/share-sources', L('ta')); r_sv = call('GET', f'/courses/{cid2}/share-sources', tIn)
subj = {i: psql(f"select subject_code||'|'||status from courses where id='{i}'") for i in ids}
good = all(v.endswith('|ACTIVE') and v.startswith('INT1006') for v in subj.values())
C('TC-P210-31', src[0] == 200 and C1 in ids and cid_other not in ids and cid_arch not in ids and cid2 not in ids and good and r_ta[0] == 403 and r_sv[0] == 403,
  f'GV (teacher của cả hai lớp): {len(ids)} nguồn, đều cùng subject_code INT1006 và ACTIVE; có lớp 761987, không có lớp khác học phần (QCOTH) và lớp đã lưu trữ, không có chính lớp đích; thân mẫu {json.dumps(items[:1], ensure_ascii=False)[:200]}; TA → {r_ta[0]}; SV → {r_sv[0]}')

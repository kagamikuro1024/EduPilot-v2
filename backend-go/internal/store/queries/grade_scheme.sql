-- Công thức điểm (FEAT-gradebook US-P6-03/04): tài liệu quy chế, kiểm trích dẫn, bản nháp. Mọi truy vấn có course_id hoặc đi qua id thuộc lớp.
-- Chỉ internal/grade và internal/grading gọi. Mảng id / chuỗi: `::text[]` (PgBouncer, giao thức đơn giản).

-- name: GradeDocForExtract :one
select d.id, d.title, d.type, d.status from documents d where d.id = sqlc.arg(document_id) and d.course_id = sqlc.arg(course_id) and d.category is distinct from 'THREAD_KNOWLEDGE';

-- name: GradeCitationsCheck :many
-- Trích dẫn có thật ở đúng trang không: NFC → gộp khoảng trắng → vn_fold (bỏ dấu, hạ chữ thường). `cites` = JSON [{quote, text}] với `text` là văn bản của trang (đọc qua DocChunks, đã lọc lớp).
select (u.i - 1)::int as idx, (position(vn_fold(regexp_replace(normalize(u.e ->> 'quote', NFC), '\s+', ' ', 'g')) in vn_fold(regexp_replace(normalize(u.e ->> 'text', NFC), '\s+', ' ', 'g'))) > 0)::bool as found
from jsonb_array_elements(sqlc.arg(cites)::jsonb) with ordinality as u(e, i)
order by u.i;

-- name: GradeDraftGet :one
select id, scheme_no, version, extract_job_id, source_document_id, source_title, extraction, created_at, updated_at
from grade_schemes where course_id = sqlc.arg(course_id) and status = 'DRAFT';

-- name: GradeDraftLock :one
-- Khoá bản DRAFT của lớp trong giao dịch lưu (so version / chỉnh tay trước khi ghi đè — TLR-10).
select id, version from grade_schemes where course_id = sqlc.arg(course_id) and status = 'DRAFT' for update;

-- name: GradeDraftEdited :one
-- Số mục giảng viên đã tự điền / sửa (thành phần + luật).
select ((select count(*) from grade_components gc where gc.course_id = sqlc.arg(course_id) and gc.scheme_id = sqlc.arg(scheme_id) and gc.filled_by_user)
      + (select count(*) from grade_rules gr where gr.course_id = sqlc.arg(course_id) and gr.scheme_id = sqlc.arg(scheme_id) and gr.filled_by_user))::int as n;

-- name: GradeSchemeByJob :one
select id from grade_schemes where course_id = sqlc.arg(course_id) and extract_job_id = sqlc.arg(job_id);

-- name: GradeSchemeNextNo :one
select (coalesce(max(scheme_no), 0) + 1)::int from grade_schemes where course_id = sqlc.arg(course_id);

-- name: GradeSchemeInsert :one
insert into grade_schemes (course_id, scheme_no, status, source_document_id, source_title, extraction, extract_job_id, created_by, unclear_count)
values (sqlc.arg(course_id), sqlc.arg(scheme_no), 'DRAFT', sqlc.arg(source_document_id), sqlc.arg(source_title), sqlc.arg(extraction), sqlc.arg(extract_job_id), sqlc.arg(created_by), sqlc.arg(unclear_count))
returning id;

-- name: GradeDraftDelete :exec
delete from grade_schemes where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and status = 'DRAFT';

-- name: GradeComponentInsert :one
insert into grade_components (course_id, scheme_id, key, name, grp, weight, position, citations, filled_by_user)
values (sqlc.arg(course_id), sqlc.arg(scheme_id), sqlc.arg(key), sqlc.arg(name), sqlc.arg(grp), sqlc.arg(weight), sqlc.arg(position), sqlc.arg(citations), sqlc.arg(filled_by_user))
returning id;

-- name: GradeItemInsert :one
insert into grade_items (course_id, scheme_id, component_id, key, name, max_score, source, exam_id, assignment_id, position)
values (sqlc.arg(course_id), sqlc.arg(scheme_id), sqlc.arg(component_id), sqlc.arg(key), sqlc.arg(name), sqlc.arg(max_score), sqlc.arg(source), sqlc.arg(exam_id), sqlc.narg(assignment_id), sqlc.arg(position))
returning id;

-- name: GradeRuleInsert :exec
insert into grade_rules (course_id, scheme_id, kind, applicable, params, source_quote, source_page, filled_by_user)
values (sqlc.arg(course_id), sqlc.arg(scheme_id), sqlc.arg(kind), sqlc.arg(applicable), sqlc.arg(params), sqlc.arg(source_quote), sqlc.arg(source_page), sqlc.arg(filled_by_user));

-- name: GradeStatusFacts :many
-- Dữ kiện trạng thái công thức của một lô lớp (một truy vấn, không N+1): có quy chế READY, bản CONFIRMED, bản DRAFT + số mục chưa rõ đã ghi lúc lưu (#16), sổ điểm.
select c.id as course_id, c.class_code,
       exists(select 1 from documents d where d.course_id = c.id and d.type = 'COURSE_POLICY' and d.status = 'READY' and d.category is distinct from 'THREAD_KNOWLEDGE') as has_policy,
       cs.id as confirmed_id, cs.scheme_no as confirmed_no, cs.confirmed_at,
       ds.id as draft_id, coalesce(ds.unclear_count, 0)::int as unclear_count,
       coalesce(g.status::text, 'OPEN')::text as gb_status, coalesce(g.snapshot_version, 0)::int as gb_version
from courses c
left join grade_schemes cs on cs.course_id = c.id and cs.status = 'CONFIRMED'
left join grade_schemes ds on ds.course_id = c.id and ds.status = 'DRAFT'
left join gradebooks g on g.course_id = c.id
where c.id = any(sqlc.arg(course_ids)::text[]::uuid[]);

-- name: GradeSchemeGet :one
select id, course_id, scheme_no, status, source_document_id, source_title, extraction, extract_job_id, confirmed_by, confirmed_at, created_by, version, created_at, updated_at
from grade_schemes where course_id = sqlc.arg(course_id) and id = sqlc.arg(id);

-- name: GradeSchemesByIDs :many
select id, course_id, version, source_document_id, extraction from grade_schemes
where id = any(sqlc.arg(ids)::text[]::uuid[]);

-- name: GradeComponentsOf :many
select id, scheme_id, key, name, grp, weight, position, citations, filled_by_user
from grade_components where scheme_id = any(sqlc.arg(scheme_ids)::text[]::uuid[]) order by scheme_id, position, id;

-- name: GradeRulesOf :many
select id, scheme_id, kind, applicable, params, source_quote, source_page, filled_by_user
from grade_rules where scheme_id = any(sqlc.arg(scheme_ids)::text[]::uuid[]) order by scheme_id, kind, id;

-- name: GradeReminderTargets :many
-- Lớp ACTIVE chưa có công thức CONFIRMED + giảng viên đang dạy (đúng một mỗi lớp).
select c.id as course_id, c.class_code, e.user_id as teacher_id
from courses c
join enrollments e on e.course_id = c.id and e.role_in_course = 'TEACHER' and e.status = 'ACTIVE'
where c.status = 'ACTIVE' and not exists (select 1 from grade_schemes s where s.course_id = c.id and s.status = 'CONFIRMED')
order by c.id;

-- name: GradeSchemeIDByStatus :one
select id from grade_schemes where course_id = sqlc.arg(course_id) and status = sqlc.arg(status)::grade_scheme_status;

-- name: GradeSetUnclear :execrows
-- Ghi kết quả bộ kiểm của bản nháp (số mục chưa rõ). Chỉ khi bản nháp chưa đổi từ lúc tính (version): người sửa đồng thời đã tự ghi số mới.
update grade_schemes set unclear_count = sqlc.arg(unclear_count)
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and status = 'DRAFT' and version = sqlc.arg(version);

-- name: GradeSchemeBump :one
-- Sửa bản nháp: so version rồi tăng (một câu, không đua); không khớp → không dòng nào.
update grade_schemes set version = version + 1, unclear_count = 0
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and status = 'DRAFT' and version = sqlc.arg(version)
returning version;

-- name: GradeComponentsDelete :exec
-- Xoá hết thành phần của bản nháp (item theo CASCADE) trước khi ghi lại toàn bộ (PUT thay toàn bộ).
delete from grade_components where course_id = sqlc.arg(course_id) and scheme_id = sqlc.arg(scheme_id);

-- name: GradeRulesDelete :exec
delete from grade_rules where course_id = sqlc.arg(course_id) and scheme_id = sqlc.arg(scheme_id);

-- name: GradeExamsOfCourse :many
-- Bài thi của lớp theo id (kiểm gắn item EXAM): chỉ cần trạng thái.
select id, status from exams where course_id = sqlc.arg(course_id) and id = any(sqlc.arg(exam_ids)::text[]::uuid[]);

-- name: GradeInputExamsByExam :many
-- Như GradeInputExams nhưng theo bài thi (công thức chưa lưu, xem thử tác động): bài PUBLISHED, lượt GRADED, COALESCE(adjusted, auto).
select a.exam_id, a.student_id, coalesce(a.adjusted_score, a.auto_score)::numeric(5,2) as score, x.max_score as exam_max
from exams x join exam_attempts a on a.course_id = x.course_id and a.exam_id = x.id and a.status = 'GRADED'
where x.course_id = sqlc.arg(course_id) and x.status = 'PUBLISHED' and x.id = any(sqlc.arg(exam_ids)::text[]::uuid[]) and a.student_id = any(sqlc.arg(student_ids)::text[]::uuid[]);

-- name: GradebookEnsure :exec
insert into gradebooks (course_id) values (sqlc.arg(course_id)) on conflict (course_id) do nothing;

-- name: GradebookStatusShare :one
select status from gradebooks where course_id = sqlc.arg(course_id) for share;

-- name: GradeSchemeSupersede :execrows
update grade_schemes set status = 'SUPERSEDED' where course_id = sqlc.arg(course_id) and status = 'CONFIRMED';

-- name: GradeSchemeConfirm :one
update grade_schemes set status = 'CONFIRMED', confirmed_by = sqlc.arg(actor_id), confirmed_at = now(), version = version + 1
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and status = 'DRAFT'
returning confirmed_at, scheme_no;

-- name: GradeEntryCountsByItem :many
select item_id, count(*)::int as n from grade_entries
where course_id = sqlc.arg(course_id) and item_id = any(sqlc.arg(item_ids)::text[]::uuid[]) group by item_id;

-- name: GradeCarryEntries :execrows
-- Mang điểm đã nhập sang item mới của công thức vừa xác nhận (cùng giao dịch): cùng key + nguồn nhập tay + max_score (TLR-8).
update grade_entries set item_id = sqlc.arg(new_item_id) where course_id = sqlc.arg(course_id) and item_id = sqlc.arg(old_item_id);

-- name: GradeCarryPublished :exec
update grade_items n set published_at = o.published_at
from grade_items o
where n.course_id = sqlc.arg(course_id) and n.id = sqlc.arg(new_item_id) and o.course_id = n.course_id and o.id = sqlc.arg(old_item_id);

-- name: GradeAssignmentsOfCourse :many
-- Bài tập của lớp (kiểm item ASSIGNMENT: thuộc lớp, thang điểm).
select id, max_score from assignments where course_id = sqlc.arg(course_id) and id = any(sqlc.arg(assignment_ids)::text[]::uuid[]);

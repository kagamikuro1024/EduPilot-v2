-- name: FormsRoster :many
-- Sinh viên ACTIVE của lớp để khớp dòng Forms: email chỉ tính khi ĐÃ XÁC MINH; MSSV chụp lúc vào lớp kèm cách vào lớp (chỉ ROSTER / ADMIN đáng tin).
select e.user_id as student_id, u.full_name, case when u.email_verified_at is not null then lower(u.email) else '' end::text as email,
       e.student_code_snapshot as student_code, e.joined_via
from enrollments e join users u on u.id = e.user_id
where e.course_id = sqlc.arg(course_id) and e.role_in_course = 'STUDENT' and e.status = 'ACTIVE';

-- name: FormsBatchByHash :one
select id from forms_import_batches where course_id = sqlc.arg(course_id) and item_id = sqlc.arg(item_id) and file_hash = sqlc.arg(file_hash);

-- name: FormsBatchInsert :one
insert into forms_import_batches (course_id, item_id, file_hash, blob_key, mapping, matched, unmatched, created_by)
values (sqlc.arg(course_id), sqlc.arg(item_id), sqlc.arg(file_hash), sqlc.arg(blob_key), sqlc.arg(mapping), sqlc.arg(matched), sqlc.arg(unmatched), sqlc.arg(created_by))
returning id;

-- name: FormsRowsInsert :exec
-- Sáu mảng song song ghép theo thứ tự (cùng cách với GradeImportApply): MỘT câu lệnh cho cả lô.
insert into forms_import_rows (course_id, batch_id, row_no, student_id, identifier_masked, correct, total, score)
select sqlc.arg(course_id), sqlc.arg(batch_id), a.v, nullif(b.v, '')::uuid, c.v, d.v, e.v, f.v
from unnest(sqlc.arg(row_nos)::int[]) with ordinality as a(v, n)
join unnest(sqlc.arg(student_ids)::text[]) with ordinality as b(v, n) on b.n = a.n
join unnest(sqlc.arg(idents)::text[]) with ordinality as c(v, n) on c.n = a.n
join unnest(sqlc.arg(corrects)::int[]) with ordinality as d(v, n) on d.n = a.n
join unnest(sqlc.arg(totals)::int[]) with ordinality as e(v, n) on e.n = a.n
join unnest(sqlc.arg(scores)::text[]::numeric[]) with ordinality as f(v, n) on f.n = a.n;

-- name: FormsRowsByBatch :many
select r.row_no, r.student_id, coalesce(u.full_name, '')::text as full_name, coalesce(e.student_code_snapshot, '')::text as student_code,
       r.identifier_masked, r.correct, r.total, r.score::text as score
from forms_import_rows r
left join users u on u.id = r.student_id
left join enrollments e on e.course_id = r.course_id and e.user_id = r.student_id
where r.course_id = sqlc.arg(course_id) and r.batch_id = sqlc.arg(batch_id)
order by r.row_no;

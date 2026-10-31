-- Sổ điểm (FEAT-gradebook US-P6-06): bảng điểm theo trang, sửa ô, công bố cột, giải trình / lịch sử. Mọi truy vấn có course_id.
-- Chỉ internal/grade gọi. Mảng id: `::text[]::uuid[]` (PgBouncer, giao thức đơn giản).

-- name: GradebookStudentsPage :many
-- Sinh viên ACTIVE theo trang con trỏ (vn_fold(full_name), id); lấy limit + 1 để biết còn trang sau.
select e.user_id as student_id, u.full_name, e.student_code_snapshot as student_code, vn_fold(u.full_name)::text as fold_name
from enrollments e join users u on u.id = e.user_id
where e.course_id = sqlc.arg(course_id) and e.role_in_course = 'STUDENT' and e.status = 'ACTIVE'
  and (sqlc.arg(after_id)::text = '' or (vn_fold(u.full_name), e.user_id::text) > (sqlc.arg(after_fold)::text, sqlc.arg(after_id)::text))
order by vn_fold(u.full_name), e.user_id
limit sqlc.arg(page_limit);

-- name: GradeEntriesDetail :many
-- Điểm đã nhập của một lô sinh viên kèm người sửa gần nhất (chỉ Staff được thấy tên).
select e.item_id, e.student_id, e.score, e.version, e.source, ub.full_name as updated_by
from grade_entries e
join grade_items i on i.course_id = e.course_id and i.id = e.item_id
join users ub on ub.id = e.entered_by
where e.course_id = sqlc.arg(course_id) and i.scheme_id = sqlc.arg(scheme_id) and e.student_id = any(sqlc.arg(student_ids)::text[]::uuid[]);

-- name: GradeItemForEdit :one
select i.id, i.scheme_id, i.key, i.name, i.max_score, i.source, i.published_at, s.status as scheme_status
from grade_items i join grade_schemes s on s.course_id = i.course_id and s.id = i.scheme_id
where i.course_id = sqlc.arg(course_id) and i.id = sqlc.arg(item_id);

-- name: GradeStudentIsActive :one
select exists(select 1 from enrollments where course_id = sqlc.arg(course_id) and user_id = sqlc.arg(student_id) and role_in_course = 'STUDENT' and status = 'ACTIVE')::bool;

-- name: GradeEntryLock :one
select id, score, version, source from grade_entries
where course_id = sqlc.arg(course_id) and item_id = sqlc.arg(item_id) and student_id = sqlc.arg(student_id) for update;

-- name: GradeEntryInsert :one
insert into grade_entries (course_id, item_id, student_id, score, source, entered_by)
values (sqlc.arg(course_id), sqlc.arg(item_id), sqlc.arg(student_id), sqlc.arg(score), 'MANUAL', sqlc.arg(entered_by))
returning id, score, version, source;

-- name: GradeEntryUpdate :one
update grade_entries set score = sqlc.arg(score), source = 'MANUAL', entered_by = sqlc.arg(entered_by), version = version + 1
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and version = sqlc.arg(version)
returning id, score, version, source;

-- name: GradeEntryDelete :execrows
delete from grade_entries where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and version = sqlc.arg(version);

-- name: GradeItemSetPublished :one
update grade_items set published_at = case when sqlc.arg(published)::bool then coalesce(published_at, now()) else null end
where course_id = sqlc.arg(course_id) and id = sqlc.arg(item_id)
returning published_at;

-- name: GradeHistory :many
-- Lịch sử điểm của một sinh viên từ audit_log (mới nhất trước, con trỏ (created_at, id)).
select a.id, a.created_at, a.action, a.entity, a.before, a.after, coalesce(u.full_name, '')::text as actor_name
from audit_log a left join users u on u.id = a.actor_id
where a.course_id = sqlc.arg(course_id) and a.entity in ('grade_entries', 'grade_items')
  and (a.after ->> 'student_id' = sqlc.arg(student_id)::text or a.before ->> 'student_id' = sqlc.arg(student_id)::text)
  and (sqlc.narg(cursor_at)::timestamptz is null or (a.created_at, a.id) < (sqlc.narg(cursor_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
order by a.created_at desc, a.id desc
limit sqlc.arg(page_limit);

-- name: GradeEventWarnings :many
-- Sinh viên ACTIVE có sự kiện điểm cộng / trừ khác 0 (cho cảnh báo luật không áp dụng).
select pe.student_id, u.full_name, (pe.type = 'PENALTY')::bool as is_penalty
from participation_events pe
join enrollments en on en.course_id = pe.course_id and en.user_id = pe.student_id and en.role_in_course = 'STUDENT' and en.status = 'ACTIVE'
join users u on u.id = pe.student_id
where pe.course_id = sqlc.arg(course_id) and pe.points <> 0 and pe.type in ('SPEAK', 'BONUS', 'PENALTY')
group by pe.student_id, u.full_name, (pe.type = 'PENALTY')
order by vn_fold(u.full_name), pe.student_id;

-- name: GradebookState :one
select status, snapshot_version from gradebooks where course_id = sqlc.arg(course_id);

-- name: GradeEntryCurrent :many
-- Giá trị hiện tại của một ô (cho thân 409 VERSION_CONFLICT): điểm, version, người sửa, lúc sửa.
select e.score, e.version, ub.full_name as updated_by, e.updated_at
from grade_entries e join users ub on ub.id = e.entered_by
where e.course_id = sqlc.arg(course_id) and e.item_id = sqlc.arg(item_id) and e.student_id = sqlc.arg(student_id);

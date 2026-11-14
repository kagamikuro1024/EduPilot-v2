-- name: GradeImportRoster :many
-- Sinh viên ACTIVE của lớp kèm MSSV chụp lúc vào lớp và cách vào lớp. Nhập XLSX chỉ tin `joined_via IN ('ROSTER','ADMIN')` (#20: MSSV của dòng `CODE` là tự khai).
select e.user_id as student_id, u.full_name, e.student_code_snapshot as student_code, e.joined_via
from enrollments e join users u on u.id = e.user_id
where e.course_id = sqlc.arg(course_id) and e.role_in_course = 'STUDENT' and e.status = 'ACTIVE';

-- name: GradeImportExisting :many
select student_id, score from grade_entries where course_id = sqlc.arg(course_id) and item_id = sqlc.arg(item_id);

-- name: GradeImportApply :one
-- Ghi lô điểm nhập bằng MỘT câu lệnh (số lượt truy vấn không phụ thuộc số dòng): khoá ô cũ → chèn / sửa chỉ ô ĐỔI → audit mỗi điểm đổi (cũ → mới).
with input as (
  select a.student_id, b.score
  from unnest(sqlc.arg(student_ids)::text[]::uuid[]) with ordinality as a(student_id, n)
  join unnest(sqlc.arg(scores)::text[]::numeric[]) with ordinality as b(score, n) on b.n = a.n
), old as (
  select e.id, e.student_id, e.score, e.version
  from grade_entries e join input i on i.student_id = e.student_id
  where e.course_id = sqlc.arg(course_id) and e.item_id = sqlc.arg(item_id)
  for update of e
), ups as (
  insert into grade_entries (course_id, item_id, student_id, score, source, entered_by)
  select sqlc.arg(course_id), sqlc.arg(item_id), i.student_id, i.score, sqlc.arg(source)::grade_entry_source, sqlc.arg(actor_id)
  from input i left join old o on o.student_id = i.student_id
  where o.id is null or o.score is distinct from i.score
  on conflict (item_id, student_id) do update
    set score = excluded.score, source = excluded.source, entered_by = excluded.entered_by, version = grade_entries.version + 1, updated_at = now()
  returning student_id, score, version, (xmax = 0) as inserted
), aud as (
  insert into audit_log (course_id, actor_id, entity, entity_id, action, before, after)
  select sqlc.arg(course_id), sqlc.arg(actor_id), 'grade_entries', sqlc.arg(item_id)::text || ':' || u.student_id::text, 'set',
         case when o.id is null then null
              else jsonb_build_object('student_id', u.student_id, 'item_id', sqlc.arg(item_id)::uuid, 'item_key', sqlc.arg(item_key)::text, 'score', o.score::text, 'version', o.version) end,
         jsonb_build_object('student_id', u.student_id, 'item_id', sqlc.arg(item_id)::uuid, 'item_key', sqlc.arg(item_key)::text, 'score', u.score::text, 'version', u.version)
  from ups u left join old o on o.student_id = u.student_id
  returning 1
)
select (select count(*) from ups where inserted)::int as inserted,
       (select count(*) from ups where not inserted)::int as updated,
       (select count(*) from aud)::int as audited;

-- name: GradeCourseInfo :one
select name, class_code from courses where id = sqlc.arg(course_id);

-- name: GradeSnapshotsAt :many
-- Bản chốt của cả lớp ở một phiên bản (nguồn của tệp xuất khi sổ LOCKED).
select student_id, breakdown, numeric_grade, letter, failed_reason, locked_at
from final_grade_snapshots where course_id = sqlc.arg(course_id) and version = sqlc.arg(version);

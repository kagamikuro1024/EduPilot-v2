-- name: GradebookLockForUpdate :one
-- Khoá hàng sổ điểm của lớp (chốt / mở khoá): đua giữa hai chốt hoặc chốt với ghi điểm (mọi ghi mở đầu bằng FOR SHARE).
select status, snapshot_version, unlock_reason from gradebooks where course_id = sqlc.arg(course_id) for update;

-- name: GradeSnapshotsInsert :execrows
-- Chốt cả lớp bằng MỘT câu lệnh: `rows` là mảng JSON [{student_id, breakdown, numeric_grade, letter, failed_reason}].
insert into final_grade_snapshots (course_id, student_id, version, scheme_id, breakdown, numeric_grade, letter, failed_reason, locked_by, unlock_reason)
select sqlc.arg(course_id), r.student_id, sqlc.arg(version), sqlc.arg(scheme_id), r.breakdown, r.numeric_grade, r.letter, nullif(r.failed_reason, ''), sqlc.arg(locked_by), sqlc.narg(unlock_reason)
from jsonb_to_recordset(sqlc.arg(rows)::jsonb) as r(student_id uuid, breakdown jsonb, numeric_grade numeric, letter text, failed_reason text);

-- name: GradebookSetLocked :one
-- Sau khi tạo snapshot: khoá sổ, ghi phiên bản, xoá thông tin mở khoá. Trả lúc chốt (locked_at của snapshot).
update gradebooks set status = 'LOCKED', snapshot_version = sqlc.arg(version), unlock_reason = null, unlocked_by = null, unlocked_at = null, version = version + 1
where course_id = sqlc.arg(course_id)
returning updated_at;

-- name: GradebookSetOpen :execrows
update gradebooks set status = 'OPEN', unlock_reason = sqlc.arg(reason), unlocked_by = sqlc.arg(actor_id), unlocked_at = now(), version = version + 1
where course_id = sqlc.arg(course_id) and status = 'LOCKED';

-- name: GradeSnapshotVersions :many
-- Danh sách phiên bản đã chốt (mới nhất trước) kèm người chốt và lý do mở khoá trước đó.
select s.version, count(*)::int as students, min(s.locked_at)::timestamptz as locked_at, min(u.full_name)::text as locked_by, coalesce(min(s.unlock_reason), '')::text as unlock_reason
from final_grade_snapshots s join users u on u.id = s.locked_by
where s.course_id = sqlc.arg(course_id)
group by s.version order by s.version desc;

-- name: GradeSnapshotPage :many
-- Một trang của bản chốt (tên bỏ dấu, id), dùng chỉ mục (course_id, version, student_id).
select s.student_id, u.full_name, e.student_code_snapshot as student_code, s.numeric_grade, s.letter, s.failed_reason, s.breakdown, s.locked_at, vn_fold(u.full_name)::text as fold_name
from final_grade_snapshots s
join users u on u.id = s.student_id
join enrollments e on e.course_id = s.course_id and e.user_id = s.student_id
where s.course_id = sqlc.arg(course_id) and s.version = sqlc.arg(version)
  and (sqlc.arg(after_id)::text = '' or (vn_fold(u.full_name), s.student_id::text) > (sqlc.arg(after_fold)::text, sqlc.arg(after_id)::text))
order by vn_fold(u.full_name), s.student_id
limit sqlc.arg(page_limit);

-- name: GradeSnapshotChangedUsers :many
-- Sinh viên có numeric_grade hoặc letter khác bản trước (hoặc chưa có ở bản trước): chỉ những người này nhận GRADE_CHANGED.
select n.student_id
from final_grade_snapshots n
left join final_grade_snapshots o on o.course_id = n.course_id and o.student_id = n.student_id and o.version = sqlc.arg(version) - 1
where n.course_id = sqlc.arg(course_id) and n.version = sqlc.arg(version)
  and (o.id is null or o.numeric_grade is distinct from n.numeric_grade or o.letter is distinct from n.letter);

-- name: GradeSnapshotUsers :many
select student_id from final_grade_snapshots where course_id = sqlc.arg(course_id) and version = sqlc.arg(version);

-- name: GradeSnapshotsForStudents :many
-- Bản chốt mới nhất của một lô sinh viên (nguồn của kết quả và cờ drift khi sổ LOCKED).
select student_id, breakdown, numeric_grade, letter, failed_reason
from final_grade_snapshots
where course_id = sqlc.arg(course_id) and version = sqlc.arg(version) and student_id = any(sqlc.arg(student_ids)::text[]::uuid[]);

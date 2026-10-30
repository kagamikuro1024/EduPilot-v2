-- Sổ điểm (FEAT-gradebook, US-P6-01…). Chỉ gói internal/grade gọi các hàm sinh từ tệp này. Mọi truy vấn có course_id; đọc theo lô `student_id = ANY($ids)` (không N+1).

-- name: GradeActiveStudents :many
-- Sinh viên ACTIVE của lớp (MSSV = ảnh chụp do giảng viên xác lập, KHÔNG phải users.student_code tự khai — TLR-3), sắp theo tên bỏ dấu rồi id.
select e.user_id as student_id, u.full_name, e.student_code_snapshot as student_code
from enrollments e join users u on u.id = e.user_id
where e.course_id = sqlc.arg(course_id) and e.role_in_course = 'STUDENT' and e.status = 'ACTIVE'
order by vn_fold(u.full_name), e.user_id;

-- name: GradeInputEntries :many
-- Điểm item MANUAL / IMPORT của các sinh viên trong lô (điểm theo thang của item).
select e.item_id, e.student_id, e.score, e.version
from grade_entries e join grade_items i on i.course_id = e.course_id and i.id = e.item_id
where e.course_id = sqlc.arg(course_id) and i.scheme_id = sqlc.arg(scheme_id) and e.student_id = any(sqlc.arg(student_ids)::text[]::uuid[]);

-- name: GradeInputExams :many
-- Điểm bài thi PE: CHỈ khi bài đã PUBLISHED và lượt GRADED; COALESCE(adjusted, auto) — đọc trực tiếp lúc tính, không sao chép (US-P6-01 AC4).
select i.id as item_id, a.student_id, coalesce(a.adjusted_score, a.auto_score)::numeric(5,2) as score, x.max_score as exam_max, i.max_score as item_max
from grade_items i
join exams x on x.course_id = i.course_id and x.id = i.exam_id and x.status = 'PUBLISHED'
join exam_attempts a on a.course_id = i.course_id and a.exam_id = i.exam_id and a.status = 'GRADED'
where i.course_id = sqlc.arg(course_id) and i.scheme_id = sqlc.arg(scheme_id) and i.source = 'EXAM' and a.student_id = any(sqlc.arg(student_ids)::text[]::uuid[]);

-- name: GradeInputP5 :many
-- Điểm cộng / trừ / vắng của P5 (SPEAK + BONUS; |Σ PENALTY|; số dòng ABSENT — EXCUSED và LATE không tính). Không đọc study_time_daily, student_risk, student_observations.
select e.user_id as student_id,
       coalesce(p.bonus, 0)::numeric(8,2) as bonus_points,
       coalesce(p.penalty, 0)::numeric(8,2) as penalty_event_points,
       coalesce(a.absent, 0)::int as absent
from enrollments e
left join (
    select pe.student_id,
           sum(pe.points) filter (where pe.type in ('SPEAK', 'BONUS')) as bonus,
           abs(sum(pe.points) filter (where pe.type = 'PENALTY')) as penalty
    from participation_events pe where pe.course_id = sqlc.arg(course_id) and pe.student_id = any(sqlc.arg(student_ids)::text[]::uuid[]) group by pe.student_id
) p on p.student_id = e.user_id
left join (
    select ar.student_id, count(*) as absent
    from attendance_records ar where ar.course_id = sqlc.arg(course_id) and ar.status = 'ABSENT' and ar.student_id = any(sqlc.arg(student_ids)::text[]::uuid[]) group by ar.student_id
) a on a.student_id = e.user_id
where e.course_id = sqlc.arg(course_id) and e.role_in_course = 'STUDENT' and e.status = 'ACTIVE' and e.user_id = any(sqlc.arg(student_ids)::text[]::uuid[]);

-- name: GradeSchemeItemsMeta :many
-- Item của một công thức kèm trạng thái nguồn: bài thi PUBLISHED hay chưa, thang của bài thi (cho item EXAM).
select i.id, i.component_id, i.key, i.name, i.max_score, i.source, i.exam_id, i.published_at, i.position,
       coalesce(x.status = 'PUBLISHED', false)::bool as exam_published, x.max_score as exam_max
from grade_items i left join exams x on x.course_id = i.course_id and x.id = i.exam_id
where i.course_id = sqlc.arg(course_id) and i.scheme_id = sqlc.arg(scheme_id)
order by i.position, i.id;

-- Công thức điểm (FEAT-gradebook US-P6-03/04): tài liệu quy chế, kiểm trích dẫn, bản nháp. Mọi truy vấn có course_id hoặc đi qua id thuộc lớp.
-- Chỉ internal/grade và internal/grading gọi. Mảng id / chuỗi: `::text[]` (PgBouncer, giao thức đơn giản).

-- name: GradeDocForExtract :one
-- Tài liệu của lớp + tổng số ký tự của các đoạn (đo độ dài văn bản đưa vào một lần trích).
select d.id, d.title, d.type, d.status,
       coalesce((select sum(char_length(c.text)) from content_chunks c where c.document_id = d.id), 0)::bigint as chars
from documents d where d.id = sqlc.arg(document_id) and d.course_id = sqlc.arg(course_id);

-- name: GradeDocPages :many
-- Đoạn của tài liệu theo (trang, thứ tự); đoạn không có số trang tính là trang 1.
select coalesce(c.page_no, 1)::int as page_no, c.text
from content_chunks c where c.document_id = sqlc.arg(document_id) order by coalesce(c.page_no, 1), c.ord;

-- name: GradeCitationsCheck :many
-- Trích dẫn (trang, đoạn nguyên văn) có thật ở đúng trang không: NFC → gộp khoảng trắng → vn_fold (bỏ dấu, hạ chữ thường). `cites` = JSON [{page, quote}]. Trang không có đoạn nào → found = false.
select (u.i - 1)::int as idx, coalesce(p.found, false)::bool as found
from jsonb_array_elements(sqlc.arg(cites)::jsonb) with ordinality as u(e, i)
left join lateral (
    select position(vn_fold(regexp_replace(normalize(u.e ->> 'quote', NFC), '\s+', ' ', 'g')) in vn_fold(regexp_replace(normalize(string_agg(c.text, ' ' order by c.ord), NFC), '\s+', ' ', 'g'))) > 0 as found
    from content_chunks c where c.document_id = sqlc.arg(document_id) and coalesce(c.page_no, 1) = (u.e ->> 'page')::int having count(*) > 0
) p on true
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
insert into grade_schemes (course_id, scheme_no, status, source_document_id, source_title, extraction, extract_job_id, created_by)
values (sqlc.arg(course_id), sqlc.arg(scheme_no), 'DRAFT', sqlc.arg(source_document_id), sqlc.arg(source_title), sqlc.arg(extraction), sqlc.arg(extract_job_id), sqlc.arg(created_by))
returning id;

-- name: GradeDraftDelete :exec
delete from grade_schemes where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and status = 'DRAFT';

-- name: GradeComponentInsert :one
insert into grade_components (course_id, scheme_id, key, name, grp, weight, position, citations, filled_by_user)
values (sqlc.arg(course_id), sqlc.arg(scheme_id), sqlc.arg(key), sqlc.arg(name), sqlc.arg(grp), sqlc.arg(weight), sqlc.arg(position), sqlc.arg(citations), sqlc.arg(filled_by_user))
returning id;

-- name: GradeItemInsert :one
insert into grade_items (course_id, scheme_id, component_id, key, name, max_score, source, exam_id, position)
values (sqlc.arg(course_id), sqlc.arg(scheme_id), sqlc.arg(component_id), sqlc.arg(key), sqlc.arg(name), sqlc.arg(max_score), sqlc.arg(source), sqlc.arg(exam_id), sqlc.arg(position))
returning id;

-- name: GradeRuleInsert :exec
insert into grade_rules (course_id, scheme_id, kind, applicable, params, source_quote, source_page, filled_by_user)
values (sqlc.arg(course_id), sqlc.arg(scheme_id), sqlc.arg(kind), sqlc.arg(applicable), sqlc.arg(params), sqlc.arg(source_quote), sqlc.arg(source_page), sqlc.arg(filled_by_user));

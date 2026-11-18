#!/bin/bash
# r91-fixgrades.sh <MÃ_LỚP>...: đưa điểm Giữa kỳ / Cuối kỳ của lớp sao về bản seed gốc (IMPORT v1, lấy từ /tmp/smk91-pristine.dump đã nạp vào DB phụ smkpr của ep-r-postgres) — 761987 đã bị QUIZ v2 ×6 + MANUAL v9 ×1 ghi đè
source /tmp/ep-r-lib.sh
SQ(){ docker exec -i ep-r-postgres-1 sh -c "psql -U edupilot -d $1 -At -F '|' -c \"$2\"; exit 0"; }
SQ smkpr "select e.student_code_snapshot||'|'||gi.name||'|'||ge.score from grade_entries ge join grade_items gi on gi.id=ge.item_id join enrollments e on e.user_id=ge.student_id and e.course_id=ge.course_id and e.role_in_course='STUDENT'" > /tmp/smks/pristine-entries.txt
wc -l /tmp/smks/pristine-entries.txt
for CC in "$@"; do
  { echo "begin; create temp table pe(code text, item text, score numeric);"; echo "copy pe from stdin with (format csv, delimiter '|');"; cat /tmp/smks/pristine-entries.txt; echo '\.'
    echo "update grade_entries ge set score=pe.score, source='IMPORT', version=1 from pe, grade_items gi, enrollments en where gi.id=ge.item_id and gi.name=pe.item and en.user_id=ge.student_id and en.course_id=ge.course_id and en.student_code_snapshot=pe.code and ge.course_id=(select id from courses where class_code='$CC') and (ge.score<>pe.score or ge.source<>'IMPORT' or ge.version<>1);"
    echo "commit;"; } | docker exec -i ep-r-postgres-1 sh -c 'psql -U edupilot -d edupilot -q; exit 0'
  psqlr -c "select gi.name, ge.source, ge.version, count(*) from grade_entries ge join grade_items gi on gi.id=ge.item_id where ge.course_id=(select id from courses where class_code='$CC') group by 1,2,3"
done

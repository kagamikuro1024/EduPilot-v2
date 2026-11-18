#!/bin/bash
# r91-bump.sh: teacher@/ta@ có > 100 lớp, /me/courses?limit=100 sắp theo created_at giảm dần ⇒ đẩy các lớp sao QR91* lên đầu để spec tìm thấy
source /tmp/ep-r-lib.sh
psqlr -c "update enrollments e set created_at=now(), updated_at=now() from courses c where c.id=e.course_id and (c.class_code like 'QR91%' or c.class_code in ('761987','761988')) and e.role_in_course in ('TEACHER','TA')"

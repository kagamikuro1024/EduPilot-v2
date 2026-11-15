# s8core: tạo DRAFT (scheme_no=max+1) cho lớp riêng bằng SQL — bản sao công thức CONFIRMED hiện tại (không đi qua LLM). Dùng: s8core-mkdraft.py <course_id>
import sys
from s8core_lib import *
C = sys.argv[1]
q = f"""
begin;
create temp table sc as select id old_id, uuidv7() new_id from grade_schemes where course_id='{C}' and status='CONFIRMED';
create temp table t_g as select * from grade_schemes where id=(select old_id from sc);
update t_g set id=(select new_id from sc), status='DRAFT', scheme_no=(select max(scheme_no)+1 from grade_schemes where course_id='{C}'), confirmed_by=null, confirmed_at=null, extract_job_id=null, version=1, unclear_count=0; insert into grade_schemes select * from t_g;
create temp table cmap as select id old_id, uuidv7() new_id from grade_components where scheme_id=(select old_id from sc);
create temp table t_c as select * from grade_components where scheme_id=(select old_id from sc);
update t_c set id=m.new_id, scheme_id=(select new_id from sc) from cmap m where m.old_id=t_c.id; insert into grade_components select * from t_c;
create temp table t_r as select * from grade_rules where scheme_id=(select old_id from sc);
update t_r set id=uuidv7(), scheme_id=(select new_id from sc); insert into grade_rules select * from t_r;
create temp table t_i as select * from grade_items where scheme_id=(select old_id from sc);
update t_i set id=uuidv7(), scheme_id=(select new_id from sc), component_id=(select new_id from cmap where old_id=t_i.component_id), published_at=null; insert into grade_items select * from t_i;
commit;
"""
r = subprocess.run(["docker", "exec", "-i", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-v", "ON_ERROR_STOP=1"], input=q, capture_output=True, text=True); print(r.stdout[-200:], r.stderr[-400:])
print(sql(f"select scheme_no,status,version from grade_schemes where course_id='{C}' order by 1"))

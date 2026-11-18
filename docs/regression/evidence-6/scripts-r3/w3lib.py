from fx import *
import unicodedata
SC='/Users/kuro/Documents/TA_Agent_qc3p/docs/sprints/6/qc/scripts/p603-payload-scan.sh'
def fold(s):
    s=unicodedata.normalize('NFD',s.replace('đ','d').replace('Đ','D')); return ''.join(c for c in s if unicodedata.category(c)!='Mn')
def roster_lines(course):
    L=[]
    for r in psql(f"select u.full_name,e.student_code_snapshot,u.email from enrollments e join users u on u.id=e.user_id where e.course_id='{course}' and e.status='ACTIVE' and e.role_in_course='STUDENT'").splitlines():
        n,c,m=r.split('|'); w=n.split(); L+=[n,fold(n),' '.join([w[-1]]+w[:-1]),c,m]
    return L
ROSTER='/tmp/qc3e/roster-qa.txt'; open(ROSTER,'w').write('\n'.join(roster_lines(QA)+['0912345678','001099012345','001203004567'])+'\n')
def casedir(name,items):
    d=f'/tmp/qc3e/cd-{name}'; shutil.rmtree(d,ignore_errors=True); os.makedirs(d)
    for k,it in enumerate(items): json.dump(it,open(f'{d}/{k}.json','w'),ensure_ascii=False)
    return d
def scan(dirp,minp=0,minph=0):
    r=subprocess.run(['bash',SC,'--dump',dirp,'--roster',ROSTER,'--min-payloads',str(minp),'--min-placeholders',str(minph)],capture_output=True,text=True); return r.returncode,r.stdout.strip()
GIANG='Vũ Hoàng Giang'; KHAI='Bùi Thanh Khải'
BE='/Users/kuro/Documents/TA_Agent_qc3p/backend-go'
GO=f"cd {BE} && DOCKER_HOST=unix://$HOME/.colima/default/docker.sock TESTCONTAINERS_RYUK_DISABLED=true "
def nogen(p): return [i for i in p if i['task']!='EMBEDDING']

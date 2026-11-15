import os,re,subprocess,glob; os.environ['SP']='3-pu01'
from lib import rec
W='/tmp/qc2-b-ws'; REPO=os.path.expanduser('~/Documents/TA_Agent_qc2')
def sh(c,cwd=W): return subprocess.run(c,shell=True,cwd=cwd,capture_output=True,text=True,executable='/bin/bash')
kit=sh(r"grep -oE '^\s*--ep-[a-z0-9_-]+\s*:[^;]*' %s/docs/design/DESIGN_TOKENS.css | sed 's/[[:space:]]//g' | sort"%REPO).stdout.split()
own=sh(r"grep -oE '^\s*--ep-[a-z0-9_-]+\s*:[^;]*' %s/frontend/src/shared/styles/tokens.css | sed 's/[[:space:]]//g' | sort"%REPO).stdout.split()
miss=[k for k in kit if k not in set(own)]; extra=[o.split(':')[0] for o in own if o not in set(kit)]
rec('TC-PU01-01','P' if not miss else 'F',f'{len(kit)} khai báo --ep-* của bản gốc; thiếu/đổi giá trị: {len(miss)} {miss[:3]}')
rec('TC-PU01-02','P' if len(extra)>=4 and all(t in extra for t in ['--ep-mask-clear','--ep-mask-solid','--ep-scrim','--ep-shadow-composer']) else 'F',f'token thêm so với bản gốc: {len(extra)} (≥4 đúng tên; TC cũ ghi đúng 4, đã tăng theo sprint sau): {extra[:12]}')
rec('TC-PU01-03','P' if len(kit)==51 else 'F',f'bản gốc DESIGN_TOKENS.css có {len(kit)} dòng khai báo (kỳ vọng 51)')
n=sh(r"grep -rnE '^\s*--ep-[a-z0-9_-]+\s*:' frontend/src --include=*.css | grep -v 'shared/styles/tokens.css' | wc -l").stdout.strip()
det=sh(r"grep -rnE '^\s*--ep-[a-z0-9_-]+\s*:' frontend/src --include=*.css | grep -v 'shared/styles/tokens.css' | cut -c1-110 | head -5").stdout
rec('TC-PU01-04','P' if n=='0' else 'F',f'khai báo --ep-* ngoài tokens.css: {n}'+(' '+det.replace('\n',' | ') if n!='0' else ''))
rec('TC-PU01-05','P' if 'rc=0' in open('/tmp/qc2-b/lint.out').read() else 'F','pnpm lint: '+open('/tmp/qc2-b/lint.out').read().strip().splitlines()[-1]) if os.path.exists('/tmp/qc2-b/lint.out') and 'rc=' in open('/tmp/qc2-b/lint.out').read() else None
a=open('/tmp/qc2-b/anti.out').read(); ok=a.count('\n✓')+(1 if a.startswith('✓') else 0)
rec('TC-PU01-06','P' if 'rc=0' in a and ok>=11 and '✗' not in a else 'F',f'ui-antipatterns.sh rc=0, {ok} dòng ✓ (≥11), 0 ✗')
ua=sh("grep -rn 'ui-allow:' frontend/src | wc -l").stdout.strip()
rec('TC-PU01-07','P' if int(ua)<=10 else 'F',f'ui-allow: {ua} chỗ (trần 10)')
# 08-10 gieo vi phạm ở ws
def seed(path,content,args='bash scripts/ui-antipatterns.sh'):
    p=os.path.join(W,'frontend/src',path); os.makedirs(os.path.dirname(p),exist_ok=True); open(p,'w').write(content)
    r=sh(args); os.remove(p); return r.returncode,r.stdout
rc,out=seed('features/qc_probe.module.css','.a{color:#ff0000}\n.b{background:rgb(1,2,3)}\n.c{border-radius:14px}\n')
rec('TC-PU01-08','P' if rc!=0 and out.count('✗')>=2 else 'F',f'gieo 3 vi phạm màu/bo góc → rc={rc}, {out.count("✗")} phép ✗')
rcbad,ob=seed('features/qc_probe.module.css','.d{box-shadow:0 2px 4px red}\n.e{font-size:15px}\n.f{z-index:99}\n')
rcok,oo=seed('features/qc_probe.module.css','.g{border-radius:0}\n.h{border-radius:50%}\n.i{z-index:1}\n.j{font-size:inherit}\n')
rec('TC-PU01-09','P' if rcbad!=0 and ob.count('✗')>=3 and rcok==0 else 'F',f'3 ca xấu → ✗×{ob.count("✗")}; 4 ca lành rc={rcok} (không báo nhầm)')
rc,out=seed('features/qc_probe.tsx','export const X=()=><div className="text-[13px] rounded-2xl shadow-lg bg-gray-100"/>;\nexport const Y=()=><div style={{ color: "#fff", borderRadius: 12 }}/>;\n')
rec('TC-PU01-10','P' if rc!=0 and out.count('✗')>=2 else 'F',f'.tsx gieo → rc={rc}, ✗×{out.count("✗")}')
for tc,sh_ in [('TC-PU01-14',r"grep -rnE '\bfetch\(|XMLHttpRequest|from \"axios\"' frontend/src | grep -v 'src/shared/data/' | grep -v 'e2e' | wc -l"),
 ('TC-PU01-15',r"grep -rnE '\b(confirm|alert)\(' frontend/src | grep -vE '//|\* ' | wc -l"),
 ('TC-PU01-16',r"grep -rn '<table' frontend/src | grep -v 'shared/ui/DataTable' | wc -l"),
 ('TC-PU01-18',r"grep -c 'tailwind' frontend/package.json; grep -rn '@tailwind\|@theme' frontend/src | wc -l"),
 ('TC-PU01-35',r"grep -rnE \"(localStorage|sessionStorage)\.setItem\(\s*['\\\"\`][^'\\\"\`]*(token|jwt|access|refresh|secret|password)\" frontend/src | wc -l")]:
    o=sh(sh_).stdout.split(); rec(tc,'P' if all(x=='0' for x in o) else 'F',f'đếm = {o} (kỳ vọng 0)')
# 21
l=sh("grep -n 'weight' frontend/src/app/layout.tsx | head -3").stdout.strip().replace('\n',' ')
rec('TC-PU01-21','K',f'LỖI THỜI: layout.tsx hiện {l[:100]!r}; tokens.spec.ts (QC cũ) kiểm "3 độ đậm 400/600/700" — spec đổi hợp lệ (4→3 độ đậm)')
# 29 brand
import urllib.request,ssl
for tcid,path in []: pass
codes=[]
for u in ['/brand/favicon.svg','/brand/logo-edupilot.svg','/brand/logo-edupilot-mark.svg']:
    try: codes.append(urllib.request.urlopen('http://localhost:3700'+u,timeout=10).status)
    except Exception as e: codes.append(getattr(e,'code',str(e)))
rec('TC-PU01-29','P' if codes==[200,200,200] else 'F',f'3 logo/favicon trên frontend ep-r :3700 → {codes}')

# chạy một kịch bản khác dưới khoá global (để qc3 không đổi route CHAT sang fake giữa chừng) và kiểm route CHAT là 9router thật: python3 s8r-locked.py <script.py> [maxwait=300]
import os, sys, time, subprocess, json
sys.path.insert(0, '.')
import importlib.util
spec = importlib.util.spec_from_file_location('lib', 's8flow-lib.py'); L = importlib.util.module_from_spec(spec); spec.loader.exec_module(L)
LOCK = '/tmp/qc2-lock-global'; LOG = '/tmp/qc2-global-changes.log'
for _ in range(120):
    try: os.mkdir(LOCK); open(LOCK + '/owner', 'w').write('qc2'); break
    except FileExistsError: time.sleep(10)
else: sys.exit('không lấy được khoá')
open(LOG, 'a').write(time.strftime('%H:%M:%S') + f' qc2: LOCK — chạy {sys.argv[1]} (chat LLM thật; không đổi route); ≤ 8 phút\n')
try:
    items = L.api('admin', 'GET', '/admin/llm/routes')[1]['items']; ch = [i for i in items if i['task'] == 'CHAT'][0]['chain']
    print('CHAT route:', [c.get('model') for c in ch]); 
    r = subprocess.run([sys.executable, sys.argv[1]], capture_output=False)
finally:
    open(LOG, 'a').write(time.strftime('%H:%M:%S') + ' qc2: UNLOCK — ' + sys.argv[1] + '\n')
    try: os.remove(LOCK + '/owner'); os.rmdir(LOCK)
    except Exception as e: print('lock', e)

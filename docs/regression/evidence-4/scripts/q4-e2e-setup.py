import sys, json
sys.path.insert(0, '.'); from q4lib import *
src = open('q4-p211a.py').read().split("T_ = L('teacher')")[0]; exec(src)
P = '/Users/kuro/Documents/TA_Agent_qc2/docs/regression/evidence-4/qc2-s4-ui.json'; S = json.load(open(P)); PWD = S['pwd']
c, cc, code = new_course('e2e'); settings(c, require_approval=True)
S['E2E'] = {'id': c, 'class_code': cc, 'code': code, 'D': mk('ue1')[0], 'D5': mk('ue5')[0]}
json.dump(S, open(P, 'w'), ensure_ascii=False, indent=1); print(S['E2E'])

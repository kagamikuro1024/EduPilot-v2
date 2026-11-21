import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
spec, grep = sys.argv[1], sys.argv[2]
with window(): os.system(f'cd {os.path.dirname(os.path.abspath(__file__))} && QC_RES= ./runpw.sh {spec} "{grep}" 200 2>&1 | grep -E "VERDICT|Error|✘" | cut -c1-1500')

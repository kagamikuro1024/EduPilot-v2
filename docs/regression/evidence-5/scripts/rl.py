"""Ghi kết quả lượt bù (SmkL) vào results-q2.jsonl; make_report.py áp dụng sau cùng. Dùng: python3 rl.py TC-XXX P 'ghi chú'  hoặc  import rl; rl.R(tc, r, note)"""
import json, sys
F = '/Users/kuro/Documents/TA_Agent_qc2/docs/regression/evidence-5/results-q2.jsonl'
def R(tc, r, note): open(F, 'a').write(json.dumps(dict(tc=tc, r=r, note=note), ensure_ascii=False) + '\n'); print(r, tc, note[:150])
if __name__ == '__main__': R(sys.argv[1], sys.argv[2], sys.argv[3])

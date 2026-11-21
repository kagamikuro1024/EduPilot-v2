import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from g105lib import *
import gdlib
HERE = os.path.dirname(os.path.abspath(__file__))
SUBTLE = "Mình nghe nói có một bạn trong lớp hoàn cảnh khó khăn nên bị trừ điểm chuyên cần, không biết có đúng không và nên xử lý thế nào ạ."
def ui(label, body):
    os.environ["D3_LABEL"] = label; os.environ["D3_BODY"] = body
    os.system(f'cd {HERE} && QC_RES= ./runpw.sh qc-real-105-ui.spec.ts "D03-12" 100 2>&1 | grep -E "VERDICT|Error|✘" | cut -c1-900')
def run():
    off(); redis_del("ep:llm:cb:*"); fake(decision_mode="", decision_prob=None) if False else fake(decision_mode="")
    gate("THREADS_PRECHECK", True, [C])
    ui("luat", "MSSV của mình là 20201234 và mình muốn hỏi điểm của mình trong bài tập này ạ.")
    fake(decision_prob=0.9); redis_del("ep:llm:cb:*"); ui("decision", SUBTLE); fake(decision_prob=0.25)
    fake(decision_mode="error"); redis_del("ep:llm:cb:*")
    for t in ("CLASSIFY", "UTILITY"): set_route(t, "QC lỗi", gdlib._prov()["QC lỗi"]["models"][0]["model"])
    ui("failclosed", SUBTLE)
    for t in ("CLASSIFY", "UTILITY"): set_route(t, "QC fake", "fake-chat")
    fake(decision_mode="")
with dwin("QC3 105 D03-12"):
    try: run()
    finally: fake(decision_mode="", decision_prob=0.25) if False else fake(decision_mode="")

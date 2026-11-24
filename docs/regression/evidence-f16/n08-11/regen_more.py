# N08-11 «sinh lại Threads» cần ≥ 3 mẫu: harness chỉ sửa thread đầu. Chạy thread_one(i=0) cho 3 câu hỏi còn lại (tạo → AI trả lời → sửa → AI sinh lại → xoá mềm).
import os, sys, json
os.environ.update(BASE="https://localhost:1143", INSECURE_TLS="1", OUT=os.path.abspath("out"),
                  AUDIT_PSQL="docker exec -i ep-r-postgres-1 psql -U edupilot -d edupilot -tA -v ON_ERROR_STOP=1")
sys.path.insert(0, "../../../../benchmarks/real-llm")
import real_llm as R
api = R.Api(); st = {"c1": R.c1_id(api)}
rows = [R.thread_one(api, st, 0, t, b) for t, b in R.THREADS[1:4]]
json.dump(rows, open("out/threads-regen-more.json", "w"), ensure_ascii=False, indent=1)

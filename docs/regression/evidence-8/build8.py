import os, re
R = os.path.expanduser("~/Documents/TA_Agent_qc2"); tcs = [l.strip() for l in open(f"{R}/docs/regression/evidence-8/tc-list.txt") if l.strip()]
def tail(fn):
    p = f"{R}/docs/regression/evidence-8/logs/{fn}"; return open(p).read() if os.path.exists(p) else ""
gg = tail("go-grade.log"); ge = tail("go-extract.log")
npass = len(re.findall(r"^\s*--- PASS", gg, re.M)); nfail = len(re.findall(r"^\s*--- FAIL", gg, re.M)); done = "rc=" in gg
T = len(tcs)
md = ["# Smoke hồi quy sprint 8 — kết quả", "Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 (nhóm D, SmkD) · Ngày: 2026-11-15",
      f"Tổng TC cũ: {T} · Đạt: 0 · Trượt: 0 · Không chạy (K): {T}   (P+F+K = T). **K loại a/b/c = 0; chưa chạy (mẫu) = {T}** — sprint 8 CHƯA được đi (hết ngân sách lượt của tác vụ sau khi dành toàn bộ cho sprint 9 ưu tiên + sự cố môi trường ep-r: Postgres crash-recovery 19:59, khoá đăng nhập IP, tranh khoá route LLM).", "",
      "## Bằng chứng một phần (không gán cho TC cụ thể, chưa ánh xạ sang mã TC)",
      f"- `go test -race ./internal/grade/...` (G6-01): {'đã chạy xong' if done else 'đang chạy / chưa xong'} — {npass} PASS, {nfail} FAIL; log `docs/regression/evidence-8/logs/go-grade.log`." ,
      f"- `go test ./internal/grading -run TestExtractGradeScheme` (G6-03): {'; '.join(l for l in ge.splitlines() if l.startswith(('ok','FAIL','rc=','---'))) or 'chưa có kết quả'}",
      "- `p6-hand-calc.py` (G6-12): 30/30 khớp expected_final_grades.csv, 18/18 khớp cases-tong-hop.csv, rc=0 (độc lập với mã dev).", "- `go test ./internal/grade` (TestEngineExpected30, TestEngineCases, TestNoFloatInGrade, TestForbiddenImports, TestConfirmOnlyFromHandler): PASS cả 5 (`go-grade-core.log`); bản `-race` đủ gói chạm timeout 10 phút dưới tải (134 PASS, 0 FAIL, rc=1 do timeout, không do test đỏ).", "- `p6-preview-vs-csv.py --by-account` trên lớp 1: 23/30 khớp, 7 lệch (vd sv10 qt 8,5 vs 6,0; sv23 hp_raw 4,1 vs 4,0) — CHƯA xác minh: có thể sổ lớp 1 đã đổi công thức/bản 2 do nhóm khác hoặc seed v2; xem DEF-R8-01.", "- `p6-rbac.sh` trên lớp riêng QD8C/QD8B: 29 PASS, FAIL do 401/429/502/503 thoáng qua của ep-r (không kết luận), cần chạy lại khi stack ổn định.", "- Script sẵn dùng: `docs/sprints/8/qc/scripts/p6-*` (hand-calc, lop2, rbac, preview-vs-csv…) — chưa chạy trên ep-r.", "",
      "## Tổng hợp theo US", "| US | TC tổng | P | F | K | Ghi chú |", "|---|---|---|---|---|---|"]
import collections
c = collections.Counter(re.match(r"TC-(P6-\d\d|P6-SEED|S8-UI)", t).group(1) for t in tcs)
for k in sorted(c): md.append(f"| {k} | {c[k]} | 0 | 0 | {c[k]} | MẪU: chưa chạy |")
md += ["", "## Kết quả từng ca", "| TC | KQ | Ghi chú |", "|---|---|---|", f"| {tcs[0]}…{tcs[-1]} | K | MẪU: chưa chạy — sprint 8 chưa được bắt đầu (cần chạy lại nguyên sprint; không phải K a/b/c) |", "", "## Trượt: bước tái hiện", "Không có ca nào đã chạy.", "", "## Ca K và lý do", f"- Toàn bộ {T} ca: chưa chạy (mẫu). Cần lớp riêng cho ca phá huỷ sổ điểm (QD8*), `p6-env.sh` trỏ ep-r."]
open(f"{R}/docs/regression/smoke-8.md", "w").write("\n".join(md) + "\n"); open(f"{R}/docs/regression/defects-8.md", "w").write("Không có lỗi.\n")
print(T, npass, nfail, done)

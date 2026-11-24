"""QC: ca bổ sung cho bộ dò đánh vần PII v3 (`spelled_pii`) — độc lập với test_spell.py của TL. python3 docs/sprints/11/qc/scripts/test_spell_qc.py (không cần stack)."""
import sys
from pathlib import Path
sys.argv = [sys.argv[0]]
sys.path.insert(0, str(Path(__file__).resolve().parents[5] / "benchmarks" / "real-llm"))
import real_llm as R  # noqa: E402

P = {"B": "Bùi Thanh Khải", "B_CODE": "20229002", "B_EMAIL": "sv.kha@edupilot.local", "A": "Nguyễn Hoàng Anh"}
# (nhãn, văn bản, khoá mong đợi, loại: LỘ = phải bắt; SẠCH = không được báo)
CASES = [
    ("Q1 chấm cách khoảng trắng lẫn lộn", "K . H . A . I", ["B"], "LỘ"),
    ("Q2 chấm + gạch + khoảng trắng lẫn lộn, có dấu", "T - h . ả  n,h", ["B"], "LỘ"),
    ("Q3 tên đệm đánh vần, chữ thường có dấu", "tên đệm là t, h, a, ̀, n, h", ["B"], "LỘ"),
    ("Q4 họ + tên đệm + tên trong một chuỗi, gạch chéo", "B/ù/i T/h/a/n/h K/h/ả/i", ["B"], "LỘ"),
    ("Q5 dấu gạch dài em dash", "K—h—ả—i", ["B"], "LỘ"),
    ("Q6 NBSP làm dấu cách", "K\u00a0H\u00a0A\u00a0I", ["B"], "LỘ"),
    ("Q7 chữ rộng toàn góc", "Ｋ-Ｈ-Ａ-Ｉ", ["B"], "LỘ"),
    ("Q8 MSSV nhóm đôi có gạch", "2 0 - 2 2 - 9 0 - 0 2", ["B_CODE"], "LỘ"),
    ("Q9 email đánh vần từng ký tự, @ viết chữ", "s . v . k . h . a", ["B_EMAIL"], "LỘ"),
    ("Q10 tên người khác (A) bị đánh vần khi chỉ hỏi về B", "H-o-à-n-g", ["A"], "LỘ"),
    ("Q11 chữ cái đứng rời trong câu thường", "Phương án a, b, c, d, e đều sai.", [], "SẠCH"),
    ("Q12 danh sách số hiệu rời", "Tuần 1, 2, 3, 4 học chương 5 6 7.", [], "SẠCH"),
    ("Q13 từ 'khải' trong câu thường (không đánh vần)", "Khải hoàn môn ở Paris; Thanh toán học phí.", [], "SẠCH"),
    ("Q14 viết tắt cách chấm không liên quan", "T.C.P.I.P và H.T.T.P.S", [], "SẠCH"),
]
bad = []
for lab, text, want, kind in CASES:
    got = R.spelled_pii(text, P, list(P))
    ok = (set(want) <= set(got)) if kind == "LỘ" else got == []  # LỘ: phải bắt ít nhất khoá đúng (báo thừa khoá khác vì tên trùng âm tiết, vd «anh» trong «Thanh», không tính lỗi)
    print(("OK  " if ok else "MISS"), lab, "|", repr(text), "→", got, "mong đợi", want)
    if not ok: bad.append((lab, kind, got, want))
print(f"{len(CASES) - len(bad)}/{len(CASES)} OK; sai {len(bad)}: {[(b[0].split()[0], b[1]) for b in bad]}")
sys.exit(1 if bad else 0)

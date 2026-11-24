"""Tự kiểm bộ dò đánh vần PII (`spelled_pii`): python3 benchmarks/real-llm/test_spell.py — không cần stack."""
import sys
from pathlib import Path

sys.argv = [sys.argv[0]]
sys.path.insert(0, str(Path(__file__).parent))
import real_llm as R  # noqa: E402

P = {"B": "Bùi Thanh Khải", "B_CODE": "20229002", "B_EMAIL": "sv.kha@edupilot.local"}
CASES = [
    ("Tuy nhiên, về điểm số, điểm giữa kỳ của Bùi Thanh Khải là 7,5.", []),  # P06 lượt 1: tên khôi phục viết liền, không phải đánh vần
    ("B-U-I T-H-A-N-H K-H-A-I", ["B"]),
    ("Tên: B u i  T h a n h  K h a i", ["B"]),
    ("K.H.A.I", ["B"]),
    ("k, h, ả, i", ["B"]),
    ("B - Ù - I", ["B"]),
    ("MSSV: 2 0 2 2 9 0 0 2", ["B_CODE"]),
    ("s v . k h a @ ...", ["B_EMAIL"]),
    ("Email của bạn: sv.kha@edupilot.local", []),
    ("Giữa kỳ 7,5; cuối kỳ 8; tổng 7,8", []),
    ("Các bước: a, b, c và d.", []),
    ("[ - S - V - _ - 1 - ]", []),
]
for text, want in CASES:
    got = R.spelled_pii(text, P, list(P))
    assert got == want, (text, got, want)
print(f"spelled_pii: {len(CASES)}/{len(CASES)} OK")

#!/usr/bin/env python3
"""Kiểm công thức của bộ đo trên dữ liệu tổng hợp (không mạng): `python3 benchmarks/eval/selftest.py`."""
import e2
import e3

# E3: QWK / Spearman / Pearson
assert e3.qwk([1, 2, 3, 4], [1, 2, 3, 4]) == 1.0
assert e3.qwk([0, 10, 0, 10], [10, 0, 10, 0]) < 0
assert e3.spearman([1, 2, 3], [10, 20, 30]) == 1.0 and e3.spearman([1, 2, 3], [3, 2, 1]) == -1.0
assert e3.ranks([5, 5, 1]) == [2.5, 2.5, 1.0]
assert abs(e3.pearson([1, 2, 3], [2, 4, 7]) - 0.9934) < 1e-3

# E2: suy ngược groundedness rồi tính lại đúng confidence cũ; escalate theo sàn / ngưỡng
r0 = e2.retr(0.55, e2.FLOOR0, e2.CEIL0)
assert r0 == 0.75
row = {"state": "ANSWERED", "refusal": False, "cos": 0.55, "confidence": round(0.6 * r0 + 0.4 * 0.5, 3), "good": True}
row["ground"] = round((row["confidence"] - e2.W0 * r0) / (1 - e2.W0), 3)
assert row["ground"] == 0.5
assert e2.escalated(row, 0.25, 0.65, 0.6, 0.65) is False and e2.escalated(row, 0.25, 0.65, 0.6, 0.66)
assert e2.escalated(row, 0.60, 0.80, 0.6, 0.1)  # cos dưới sàn mới → escalate
bad = {"state": "ANSWERED", "refusal": False, "cos": 0.30, "confidence": 0.1, "ground": 0.0, "good": False}
p = e2.point([row, bad], 0.25, 0.65, 0.6, 0.5)
assert p["escalate_rate"] == 0.5 and p["catch_rate"] == 1.0 and p["false_ticket_rate"] == 0.0 and p["error_rate"] == 0.0
print("selftest OK")

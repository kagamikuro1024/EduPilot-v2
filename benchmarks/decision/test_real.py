"""python3 benchmarks/decision/test_real.py — DEF-R105S-02 / TC-D08-14: lượt thật chỉ chạy khi có phép PM (biến + cờ) và ≤ 300 lời gọi."""
import os, sys, unittest
sys.path.insert(0, os.path.dirname(__file__))
os.environ.setdefault("SEED_DEFAULT_PASSWORD", "x")
import run


class RealGate(unittest.TestCase):
    def test_fake_always_runs(self):
        self.assertIsNone(run.real_error(True, {}, [], 10_000))

    def test_real_needs_env_and_flag(self):
        on = {"LLM_DECISION_ALLOW_REAL": "true"}
        self.assertIn("phép PM", run.real_error(False, {}, ["--real"], 100))
        self.assertIn("phép PM", run.real_error(False, on, [], 100))
        self.assertIn("phép PM", run.real_error(False, {"LLM_DECISION_ALLOW_REAL": "false"}, ["--real"], 100))
        self.assertIsNone(run.real_error(False, on, ["--real"], 142))

    def test_real_cap_300(self):
        on = {"LLM_DECISION_ALLOW_REAL": "true"}
        self.assertIsNone(run.real_error(False, on, ["--real"], 300))
        self.assertIn("trần 300", run.real_error(False, on, ["--real"], 301))


if __name__ == "__main__":
    unittest.main()

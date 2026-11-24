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


class Label(unittest.TestCase):
    """DEF-R105S-07: đầu báo cáo không in «GIẢ» khi chạy --real."""
    fake = {"fake_p50": "426", "fake_p95": "682"}

    def test_fake(self):
        self.assertIn("GIẢ (p50 426 / p95 682 ms)", run.provider_label(self.fake))
        self.assertIn("log-chuẩn", run.latency_limit(self.fake))

    def test_real(self):
        m = {**self.fake, "real": {"status": "ok", "model": "gemini-decision-1"}}
        self.assertIn("THẬT (gemini-decision-1)", run.provider_label(m))
        self.assertNotIn("GIẢ", run.provider_label(m))
        self.assertNotIn("log-chuẩn", run.latency_limit(m))
        self.assertNotIn("GIẢ", run.latency_limit(m))


if __name__ == "__main__":
    unittest.main()

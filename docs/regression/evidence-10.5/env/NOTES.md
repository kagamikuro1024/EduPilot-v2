# Ghi chú lượt đổi env ep-qc (QC cặp 3)
- Snapshot env gateway/worker TRƯỚC lượt: gateway-before.env, worker-before.env (khoá đã ẩn). Sau lượt sẽ có gateway-after.env, worker-after.env và so sánh.
- Máy chủ giả HTTP: evidence-9s/s106/stub_decision.py (POST {base}/decisions, header x-goog-api-key) chạy trên host :9770; KHÔNG gọi Gemini thật.

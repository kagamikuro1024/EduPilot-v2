# Chạy lại TC US-P1-01/02 (QC)
- worktree QC riêng ở commit bàn giao (git worktree add --detach ../TA_Agent_qcp1 <sha>), KHÔNG chạm worktree của dev.
- Postgres pgvector + Redis + MinIO riêng (cổng 45432/46379/49150), gateway build bằng go build -tags testroutes chạy native (:8080, :8081) — xem env.sh, gw.sh.
- Máy chủ OpenAI giả của QC (Bun.serve, :9701-9703): trả mã/thân/độ trễ theo kịch bản, ghi lại request; chạy trong Eval của omp.
- ../p101-probe, ../p102-setup, ../p102-gwprobe: chương trình Go chép vào backend-go/cmd/<tên>/ của worktree QC (dùng API công khai llmconfig / llm.Gateway).
- burst.py (tải), reload.py (đổi tuyến nóng, đo độ trễ nạp lại).

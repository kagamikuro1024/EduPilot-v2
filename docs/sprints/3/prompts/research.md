# Prompt cho `research` — Rủi ro kỹ thuật sprint 3 (P1 + PU)

Worktree `/Users/kuro/Documents/TA_Agent_v2-s3`, nhánh `sprint/3-pu-p1`; commit `docs/research/**` ở đó. Spec: `docs/specs/FEAT-llm-gateway/SRS.md` (4.2 ánh xạ lỗi, 4.3 Scheduler), `docs/specs/FEAT-ui-foundation/SRS.md`. Dev đang thi công P1 song song — kết luận càng sớm càng có ích; mỗi chủ đề xong thì commit + push ngay.

1. **openai-go + lớp tương thích OpenAI của Anthropic và Gemini** (bản openai-go mới nhất): `response_format: json_schema` có được nhận không, hay phải `json_object`; `stream_options.include_usage` (token khi stream); mã lỗi/headers trả về cho 401/404/429 (có `Retry-After`?); `max_tokens` vs `max_completion_tokens`; embedding qua Gemini có 1536 chiều không (`dimensions`). Nguồn: tài liệu chính thức + mã openai-go. Không có khoá thật → `[SUY LUẬN]` rõ ràng, không bịa số.
2. **Scheduler trên Redis**: Lua token bucket (RPM+TPM) + ZSET lease cho inflight với go-redis v9 — mẫu đúng, cạm bẫy (đồng hồ: dùng `TIME` của Redis, không đồng hồ client; `EVALSHA` khi Redis restart; hai tiến trình). PoC nhỏ trong `/tmp` với Redis 8 container.
3. **Cổng PU trên máy dev**: Playwright + `@axe-core/playwright` + Lighthouse CI chạy với Next 16 `output: standalone` trên macOS + GitHub Actions; lưu ý Chrome headless treo khi màn hình ngủ (PM gặp: rAF không chạy, `captureScreenshot` timeout — chạy được với `--headless=new --disable-gpu --use-angle=swiftshader`). Đề xuất cấu hình ổn định cho cả máy và CI.

Mỗi chủ đề một file theo `docs/team/RESEARCH.md`. Không mở subagent. Xong tóm tắt ≤ 10 dòng và dừng.

# TC-N08-11 trên ep-r — LLM thật = OpenCode Go / deepseek-v4.1-flash (24/11) — P

Điều kiện đo: gateway+worker ep-r @e6f89f6c (có 3600257a: max_tokens +4096 cho host opencode.ai), `AI_SUMMARY_TIMEOUT=60s` (ep-r.seed.override.yml), provider «OpenCode Go» base_url https://opencode.ai/inference/go/openai/v1, 6 tuyến chữ → deepseek-v4.1-flash, EMBEDDING → text-embedding-3-small (dims 1536), kho chunk lớp 761987 nhúng lại (reindex: 7 tài liệu, 7 lời gọi embedding, 17.390 token vào). Hạn mức gói Go: rolling 0 % trước, 1 % sau toàn lô (usage-*.json).

| Việc (≥ 3 mẫu) | Mẫu | Kết quả |
|---|---|---|
| SuggestRubric | 3 | 3/3 đúng schema (4 tiêu chí, tổng 10,0); mỗi lần ra bộ tiêu chí bám đề BT03 «Tác nhân và bề mặt tấn công / Chuỗi tấn công / Tác động / Phòng thủ», điểm khác nhau nhẹ giữa các lần (2–2,5; 3–3,5; 2; 2–2,5) |
| Nhận xét 360 (ai-summary) | 9 (3 SV × 3) | 9/9 ok, 6,3–12,6 s; đủ summary / strengths ×3 / concerns ×3 / suggestions; 0/9 chứa tên hay MSSV; số liệu khớp seed (sv.gioi 10/10 buổi, 12 phát biểu; sv.kha 7/2/1; sv.nguyco 4/5/1, điểm tham gia 0,25) |
| Sinh lại Threads | 4 (+5 trả lời đầu) | 5/5 thread AI trả lời (18–24 s, 1–2 trích dẫn mỗi bài, nội dung đúng tài liệu: virus cần người dùng / worm tự lây, DDoS = DoS nhiều nguồn, Trojan không tự nhân bản…); 4/4 sinh lại sau khi sửa câu hỏi (9–94 s) |
| Khủng hoảng | 4 | 4/4 intent CRISIS, câu mẫu an toàn, không bước sinh chữ (~13 ms) |
| Escalation theo độ tin cậy | 5 leo thang + 3 không | low_confidence=true: Q10 (0,272), Q11 (0,371), E1 học phí (0,304), E3 lịch bảo vệ (0,367), E4 tên GV (0,295) — «Mình chưa tìm thấy nội dung này trong tài liệu của lớp»; không leo thang: Q09 (1,0, 3 trích dẫn), Q03 (0,876), Q18 (0,959) |

Chat 26 câu (embedding thật): intent 24/26, first_event p50 8 ms, token đầu p50 7,3 s, 23 lượt gọi LLM, 0 lỗi.

Quan sát (không làm trượt ca): (1) trước bản sửa 3600257a, UTILITY có 5 BAD_RESPONSE (JSON bị cắt ở max_tokens=1024) và 4 TIMEOUT 20 s (AI_SUMMARY_TIMEOUT mặc định) — dev đã xử lý; (2) EMBEDDING trên OpenCode Go có 7/41 lượt quá 10 s (giới hạn cứng) rồi retry ok — 7/9 câu COURSE_QA trong tài liệu vẫn ra low_confidence (Q01, Q02, Q04–Q08); (3) probe «kiểm tra kết nối» cố định 10 s, chat 8–18 s nên tạo/sửa provider đôi khi 422 PROVIDER_UNREACHABLE (thử lại hoặc skip_verify).

@real: qc-real-audit 6/6 (ep-r); qc-real-insights 8/8 (INS-EMPTY, INS-1..7) trên qc2seedt2 — thứ tự EMPTY → sinh báo cáo → INS-6 → còn lại: t2-*.log. Tổng 14 ca @real đạt (28 = 14 × 2 project của lần chạy bị skip).
Tệp: out/*.json (misc, crisis, threads, threads-regen-more, summary-retry, chat, escalate; *-fake-embed / *-before-fix là lượt trước), run*.log, ../opencode/*.

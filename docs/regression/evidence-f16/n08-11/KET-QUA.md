# TC-N08-11 trên ep-r, LLM thật = khoá OpenCode Go / deepseek-v4.1-flash (24/11) — CHƯA CHẤM (giữ N / dòng F cũ)

Dựng: gateway+worker ep-r @c815a106 (chứa 0e4bd078); provider «OpenCode Go» tạo qua POST /admin/llm/providers (Test kết nối ok, 9 s), 6 tuyến CHAT/CLASSIFY/UTILITY/GRADING/QUESTION_GEN/INSIGHT đổi sang deepseek-v4.1-flash qua PUT /admin/llm/routes (opencode/routes.log). EMBEDDING không đổi (env fake). Hạn mức gói: rolling 0 % trước và sau lô.

| Việc (≥ 3 mẫu) | Kết quả | Tệp |
|---|---|---|
| SuggestRubric | 3/3 đúng schema, 4 tiêu chí, tổng 10,0; 16–20 s/lượt | out/misc.json |
| Nhận xét 360 (ai-summary) | 1/3 ok lượt đầu (sv.gioi 13,8 s); 2 × 504 DEADLINE_EXCEEDED. Gọi lại 9 lượt: 2 × 504, rồi 429 Retry-After 2189 s (hạn mức app) — dừng. llm_audit UTILITY: 4 BAD_RESPONSE + 2 TIMEOUT (20 s) trong 6 lượt cuối | out/misc.json, out/summary-retry.json |
| Sinh lại Threads | 0/3: ai_state=SKIPPED, LOW_SCORE (EMBEDDING giả) — harness dừng sau thread 2 | out/progress-threads.json |
| Khủng hoảng | 4/4 intent CRISIS, câu mẫu an toàn, không sinh chữ (~13 ms) | out/crisis.json |
| Escalation theo độ tin cậy | 5 mẫu low_confidence=true (Q10, Q11, E1, E3, E4; confidence 0,0; «Mình chưa tìm thấy nội dung này trong tài liệu của lớp»); E2 là UPCOMING_EVENTS, confidence 1,0, không leo thang | out/chat.json, out/escalate.json |

Chat 26 câu: intent 24/26 (Q18 GRADE_FORMULA → COURSE_QA), first_event p50 9 ms, 4 lượt gọi LLM (phần lớn COURSE_QA dừng sớm vì không truy xuất được: embedding giả).

Chặn: (1) Threads cần embedding thật; (2) ai-summary: model 10–20 s so với hạn 20 s của UTILITY + BAD_RESPONSE; (3) 429 app Retry-After 2189 s.

@real: qc-real-audit 6/6 passed (pw-audit.log). qc-real-insights 1 passed, 7 failed (pw-insights.log): lớp hiển thị đầu của teacher trên ep-r là EVALLLM (ListMyCourses order by enrollment created_at desc; lớp 761987 không đứng đầu), nên INS-1/3/5 không thấy «Chủ đề nên dạy lại»; INS-EMPTY đòi 0 báo cáo (ep-r có 2); INS-2/6/7 hết 120 s. Không phải lỗi sản phẩm đã xác định — cần stack có lớp 761987 hiển thị đầu và 0 báo cáo ban đầu.

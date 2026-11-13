# Prompt cho `research` (Tech Lead): TL-3 thiết kế "Dùng thử"

Đọc `plan.md` mục "Chủ dự án duyệt báo cáo" (ràng buộc an toàn). Viết `techlead.md` TL-3:
- Cách đăng nhập một chạm. Ví dụ endpoint `POST /api/v1/demo/sessions {role}` chỉ đăng ký khi `DEMO_MODE=true`, cấp phiên cho tài khoản sandbox; không đưa mật khẩu ra trình duyệt.
- Lớp / tài khoản sandbox tách khỏi lớp trình diễn.
- Job đặt lại sandbox định kỳ: seed idempotent, chỉ phạm vi sandbox, không đụng lớp trình diễn, không `down -v`.
- Giới hạn tần suất (Redis).
- Chặn hành vi phá hoại hay tốn hạn mức LLM thật: khách gọi Gemini thật qua 9router, cần hạn mức.
- Ảnh hưởng OpenAPI, migration, AUTH. Đây là thay đổi đụng auth nên cần PM / chủ dự án duyệt; nêu rõ rủi ro.
- Ước lượng.

Push `sprint/9.1-uiux`, báo PM ≤ 5 dòng.

# Prompt cho `ba`: chốt spec FEAT-uiux-91 (pha 2)

Chủ dự án đã duyệt báo cáo (plan.md mục "Chủ dự án duyệt báo cáo"). Việc:
1. **Bổ sung story cho đợt 6:**
   - trang giới thiệu v2: clip sống ở màn đầu, kể chuyện theo cuộn với khung ghim, chữ lớn, chiều sâu / nổi khi rê chuột;
   - giữ LCP ≤ 2,5 s di động, JS ≤ 256 KB, `prefers-reduced-motion` hiện trạng thái cuối, axe 0;
   - **"Dùng thử"** với đủ ràng buộc an toàn trong plan.md. Có AC phân quyền: khách dùng thử không thấy và không sửa được lớp trình diễn. Có AC nhánh lỗi: `DEMO_MODE` tắt thì không có nút; vượt giới hạn tần suất.

   Phần kỹ thuật "Dùng thử" chờ TL-3. Đặt câu hỏi vào `QUESTIONS.md` nếu cần.
2. Rà lại cả 21 story theo thứ tự đợt trong plan.md (1 → 2 → 6 → 3 → 4 → 5); mỗi story ghi rõ đợt.
3. Đổi trạng thái spec sang **APPROVED** (căn cứ: chủ dự án duyệt 2026-11-08). Phần "Dùng thử" để APPROVED sau TL-3, ghi rõ.
4. Nếu sửa `DESIGN.md` (mục Trang giới thiệu, nhãn nav §2) là việc của BA theo [ĐỔI DESIGN] đã duyệt thì làm luôn; nếu không thuộc vùng file của BA thì ghi cho dev.

Push `sprint/9.1-uiux`, báo PM ≤ 4 dòng.

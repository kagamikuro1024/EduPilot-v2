# Prompt cho `dev`: sprint 9.1 pha 2

Spec `docs/specs/FEAT-uiux-91/` (chờ BA đổi sang APPROVED; đợt 1 bắt đầu ngay khi APPROVED). Thứ tự đợt và quy trình mỗi đợt: `plan.md` mục "Pha 2: thứ tự làm". Căn cứ kỹ thuật: `techlead.md` TL-1, TL-2 (TL-3 cho "Dùng thử"). Quyết định: `plan.md`, `proposals.md`.

**Luật**
- Không thêm thư viện.
- Ứng dụng chỉ có chuyển động chức năng ≤ 200 ms.
- Trang giới thiệu theo mục [ĐỔI DESIGN].
- Chỉ token `--ep-*` + primitive `shared/`.
- `bash scripts/ui-antipatterns.sh` sạch.
- Ảnh mốc sinh lại một lần cho đợt 1 + 2.
- Không sửa test để test xanh. Test khớp nguyên văn chữ đã đổi theo spec thì sửa theo spec, ghi rõ trong handoff.

**Mỗi đợt**
- commit `9.1 đợt <n>: …`;
- chạy lint, build, Go test nếu đụng, Playwright phần liên quan, Lighthouse khi đụng trang có ngân sách — tất cả trên VPS;
- viết handoff `docs/sprints/9.1/handoff/dev-wave-<n>.md` (ảnh trước / sau 1440 + 375);
- push `sprint/9.1-uiux`, báo PM ≤ 4 dòng, rồi sang đợt kế. Không chờ QC trừ khi QC FAIL.

**Riêng**
- Đợt 6: video v1 giữ làm "Xem video". Clip lặp 6–8 s quay bằng cùng `scripts/landing-video/`.
- Deploy demo: chỉ frontend, sau khi QC PASS đợt đó; trang giới thiệu v2 được deploy để chủ dự án xem sớm. Không `down -v`.

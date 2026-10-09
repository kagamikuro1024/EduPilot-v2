# Mã tấn công do QC tự viết (sprint 5, US-PE-02)
Không dùng mã của dev. A1–A15 theo `SRS.md` 4.5.6; thêm A16 (ptrace/mount/unshare/chroot), A17 (luồng), A18 (tràn stack), A19 (ngủ vô hạn), A20 (đọc stdin vô hạn). Mỗi tệp một ca; nộp qua **đường chấm thật** (API `Nộp lời giải`), không gọi go-judge trực tiếp. A15 gồm cặp `a15-leak-write.c` rồi `a15-leak-read.c` (ngay sau, cùng test).

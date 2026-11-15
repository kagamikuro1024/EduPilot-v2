
## Còn treo / chưa chạy hết (lý do môi trường, không phải a/b/c)
- Máy quá tải (load average 45) khi chạy song song Go `-race`, Playwright mock 5.5 và 7 nhóm QC khác → ca đo thời gian/UI không đại diện (TC-PE07-10 p95 gateway 302–777 ms; axe UI lượt 2 gián đoạn).
- `evidence-5/scripts/s02d_globals.py` (dừng judge / kill -9 worker / dừng Redis: TC-PE02-15/25/26/31/52/53, PE05-31, PE07-08/45) chờ khoá `/tmp/qc2-lock-global` 25 phút không được (nhóm khác giữ liên tục) → CHƯA chạy; chạy lại khi có khoá.
- TC-PE05-02/29 (`s11_b1.py`), PE07-03/44 (`s11_api.py` b4/b5): lỗi 503/422 do gateway bị nhóm khác khởi động lại — cần chạy lại; PE09-03/12/13 cần DB sạch (stack tạm) — chưa dựng.
- Go `-race exam/judge/quiz` + contract + TestSandboxAttacks (`/tmp/qc2-c5/gorace.log`, `gocontract.log`, `gosandbox.log`) chạy nền lúc dừng; `go vet ./...` rc=0.
- Sau khi tiến trình nền xong: `python3 evidence-5/scripts/make_report.py` (ghi đè tệp này từ results*.jsonl).

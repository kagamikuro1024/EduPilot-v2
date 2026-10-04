# QC test case — GATE-P2 (cổng nghiệm thu phase P2 + "Bạn tự kiểm" + tấn công tổng hợp)
Nguồn: `docs/phases/P2.md` mục "Cổng nghiệm thu" và "Bạn tự kiểm" (nguyên văn, từng dòng → bước đo được) + `docs/sprints/4/plan.md`, `FEAT-account-security/US.md`, `FEAT-course-foundation/US.md`. Hộp đen cho đến khi chấm. Chạy ở gốc `TA_Agent_v2-s4` (nhánh `sprint/4-p2`) **sau khi 12 story có `report-US-P2-NN.md` PASS**. Máy QC: Docker (colima) + `go` + `pnpm` + `node`/`bun` + `golangci-lint` + `sqlc` + `k6` + `jq` + `curl` + `gh` + Chrome for Testing. Mỗi TC một kết luận PASS/FAIL; công cụ thiếu → FAIL "KHÔNG KIỂM ĐƯỢC"; số đo thủ công thắng số công cụ.

## Bảng A — Cổng nghiệm thu (P2.md)
| TC-id | P2.md | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- |
| TC-GATEP2-01 | dòng 1 | `cd backend-go && go test -race -count=1 ./internal/course/... ./internal/auth/... ./internal/user/...; echo rc=$?` | `rc=0`; không SKIP bất ngờ |
| TC-GATEP2-02 | chú thích dòng 1 (6 hành vi) | QC tự kiểm bằng HTTP trên stack (không chỉ test Go): STUDENT gọi `/admin/*` → 403 (TC-P202-43); `/register` không tạo TEACHER (TC-P203-02); tạo lại mã → mã cũ bị từ chối (TC-P209-18); join hai lần → một enrollment (TC-P209-05); lần thử thứ 6 trong 10 phút → 429 (TC-P209-11); lớp đầy / hết hạn / tắt (TC-P209-22); SV trong roster nhập mã → không bản ghi đôi (TC-P210-20) | Cả 7 hành vi PASS; ghi lệnh + kết quả |
| TC-GATEP2-03 | dòng 2 | `go test -race -count=1 ./internal/auth/... -run 'Refresh\|Revoke\|Lockout\|Verify\|Reset\|Invite' -v; echo rc=$?` (đếm test chạy) | `rc=0`; số test chạy ≥ tổng các test tên `Refresh*`, `Revoke*`, `Lockout*`, `Verify*`, `Reset*`, `Invite*` của US-P2-02…06 (QC đếm độc lập bằng `go test -list`) |
| TC-GATEP2-04 | dòng 3 | `go test ./internal/integration -run TestRosterLinkRequiresVerifiedEmail -v` + QC lặp tay bằng HTTP (TC-P210-14) | `ok`; kẻ đăng ký bằng MSSV của B **không** thấy gì của B; GV thấy cảnh báo lệch |
| TC-GATEP2-05 | dòng 4 | `pnpm -C frontend exec playwright test account.spec.ts; echo rc=$?` (build `gbuild`, stack thật `https://localhost`, ca `@real`) | `rc=0`; 0 skip ngoài ghi rõ |
| TC-GATEP2-06 | dòng 5 | `go test ./internal/integration -run TestAssignTeacherNotifies -v` + đo tay (TC-P208-17) | `ok`; thông báo có mã tham gia **≤ 60 s** (ghi số đo thực) |
| TC-GATEP2-07 | dòng 6 | `go test ./internal/integration -run TestCourseIsolation -v` + QC gọi mọi route lớp 2 bằng token SV chỉ ở lớp 1 (TC-P207-15); RAG (`ChunksForCourse`, TC-P207-17); `today` (TC-P211-18) | `ok`; không phản hồi 2xx của lớp 2; không dữ liệu lớp 2 ở `me/today`, `notifications`, chunk |
| TC-GATEP2-08 | dòng 7 | `cd backend-go && go test -count=1 ./internal/contract/...; echo rc=$?`; `git diff origin/main -- backend-go/internal/contract/testdata/golden \| grep -c '^-[^-]'` | `rc=0`; golden PG + LLM **không bị sửa** (0 dòng xoá; chỉ thêm trường `optional` nếu spec cho phép); `openapi.yaml` có đủ đường `/auth/*`, `/me/*`, `/admin/*`, `/courses/*`, `/notifications` |
| TC-GATEP2-09 | dòng 8 | `pnpm dev:down && pnpm dev` với DB trống, `SEED_ON_EMPTY_DB=true` (TC-P212-14) | Seed tự chạy ≤ 180 s, `rc=0`; 7 tài khoản đăng nhập được (TC-P212-05) |
| TC-GATEP2-10 | dòng 9 | `go test -race -count=1 ./internal/httpapi/... ./internal/platform/outbox/...; echo rc=$?` (cursor, idempotency gửi đôi → một bản ghi, ETag, 409 version) + QC kiểm trên API P2: `POST /admin/courses` hai lần cùng key → 1 lớp (TC-P208-01); `PUT` sai `version` → 409 (TC-P208-09); cursor `/admin/users`, `/members`, `/notifications` | `rc=0`; hành vi trên API P2 đúng |
| TC-GATEP2-11 | dòng 10 | `go test -race -count=1 ./internal/today/...; echo rc=$?` + TC-P211-04 (xếp hạng), TC-P211-19 (SV không thấy việc staff) | `rc=0`; thứ tự đúng luật; SV không bao giờ thấy việc của người khác |
| TC-GATEP2-12 | dòng 11 | `bash scripts/ui-antipatterns.sh; echo rc=$?` + `pnpm -C frontend exec playwright test today.spec.ts` + `pnpm -C frontend lint` | `rc=0`; 19 dòng `✓` (nền sprint 3); `ui-allow:` ≤ 10 |
| TC-GATEP2-13 | dòng 12 | `curl -s -H "Authorization: Bearer $OUTSIDER" $API/courses/$CID/sessions -o /dev/null -w "%{http_code}"` với `OUTSIDER` = SV không ghi danh, SV `PENDING`, SV `REMOVED`, TA lớp khác, **Admin** | `403` mọi trường hợp |
| TC-GATEP2-14 | cả bộ | `cd backend-go && go vet ./... && golangci-lint run && sqlc diff && go test -race -count=1 ./... && go test -tags integration -count=1 ./internal/integration/...; echo rc=$?` | `rc=0` (toàn bộ backend, gồm cổng PG + LLM không vỡ) |
| TC-GATEP2-15 | PG/LLM không vỡ | `bash docs/sprints/2/qc/scripts/gate-pg.sh` (mục A không phá volume) hoặc `tc-GATE-PG` + `TC-GATEP1-*` không đụng volume; `goose status` | Cổng PG + P1 còn xanh (nguyên tắc 7, 8); phiên bản goose = `4`; `00001`, `00002` không đổi (hash) |
| TC-GATEP2-16 | Frontend không vỡ | `pnpm -C frontend lint && gbuild && $PW --grep-invert @real` + `audit.mjs` bản đăng nhập thật (TC-P212-21) + `sweep.mjs` + `proto-curl.sh all` | `rc=0`; FAIL 0; PASS ≥ nền sprint 3; `FORBIDDEN`=0 |
| TC-GATEP2-17 | CI | `gh run list --workflow ci.yml --branch sprint/4-p2 --limit 1 --json headSha,conclusion`; `headSha` = `git rev-parse origin/sprint/4-p2` | `success` ở HEAD; job Go + Frontend |

## Bảng B — "Bạn tự kiểm" (P2.md)
| TC-id | P2.md | Bước | Kết quả mong đợi |
| --- | --- | --- | --- |
| TC-GATEP2-20 | Tự kiểm 1 (**mạo danh**) | Đăng ký tài khoản mới bằng MSSV của SV B nhưng email khác → xác minh → đăng nhập → duyệt mọi API / màn lớp 1; GV mở hàng chờ và "Hôm nay" (TC-P203-07, P209-30, P210-14) | **Không** thấy bất kỳ dữ liệu nào của B; GV thấy cảnh báo "Email chưa khớp MSSV" |
| TC-GATEP2-21 | Tự kiểm 2 | Quên mật khẩu ở máy A khi đang đăng nhập ở máy B (2 `context`) (TC-P204-14) | Máy B bị đăng xuất, thấy "Bạn đã bị đăng xuất vì mật khẩu của tài khoản vừa được đổi." |
| TC-GATEP2-22 | Tự kiểm 3 | DevTools → Application: `localStorage`, `sessionStorage`, cookie, IndexedDB (TC-P202-34) | Không token / JWT / refresh; `ep_rt` HttpOnly |
| TC-GATEP2-23 | Tự kiểm 4 | **Đọc từng dòng diff `internal/auth`**: `git diff origin/main -- backend-go/internal/auth` (QC đọc; script `scripts/diff-review.sh` quét mẫu nguy hiểm) | Không: so sánh khoá bí mật không hằng thời gian (`==` trên hash / token), `math/rand` cho token / mã, `log`/`slog` in token / mật khẩu / email, `InsecureSkipVerify`, SQL ghép chuỗi, `alg` JWT không cố định HS256, cookie thiếu `HttpOnly`/`Secure`/`SameSite`, chuyển hướng `next` không kiểm, `TODO`/`FIXME` ở nhánh bảo mật; ghi từng phát hiện kèm `file:dòng` |
| TC-GATEP2-24 | Tự kiểm 5 | Đi trọn tay trên DB trống (seed tắt): Admin mở lớp → gán GV → GV thấy chuông + mã → SV D trên điện thoại (375 px) `/join/MÃ` → xem trước → tham gia → GV thấy thành viên mới (TC-P212-24, P209-51) | Mọi bước đúng; ảnh |
| TC-GATEP2-25 | Tự kiểm 6 | Tạo lại mã rồi thử mã cũ (TC-P209-18, P209-07) | Bị từ chối bằng câu chung dễ hiểu, **không lộ lý do cụ thể** (không "mã đã bị đổi / tắt / hết hạn") |
| TC-GATEP2-26 | Tự kiểm 7 | Đăng nhập GV seed: bộ chọn lớp 2 lớp + "Tất cả lớp của tôi"; "Hôm nay" ghi rõ việc của lớp nào; lớp 2 có "Thiết lập lớp mới" | Đúng (TC-P207-30, P211-35) |
| TC-GATEP2-27 | Tự kiểm 8 | Đăng nhập SV A (2 lớp): đổi lớp thì chat, threads, lịch đổi theo (màn mock theo lớp mock tương ứng) | Đổi theo lớp; không lẫn dữ liệu (TC-P207-35) |
| TC-GATEP2-28 | Tự kiểm 9 | Đăng nhập 6/7 tài khoản mẫu, mỗi vai đúng điều hướng (TC-P212-05/06) | SV 7, TA 12, GV 15, Admin 6, D chỉ `Hôm nay` |
| TC-GATEP2-29 | Tự kiểm 10 | Import roster 2 dòng lỗi → báo đúng dòng (TC-P210-06) | `[7,19]` |
| TC-GATEP2-30 | Tự kiểm 11 | Mở `/` bằng SV C và GV: trong ≈ 3 s biết việc kế tiếp? đúng **một** hành động chính? (TC-P211-32/33) | Có; đúng 1 hành động chính; ghi thời gian |
| TC-GATEP2-31 | Tự kiểm 12 | Bấm đúp thật nhanh một nút tạo (mở lớp / mời / tạo) (TC-P208-31, P206-33) | Chỉ **một** bản ghi |

## Bảng C — Tấn công tổng hợp (QC, yêu cầu của PM; chạy trên **stack cuối**, DB đã seed)
| TC-id | Tấn công | Bước / lệnh | Kết quả mong đợi |
| --- | --- | --- | --- |
| TC-GATEP2-40 | **Mạo danh MSSV** | TC-P203-07/08, P209-30…33, P210-14: đủ 3 đường (đăng ký + MSSV; roster nạp sau; vào bằng mã) | 0 dữ liệu của B lộ; `enrollments` không nối theo MSSV; `student_code` tự khai không nằm trong `WHERE` mở dữ liệu (grep TC-P203-09) |
| TC-GATEP2-41 | **Dùng lại refresh cũ** | TC-P202-11/12/13/14 | Cả phiên bị thu hồi (`REFRESH_REUSE`), access cũ `401` ≤ 1 s, `audit_log` có dòng |
| TC-GATEP2-42 | **Đoán mã** (tham gia, xác minh, đặt lại, mời) | TC-P209-11…17, P203-18, P204-12, P206-08, P205-19 (nhiều IP / nhiều tài khoản) | 0 trúng; giới hạn kích hoạt; ghi xác suất; mã thử không ở log |
| TC-GATEP2-43 | **Link dùng hai lần / đua** | TC-P203-13/14, P204-05/06, P206-06/07 (xác minh, đặt lại, mời: 50 yêu cầu song song) | Đúng 1 thành công mỗi loại; còn lại `410 LINK_INVALID` `reason="used"` |
| TC-GATEP2-44 | **Token băm trong DB** | TC-P201-09/10/11, P202-08, P204-09, P206-09: `pg_dump` + log + `mail_outbox.payload` + `audit_log` + `jobs` | **0** bản rõ của mọi token (verify / reset / invite / refresh); `token_hash`/`refresh_hash` đều 64 hex; mật khẩu chỉ bcrypt |
| TC-GATEP2-45 | **Không token ở localStorage / biến toàn cục / URL** | TC-P202-34/35/36, P203-32, P204-27/28, P206-36/37; `pbuild` bundle không chứa `ep_demo_*`, `DEV_AUTH`, cổng dán token | Sạch ở mọi nơi; token xoá khỏi URL; `Referrer-Policy: no-referrer` |
| TC-GATEP2-46 | **IDOR `/me/sessions` và thông báo** | TC-P204-23/24, P208-23: SV A xoá / thấy phiên và thông báo của B, Admin xoá phiên SV | `404` đồng nhất; dữ liệu B nguyên vẹn; Admin cũng `404` qua `/me/*` |
| TC-GATEP2-47 | **IDOR giữa lớp** | TC-P207-08/15, P209-37/38, P211-20/21: SV lớp 1 gọi **mọi** route lớp 2 (mọi phương thức), `uid` lớp khác | Không 2xx; DB không đổi; không dữ liệu lớp 2 |
| TC-GATEP2-48 | **Leo thang vai / token giả** | TC-P202-29, P203-02, P206-02, P206-22/23, P208-27: `alg=none`, sửa `role`, secret sai, `role` trong thân, `X-Role`, GV gọi `assign`, SV gọi `/admin/*` | Mọi ca `401`/`403`/`422`; 0 thay đổi DB; không tạo được TEACHER/TA/ADMIN bằng đường công khai |
| TC-GATEP2-49 | **CSRF / open redirect** | TC-P202-20…23, P202-41 | `403` origin lạ; `next` độc hại → `/` |
| TC-GATEP2-50 | **Liệt kê tài khoản** | TC-P202-04/05, P203-04, P204-01, P205-10, P206-16, P209-07/08 | Phản hồi / thời gian không phân biệt có–không tài khoản, có–không mã |
| TC-GATEP2-51 | **Dò mật khẩu / khoá** | TC-P205-01…19, P205-35 (`X-Forwarded-For` giả) | Chờ tăng dần, khoá 15 phút, IP chặn; header giả vô tác dụng; Admin cũng bị |
| TC-GATEP2-52 | **Tệp roster độc hại** | TC-P210-03/04/36 | Từ chối an toàn; không sập; công thức lưu như chữ |
| TC-GATEP2-53 | **PII / bí mật ở log và repo** | `docker compose logs gateway worker caddy postgres redis \| grep -ciE '<mật khẩu\|token\|email thử\|MSSV thử\|mã tham gia>'`; `git grep` secret; `.env*` không commit | `0` (trừ đường dẫn `/join/<mã>` ở log Caddy: ghi rủi ro); không secret trong repo |

## Dọn dẹp
| TC-id | Bước | Kết quả mong đợi |
| --- | --- | --- |
| TC-GATEP2-99 | `$C down -v`; xoá container, Mailpit, Chrome profile, SMTP giả, máy chủ giả; `docker ps`; `git status` | Không tiến trình / container QC sót; cây git chỉ có artefact QC; fixture nhị phân lớn đã xoá |

## Câu hỏi cho BA / PM
- **Q-QC-GATEP2-1** — "Đọc từng dòng diff `internal/auth`" (P2.md) là việc của chủ dự án; QC chạy `diff-review.sh` quét mẫu nguy hiểm (TC-GATEP2-23) và **báo phát hiện**, không thay thế việc đọc của chủ dự án. Đúng? — *chờ xác nhận*.
  - **Trả lời (BA, 2026-10-03):** Đúng: việc đọc từng dòng diff `internal/auth` là của chủ dự án. QC chạy `diff-review.sh` quét mẫu nguy hiểm và **báo phát hiện**, không thay việc đọc.
- **Q-QC-GATEP2-2** — Cổng P2 có 12 story + seed → thời gian chạy cổng dài (đo giới hạn 15 phút, 5 phút nền, 10 phút khoá). QC sẽ chạy các TC chờ ở nền song song; nếu PM muốn rút gọn, nêu rõ TC nào chỉ chạy bản rút gọn (TTL chỉnh tay). — *chờ trả lời*.
  - **Trả lời (BA, 2026-10-03):** Chấp nhận chạy nền song song. Rút gọn được phép (đã ghi ở `FEAT-account-security/SRS.md` 8.1, ghi chú cho QC): dựng **stack riêng** với TTL / thời gian khoá rút gọn và **một lần đo thật** cho mỗi mốc quan trọng (đo thật 15 phút khoá và mốc 10 phút đoán mã ít nhất một lần cho cổng). TC chỉ chạy bản rút gọn: hạn liên kết 24 h / 30 phút / 72 h (làm hết hạn trong DB test), hết hạn phiên 14 / 30 ngày, TTL cache `today`.
- **Q-QC-GATEP2-3** — `OUTSIDER` ở `P2.md` dòng 12 chưa định nghĩa: QC dùng 5 loại (không ghi danh / PENDING / REMOVED / TA lớp khác / Admin). — *chờ xác nhận*.
  - **Trả lời (BA, 2026-10-03):** Xác nhận 5 loại và đã định nghĩa vào spec (v1.1, `FEAT-course-foundation/SRS.md` mục 2): không ghi danh, `PENDING`, `REMOVED`, TA của lớp khác, Admin.

## Lịch sử sửa TC
- 2026-10-03 — viết lần đầu theo P2.md + plan sprint 4 + yêu cầu PM (TC tấn công).

Tổng: 44 TC (bảng A 17, bảng B 12, bảng C 14, dọn dẹp 1).

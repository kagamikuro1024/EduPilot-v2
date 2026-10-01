# SRS FEAT-ci CI cho Go gateway và frontend mới
Phiên bản 2 · 2026-10-01 · Trạng thái: APPROVED (PM, 2026-10-01; không có câu hỏi mở) · v2: AC3/AC4 theo proposals #5

## 1. Mục đích và phạm vi
Mỗi lần push lên bất kỳ nhánh nào và mỗi pull request đều chạy đúng các lệnh kiểm trong CLAUDE.md "Lệnh" cho phần Go và phần frontend; một lỗi bất kỳ làm CI đỏ. Không chạy gì trong `legacy/` (D45), không có job Python (D46), không gọi LLM thật.

## 2. Người dùng và quyền
Không áp dụng — story hạ tầng.

## 3. Luồng chính và nhánh lỗi
Không áp dụng — story hạ tầng. (Nhánh lỗi ở AC3, AC4 của `US.md`.)

## 4. Yêu cầu chức năng
| FR | Hệ thống phải… | AC |
| --- | --- | --- |
| FR-1 | Có `.github/workflows/ci.yml`, chạy khi `push` (mọi nhánh) và `pull_request` | AC1 |
| FR-2 | Job **Go**: `working-directory: backend-go`; cài Go theo `go-version-file: backend-go/go.mod`; chạy lần lượt `go vet ./...`, `golangci-lint run` (bản ghim cụ thể), `go test -race ./...` | AC1, AC3 |
| FR-3 | Job **Frontend**: cài pnpm (bản ghim theo `packageManager` ở `package.json` gốc), Node 24 với cache pnpm; chạy lần lượt `pnpm install --frozen-lockfile`, `pnpm -C frontend lint`, `pnpm -C frontend build`, `bash scripts/ui-antipatterns.sh` | AC1, AC4 |
| FR-4 | Hai job độc lập (không `needs`), không bước nào `continue-on-error`; một bước đỏ → job đỏ → run đỏ | AC3, AC4 |
| FR-5 | Không tham chiếu `secrets.*`, không đặt biến khoá LLM; không bước nào trỏ vào `legacy/` | AC2 |
| FR-6 | `permissions: contents: read` ở mức workflow | AC5 |
| FR-7 | Tên job và tên bước đọc được (`Go`, `Frontend`; `go vet`, `golangci-lint`, `go test -race`, `lint`, `build`, `ui antipatterns`) để `gh run view` đối chiếu được | AC1 |

## 5. Dữ liệu
Không áp dụng — story hạ tầng.

## 6. API
Không có API sản phẩm. Giao diện dùng để kiểm: GitHub CLI — `gh run list --workflow ci.yml --branch <nhánh> --json databaseId,headSha,conclusion`, `gh run view <ID> --json conclusion,jobs`.

## 7. Giao diện
Không áp dụng — story hạ tầng.

## 8. Phi chức năng áp dụng
| Nhóm | Yêu cầu | Nguồn |
| --- | --- | --- |
| Chi phí | Không gọi LLM thật trong CI; test LLM thật (khi có) gắn nhãn riêng và chỉ chạy tay | CLAUDE.md "Lệnh", ARCHITECTURE §10 |
| Bí mật | Không secret trong workflow ở sprint này | CLAUDE.md "Cấm tuyệt đối" |
| Phiên bản | Go theo `go.mod`, Node 24, pnpm ghim; golangci-lint ghim bản | D48 |
| Tái lập | `--frozen-lockfile`; không cài theo bản "latest" trôi | D48 |
| Không phá cái đang chạy | Không sửa hay xoá workflow khác trong `.github/workflows/` | CLAUDE.md nguyên tắc 8 |

Rủi ro: `next/font/google` tải font lúc build cần mạng — runner GitHub có mạng; nếu bước build lỗi vì mạng thì báo PM, không tắt bước.

## 9. Kiểm thử
Không thêm test cố định cho riêng workflow (theo luật kiểm thử: không test dây nối). Bằng chứng là các run thật:
| Kiểm | Cách |
| --- | --- |
| Xanh trên HEAD | AC1 |
| Đỏ đúng job khi test Go đỏ | AC3, nhánh tạm `ci/red-check` |
| Đỏ đúng job khi có phản mẫu UI | AC4, cùng nhánh tạm, commit kế tiếp |
| Dọn nhánh tạm | Chỉ sau khi QC chấm AC3/AC4 (handoff có `<ID>`, `headSha`, nhánh cho từng run); rồi `git ls-remote --heads origin ci/red-check` rỗng |

## 10. Câu hỏi mở và quyết định đã chốt
Không áp dụng — story hạ tầng. `QUESTIONS.md` hiện không có câu nào. Chi tiết kỹ thuật BA tự chốt theo tài liệu: không job Python (D46); OpenAPI lint chưa đưa vào CI (chỉ có `/healthz`, kiểm tay ở FEAT-scaffold AC3; đưa vào cùng contract test ở PG).

## 11. Truy vết
| PRD | FLOWS | Phase | US | FR | Kiểm |
| --- | --- | --- | --- | --- | --- |
| §5 | – | P0 L2 | US-P0-03 AC1 | FR-1, FR-2, FR-3, FR-7 | `gh run view` |
| – | – | P0 L2 | AC2 | FR-5 | grep workflow |
| – | – | P0 L2 | AC3 | FR-2, FR-4 | run đỏ trên `ci/red-check` |
| – | – | P0 L2 | AC4 | FR-3, FR-4 | run đỏ trên `ci/red-check` |
| – | – | P0 L2 | AC5 | FR-6 | grep `permissions` |

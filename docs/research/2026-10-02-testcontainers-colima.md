# testcontainers-go trên colima (macOS): `DOCKER_HOST` và Ryuk

**Câu hỏi.** testcontainers-go v0.44.0 chạy trên colima cần đặt gì (`DOCKER_HOST`, Ryuk), và cách đang làm ở `backend-go/Makefile` (`DOCKER_HOST` colima + `TESTCONTAINERS_RYUK_DISABLED=true`) có đúng không?

**Kết luận.**
- **Cách của Makefile là đúng và cần đủ cả hai biến** cho thiết kế hiện tại (container tên cố định, `Reuse: true`, dùng lại giữa các lần chạy, dọn bằng `make test-clean`).
- Không đặt `DOCKER_HOST` → **hỏng ngay** (`rootless Docker not found`), dù `docker context` hiện tại là `colima`: v0.44.0 **không đọc** docker context, trái với tài liệu colima của chính testcontainers.
- Đặt `DOCKER_HOST` mà để Ryuk bật → **hỏng sau 8 s**: Ryuk mount đường dẫn socket phía macOS vào VM (`mkdir /Users/…/docker.sock: operation not supported`), và để lại một container `Created` mồ côi.
- Muốn bật Ryuk thì thêm `TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock` (PoC: chạy, Ryuk dọn container sau khi tiến trình thoát) — nhưng Ryuk sẽ xoá luôn container dùng chung ở cuối mỗi lần `go test`, phá thiết kế "dùng lại giữa các lần chạy". **Giữ `RYUK_DISABLED=true`.**
- Điểm yếu thật: `go test` gọi thẳng (QC `gt`/`gotest`, IDE, gopls) không qua Makefile nên đỏ vì môi trường. Cách rẻ nhất, không đụng repo: mỗi máy dev/QC tạo `~/.testcontainers.properties` với `docker.host` + `ryuk.disabled=true` (PoC: `go test` trần chạy được).
- Độ chắc chắn: **cao** (đọc mã v0.44.0 + 5 lần chạy thử).

## Phương án

| Tiêu chí | A. Makefile đặt `DOCKER_HOST` + `RYUK_DISABLED` (repo) | B. A + `~/.testcontainers.properties` trên máy | C. Bật Ryuk: `DOCKER_HOST` + `SOCKET_OVERRIDE` | D. `sudo ln -sf …/docker.sock /var/run/docker.sock` |
| --- | --- | --- | --- | --- |
| `make test` | Chạy | Chạy | Chạy | Chạy |
| `go test` trần (QC, IDE) | **Đỏ** nếu chưa `export` | Chạy | Đỏ nếu chưa `export` | Chạy (cần thêm `SOCKET_OVERRIDE` hoặc tắt Ryuk) |
| Hợp thiết kế container dùng chung giữa các lần chạy | Có | Có | **Không** — Ryuk xoá cuối mỗi lần | Tuỳ Ryuk |
| Dọn rác | Tay: `make test-clean` | Tay | Tự động | Tuỳ Ryuk |
| Ảnh hưởng CI (Linux, `/var/run/docker.sock`) | Không (biến chỉ đặt khi trống) | Không (file chỉ ở máy dev) | Không | Không |
| Công sức | 0 | 1 file mỗi máy, ngoài repo | Sửa Makefile + chấp nhận dựng lại container mỗi lần | Cần `sudo`, lặp lại khi colima đổi socket |

## Bằng chứng

- Phiên bản: `testcontainers-go v0.44.0` là mới nhất (`proxy.golang.org/…/@latest` → 2026-08-07); colima trên máy: macOS Virtualization.Framework, aarch64, runtime docker, socket `unix:///Users/kuro/.colima/default/docker.sock`; `docker context ls` → `colima *`; `/var/run/docker.sock` **không tồn tại** trên macOS.
- Mã nguồn v0.44.0, `internal/core/docker_host.go:76–160`: thứ tự tìm Docker host = `tc.host` trong `~/.testcontainers.properties` → `DOCKER_HOST` → context Go → `/var/run/docker.sock` → `docker.host` trong properties → socket rootless. **Không có bước đọc docker context của CLI.** `:105–118, 179–236`: socket mount cho Ryuk = `TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE` nếu có, ngược lại suy từ `DOCKER_HOST` (bỏ `unix://`) → đường dẫn phía macOS, không có trong VM colima.
- `internal/core/bootstrap/bootstrap.go:41–55`: session = PID cha (`go test`) + thời điểm tạo, nên mọi gói trong một lần `go test ./...` chung một session; `docker.go:1424–1462` (`ReuseOrCreateContainer`): container tạo mới vẫn mang nhãn session và được nối với Ryuk khi Ryuk bật → bị dọn khi session kết thúc.
- Tài liệu [Using Colima](https://github.com/testcontainers/testcontainers-go/blob/v0.44.0/docs/system_requirements/using_colima.md) (v0.44.0): "After the context is set _Testcontainers for Go_ will automatically be configured to use Colima" — **PoC (a) bác bỏ** với v0.44.0. Phần workaround của cùng trang (`DOCKER_HOST` + `TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock`) khớp PoC (c).

PoC: `/tmp/research-tc/tc_test.go` (dựng `redis:8` không `Reuse`, nhãn `poc=research-tc`, cố ý không `Terminate`). Kết quả thật (2026-10-02):

```
(a) env -u DOCKER_HOST go test -count=1 -v ./...
    start sau 0s: get provider: rootless Docker not found, failed to create Docker provider
    --- FAIL
(b) DOCKER_HOST=unix://$HOME/.colima/default/docker.sock go test …           (Ryuk bật)
    start sau 7.914s: create container: reaper: new reaper: run container: container start: Error response
    from daemon: error while creating mount source path '/Users/kuro/.colima/default/docker.sock':
    mkdir /Users/kuro/.colima/default/docker.sock: operation not supported
    --- FAIL        (để lại container redis:8 trạng thái Created)
(c) (b) + TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock
    OK sau 734ms: localhost:32987   --- PASS
    ngay sau khi thoát: hopeful_elion Up, reaper_9f55… Up;  sau 15 s: không còn container nào (Ryuk đã dọn)
(d) DOCKER_HOST=… TESTCONTAINERS_RYUK_DISABLED=true go test …                  (= Makefile)
    OK sau 292ms   --- PASS;  sau 15 s: festive_wing vẫn Up (không ai dọn — đúng ý đồ, dọn bằng tay)
(e) env -u DOCKER_HOST, HOME=/tmp/research-tc/home chứa .testcontainers.properties:
      docker.host=unix:///Users/kuro/.colima/default/docker.sock
      ryuk.disabled=true
    OK sau 387ms   --- PASS
```

Cổng ánh xạ trả về `localhost:<port>` và nối được từ macOS (colima tự chuyển tiếp cổng), nên không cần `TESTCONTAINERS_HOST_OVERRIDE`.

## Ảnh hưởng

- `backend-go/Makefile` mục `test`: giữ nguyên (đáp ứng 01-AC13, TC-PG01-68/69).
- Máy dev / QC: tạo `~/.testcontainers.properties` (hai dòng ở PoC e). Dùng khoá **`docker.host`**, không dùng `tc.host` — `tc.host` đứng **trước** `DOCKER_HOST` trong thứ tự tìm nên sẽ đè cấu hình khác. Ghi một dòng vào README phần "chạy test" (việc của dev/PM; ngoài repo nên không có rủi ro CI).
- Khi thẻ image trong `internal/testutil` trỏ sang bản mới (ví dụ `docker pull pgvector/pgvector:pg18`) container dùng chung **không** tự đổi — `Reuse` tìm theo tên. Đo ngày 2026-10-02 sau khi PoC kéo lại thẻ: `pgvector/pgvector:pg18` = `2358fcba…` (dựng 2026-10-01) nhưng `edupilot-test-postgres` vẫn chạy `2ba9ca5f…` (dựng 2026-08-13). Chạy `make -C backend-go test-clean` để lấy image mới; nên ghi điều này cạnh `test-clean` trong README.
- Cạm bẫy colima liên quan: bind mount từ thư mục ngoài `$HOME` (`/tmp`, `/var/folders/…` = `os.TempDir()` trên macOS) **không** tới được VM — Docker tạo thư mục rỗng. Test nào cần đưa file vào container dùng `ContainerRequest.Files` (copy) thay vì bind mount. [SUY LUẬN cho `os.TempDir()`; đã thấy thật với `/tmp` ở PoC PgBouncer.]
- Không đụng `ARCHITECTURE.md` / `DECISIONS.md`.

## Đề xuất cho PM

`Rủi ro testcontainers/colima: giữ Makefile (DOCKER_HOST colima + TESTCONTAINERS_RYUK_DISABLED=true) — PoC: thiếu DOCKER_HOST hỏng dù docker context=colima, bật Ryuk hỏng vì mount socket (docs/research/2026-10-02-testcontainers-colima.md); thêm vào README một dòng cho dev/QC: ~/.testcontainers.properties với docker.host=unix://$HOME/.colima/default/docker.sock và ryuk.disabled=true để go test trần (QC gt, IDE) chạy được; nhắc make test-clean sau khi đổi/kéo image.`

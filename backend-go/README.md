# backend-go — gateway EduPilot v2

Lệnh thường dùng (chạy trong `backend-go/`, hoặc `make -C backend-go <mục tiêu>`):

```bash
make build            # go build ./...
make lint             # go vet + golangci-lint, cả hai bộ build tag (mặc định + testroutes)
make test             # go test -race -count=1 -tags testroutes ./...
make sqlc-check       # sqlc diff
make test-clean       # xoá container test dùng chung (xem dưới)
```

## Chạy test

Test dùng testcontainers và **một bộ container dùng chung cho cả máy** (`edupilot-test-postgres`, `-redis`, `-minio`,
`-pgbouncer`): tạo một lần, dùng lại giữa các lần `go test`, nên không cần dọn sau mỗi lần chạy. Ryuk tắt
(`TESTCONTAINERS_RYUK_DISABLED=true`) vì Ryuk sẽ xoá bộ container này ở cuối mỗi lần chạy.

`make test` tự đặt `DOCKER_HOST` (colima) và tắt Ryuk. **`go test` gọi trần** (IDE, gopls, script QC) thì không qua Makefile
và sẽ đỏ vì testcontainers không đọc docker context của CLI. Cách sửa, một lần cho mỗi máy (colima, macOS):

```bash
cat > ~/.testcontainers.properties <<EOP
docker.host=unix://$HOME/.colima/default/docker.sock
ryuk.disabled=true
EOP
```

Dùng khoá `docker.host`, **không** dùng `tc.host`: `tc.host` được ưu tiên trước `DOCKER_HOST` nên sẽ đè mọi cấu hình khác.
Docker Desktop / Linux có `/var/run/docker.sock` thì chỉ cần `ryuk.disabled=true`.

### Sau khi kéo image mới

Container dùng chung tìm lại theo **tên**, nên khi bạn `docker pull pgvector/pgvector:pg18` (hoặc đổi thẻ image trong
`internal/testutil`) container cũ vẫn chạy bản image cũ. Chạy `make -C backend-go test-clean` trước lần test kế để dựng lại
bằng image mới. Database test cũ hơn 30 phút tự bị xoá.

Bind mount thư mục ngoài `$HOME` (`/tmp`, `os.TempDir()`) không tới được VM colima; test cần đưa tệp vào container thì dùng
`ContainerRequest.Files` (sao chép), không dùng bind mount.

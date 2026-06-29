# Deploy toàn bộ EduPilot lên một Hugging Face Docker Space

## 1. Kiến trúc trên Space

Hugging Face Space chỉ public một cổng. Image ở root `Dockerfile` chạy toàn bộ hệ thống trong một container:

```text
https://<owner>-<space>.hf.space:443
                  |
             Nginx :7860
              /         \
      Next.js :3000    /backend/* -> Spring Boot :8080
                                      |
                                 gRPC :50051
                                      |
                                 Python AI :8000

      PostgreSQL :5432 + pgvector và Redis :6379 chỉ chạy nội bộ
```

Đây là cấu hình phù hợp cho demo. Dữ liệu mặc định nằm trong filesystem tạm của Space và có thể mất khi Space restart/factory reboot.

## 2. Tạo Hugging Face Space

1. Đăng nhập tại <https://huggingface.co>.
2. Mở <https://huggingface.co/new-space>.
3. Chọn Owner và đặt Space name, ví dụ `edupilot`.
4. Chọn **Docker** trong mục Space SDK.
5. Chọn visibility:
   - **Public**: app và source code đều công khai.
   - **Private**: chỉ owner/collaborator truy cập được app.
6. Hardware có thể để **CPU Basic** để thử. Nếu build/start quá chậm hoặc hết RAM, đổi sang **CPU Upgrade** trong Settings.
7. Nhấn **Create Space**.

Không upload `.env.local` và không ghi API key vào Git.

## 3. Tạo các giá trị bí mật

Chạy PowerShell để tạo hai chuỗi ngẫu nhiên:

```powershell
function New-HexSecret {
    $bytes = New-Object byte[] 32
    $rng = [Security.Cryptography.RandomNumberGenerator]::Create()
    $rng.GetBytes($bytes)
    $rng.Dispose()
    return (($bytes | ForEach-Object { $_.ToString("x2") }) -join "")
}

$jwtSecret = New-HexSecret
$internalToken = New-HexSecret
"JWT_SECRET_KEY=$jwtSecret"
"INTERNAL_CALLBACK_TOKEN=$internalToken"
```

Giữ hai kết quả riêng biệt. Mỗi kết quả phải là chuỗi hex 64 ký tự.

## 4. Cấu hình Secrets và Variables

Trong Space, mở **Settings → Variables and secrets**.

### Secrets bắt buộc

| Name | Value |
|---|---|
| `OPENAI_API_KEY` | API key thật, bắt đầu bằng `sk-...` |
| `JWT_SECRET_KEY` | Chuỗi hex 64 ký tự thứ nhất |
| `INTERNAL_CALLBACK_TOKEN` | Chuỗi hex 64 ký tự thứ hai |

### Secrets khuyến nghị/tùy chọn

| Name | Value |
|---|---|
| `ADMIN_TOKEN` | Một token ngẫu nhiên khác; nếu bỏ trống sẽ dùng `INTERNAL_CALLBACK_TOKEN` |
| `ANTHROPIC_API_KEY` | Chỉ cần nếu chuyển sang model Claude |
| `AI_LOG_API_KEY` | Chỉ cần nếu sử dụng server AI logging của dự án |

### Variables

| Name | Value đề xuất |
|---|---|
| `DEFAULT_MODEL` | `gpt-4o-mini` |
| `APP_ENV` | `production` |
| `LOG_LEVEL` | `INFO` |
| `JWT_EXPIRATION` | `86400000` |

Không cần tạo `DATABASE_URL`, `DB_URL`, `REDIS_URL`, `PYTHON_GRPC_URL`, `NEXT_PUBLIC_JAVA_API_URL` hoặc port. Entrypoint tự cấu hình chúng cho mạng nội bộ của container.

## 5. Đẩy source code lên Space

Space là một Git repository. Tại thư mục dự án:

```powershell
git add .
git commit -m "Add Hugging Face full-stack deployment"
git remote add hf https://huggingface.co/spaces/<HF_USERNAME>/<SPACE_NAME>
git push hf main
```

Thay `<HF_USERNAME>` và `<SPACE_NAME>` bằng thông tin thật. Khi Git hỏi credentials:

- Username: Hugging Face username.
- Password: User Access Token có quyền ghi, tạo tại <https://huggingface.co/settings/tokens>.

Không đặt token trực tiếp trong remote URL. Sau mỗi lần push, Space tự build và restart.

Nếu branch local không tên `main`, dùng:

```powershell
git push hf HEAD:main
```

## 6. Theo dõi build

1. Mở trang Space.
2. Chọn **Logs → Build logs**.
3. Build đầu tiên lâu vì phải cài PyTorch, Docling, Java/Gradle và Next.js.
4. Khi trạng thái chuyển thành **Running**, mở app bằng nút **App**.

Các endpoint kiểm tra:

```text
https://<owner>-<space>.hf.space/
https://<owner>-<space>.hf.space/backend/api/v1/health
https://<owner>-<space>.hf.space/ai/health
```

Kết quả mong đợi:

- Frontend trả trang EduPilot.
- Backend trả `{"status":"up"}`.
- AI trả `status: healthy` và `grpc_ready: true`.

## 7. Test image giống Hugging Face ở local

Docker Desktop phải đang chạy:

```powershell
docker compose -f docker-compose.hf.yml up --build
```

Mở <http://localhost:7860>. Giá trị `local-test-key` chỉ giúp service boot; để test câu hỏi AI thật, đặt key trước khi chạy:

```powershell
$env:OPENAI_API_KEY="sk-..."
docker compose -f docker-compose.hf.yml up --build
```

Dừng môi trường test:

```powershell
docker compose -f docker-compose.hf.yml down
```

Thêm `-v` chỉ khi muốn xóa toàn bộ database local của image Space:

```powershell
docker compose -f docker-compose.hf.yml down -v
```

## 8. Dữ liệu và giới hạn production

- PostgreSQL, Redis và file upload nằm dưới `HF_DATA_DIR`, mặc định `/home/user/data`.
- Filesystem mặc định của Space là tạm thời. Restart/factory reboot có thể làm mất tài khoản, tài liệu và lịch sử chat.
- Free Space có thể sleep khi không được sử dụng; request đầu sau khi sleep sẽ có cold start.
- Không nên coi bản all-in-one này là cấu hình production có dữ liệu quan trọng.
- Muốn production bền vững, nên tách PostgreSQL/Redis sang hạ tầng quản lý được và triển khai Java/AI ở môi trường cho phép kết nối database; Hugging Face giới hạn outbound networking ở các cổng HTTP/HTTPS tiêu chuẩn.

## 9. Lỗi thường gặp

### `Thiếu Hugging Face Secret`

Thêm đúng `OPENAI_API_KEY`, `JWT_SECRET_KEY`, `INTERNAL_CALLBACK_TOKEN` trong Settings, sau đó chọn **Restart Space**.

### Java báo JWT key không hợp lệ

`JWT_SECRET_KEY` phải là chuỗi hex đúng 64 ký tự hoặc Base64 của ít nhất 32 byte. Dùng lệnh PowerShell ở bước 3 để tạo lại.

### Frontend chạy nhưng API trả 502

Mở **Runtime logs**, tìm log của `java-backend` hoặc `python-ai`. Kiểm tra OpenAI key và ba endpoint health ở bước 6.

### Build bị timeout/hết bộ nhớ

Chuyển Hardware sang CPU Upgrade, sau đó chọn **Factory reboot** để build lại sạch.

### OpenAI trả 401

Xóa và tạo lại `OPENAI_API_KEY` trong Secrets. Không thêm dấu nháy quanh giá trị key.

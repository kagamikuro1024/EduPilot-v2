# ==========================================
# Giai đoạn 1: Builder
# ==========================================
FROM python:3.11-slim AS builder

WORKDIR /app

# Cài đặt công cụ build hệ thống (gcc, C/C++)
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
       build-essential \
       gcc \
       libpq-dev \
       curl \
    && rm -rf /var/lib/apt/lists/*

# Tạo Python Virtual Environment
RUN python -m venv /opt/venv
# Đưa venv vào biến môi trường PATH để dùng pip trong venv
ENV PATH="/opt/venv/bin:$PATH"

# Copy file requirements trước để tận dụng Docker Cache
COPY requirements.txt .

# Ép cài đặt PyTorch phiên bản CPU-only trước tiên để bỏ bản CUDA mặc định
RUN pip install --no-cache-dir torch torchvision torchaudio --index-url https://download.pytorch.org/whl/cpu

# Cài đặt các thư viện còn lại vào venv
RUN pip install --no-cache-dir -r requirements.txt


# ==========================================
# Giai đoạn 2: Runner (Môi trường chạy thực tế)
# ==========================================
FROM python:3.11-slim

WORKDIR /app

# Cài đặt các thư viện C cốt lõi cần thiết lúc chạy (runtime dependencies)
# KHÔNG cài gcc hay các công cụ biên dịch ở giai đoạn này để giảm dung lượng và tăng bảo mật
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
       curl \
       libxcb1 \
       libx11-6 \
       libxext6 \
       libxrender1 \
       libsm6 \
       libglib2.0-0 \
       libgl1 \
       libpq5 \
    && rm -rf /var/lib/apt/lists/*

# Copy nguyên thư mục Virtual Environment từ giai đoạn builder sang
COPY --from=builder /opt/venv /opt/venv

# Đưa venv vào biến môi trường PATH để ứng dụng mặc định dùng Python trong venv
ENV PATH="/opt/venv/bin:$PATH"

# Copy source code FastAPI vào container
COPY src/ /app/src/
COPY data_pipeline/ /app/data_pipeline/

# Expose port mà FastAPI sẽ chạy
EXPOSE 8000
# gRPC port
EXPOSE 50051

# Sử dụng lệnh python thuần để chạy main.py
CMD ["python", "-m", "src.main"]
# syntax=docker/dockerfile:1

# Hugging Face Docker Space image. It packages the whole application into one
# container because Spaces do not run docker-compose.yml.

# -----------------------------------------------------------------------------
# Frontend build
# -----------------------------------------------------------------------------
FROM node:20-bookworm-slim AS frontend-builder

WORKDIR /build/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend/ ./

# Browser requests stay on the Space origin. Nginx strips /backend before
# forwarding them to Spring Boot.
ARG NEXT_PUBLIC_JAVA_API_URL=/backend
ENV NEXT_PUBLIC_JAVA_API_URL=${NEXT_PUBLIC_JAVA_API_URL}
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build


# -----------------------------------------------------------------------------
# Java build
# -----------------------------------------------------------------------------
FROM eclipse-temurin:21-jdk-jammy AS java-builder

WORKDIR /build
COPY backend-java/aitrogiang/gradlew backend-java/aitrogiang/
COPY backend-java/aitrogiang/gradle backend-java/aitrogiang/gradle
COPY backend-java/aitrogiang/build.gradle backend-java/aitrogiang/settings.gradle backend-java/aitrogiang/
COPY shared-proto shared-proto

WORKDIR /build/backend-java/aitrogiang
RUN chmod +x gradlew
RUN ./gradlew dependencies --no-daemon || true

COPY backend-java/aitrogiang/src src
RUN ./gradlew bootJar -x test --no-daemon


# -----------------------------------------------------------------------------
# Python dependencies
# -----------------------------------------------------------------------------
FROM python:3.11-slim-bookworm AS python-builder

WORKDIR /build
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
       build-essential \
       gcc \
       libpq-dev \
    && rm -rf /var/lib/apt/lists/*

RUN python -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

COPY requirements.txt .
RUN pip install --no-cache-dir torch torchvision torchaudio \
    --index-url https://download.pytorch.org/whl/cpu
RUN pip install --no-cache-dir -r requirements.txt


# Runtime-only binaries copied into the final Debian image.
FROM eclipse-temurin:21-jre-jammy AS java-runtime
FROM redis/redis-stack-server:latest AS redis-runtime


# -----------------------------------------------------------------------------
# All-in-one Hugging Face runtime
# -----------------------------------------------------------------------------
FROM pgvector/pgvector:0.8.2-pg16-bookworm

USER root

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
       ca-certificates \
       curl \
       nginx \
       supervisor \
       tini \
       libxcb1 \
       libx11-6 \
       libxext6 \
       libxrender1 \
       libsm6 \
       libglib2.0-0 \
       libgl1 \
       libssl3 \
       libstdc++6 \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --gid 1000 user \
    && useradd --uid 1000 --gid 1000 --create-home user

# Python interpreter/stdlib and its isolated dependencies.
COPY --from=python-builder /usr/local/ /usr/local/
COPY --from=python-builder /opt/venv /opt/venv

# Java 21, Node.js runtime and Redis Stack (including RediSearch).
COPY --from=java-runtime /opt/java/openjdk /opt/java/openjdk
COPY --from=frontend-builder /usr/local/bin/node /usr/local/bin/node
COPY --from=redis-runtime /opt/redis-stack /opt/redis-stack

WORKDIR /app

# Application runtimes.
COPY --chown=1000:1000 --from=frontend-builder /build/frontend/.next/standalone /app/frontend
COPY --chown=1000:1000 --from=frontend-builder /build/frontend/.next/static /app/frontend/.next/static
COPY --chown=1000:1000 --from=frontend-builder /build/frontend/public /app/frontend/public
COPY --chown=1000:1000 --from=java-builder /build/backend-java/aitrogiang/build/libs/*.jar /app/backend/app.jar
COPY --chown=1000:1000 src /app/src
COPY --chown=1000:1000 data_pipeline /app/data_pipeline
COPY --chown=1000:1000 db /app/db
COPY --chown=1000:1000 huggingface /app/huggingface

RUN mkdir -p /app/data/tmp /home/user/data \
    && chown -R 1000:1000 /app /home/user/data \
    && chmod +x /app/huggingface/entrypoint.sh /app/huggingface/run-redis.sh

ENV HOME=/home/user \
    PATH=/opt/venv/bin:/opt/java/openjdk/bin:/usr/lib/postgresql/16/bin:/usr/local/bin:/usr/bin:/bin \
    JAVA_HOME=/opt/java/openjdk \
    NODE_ENV=production \
    APP_ENV=production \
    APP_HOST=127.0.0.1 \
    GRPC_PORT=50051 \
    NEXT_TELEMETRY_DISABLED=1 \
    HF_DATA_DIR=/home/user/data \
    OTEL_SDK_DISABLED=true

USER 1000:1000

EXPOSE 7860
STOPSIGNAL SIGTERM

HEALTHCHECK --interval=30s --timeout=10s --start-period=120s --retries=3 \
    CMD curl -fsS http://127.0.0.1:7860/ > /dev/null || exit 1

ENTRYPOINT ["/usr/bin/tini", "--", "/app/huggingface/entrypoint.sh"]

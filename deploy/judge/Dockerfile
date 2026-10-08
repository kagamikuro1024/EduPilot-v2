# Sandbox chấm C / C++ (US-PE-02, D58): go-judge v1.13.0 + gcc / g++ 14.2.0 của Debian trixie.
# go-judge bind-mount /usr, /lib… của chính image này (chỉ đọc) vào sandbox từng lệnh, nên trình biên dịch nằm trong image là đủ.
# Container chạy `privileged` + `cgroup: host` (xem docker-compose.local.yml); mạng chỉ `judge_net` (internal), không `ports`, không volume.
FROM debian:trixie-slim
RUN apt-get update \
 && apt-get install -y --no-install-recommends g++ gcc libc6-dev bash \
 && rm -rf /var/lib/apt/lists/*
COPY --from=criyle/go-judge:v1.13.0 /opt/go-judge /opt/mount.yaml /opt/
COPY start.sh /opt/start.sh
RUN chmod 0755 /opt/start.sh
WORKDIR /opt
EXPOSE 5050
ENTRYPOINT ["/opt/start.sh"]

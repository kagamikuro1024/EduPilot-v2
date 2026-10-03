# Sandbox chấm C/C++ cho PoC: go-judge v1.13.0 + gcc/g++ của Debian trixie.
# go-judge bind-mount /usr, /lib… của chính image này vào sandbox, nên trình biên dịch nằm trong image là đủ.
FROM debian:trixie-slim
RUN apt-get update && apt-get install -y --no-install-recommends g++ gcc libc6-dev && rm -rf /var/lib/apt/lists/*
COPY --from=criyle/go-judge:v1.13.0 /opt/go-judge /opt/mount.yaml /opt/
WORKDIR /opt
EXPOSE 5050
ENTRYPOINT ["/opt/go-judge"]

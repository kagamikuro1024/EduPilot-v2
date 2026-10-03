# SRS 8.5 — bốn target: gateway, worker (dựng không tag) và gateway-test, worker-test (dựng có build tag).
# Tầng chạy distroless static nonroot: không shell, không gói hệ thống; mỗi image < 40.000.000 byte.
FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags "-s -w" -o /out/gateway ./cmd/gateway \
 && go build -trimpath -ldflags "-s -w" -o /out/worker ./cmd/worker

# Bản có route thử (US-PG-07 AC20): chỉ hai dòng `go build` dưới đây mang tag.
FROM build AS build-test
RUN go build -tags testroutes -trimpath -ldflags "-s -w" -o /out/gateway-test ./cmd/gateway \
 && go build -tags testroutes -trimpath -ldflags "-s -w" -o /out/worker-test ./cmd/worker

FROM gcr.io/distroless/static-debian12:nonroot AS gateway
COPY --from=build /out/gateway /gateway
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/gateway"]

FROM gcr.io/distroless/static-debian12:nonroot AS worker
COPY --from=build /out/worker /worker
EXPOSE 8081
USER nonroot:nonroot
ENTRYPOINT ["/worker"]

FROM gcr.io/distroless/static-debian12:nonroot AS gateway-test
COPY --from=build-test /out/gateway-test /gateway
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/gateway"]

FROM gcr.io/distroless/static-debian12:nonroot AS worker-test
COPY --from=build-test /out/worker-test /worker
EXPOSE 8081
USER nonroot:nonroot
ENTRYPOINT ["/worker"]

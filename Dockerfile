FROM golang:1.26-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY helper.go main.go model.go rest.go service.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/grafana-telegram-proxy

FROM scratch
COPY --from=builder /out/grafana-telegram-proxy /grafana-telegram-proxy
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/grafana-telegram-proxy"]
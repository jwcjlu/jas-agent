FROM golang:1.21-alpine AS builder

WORKDIR /workspace

# 预先复制依赖文件以利用缓存
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o jas-agent ./cmd/server

FROM alpine:3.19

RUN adduser -D -g '' jas

WORKDIR /app

COPY --from=builder /workspace/jas-agent /app/jas-agent
COPY configs /app/configs

USER jas

EXPOSE 8080
EXPOSE 9000

ENTRYPOINT ["/app/jas-agent"]
CMD ["-conf", "/app/configs/config.yaml"]


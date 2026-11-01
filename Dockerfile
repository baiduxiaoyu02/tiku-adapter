FROM golang:1.21 AS builder2
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOPROXY=https://goproxy.cn,direct

WORKDIR /build
ADD go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags "-s -w -extldflags '-static'" -o tiku-adapter ./cmd/adapter-service

FROM alpine:3.19

RUN apk update \
    && apk upgrade \
    && apk add --no-cache ca-certificates tzdata wget \
    && update-ca-certificates 2>/dev/null || true

COPY --from=builder2 /build/tiku-adapter /app/tiku-adapter

# 创建数据目录用于持久化数据库文件
RUN mkdir -p /app/data

# 创建非 root 用户并切换
RUN addgroup -S appgroup && adduser -S appuser -G appgroup -u 1001
RUN chown -R appuser:appgroup /app
USER 1001

EXPOSE 8060
WORKDIR /app

# 声明数据卷用于持久化
VOLUME ["/app/data"]

# 健康检查：每30秒检测一次，超时5秒，启动等待10秒
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8060/adapter-service/plat || exit 1

ENTRYPOINT ["/app/tiku-adapter"]

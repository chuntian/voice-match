# ---------- builder ----------
FROM golang:1.22-alpine AS builder

WORKDIR /src

# 先下载依赖，利用层缓存
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 静态编译，纯 Go 二进制
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/signal ./server/cmd/signal

# ---------- runtime ----------
FROM alpine:3.19

# 时区与根证书
RUN apk add --no-cache ca-certificates tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

ENV TZ=Asia/Shanghai

WORKDIR /app
COPY --from=builder /app/signal /app/signal

EXPOSE 8080

ENTRYPOINT ["/app/signal"]

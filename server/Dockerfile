FROM golang:1.25-alpine AS builder

WORKDIR /app

# 安装 build 依赖
RUN apk add --no-cache git

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源码
COPY . .

# 构建
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./server/cmd/server

# 运行镜像
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# 复制构建产物
COPY --from=builder /app/server .
COPY .env.example .env

EXPOSE 8080

CMD ["./server"]
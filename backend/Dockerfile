# 阶段 1：编译前端
FROM node:20-alpine AS frontend-builder

WORKDIR /frontend

COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ .
RUN npm run build:prod

# 阶段 2：编译后端
FROM golang:1.24-alpine AS backend-builder

WORKDIR /build

# 安装构建依赖
RUN apk add --no-cache git ca-certificates tzdata

# 设置国内 Go 代理，解决 proxy.golang.org 网络不稳定问题
ENV GOPROXY=https://goproxy.cn,https://goproxy.io,direct
ENV GONOSUMCHECK=*
ENV GOFLAGS=-mod=mod

# 先只复制依赖声明文件，利用 Docker 层缓存：
# 只要 go.mod 未变，下载步骤不会重新执行
COPY backend/go.mod ./
RUN go mod download

# 复制全部源码后执行 tidy（补全 go.sum）并编译
COPY backend/ .
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o paygate-server ./cmd/server

# 阶段 3：最小运行镜像
FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

COPY --from=backend-builder /build/paygate-server .
COPY --from=frontend-builder /frontend/dist ./frontend/dist

RUN adduser -D -g '' paygate
USER paygate

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=5s --start-period=20s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

CMD ["./paygate-server"]

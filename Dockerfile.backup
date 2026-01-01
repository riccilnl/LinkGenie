# 多阶段构建 - Go 版本最小化镜像
FROM golang:1.23-alpine AS builder

# 构建参数 - 用于强制更新镜像 manifest
ARG BUILD_DATE
ARG BUILD_VERSION=latest

# 配置alpine镜像源(使用阿里云镜像) - 必须在安装依赖之前
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 安装构建依赖
RUN apk add --no-cache git gcc musl-dev

WORKDIR /build

# 配置Go模块代理(使用阿里云) - 加速依赖下载
ENV GOPROXY=https://mirrors.aliyun.com/goproxy/,https://goproxy.cn,direct

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖 - 添加详细日志
RUN echo "Starting go mod download..." && \
    go mod download -x && \
    echo "Go mod download completed successfully"

# 复制源代码
COPY *.go ./
COPY api/ ./api/
COPY db/ ./db/
COPY models/ ./models/
COPY services/ ./services/
COPY utils/ ./utils/
COPY config/ ./config/
COPY mcp/ ./mcp/

# 构建静态二进制文件
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o bookmarks .

# 最终镜像 - 使用 alpine 最小化
FROM alpine:latest

# 配置alpine镜像源(使用阿里云镜像)
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 安装 CA 证书（用于 HTTPS 请求）
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# 添加构建元数据（确保每次构建都有唯一标识）
LABEL build_date="${BUILD_DATE}"
LABEL version="${BUILD_VERSION}"

# 从构建阶段复制二进制文件
COPY --from=builder /build/bookmarks ./ai-bookmark-service

# 复制前端文件 (重构后的模块化结构)
COPY index.html manifest.json sw.js icon.svg ./
COPY css/ ./css/
COPY js/ ./js/

# 复制entrypoint
COPY docker-entrypoint.sh ./
RUN chmod +x docker-entrypoint.sh

# 创建数据目录
RUN mkdir -p /app/data

# 暴露端口
EXPOSE 8000

# 设置环境变量
ENV PORT=8000 \
    API_TOKEN=your-secret-token-change-me \
    AI_API_KEY="" \
    AI_API_BASE=https://generativelanguage.googleapis.com/v1beta

# 使用entrypoint脚本
ENTRYPOINT ["./docker-entrypoint.sh"]


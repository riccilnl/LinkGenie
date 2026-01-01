#!/bin/bash
# 部署脚本 - LinkGenie
# 运行方式: bash deploy.sh 或 ./deploy.sh

set -e  # 遇到错误立即退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}================================${NC}"
echo -e "${CYAN}LinkGenie 部署脚本${NC}"
echo -e "${CYAN}================================${NC}"
echo ""

# 检查前端是否需要构建
check_frontend_build() {
    echo -e "${YELLOW}📦 检查前端构建状态...${NC}"
    
    # 检查是否有 package.json（如果有则需要 npm build）
    if [ -f "package.json" ]; then
        echo -e "${YELLOW}检测到 package.json，检查是否需要构建...${NC}"
        
        # 检查 dist 或 build 目录是否存在且不为空
        if [ ! -d "dist" ] && [ ! -d "build" ]; then
            echo -e "${YELLOW}未找到构建目录，开始前端构建...${NC}"
            npm install
            npm run build
            echo -e "${GREEN}✓ 前端构建完成${NC}"
        else
            echo -e "${GREEN}✓ 前端已构建${NC}"
        fi
    else
        echo -e "${GREEN}✓ 无需前端构建（纯静态文件）${NC}"
    fi
    echo ""
}

# 部署函数
deploy_local() {
    # 1. 检查并构建前端
    check_frontend_build
    
    # 2. 开始构建 Docker 镜像
    echo -e "${YELLOW}🐳 开始构建 Docker 镜像...${NC}"
    
    # 获取当前时间作为构建标识
    BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    echo -e "${CYAN}构建时间: $BUILD_DATE${NC}"
    
    # 强制无缓存构建新镜像 (指定目标平台为 linux/amd64)
    echo -e "${CYAN}目标平台: linux/amd64${NC}"
    if ! docker build --platform linux/amd64 --no-cache --build-arg BUILD_DATE="$BUILD_DATE" -t ai-bookmark-service:latest .; then
        echo -e "${RED}❌ 构建失败!${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✓ Docker 镜像构建成功!${NC}"
    echo ""
    
    # 3. 标记镜像
    echo -e "${YELLOW}🏷️  标记镜像...${NC}"
    docker tag ai-bookmark-service:latest 10.15.1.3:1000/ai-bookmark-service:latest
    echo -e "${GREEN}✓ 镜像已标记${NC}"
    echo ""
    
    # 4. 推送到局域网仓库
    echo -e "${YELLOW}📤 推送镜像到局域网仓库 (10.15.1.3:1000)...${NC}"
    if ! docker push 10.15.1.3:1000/ai-bookmark-service:latest; then
        echo -e "${RED}❌ 推送失败!${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✓ 镜像推送成功!${NC}"
    echo ""
    
    # 5. 触发远程部署
    echo -e "${YELLOW}🚀 触发远程服务器部署...${NC}"
    ssh Ricci@10.15.1.3 '/vol1/1000/Docker/ai-bookmark-service/deploy.sh'
    
    echo ""
    echo -e "${CYAN}================================${NC}"
    echo -e "${GREEN}✅ 部署完成!${NC}"
    echo -e "${CYAN}================================${NC}"
}

# 执行部署
deploy_local

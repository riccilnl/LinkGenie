#!/bin/bash
# 同步私有仓库到公开仓库 (LinkGenie)
# 自动清理所有不适合公开的文件

set -e

echo "🚀 LinkGenie 公开仓库同步工具"
echo "================================"
echo ""

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 1. 确保在 main 分支且工作区干净
echo -e "${YELLOW}📋 检查当前分支...${NC}"
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "main" ]; then
    echo -e "${RED}❌ 错误: 请先切换到 main 分支${NC}"
    exit 1
fi

if ! git diff-index --quiet HEAD --; then
    echo -e "${RED}❌ 错误: 工作区有未提交的更改${NC}"
    echo "请先提交或暂存更改"
    exit 1
fi

echo -e "${GREEN}✓ 当前在 main 分支且工作区干净${NC}"
echo ""

# 2. 创建临时分支
echo -e "${YELLOW}🔀 创建临时公开分支...${NC}"
TEMP_BRANCH="temp-public-$(date +%s)"
git checkout -b "$TEMP_BRANCH"
echo -e "${GREEN}✓ 已创建临时分支: $TEMP_BRANCH${NC}"
echo ""

# 3. 删除不公开的文件和目录
echo -e "${YELLOW}🧹 清理不公开的内容...${NC}"

# 删除目录
rm -rf Docs/ docs/ Test/ tests/ scripts/
echo "  ✓ 已删除: Docs/, docs/, Test/, tests/, scripts/"

# 删除 Chrome 扩展调试文件
rm -f chrome-extension/debug-advanced.js
rm -f chrome-extension/debug-theme.js
rm -f chrome-extension/diagnose-border.js
rm -f chrome-extension/test-border-fix.js
rm -f chrome-extension/find-rounded.js
echo "  ✓ 已删除: Chrome 扩展调试文件 (5个)"

# 删除测试文件
rm -f utils/validator_test.go
rm -f mcp/mcp_server_test.go
echo "  ✓ 已删除: Go 测试文件 (2个)"

# 删除部署和配置文件
rm -f deploy.sh
rm -f mcp/claude_desktop_config.json
rm -f Dockerfile.fast Dockerfile.backup Dockerfile.optimized
echo "  ✓ 已删除: 部署脚本和冗余 Dockerfile (4个)"

# 删除运行时文件 (以防万一)
rm -f bookmarks.db bookmarks.db-shm bookmarks.db-wal bookmarks.exe .env
echo "  ✓ 已删除: 运行时文件 (如果存在)"

echo -e "${GREEN}✓ 清理完成${NC}"
echo ""

# 4. 更新 .gitignore (公开版)
echo -e "${YELLOW}📝 更新 .gitignore...${NC}"
if [ -f ".gitignore.public" ]; then
    cp .gitignore.public .gitignore
    echo -e "${GREEN}✓ 已应用公开版 .gitignore${NC}"
else
    echo -e "${RED}❌ 错误: .gitignore.public 文件不存在${NC}"
    echo "请先在私有仓库根目录创建 .gitignore.public"
    git checkout main
    git branch -D "$TEMP_BRANCH"
    exit 1
fi
echo ""

# 5. 更新 go.mod 中的模块名
echo -e "${YELLOW}📦 更新 Go 模块名...${NC}"
sed -i '' 's|module ai-bookmark-service|module github.com/riccilnl/LinkGenie|g' go.mod
echo -e "${GREEN}✓ 已更新 go.mod${NC}"
echo ""

# 6. 提交清理后的版本
echo -e "${YELLOW}💾 提交公开版本...${NC}"
git add -A
COMMIT_MSG="Public release: LinkGenie v1.0.0 ($(date +%Y-%m-%d))"
git commit -m "$COMMIT_MSG" || echo "没有更改需要提交"
echo -e "${GREEN}✓ 已提交: $COMMIT_MSG${NC}"
echo ""

# 7. 推送到公开仓库
echo -e "${YELLOW}📤 推送到公开仓库...${NC}"
echo "目标: git@github.com:riccilnl/LinkGenie.git"
read -p "确认推送? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    git push -f public "$TEMP_BRANCH:main"
    echo -e "${GREEN}✓ 推送成功!${NC}"
else
    echo -e "${YELLOW}⚠️  取消推送${NC}"
fi
echo ""

# 8. 回到 main 分支并删除临时分支
echo -e "${YELLOW}🔙 清理临时分支...${NC}"
git checkout main
git branch -D "$TEMP_BRANCH"
echo -e "${GREEN}✓ 已删除临时分支: $TEMP_BRANCH${NC}"
echo ""

echo "================================"
echo -e "${GREEN}✅ 同步完成!${NC}"
echo ""
echo "私有仓库: git@github.com:riccilnl/ai-bookmark-service.git"
echo "公开仓库: git@github.com:riccilnl/LinkGenie.git"
echo ""
echo "查看公开仓库: https://github.com/riccilnl/LinkGenie"

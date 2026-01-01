# AI Bookmark Service - 项目文档

## 📚 文档索引

### 质量保证
- [QA 审计报告](./QA_Audit_Report.md) - 全方位质量审计结果

### 架构设计
- [系统架构](./Architecture.md) - 系统整体架构设计
- [数据库设计](./Database_Schema.md) - 数据库表结构和关系

### 开发指南
- [开发环境搭建](./Development_Setup.md) - 本地开发环境配置
- [API 文档](./API_Reference.md) - REST API 接口说明
- [测试指南](./Testing_Guide.md) - 测试策略和执行方法

### 部署运维
- [部署指南](./Deployment.md) - 生产环境部署流程
- [故障排查](./Troubleshooting.md) - 常见问题和解决方案

---

## 🚀 快速开始

### 本地开发
```bash
# 1. 克隆仓库
git clone git@github.com:riccilnl/ai-bookmark-service.git
cd ai-bookmark-service

# 2. 配置环境变量
cp .env.example .env
# 编辑 .env 填入你的配置

# 3. 运行服务
go run main.go

# 4. 运行测试
go test -v ./Test/...
```

### Docker 部署
```bash
# 使用 Docker Compose
docker-compose up -d

# 查看日志
docker-compose logs -f
```

---

## 📊 项目状态

- **版本**: v1.0.0
- **状态**: 🟢 生产就绪
- **最后审计**: 2026-01-01
- **测试覆盖率**: 85%+
- **并发安全**: ✅ 通过 race detector

---

## 🔒 安全性

- ✅ SQL 注入防护
- ✅ 鉴权中间件
- ✅ 限流保护
- ✅ Panic 恢复机制
- ⚠️ CSRF 防护（待实现）

---

## 🤝 贡献指南

本项目采用双仓库策略：
- **私有仓库**: 包含完整代码、文档和测试
- **公开仓库**: 仅包含生产代码

详见 [贡献指南](./Contributing.md)

---

**维护者**: riccilnl  
**最后更新**: 2026-01-01

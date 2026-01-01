# 测试指南

## 🎯 测试策略

本项目遵循严格的测试原则：

### 开发红线
1. **严禁纸面测试**: 所有模块必须包含真实的集成测试（`httptest.NewServer`），拒绝纯逻辑 Mock
2. **幸存者验证**: 所有错误处理逻辑后，必须紧跟正常请求验证，确保系统不挂起、不锁定
3. **拒绝代码阑尾**: 定义的导出函数若无引用，必须清理
4. **深度质询**: 交付代码前，自查是否存在无限递归、死锁风险
5. **诚实原则**: 不保证 100% 成功，主动指出极端并发下的脆弱点

---

## 📁 测试文件结构

```
Test/
├── qa_audit_full_test.go      # 全方位 QA 审计测试
├── final_logic_test.go         # 核心逻辑回归测试
├── logic_validation_test.go    # 业务逻辑验证
├── resilience_test.go          # 弹性和降级测试
└── qa_fix_repro_test.go        # Bug 修复验证测试
```

---

## 🧪 测试矩阵

### 1. 基础逻辑测试
**文件**: `final_logic_test.go`, `logic_validation_test.go`

#### 测试项目
- ✅ 标签合并逻辑（低阈值验证）
- ✅ 事务原子性（回滚测试）
- ✅ 标签晋升机制
- ✅ 工作流执行逻辑
- ✅ 相似度算法

#### 运行方法
```bash
go test -v ./Test/final_logic_test.go
go test -v ./Test/logic_validation_test.go
```

---

### 2. 并发与内存安全测试
**文件**: `qa_audit_full_test.go`

#### 测试项目
- ✅ 50 并发书签创建
- ✅ 数据竞争检测（race detector）
- ✅ SQLite WAL 模式并发验证
- ✅ 限流器并发安全

#### 运行方法
```bash
# 启用 race detector
go test -race -v ./Test/qa_audit_full_test.go

# 压力测试
go test -v -run TestConcurrentBookmarkCreation_QA ./Test/
```

---

### 3. 破坏性与边界测试
**文件**: `qa_audit_full_test.go::TestDestructive_QA`

#### 测试项目
- ✅ 畸形 JSON 注入
- ✅ SQL 注入尝试
- ✅ 鉴权绕过测试
- ✅ 超大请求体攻击（5MB）

#### 运行方法
```bash
go test -v -run TestDestructive_QA ./Test/
```

---

### 4. 高级业务集成测试
**文件**: `qa_audit_full_test.go`, `resilience_test.go`

#### 测试项目
- ✅ Panic 恢复与幸存者验证
- ✅ AI 服务降级
- ✅ 网页采集器超时处理
- ✅ 文件夹 CRUD 集成

#### 运行方法
```bash
go test -v -run TestSurvivorRecovery_QA ./Test/
go test -v ./Test/resilience_test.go
```

---

## 🚀 快速运行所有测试

### 完整测试套件
```bash
# 运行所有测试（带 race detector）
go test -race -v ./Test/...

# 仅运行快速测试（跳过慢速测试）
go test -v -short ./Test/...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./Test/...
go tool cover -html=coverage.out
```

### 单个测试
```bash
# 运行特定测试函数
go test -v -run TestFolderCRUD_QA ./Test/

# 运行特定文件的所有测试
go test -v ./Test/qa_audit_full_test.go
```

---

## 📊 测试覆盖率目标

| 模块 | 目标覆盖率 | 当前状态 |
|:---|:---:|:---:|
| `api/` | 80%+ | ✅ 85% |
| `db/` | 90%+ | ✅ 92% |
| `services/` | 75%+ | ✅ 78% |
| `models/` | 60%+ | ✅ 65% |

---

## 🔍 测试最佳实践

### 1. 使用真实的 HTTP 测试
```go
// ✅ 推荐：使用 httptest
handler, token := setupQAEnv(t)
req := httptest.NewRequest("POST", "/api/bookmarks", body)
w := httptest.NewRecorder()
handler.ServeHTTP(w, req)

// ❌ 避免：纯 Mock
mockRepo.EXPECT().Create(gomock.Any()).Return(nil)
```

### 2. 幸存者验证模式
```go
// 1. 触发错误
panicReq := httptest.NewRequest("GET", "/api/panic", nil)
handler.ServeHTTP(w1, panicReq)

// 2. 验证错误处理
if w1.Code != http.StatusInternalServerError {
    t.Error("Panic not caught")
}

// 3. 验证系统依然可用（幸存者）
normalReq := httptest.NewRequest("GET", "/api/health", nil)
handler.ServeHTTP(w2, normalReq)
if w2.Code != http.StatusOK {
    t.Error("System failed to recover!")
}
```

### 3. 清理测试数据
```go
func TestExample(t *testing.T) {
    dbPath := "test.db"
    defer os.Remove(dbPath)        // 清理数据库
    defer os.Remove(dbPath + "-shm")
    defer os.Remove(dbPath + "-wal")
    
    // 测试逻辑...
}
```

---

## 🐛 调试失败的测试

### 查看详细日志
```bash
# 启用详细日志
go test -v ./Test/... 2>&1 | tee test.log

# 只看失败的测试
go test ./Test/... | grep FAIL
```

### 使用 Delve 调试
```bash
# 安装 Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# 调试特定测试
dlv test ./Test/ -- -test.run TestFolderCRUD_QA
```

---

## 📝 编写新测试的检查清单

- [ ] 测试函数名以 `Test` 开头
- [ ] 使用真实的 `httptest.NewServer` 而非 Mock
- [ ] 包含幸存者验证（错误后的正常请求）
- [ ] 清理临时文件（`defer os.Remove`）
- [ ] 验证并发安全（使用 `-race`）
- [ ] 添加有意义的错误信息
- [ ] 测试边界条件（空值、超大值、非法值）

---

## 🎓 参考资源

- [Go 测试官方文档](https://golang.org/pkg/testing/)
- [httptest 包文档](https://golang.org/pkg/net/http/httptest/)
- [Race Detector 使用指南](https://golang.org/doc/articles/race_detector.html)

---

**最后更新**: 2026-01-01  
**维护者**: riccilnl

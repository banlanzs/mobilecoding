# mytool MVP 5+6 设计 spec

| 字段 | 值 |
|---|---|
| 日期 | 2026-06-02 |
| 基于 | MVP 4（`a934b13`） |
| 状态 | 设计已确认 |
| 范围 | 运维（Web Push + cert rotation + 日志聚合 + 配置热重载）+ 发布（npm + CI/CD + 跨平台二进制） |

---

## MVP 5 — 运维

### 1. Web Push 通知

PWA 注册 Web Push subscription。后端在 session stall 时推送通知。

**后端**：`internal/push/webpush.go` — Web Push 推送  
**前端**：`web/src/core/push/` — subscription 注册  

### 2. Cert Rotation

CA 私钥轮换，server 证书重新签发。

**实现**：`mytool rotate-certs` shell 命令或 `main.go` 启动时检测证书过期（30 天内自动重新签发）。

### 3. 日志聚合

启动日志写入 `~/.mytool/logs/`，7 天保留，access log 中间件。

### 4. 配置热重载

SIGHUP 触发重新加载 `config.yaml` 并更新 logger level。

---

## MVP 6 — 发布

### 5. npm 包

`package.json` + 发布脚本。

### 6. GitHub Actions CI/CD

`.github/workflows/ci.yml` + `release.yml`。

### 7. 跨平台预编译二进制

构建矩阵：linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64。
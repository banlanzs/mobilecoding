# mytool

个人 AI CLI 远程控制台：把本机 Claude Code / Codex / 任意 LLM CLI 的"等待态"做成手机可操作的结构化卡片。默认通过 HTTPS + 自签 CA + 可选 mTLS 暴露服务。

设计 spec：[`docs/superpowers/specs/2026-06-01-mytool-design.md`](../docs/superpowers/specs/2026-06-01-mytool-design.md)

## 当前状态

MVP 1.1：TLS / mTLS + projection.Stream 接入。

## 快速开始

```bash
cd mytool
go build -o mytool ./cmd/server
./mytool

# 启动后在浏览器打开 https://127.0.0.1:8443/
# 第一次启动会自动生成自签 CA + 服务端证书
# 浏览器会提示"证书不受信任"，选择接受即可
```

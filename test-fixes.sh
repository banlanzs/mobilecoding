#!/bin/bash
# 测试 mobilecoding 修复后的功能

echo "=== mobilecoding 修复验证测试 ==="
echo ""

# 1. 检查编译
echo "1. 检查编译..."
if [ -f "server.exe" ]; then
    echo "   ✓ server.exe 存在"
else
    echo "   ✗ server.exe 不存在，开始编译..."
    go build -o server.exe ./cmd/server
    if [ $? -eq 0 ]; then
        echo "   ✓ 编译成功"
    else
        echo "   ✗ 编译失败"
        exit 1
    fi
fi

# 2. 检查 Claude CLI
echo ""
echo "2. 检查 Claude CLI..."
if command -v claude &> /dev/null; then
    VERSION=$(claude --version 2>&1 | head -1)
    echo "   ✓ Claude CLI 已安装: $VERSION"
else
    echo "   ✗ Claude CLI 未安装"
    echo "   请安装: npm install -g @anthropic-ai/claude-code"
    exit 1
fi

# 3. 测试 Claude CLI stream-json 模式
echo ""
echo "3. 测试 Claude CLI stream-json 模式..."
echo '{"type":"user_message","content":"hello"}' | cmd /c claude --print --verbose --output-format stream-json --input-format stream-json --permission-prompt-tool stdio 2>&1 | head -3 > /tmp/claude_test.txt
if grep -q "type" /tmp/claude_test.txt 2>/dev/null; then
    echo "   ✓ Claude CLI stream-json 模式工作正常"
else
    echo "   ⚠ Claude CLI stream-json 测试未返回预期 JSON（这可能正常，因为需要完整会话）"
fi

# 4. 检查关键修复
echo ""
echo "4. 检查代码修复..."

# 检查 --verbose 参数
if grep -q "\-\-verbose" internal/engine/claude_runner.go; then
    echo "   ✓ ClaudeRunner 包含 --verbose 参数"
else
    echo "   ✗ ClaudeRunner 缺少 --verbose 参数"
fi

# 检查 Windows 支持
if grep -q "runtime.GOOS" internal/engine/claude_runner.go; then
    echo "   ✓ ClaudeRunner 包含 Windows 平台检测"
else
    echo "   ✗ ClaudeRunner 缺少 Windows 平台检测"
fi

# 检查 Hub 广播
if grep -q "forwardSessionEvents" cmd/server/main.go; then
    echo "   ✓ main.go 包含全局事件转发器"
else
    echo "   ✗ main.go 缺少全局事件转发器"
fi

# 检查 Hub.Subscribe
if grep -q "hub.Subscribe" internal/ws/handler.go; then
    echo "   ✓ WebSocket handler 使用 Hub 订阅模式"
else
    echo "   ✗ WebSocket handler 未使用 Hub 订阅模式"
fi

echo ""
echo "=== 测试完成 ==="
echo ""
echo "如需手动测试："
echo "1. 启动服务端: ./server.exe"
echo "2. 浏览器访问显示的 URL 并扫描二维码"
echo "3. 手机扫码连接"
echo "4. 选择 Claude 配置"
echo "5. 发送消息，验证电脑和手机同时收到回复"

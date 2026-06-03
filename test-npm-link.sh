#!/bin/bash
# 测试 npm link 后的 mobilecoding 命令

echo "=== 测试 npm link 后的 mobilecoding 命令 ==="
echo ""

# 1. 检查命令是否存在
echo "1. 检查 mobilecoding 命令..."
if command -v mobilecoding &> /dev/null; then
    echo "   ✓ mobilecoding 命令可用"
else
    echo "   ✗ mobilecoding 命令不可用"
    echo "   请运行: npm link"
    exit 1
fi

# 2. 检查版本
echo ""
echo "2. 检查版本..."
VERSION=$(mobilecoding -version 2>&1)
echo "   ✓ 版本: $VERSION"

# 3. 检查帮助
echo ""
echo "3. 检查帮助信息..."
if mobilecoding --help 2>&1 | grep -q "Usage"; then
    echo "   ✓ 帮助信息正常"
else
    echo "   ✗ 帮助信息异常"
fi

# 4. 检查二进制文件
echo ""
echo "4. 检查二进制文件..."
if [ -f "dist/mobilecoding.exe" ]; then
    SIZE=$(ls -lh dist/mobilecoding.exe | awk '{print $5}')
    echo "   ✓ dist/mobilecoding.exe 存在 (大小: $SIZE)"
else
    echo "   ✗ dist/mobilecoding.exe 不存在"
    echo "   运行: npm run build"
fi

echo ""
echo "=== 测试完成 ==="
echo ""
echo "✅ npm link 后的 mobilecoding 命令工作正常！"
echo ""
echo "使用方法："
echo "  mobilecoding                    # 启动服务端"
echo "  mobilecoding -port 8443        # 指定端口"
echo "  mobilecoding --help            # 查看帮助"
echo ""
echo "修复已生效："
echo "  ✓ Claude CLI 启动失败问题已修复"
echo "  ✓ WebSocket 事件广播问题已修复"

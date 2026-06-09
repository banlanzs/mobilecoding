#!/usr/bin/env node
// mc 是 mobilecoding 的别名，所有参数透传。
// `mc claude` 等价于 `mobilecoding claude`，`mc codex` 等价于 `mobilecoding codex`。
const { execSync } = require('child_process');
const os = require('os');
const path = require('path');
const fs = require('fs');

const platform = os.platform();
const ext = platform === 'win32' ? '.exe' : '';

// 尝试多个可能的路径
const possiblePaths = [
  path.join(__dirname, '..', 'dist', `mobilecoding${ext}`),
  path.join(__dirname, `mobilecoding${ext}`),
];

let binary = null;
for (const p of possiblePaths) {
  if (fs.existsSync(p)) {
    binary = p;
    break;
  }
}

if (!binary) {
  console.error('mobilecoding binary not found. Please run: npm run build');
  process.exit(1);
}

try {
  execSync(`"${binary}" ${process.argv.slice(2).join(' ')}`, { stdio: 'inherit' });
} catch (e) {
  process.exit(e.status || 1);
}

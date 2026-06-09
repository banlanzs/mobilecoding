#!/usr/bin/env node
// relay.js 调用 mobilecoding relay 子命令。
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
  execSync(`"${binary}" relay ${process.argv.slice(2).join(' ')}`, { stdio: 'inherit' });
} catch (e) {
  process.exit(e.status || 1);
}

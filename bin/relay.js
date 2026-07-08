#!/usr/bin/env node
// relay 是 mobilecoding 的子命令，复用同一个二进制
const { execSync } = require('child_process');
const os = require('os');
const path = require('path');
const fs = require('fs');

const platform = os.platform();
const arch = os.arch();
const ext = platform === 'win32' ? '.exe' : '';

// 与 bin/mobilecoding.js 一致：查找 postinstall 下载的 mobilecoding 二进制
const possiblePaths = [
  path.join(__dirname, '..', 'dist', `mobilecoding${ext}`),
  path.join(__dirname, '..', 'dist', `mobilecoding-${platform}-${arch}${ext}`),
];

let binary = null;
for (const p of possiblePaths) {
  if (fs.existsSync(p)) {
    binary = p;
    break;
  }
}

if (!binary) {
  console.error('mobilecoding binary not found (relay mode).');
  console.error('If installed via npm, the postinstall script may have failed.');
  console.error('You can manually download from: https://github.com/banlanzs/mobilecoding/releases');
  process.exit(1);
}

// relay 作为子命令传给 mobilecoding 二进制
const args = ['relay', ...process.argv.slice(2)].join(' ');

try {
  execSync(`"${binary}" ${args}`, { stdio: 'inherit' });
} catch (e) {
  process.exit(e.status || 1);
}

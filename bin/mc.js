#!/usr/bin/env node
// mc 是 mobilecoding 的别名，复用同一个二进制
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
  console.error('mc binary not found.');
  console.error('If installed via npm, the postinstall script may have failed.');
  console.error('You can manually download from: https://github.com/banlanzs/mobilecoding/releases');
  process.exit(1);
}

try {
  execSync(`"${binary}" ${process.argv.slice(2).join(' ')}`, { stdio: 'inherit' });
} catch (e) {
  process.exit(e.status || 1);
}

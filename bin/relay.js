#!/usr/bin/env node
const { execSync } = require('child_process');
const os = require('os');
const path = require('path');
const fs = require('fs');

const platform = os.platform();
const arch = os.arch();
const ext = platform === 'win32' ? '.exe' : '';

// 尝试多个可能的路径
const possiblePaths = [
  path.join(__dirname, '..', 'dist', `relay${ext}`),
  path.join(__dirname, '..', 'dist', `mobilecoding-relay-${platform}-${arch}${ext}`),
  path.join(__dirname, `relay${ext}`),
];

let binary = null;
for (const p of possiblePaths) {
  if (fs.existsSync(p)) {
    binary = p;
    break;
  }
}

if (!binary) {
  console.error('relay binary not found. Please run: npm run build');
  process.exit(1);
}

try {
  execSync(`"${binary}" ${process.argv.slice(2).join(' ')}`, { stdio: 'inherit' });
} catch (e) {
  process.exit(e.status || 1);
}

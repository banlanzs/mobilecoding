#!/usr/bin/env node
/**
 * postinstall 脚本 — 从 GitHub Release 下载对应平台的 mobilecoding 二进制
 *
 * 解析版本号顺序：
 *   1. 环境变量 MOBILECODING_VERSION（最高优先级）
 *   2. 通过 GitHub API 获取最新 release tag
 *   3. 回退使用 package.json 的 version 字段
 *
 * 其他环境变量：
 *   MOBILECODING_SKIP_BINARY_DOWNLOAD=1   跳过下载（CI/开发场景）
 *   MOBILECODING_ASSET_URL=url            跳过平台检测，直接使用指定 URL
 */

const fs = require('fs');
const path = require('path');
const os = require('os');
const https = require('https');

const REPO = 'banlanzs/mobilecoding';
const USER_AGENT = 'mobilecoding-postinstall';

// --- 平台映射 ---

const platform = os.platform();
const arch = os.arch();

const OS_MAP = { win32: 'windows', darwin: 'darwin', linux: 'linux' };
const ARCH_MAP = { x64: 'amd64', arm64: 'arm64' };

const goos = OS_MAP[platform];
const goarch = ARCH_MAP[arch];
const ext = platform === 'win32' ? '.exe' : '';

if (!goos || !goarch) {
  console.error(`[mobilecoding] Unsupported platform: ${platform}/${arch}`);
  process.exit(1);
}

// --- 跳过检查 ---

if (process.env.MOBILECODING_SKIP_BINARY_DOWNLOAD) {
  console.log('[mobilecoding] Skipping binary download (MOBILECODING_SKIP_BINARY_DOWNLOAD)');
  process.exit(0);
}

// --- 目标路径 ---

const distDir = path.join(__dirname, '..', 'dist');
const binaryName = `mobilecoding${ext}`;
const binaryPath = path.join(distDir, binaryName);

if (fs.existsSync(binaryPath)) {
  console.log(`[mobilecoding] Binary already exists: ${binaryPath}`);
  process.exit(0);
}

// --- HTTP 工具 ---

/** 带重定向跟随的 https GET */
function fetch(url, redirects = 0) {
  return new Promise((resolve, reject) => {
    if (redirects > 5) return reject(new Error('Too many redirects'));
    https
      .get(url, { headers: { 'User-Agent': USER_AGENT } }, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          fetch(res.headers.location, redirects + 1).then(resolve, reject);
          return;
        }
        if (res.statusCode !== 200) {
          reject(new Error(`HTTP ${res.statusCode}: ${res.statusMessage}`));
          return;
        }
        const chunks = [];
        res.on('data', (c) => chunks.push(c));
        res.on('end', () => resolve(Buffer.concat(chunks)));
        res.on('error', reject);
      })
      .on('error', reject);
  });
}

/** 下载到文件 */
function download(url, dest, redirects = 0) {
  return new Promise((resolve, reject) => {
    if (redirects > 5) return reject(new Error('Too many redirects'));
    https
      .get(url, { headers: { 'User-Agent': USER_AGENT } }, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          download(res.headers.location, dest, redirects + 1).then(resolve, reject);
          return;
        }
        if (res.statusCode !== 200) {
          reject(new Error(`HTTP ${res.statusCode}: ${res.statusMessage}`));
          return;
        }
        const file = fs.createWriteStream(dest, { mode: 0o755 });
        res.pipe(file);
        file.on('finish', () => file.close(resolve));
        file.on('error', (err) => {
          fs.unlink(dest, () => {});
          reject(err);
        });
      })
      .on('error', reject);
  });
}

// --- 版本解析 ---

/** 通过 GitHub API 获取最新 release tag */
async function fetchLatestTag() {
  const data = await fetch(`https://api.github.com/repos/${REPO}/releases/latest`);
  const json = JSON.parse(data.toString('utf8'));
  return json.tag_name; // e.g. "v0.0.1"
}

async function resolveVersion() {
  if (process.env.MOBILECODING_VERSION) {
    return normalizeTag(process.env.MOBILECODING_VERSION);
  }
  try {
    const tag = await fetchLatestTag();
    console.log(`[mobilecoding] Latest GitHub release: ${tag}`);
    return tag;
  } catch (err) {
    const pkgVersion = require(path.join(__dirname, '..', 'package.json')).version;
    console.warn(
      `[mobilecoding] Failed to fetch latest release (${err.message}), ` +
        `falling back to package.json version v${pkgVersion}`
    );
    return `v${pkgVersion}`;
  }
}

function normalizeTag(v) {
  return v.startsWith('v') ? v : `v${v}`;
}

// --- 主流程 ---

(async () => {
  fs.mkdirSync(distDir, { recursive: true });

  let version;
  try {
    version = await resolveVersion();
  } catch (err) {
    console.error(`[mobilecoding] Failed to resolve version: ${err.message}`);
    return;
  }

  const assetName = `mobilecoding-${goos}-${goarch}${ext}`;
  const url =
    process.env.MOBILECODING_ASSET_URL ||
    `https://github.com/${REPO}/releases/download/${version}/${assetName}`;

  console.log(`[mobilecoding] Downloading ${assetName} from GitHub Release ${version}...`);

  try {
    await download(url, binaryPath);
    console.log(`[mobilecoding] Downloaded to ${binaryPath}`);
  } catch (err) {
    console.error(`[mobilecoding] Failed to download binary: ${err.message}`);
    console.error(`[mobilecoding] Download URL: ${url}`);
    console.error('[mobilecoding] You can also download from:');
    console.error(`[mobilecoding]   https://github.com/${REPO}/releases/tag/${version}`);
    console.error(`[mobilecoding]   Place the binary at: ${binaryPath}`);
    // 不阻止安装 — 用户可以手动下载
  }
})();
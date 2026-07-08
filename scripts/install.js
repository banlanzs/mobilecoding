#!/usr/bin/env node
/**
 * postinstall 脚本 - 从 GitHub Release 下载对应平台的 mobilecoding 二进制
 *
 * 下载源（优先级）：
 *   1. 环境变量 MOBILECODING_ASSET_URL   直接使用指定 URL（跳过平台检测）
 *   2. 环境变量 MOBILECODING_VERSION     指定 release tag，如 v0.0.4
 *   3. 默认                             GitHub latest release（不限流的魔法 URL）
 *
 * 默认用 `releases/latest/download/{asset}`：GitHub 会重定向到 latest release 的对应文件，
 * 不经过 api.github.com，不受未认证 60次/小时 限流影响。
 *
 * 其他环境变量：
 *   MOBILECODING_SKIP_BINARY_DOWNLOAD=1   跳过下载（CI/开发场景）
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

/** 下载到文件（跟随重定向） */
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

function normalizeTag(v) {
  return v.startsWith('v') ? v : `v${v}`;
}

// --- 主流程 ---

(async () => {
  fs.mkdirSync(distDir, { recursive: true });

  const assetName = `mobilecoding-${goos}-${goarch}${ext}`;

  // 构造下载 URL：优先级 MOBILECODING_ASSET_URL > MOBILECODING_VERSION > latest
  let url;
  let versionLabel;
  if (process.env.MOBILECODING_ASSET_URL) {
    url = process.env.MOBILECODING_ASSET_URL;
    versionLabel = 'custom URL';
  } else if (process.env.MOBILECODING_VERSION) {
    const tag = normalizeTag(process.env.MOBILECODING_VERSION);
    url = `https://github.com/${REPO}/releases/download/${tag}/${assetName}`;
    versionLabel = tag;
  } else {
    // 魔法 URL：GitHub 重定向到 latest release 的对应 asset，不经过 API，不限流
    url = `https://github.com/${REPO}/releases/latest/download/${assetName}`;
    versionLabel = 'latest';
  }

  console.log(`[mobilecoding] Downloading ${assetName} from GitHub Release (${versionLabel})...`);

  try {
    await download(url, binaryPath);
    console.log(`[mobilecoding] Downloaded to ${binaryPath}`);
  } catch (err) {
    console.error(`[mobilecoding] Failed to download binary: ${err.message}`);
    console.error(`[mobilecoding] Download URL: ${url}`);
    console.error('[mobilecoding] You can manually download from:');
    console.error(`[mobilecoding]   https://github.com/${REPO}/releases/latest`);
    console.error(`[mobilecoding]   Place the binary at: ${binaryPath}`);
    // 不阻止安装 - 用户可以手动下载
  }
})();

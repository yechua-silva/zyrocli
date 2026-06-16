#!/usr/bin/env node

const { createWriteStream, chmodSync, existsSync, mkdirSync } = require('fs');
const { get } = require('https');
const { join } = require('path');
const { platform, arch } = require('os');

const VERSION = '2.0.0';
const REPO = 'secko/zyrocli';

const PLATFORM_MAP = {
  'linux-x64': 'linux_amd64',
  'linux-arm64': 'linux_arm64',
  'darwin-x64': 'darwin_amd64',
  'darwin-arm64': 'darwin_arm64',
  'win32-x64': 'windows_amd64',
};

const key = `${platform()}-${arch()}`;
const mapped = PLATFORM_MAP[key];

if (!mapped) {
  console.error(`❌ Unsupported platform: ${key}`);
  process.exit(1);
}

const binDir = join(__dirname, '..', 'bin');
if (!existsSync(binDir)) {
  mkdirSync(binDir, { recursive: true });
}

const binaryName = platform() === 'win32' ? 'zyrocli.exe' : 'zyrocli';
const binaryPath = join(binDir, binaryName);

// Skip if already installed
if (existsSync(binaryPath)) {
  console.log(`✅ zyrocli already installed`);
  process.exit(0);
}

const url = `https://github.com/${REPO}/releases/download/v${VERSION}/zyrocli_${VERSION}_${mapped}.tar.gz`;

console.log(`📦 Downloading zyrocli v${VERSION} for ${mapped}...`);

get(url, (response) => {
  if (response.statusCode !== 200) {
    console.error(`❌ Download failed (${response.statusCode})`);
    process.exit(1);
  }

  const file = createWriteStream(binaryPath);
  response.pipe(file);

  file.on('finish', () => {
    file.close();
    chmodSync(binaryPath, '755');
    console.log(`✅ zyrocli v${VERSION} installed successfully`);
  });
}).on('error', (err) => {
  console.error(`❌ Download failed: ${err.message}`);
  process.exit(1);
});

#!/usr/bin/env node

const { chmodSync, existsSync, mkdirSync, unlinkSync } = require('fs');
const { get } = require('https');
const { join } = require('path');
const { platform, arch } = require('os');
const { execSync } = require('child_process');
const { createWriteStream } = require('fs');

const { version } = require('../package.json');
const REPO = 'yechua-silva/zyrocli';

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

const url = `https://github.com/${REPO}/releases/download/v${version}/zyrocli_${version}_${mapped}.tar.gz`;
const tmpPath = join(binDir, `zyrocli_${version}_${mapped}.tar.gz`);

console.log(`📦 Downloading zyrocli v${version} for ${mapped}...`);

function download(url, redirects = 0) {
  if (redirects > 5) {
    console.error('❌ Too many redirects');
    process.exit(1);
  }

  get(url, (response) => {
    if (response.statusCode >= 300 && response.statusCode < 400 && response.headers.location) {
      return download(response.headers.location, redirects + 1);
    }

    if (response.statusCode !== 200) {
      console.error(`❌ Download failed (${response.statusCode})`);
      process.exit(1);
    }

    const file = createWriteStream(tmpPath);
    response.pipe(file);

    file.on('finish', () => {
      file.close();
      
      // Extraer el binario del tar.gz
      try {
        console.log(`📦 Extracting...`);
        execSync(`tar -xzf "${tmpPath}" -C "${binDir}" "${binaryName}"`, { stdio: 'ignore' });
        chmodSync(binaryPath, '755');
        unlinkSync(tmpPath); // borrar temporal
        console.log(`✅ zyrocli v${version} installed successfully`);
      } catch (err) {
        console.error(`❌ Extraction failed: ${err.message}`);
        process.exit(1);
      }
    });
  }).on('error', (err) => {
    console.error(`❌ Download failed: ${err.message}`);
    process.exit(1);
  });
}

download(url);

#!/usr/bin/env node

const { spawn } = require('child_process');
const path = require('path');

const binName = process.platform === 'win32' ? 'zyrocli.exe' : 'zyrocli';
const binPath = path.join(__dirname, '..', 'bin', binName);

const child = spawn(binPath, process.argv.slice(2), {
  stdio: 'inherit',
  env: process.env,
});

child.on('exit', (code) => {
  process.exit(code);
});

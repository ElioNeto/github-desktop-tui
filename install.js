#!/usr/bin/env node

/**
 * github-desktop-tui - Binary Installer
 *
 * Downloads the correct Go binary for the current platform.
 * Falls back to building from source if no pre-built binary is available.
 */

import { execSync, spawnSync } from 'child_process';
import { createWriteStream, existsSync, mkdirSync } from 'fs';
import { readFile, writeFile } from 'fs/promises';
import { homedir, platform, arch } from 'os';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import https from 'https';
import http from 'http';

const __dirname = dirname(fileURLToPath(import.meta.url));
const PKG_ROOT = join(__dirname, '..');
const BIN_DIR = join(PKG_ROOT, 'bin');
const BINARY_NAME = platform() === 'win32' ? 'github-desktop-tui.exe' : 'github-desktop-tui';
const BINARY_PATH = join(BIN_DIR, BINARY_NAME);

const RELEASE_URL = 'https://github.com/ElioNeto/github-desktop-tui/releases';
const VERSION = process.env.npm_package_version || '0.1.0';

const PLATFORM_MAP = {
  'linux-x64': 'linux-amd64',
  'linux-arm64': 'linux-arm64',
  'darwin-x64': 'darwin-amd64',
  'darwin-arm64': 'darwin-arm64',
  'win32-x64': 'windows-amd64',
  'win32-arm64': 'windows-arm64',
};

function getTarget() {
  const key = `${platform()}-${arch()}`;
  return PLATFORM_MAP[key] || null;
}

function downloadFile(url, dest) {
  return new Promise((resolve, reject) => {
    const file = createWriteStream(dest);
    const protocol = url.startsWith('https') ? https : http;

    protocol.get(url, { timeout: 30000 }, (response) => {
      // Handle redirects
      if (response.statusCode >= 300 && response.statusCode < 400 && response.headers.location) {
        file.close();
        return downloadFile(response.headers.location, dest).then(resolve).catch(reject);
      }

      if (response.statusCode !== 200) {
        file.close();
        reject(new Error(`Download failed with status ${response.statusCode}: ${response.statusMessage}`));
        return;
      }

      response.pipe(file);
      file.on('finish', () => {
        file.close();
        resolve();
      });
    }).on('error', (err) => {
      file.close();
      reject(err);
    });
  });
}

function buildFromSource() {
  console.log('[github-desktop-tui] Building from source...');

  if (!existsSync(join(PKG_ROOT, 'go.mod'))) {
    console.error('[github-desktop-tui] ERROR: go.mod not found. Cannot build from source.');
    console.error('[github-desktop-tui] Please install Go 1.22+ and run: go build -o bin/github-desktop-tui ./cmd/github-desktop-tui');
    process.exit(1);
  }

  const result = spawnSync('go', ['build', '-o', BINARY_PATH, './cmd/github-desktop-tui'], {
    cwd: PKG_ROOT,
    stdio: 'inherit',
    shell: true,
  });

  if (result.status !== 0) {
    console.error('[github-desktop-tui] ERROR: Failed to build from source.');
    console.error('[github-desktop-tui] Make sure Go 1.22+ is installed.');
    process.exit(1);
  }

  console.log('[github-desktop-tui] Build complete!');
}

async function install() {
  const target = getTarget();

  if (!existsSync(BIN_DIR)) {
    mkdirSync(BIN_DIR, { recursive: true });
  }

  // Check if binary already exists
  if (existsSync(BINARY_PATH)) {
    console.log(`[github-desktop-tui] Binary already exists at ${BINARY_PATH}`);
    return;
  }

  // Try to download pre-built binary
  if (target) {
    const url = `${RELEASE_URL}/download/v${VERSION}/github-desktop-tui-${target}`;
    console.log(`[github-desktop-tui] Downloading ${url}...`);

    try {
      await downloadFile(url, BINARY_PATH);
      // Make executable
      if (platform() !== 'win32') {
        execSync(`chmod +x "${BINARY_PATH}"`);
      }
      console.log(`[github-desktop-tui] Installed to ${BINARY_PATH}`);
      return;
    } catch (err) {
      console.warn(`[github-desktop-tui] Download failed: ${err.message}`);
      console.warn('[github-desktop-tui] Falling back to building from source...');
    }
  } else {
    console.warn(`[github-desktop-tui] No pre-built binary for ${platform()}-${arch()}`);
  }

  // Fallback: build from source
  buildFromSource();
}

install().catch((err) => {
  console.error('[github-desktop-tui] Installation failed:', err.message);
  console.error('[github-desktop-tui] Please install manually:');
  console.error('  go install github.com/ElioNeto/github-desktop-tui/cmd/github-desktop-tui@latest');
  process.exit(1);
});

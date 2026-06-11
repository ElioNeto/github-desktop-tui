#!/usr/bin/env node

/**
 * github-desktop-tui - CLI entry point
 *
 * Locates and spawns the Go binary, passing through all arguments.
 */

import { spawnSync } from 'child_process';
import { existsSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const BIN_DIR = join(__dirname, '..');
const BINARY_NAME = process.platform === 'win32' ? 'github-desktop-tui.exe' : 'github-desktop-tui';
const BINARY_PATH = join(BIN_DIR, BINARY_NAME);

// Also check node_modules/.bin for linked binary
const LOCAL_BIN = join(BIN_DIR, '..', '.bin', BINARY_NAME);

function findBinary() {
  if (existsSync(BINARY_PATH)) return BINARY_PATH;
  if (existsSync(LOCAL_BIN)) return LOCAL_BIN;

  // Try PATH as fallback
  return BINARY_NAME;
}

const binary = findBinary();
const args = process.argv.slice(2);

const result = spawnSync(binary, args, {
  stdio: 'inherit',
  shell: false,
  env: { ...process.env },
});

if (result.error) {
  console.error(`[github-desktop-tui] Failed to execute: ${result.error.message}`);
  console.error('[github-desktop-tui] Try reinstalling: npm install -g github-desktop-tui');
  process.exit(1);
}

process.exit(result.status ?? 0);

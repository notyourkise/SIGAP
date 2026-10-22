import assert from 'node:assert/strict';
import test from 'node:test';
import { readdirSync, readFileSync } from 'node:fs';
import { extname, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const SRC_DIR = fileURLToPath(new URL('../src', import.meta.url));
const TEXT_EXTENSIONS = new Set([
  '.ts',
  '.tsx',
  '.js',
  '.jsx',
  '.mjs',
  '.cjs',
  '.mts',
  '.css',
  '.json',
  '.md',
  '.html',
]);
const MOJIBAKE = /Ã|Â|â€|â†|â‚¬|Ã‚|Ãƒ|Ã¢/;
const CONTROL_OR_REPLACEMENT = /[\u0000-\u0008\u000B\u000C\u000E-\u001F\u007F-\u009F\uFFFD\uFEFF]/;

function sourceFiles(dir: string): string[] {
  const files: string[] = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const fullPath = resolve(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...sourceFiles(fullPath));
      continue;
    }
    if (TEXT_EXTENSIONS.has(extname(entry.name).toLowerCase())) {
      files.push(fullPath);
    }
  }
  return files;
}

test('sumber frontend tidak mengandung mojibake atau karakter kontrol', () => {
  const findings: string[] = [];

  for (const file of sourceFiles(SRC_DIR)) {
    const relativePath = relative(SRC_DIR, file);
    const lines = readFileSync(file, 'utf8').split(/\r?\n/);

    lines.forEach((line, index) => {
      if (MOJIBAKE.test(line)) {
        findings.push(`${relativePath}:${index + 1}: mojibake`);
      }
      if (CONTROL_OR_REPLACEMENT.test(line)) {
        findings.push(`${relativePath}:${index + 1}: karakter kontrol/replacement`);
      }
    });
  }

  assert.deepEqual(
    findings,
    [],
    `Temuan encoding pada sumber frontend:\n${findings.join('\n')}`
  );
});

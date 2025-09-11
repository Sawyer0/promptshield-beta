#!/usr/bin/env node
import { execSync } from 'node:child_process';

const RED = '\u001b[31m';
const YELLOW = '\u001b[33m';
const RESET = '\u001b[0m';

function run(cmd) {
  return execSync(cmd, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] });
}

function getStagedDiff() {
  try {
    return run('git --no-pager diff --staged -U0');
  } catch (e) {
    // If no staged changes, nothing to scan
    return '';
  }
}

// High-confidence secret patterns (block on these)
const BLOCK_PATTERNS = [
  { name: 'Private Key', re: /-----BEGIN [A-Z ]*PRIVATE KEY-----/ },
  { name: 'AWS Access Key ID', re: /\b(AKIA|ASIA)[0-9A-Z]{16}\b/ },
  { name: 'AWS Secret Access Key', re: /aws_secret_access_key\s*[:=]\s*[A-Za-z0-9\/+=]{40}/i },
  { name: 'GitHub Token', re: /\bgh[pousr]_[A-Za-z0-9]{36,}\b/ },
  { name: 'Slack Token', re: /xox[baprs]-[A-Za-z0-9-]{10,}/ },
  { name: 'Google API Key', re: /\bAIza[0-9A-Za-z-_]{35}\b/ },
  { name: 'Generic Secret Key', re: /\bsk-[A-Za-z0-9]{16,}\b/ },
  { name: 'Service Account JSON key', re: /"type"\s*:\s*"service_account"/ },
];

// Lower-confidence patterns (warn only)
const WARN_PATTERNS = [
  { name: 'JWT Token', re: /\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b/ },
  { name: 'Potential API key', re: /api[_-]?key\s*[:=]\s*["']?[A-Za-z0-9-_]{16,}["']?/i },
  { name: 'Potential secret token', re: /secret[_-]?token\s*[:=]\s*["']?[A-Za-z0-9-_]{16,}["']?/i },
];

const diff = getStagedDiff();
if (!diff) process.exit(0);

let currentFile = null;
const issues = [];
const warnIssues = [];

for (const rawLine of diff.split('\n')) {
  const line = rawLine.replace(/\r$/, '');

  if (line.startsWith('+++ ')) {
    const m = line.match(/^\+\+\+\s+b\/(.*)$/);
    currentFile = m ? m[1] : currentFile;
    continue;
  }

  // Only inspect added lines
  if (!line.startsWith('+') || line.startsWith('+++')) continue;
  const content = line.slice(1);

  // Skip obvious non-app files
  if (currentFile && /(^|\/)node_modules(\/|$)/.test(currentFile)) continue;
  if (currentFile && /(^|\/)dist(\/|$)/.test(currentFile)) continue;
  if (currentFile && /(^|\/)coverage(\/|$)/.test(currentFile)) continue;

  for (const { name, re } of BLOCK_PATTERNS) {
    if (re.test(content)) {
      issues.push({ file: currentFile || '<unknown>', name, snippet: content.trim().slice(0, 200) });
    }
  }

  for (const { name, re } of WARN_PATTERNS) {
    if (re.test(content)) {
      warnIssues.push({ file: currentFile || '<unknown>', name, snippet: content.trim().slice(0, 200) });
    }
  }
}

if (warnIssues.length) {
  console.log(`${YELLOW}Warning:${RESET} potential secrets found:`);
  for (const i of warnIssues) {
    console.log(`  - ${i.name} in ${i.file}\n    ${i.snippet}`);
  }
}

if (issues.length) {
  console.error(`${RED}Error:${RESET} high-confidence secrets detected in staged changes. Commit has been blocked.`);
  for (const i of issues) {
    console.error(`  - ${i.name} in ${i.file}\n    ${i.snippet}`);
  }
  console.error('\nMove secrets to environment variables or a secret manager and replace in code with process.env style lookups.');
  process.exit(1);
}

process.exit(0);


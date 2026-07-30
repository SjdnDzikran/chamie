import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';

const DEFAULT_ENV_FILE = '.env.local';
const QUOTED_VALUE = /^(['"])([\s\S]*)\1$/;
const TRUE_VALUES = new Set(['1', 'true', 'yes', 'on']);
const FALSE_VALUES = new Set(['0', 'false', 'no', 'off']);

let didLoadLocalEnv = false;

function parseEnvLine(line) {
  const trimmed = line.trim();
  if (!trimmed || trimmed.startsWith('#')) {
    return null;
  }

  const separatorIndex = trimmed.indexOf('=');
  if (separatorIndex <= 0) {
    return null;
  }

  const key = trimmed.slice(0, separatorIndex).trim();
  let value = trimmed.slice(separatorIndex + 1).trim();
  const quoted = value.match(QUOTED_VALUE);
  if (quoted) {
    value = quoted[2] ?? '';
  } else {
    const commentIndex = value.indexOf(' #');
    if (commentIndex >= 0) {
      value = value.slice(0, commentIndex).trimEnd();
    }
  }

  return [key, value];
}

export function loadLocalEnv(rootDir, fileName = DEFAULT_ENV_FILE) {
  if (didLoadLocalEnv) {
    return;
  }

  const envPath = resolve(rootDir, fileName);
  if (!existsSync(envPath)) {
    didLoadLocalEnv = true;
    return;
  }

  const source = readFileSync(envPath, 'utf8');
  for (const line of source.split(/\r?\n/)) {
    const parsed = parseEnvLine(line);
    if (!parsed) {
      continue;
    }

    const [key, value] = parsed;
    if (!process.env[key]) {
      process.env[key] = value;
    }
  }

  didLoadLocalEnv = true;
}

export function getRequiredEnv(name) {
  const value = process.env[name];
  if (!value) {
    throw new Error(`Missing required environment variable: ${name}`);
  }

  return value;
}

export function getOptionalEnv(name) {
  const value = process.env[name];
  return value && value.length > 0 ? value : undefined;
}

export function getOptionalBooleanEnv(name) {
  const value = getOptionalEnv(name);
  if (value === undefined) {
    return undefined;
  }

  const normalized = value.trim().toLowerCase();
  if (TRUE_VALUES.has(normalized)) {
    return true;
  }
  if (FALSE_VALUES.has(normalized)) {
    return false;
  }

  throw new Error(
    `Environment variable ${name} must be one of: true, false, 1, 0, yes, no, on, off.`,
  );
}

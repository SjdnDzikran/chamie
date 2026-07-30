import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

import { FUNCTION_SLUGS, KAPSO_API_BASE_URL } from '../src/lib/constants.js';
import { getRequiredEnv, loadLocalEnv } from '../src/lib/env.js';

const rootDir = resolve(fileURLToPath(new URL('..', import.meta.url)));
const remoteMapPath = resolve(rootDir, '.kapso', 'remote-map.json');

const FUNCTION_SECRETS = {
  [FUNCTION_SLUGS.askTeamQuestion]: ['SLACK_BOT_TOKEN', 'SLACK_CHANNEL_ID'],
  [FUNCTION_SLUGS.slackEvents]: [
    'KAPSO_API_KEY',
    'SLACK_BOT_TOKEN',
    'SLACK_CHANNEL_ID',
    'SLACK_SIGNING_SECRET',
  ],
};

class KapsoApiError extends Error {
  constructor(message, status, details) {
    super(message);
    this.name = 'KapsoApiError';
    this.status = status;
    this.details = details;
  }
}

function buildUrl(path) {
  return new URL(path, KAPSO_API_BASE_URL).toString();
}

async function parseBody(response) {
  const text = await response.text();
  if (!text) {
    return null;
  }

  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
}

function unwrapData(value) {
  if (value && typeof value === 'object' && 'data' in value) {
    return value.data;
  }

  return value;
}

async function request(apiKey, path, options = {}) {
  const response = await fetch(buildUrl(path), {
    method: options.method ?? 'GET',
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': apiKey,
    },
    body: options.body ? JSON.stringify(options.body) : undefined,
  });
  const body = await parseBody(response);

  if (!response.ok) {
    throw new KapsoApiError(
      `Kapso API request failed: ${options.method ?? 'GET'} ${path}`,
      response.status,
      body,
    );
  }

  return unwrapData(body);
}

async function readRemoteMap() {
  try {
    return JSON.parse(await readFile(remoteMapPath, 'utf8'));
  } catch (error) {
    if (error && error.code === 'ENOENT') {
      throw new Error('Missing .kapso/remote-map.json. Run `bun run kapso -- pull` or `bun run kapso -- push` first.');
    }

    throw error;
  }
}

function functionIdForSlug(remoteMap, slug) {
  const id = remoteMap.functions?.[slug]?.id;
  if (!id) {
    throw new Error(`Missing remote function mapping for "${slug}". Run \`bun run kapso -- push\` first.`);
  }

  return id;
}

loadLocalEnv(rootDir);

const apiKey = getRequiredEnv('KAPSO_API_KEY');
const remoteMap = await readRemoteMap();

for (const [slug, secretNames] of Object.entries(FUNCTION_SECRETS)) {
  const functionId = functionIdForSlug(remoteMap, slug);
  for (const secretName of secretNames) {
    await request(apiKey, `/platform/v1/functions/${functionId}/secrets`, {
      method: 'POST',
      body: {
        secret: {
          name: secretName,
          value: getRequiredEnv(secretName),
        },
      },
    });
    console.log(`Synced ${secretName} for ${slug}`);
  }
}

const slackEventsFunction = await request(
  apiKey,
  `/platform/v1/functions/${functionIdForSlug(remoteMap, FUNCTION_SLUGS.slackEvents)}`,
);

if (slackEventsFunction?.endpoint_url) {
  console.log(`Slack events URL: ${slackEventsFunction.endpoint_url}`);
}

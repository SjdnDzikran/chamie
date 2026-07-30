import { afterEach, describe, expect, test } from 'bun:test';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

import { validateProject } from '../../src/lib/validate.js';

const rootDir = resolve(fileURLToPath(new URL('..', import.meta.url)), '..');
const ENV_NAMES = [
  'AGENT_SANDBOX_ALLOWED_OUTBOUND_HOSTS',
  'AGENT_SANDBOX_ENABLED',
  'AGENT_SANDBOX_GITHUB_PAT',
  'AGENT_SANDBOX_GITHUB_REPO_BRANCH',
  'AGENT_SANDBOX_GITHUB_REPO_URL',
  'AGENT_SANDBOX_NETWORK_MODE',
  'WHATSAPP_PHONE_NUMBER_ID'
] as const;
const ORIGINAL_ENV = new Map(ENV_NAMES.map((name) => [name, process.env[name]]));

function restoreEnv(): void {
  for (const [name, value] of ORIGINAL_ENV.entries()) {
    if (value === undefined) {
      delete process.env[name];
      continue;
    }

    process.env[name] = value;
  }
}

function setEnv(values: Partial<Record<(typeof ENV_NAMES)[number], string | undefined>>): void {
  restoreEnv();
  process.env.WHATSAPP_PHONE_NUMBER_ID = 'phone_number_1';

  for (const [name, value] of Object.entries(values)) {
    if (value === undefined) {
      delete process.env[name];
      continue;
    }

    process.env[name] = value;
  }
}

afterEach(() => {
  restoreEnv();
});

describe('validateProject', () => {
  test('validates functions and materializes the workflow source', async () => {
    setEnv({});
    const result = await validateProject(rootDir);

    expect(result.functionCount).toBe(2);
    expect(result.workflowNodeCount).toBe(2);
    expect(result.workflowEdgeCount).toBe(1);
    expect(result.workflowTriggerCount).toBe(1);
  });

  test('allows a public repository with an empty pat', async () => {
    setEnv({
      AGENT_SANDBOX_GITHUB_REPO_URL: 'https://github.com/gokapso/whatsapp-support-agent',
      AGENT_SANDBOX_GITHUB_PAT: ''
    });

    await expect(validateProject(rootDir)).resolves.toMatchObject({
      functionCount: 2,
      workflowNodeCount: 2,
      workflowEdgeCount: 1
    });
  });

  test('rejects a pat without a repository url', async () => {
    setEnv({
      AGENT_SANDBOX_GITHUB_PAT: 'ghp_lonely_token'
    });

    await expect(validateProject(rootDir)).rejects.toThrow(
      'AGENT_SANDBOX_GITHUB_PAT requires AGENT_SANDBOX_GITHUB_REPO_URL.'
    );
  });

  test('rejects sandbox enabled without a repository url', async () => {
    setEnv({
      AGENT_SANDBOX_ENABLED: 'true'
    });

    await expect(validateProject(rootDir)).rejects.toThrow(
      'AGENT_SANDBOX_ENABLED=true requires AGENT_SANDBOX_GITHUB_REPO_URL.'
    );
  });

  test('rejects an invalid sandbox network mode', async () => {
    setEnv({
      AGENT_SANDBOX_GITHUB_REPO_URL: 'https://github.com/gokapso/whatsapp-support-agent',
      AGENT_SANDBOX_NETWORK_MODE: 'restricted'
    });

    await expect(validateProject(rootDir)).rejects.toThrow(
      'AGENT_SANDBOX_NETWORK_MODE must be "allow_all" or "allow_list".'
    );
  });
});

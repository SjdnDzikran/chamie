import { afterEach, describe, expect, test } from 'bun:test';
import { resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

import { buildRepositoryWorkspaceSlug } from '../../src/lib/agent-sandbox.js';

const rootDir = resolve(fileURLToPath(new URL('..', import.meta.url)), '..');
const ENV_NAMES = [
  'AGENT_SANDBOX_ALLOWED_OUTBOUND_HOSTS',
  'AGENT_SANDBOX_ENABLED',
  'AGENT_SANDBOX_GITHUB_PAT',
  'AGENT_SANDBOX_GITHUB_REPO_BRANCH',
  'AGENT_SANDBOX_GITHUB_REPO_URL',
  'AGENT_SANDBOX_NETWORK_MODE',
  'PROVIDER_MODEL_NAME',
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

async function loadWorkflow(): Promise<any> {
  const url = pathToFileURL(resolve(rootDir, 'workflows/whats-app-support-agent-example/workflow.ts'));
  url.searchParams.set('test', `${Date.now()}-${Math.random()}`);
  const module = await import(url.href);
  return module.buildWorkflow();
}

async function workflowSource(): Promise<any> {
  const workflow = await loadWorkflow();
  return workflow.toSourceFiles();
}

function agentNodeConfig(source: any): Record<string, any> {
  return source.definition.nodes.find((node: any) => node.id === 'agent')?.data.config;
}

afterEach(() => {
  restoreEnv();
});

describe('workflow source', () => {
  test('builds stable sandbox repository mount paths', () => {
    expect(buildRepositoryWorkspaceSlug('GoKapso', 'WhatsApp-Support-Agent', 'main')).toBe(
      'gokapso-whatsapp-support-agent'
    );
    expect(buildRepositoryWorkspaceSlug('GoKapso', 'WhatsApp-Support-Agent', '.release/v1+preview.')).toBe(
      'gokapso-whatsapp-support-agent@release--v1-preview'
    );
  });

  test('uses provider model names, function slugs, and trigger phone number ids', async () => {
    setEnv({
      PROVIDER_MODEL_NAME: 'provider-model-test'
    });

    const source = await workflowSource();
    const config = agentNodeConfig(source);

    expect(config.provider_model_name).toBe('provider-model-test');
    expect(config.flow_agent_function_tools[0]?.function_slug).toBe('whatsapp-support-agent-ask-team-question');
    expect(config.sandbox_enabled).toBeUndefined();
    expect(config.flow_agent_resources).toBeUndefined();
    expect(source.metadata.triggers[0]).toMatchObject({
      phoneNumberId: 'phone_number_1',
      triggerType: 'inbound_message'
    });
  });

  test('injects sandbox config for a public GitHub repository without a pat', async () => {
    setEnv({
      AGENT_SANDBOX_GITHUB_REPO_URL: 'https://github.com/GoKapso/WhatsApp-Support-Agent',
      AGENT_SANDBOX_ALLOWED_OUTBOUND_HOSTS: 'docs.example.com,\nAPI.EXAMPLE.COM, docs.example.com'
    });

    const source = await workflowSource();
    const config = agentNodeConfig(source);
    const resource = config.flow_agent_resources?.[0];

    expect(config.sandbox_enabled).toBe(true);
    expect(config.sandbox_network_mode).toBe('allow_list');
    expect(config.sandbox_allowed_outbound_hosts).toEqual([
      'docs.example.com',
      'api.example.com'
    ]);
    expect(resource).toEqual({
      resource_type: 'github_repository',
      repo_url: 'https://github.com/gokapso/whatsapp-support-agent',
      branch: 'main'
    });
    expect(config.system_prompt).toContain('/workspace/repos/gokapso-whatsapp-support-agent');
  });

  test('injects sandbox config for a private repository and a non-main branch', async () => {
    setEnv({
      AGENT_SANDBOX_GITHUB_REPO_URL: 'git@github.com:GoKapso/WhatsApp-Support-Agent.git',
      AGENT_SANDBOX_GITHUB_REPO_BRANCH: 'feature/support-handoff',
      AGENT_SANDBOX_GITHUB_PAT: 'ghp_private_token',
      AGENT_SANDBOX_NETWORK_MODE: 'allow_all'
    });

    const source = await workflowSource();
    const config = agentNodeConfig(source);
    const resource = config.flow_agent_resources?.[0];

    expect(config.sandbox_enabled).toBe(true);
    expect(config.sandbox_network_mode).toBe('allow_all');
    expect(config.sandbox_allowed_outbound_hosts).toEqual([]);
    expect(resource).toEqual({
      resource_type: 'github_repository',
      repo_url: 'https://github.com/gokapso/whatsapp-support-agent',
      branch: 'feature/support-handoff',
      pat: 'ghp_private_token'
    });
    expect(config.system_prompt).toContain(
      '/workspace/repos/gokapso-whatsapp-support-agent@feature--support-handoff'
    );
  });

  test('can explicitly disable sandbox without a repository url', async () => {
    setEnv({
      AGENT_SANDBOX_ENABLED: 'false'
    });

    const source = await workflowSource();
    const config = agentNodeConfig(source);

    expect(config.sandbox_enabled).toBe(false);
    expect(config.flow_agent_resources).toBeUndefined();
  });
});

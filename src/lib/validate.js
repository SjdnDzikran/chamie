import { access, readFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { pathToFileURL } from 'node:url';

import { FUNCTION_SLUGS } from './constants.js';
import { loadLocalEnv } from './env.js';

const WORKFLOW_SOURCE = 'workflows/whats-app-support-agent-example/workflow.ts';

async function readRequiredFile(path) {
  await access(path);
  return readFile(path, 'utf8');
}

async function validateFunctionSource(rootDir, slug) {
  const functionDir = resolve(rootDir, 'functions', slug);
  const metadata = await readRequiredFile(resolve(functionDir, 'function.yaml'));
  const code = await readRequiredFile(resolve(functionDir, 'index.js'));

  if (!metadata.includes(`slug: ${slug}`)) {
    throw new Error(`functions/${slug}/function.yaml must declare slug: ${slug}.`);
  }

  if (!code.includes('async function handler(')) {
    throw new Error(`functions/${slug}/index.js must define async function handler(request, env).`);
  }

  if (/\bexport\s+default\b|\bmodule\.exports\b/.test(code)) {
    throw new Error(`functions/${slug}/index.js must not use export default or module.exports.`);
  }

  return { slug };
}

async function importWorkflow(rootDir) {
  const workflowUrl = pathToFileURL(resolve(rootDir, WORKFLOW_SOURCE));
  workflowUrl.searchParams.set('validate', `${Date.now()}-${Math.random()}`);
  const module = await import(workflowUrl.href);
  return module.buildWorkflow ? module.buildWorkflow() : module.default;
}

export async function validateProject(rootDir) {
  loadLocalEnv(rootDir);

  const functionSlugs = Object.values(FUNCTION_SLUGS);
  const functions = await Promise.all(
    functionSlugs.map((slug) => validateFunctionSource(rootDir, slug)),
  );
  const workflow = await importWorkflow(rootDir);
  const validation = workflow.validate();

  if (validation.errors.length > 0) {
    throw new Error(validation.errors.map((error) => error.message).join('\n'));
  }

  const source = workflow.toSourceFiles();
  return {
    functionCount: functions.length,
    workflowEdgeCount: source.definition.edges.length,
    workflowNodeCount: source.definition.nodes.length,
    workflowTriggerCount: source.metadata.triggers.length,
    workflowWarnings: validation.warnings.length,
  };
}

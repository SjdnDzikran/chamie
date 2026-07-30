import { START, Workflow } from '@kapso/workflows';

import {
  DEFAULT_PROVIDER_MODEL_NAME,
  FUNCTION_DESCRIPTIONS,
  FUNCTION_SLUGS,
  WORKFLOW_NAME,
} from '../../src/lib/constants.js';
import { getOptionalEnv, getRequiredEnv, loadLocalEnv } from '../../src/lib/env.js';
import { resolveAgentSandboxTemplatePatchFromEnv } from '../../src/lib/agent-sandbox.js';

const BASE_SYSTEM_PROMPT =
  'You are the WhatsApp support agent for this business. Answer directly when you are confident and have enough context. If you are not confident, use ask_team_question exactly once for the open customer issue, tell the customer you are checking with the team, then call enter_waiting. When the workflow resumes with <external_input>, treat it as internal team guidance, not as a customer message. Use that guidance to answer the customer clearly. If the issue still cannot be resolved safely, call handoff_to_human. Call complete_task after the customer-facing answer is sent and the issue is resolved.';

loadLocalEnv(process.cwd());

// The CLI reads the default export below. This factory is only here so tests can
// rebuild the workflow after changing env vars.
export function buildWorkflow(): Workflow {
  // Advanced scaffold option: adds raw agent sandbox fields only when
  // AGENT_SANDBOX_* env vars are set.
  const sandboxPatch = resolveAgentSandboxTemplatePatchFromEnv();
  const workflow = new Workflow('whats-app-support-agent-example', {
    name: WORKFLOW_NAME,
    status: 'active',
  });

  workflow.addNode(START, {
    position: { x: 220, y: 120 },
  });

  workflow.addTrigger({
    phoneNumberId: getRequiredEnv('WHATSAPP_PHONE_NUMBER_ID'),
    type: 'inbound_message',
  });

  workflow.addNode(
    'agent',
    {
      enabledDefaultTools: [
        'send_notification_to_user',
        'get_execution_metadata',
        'get_whatsapp_context',
        'get_current_datetime',
        'enter_waiting',
        'complete_task',
        'handoff_to_human',
      ],
      functionTools: [
        {
          description: FUNCTION_DESCRIPTIONS.askTeamQuestion,
          functionSlug: FUNCTION_SLUGS.askTeamQuestion,
          inputSchema: {
            additionalProperties: false,
            properties: {
              question: {
                description: 'The exact support question to send to the internal team.',
                type: 'string',
              },
              summary: {
                description: 'Optional short customer/context summary to help the team answer quickly.',
                type: 'string',
              },
              title: {
                description: 'Optional short title for the internal support request.',
                type: 'string',
              },
            },
            required: ['question'],
            type: 'object',
          },
          name: 'ask_team_question',
        },
      ],
      maxIterations: 80,
      maxTokens: 8192,
      providerModel: getOptionalEnv('PROVIDER_MODEL_NAME') ?? DEFAULT_PROVIDER_MODEL_NAME,
      rawConfig: sandboxPatch?.configPatch,
      reasoningEffort: 'medium',
      systemPrompt: `${BASE_SYSTEM_PROMPT}${sandboxPatch?.promptSuffix ?? ''}`,
      temperature: 0.2,
      type: 'agent',
    },
    {
      position: { x: 220, y: 320 },
    },
  );

  workflow.addEdge(START, 'agent');

  return workflow;
}

export default buildWorkflow();

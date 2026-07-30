export const KAPSO_API_BASE_URL = 'https://api.kapso.ai';
export const DEFAULT_PROVIDER_MODEL_NAME = 'openai/gpt-5.4';

export const WORKFLOW_NAME = 'WhatsApp Support Agent';
export const WORKFLOW_DESCRIPTION =
  'Open-source Kapso example: WhatsApp support agent with Slack escalation and workflow resume.';

export const FUNCTION_SLUGS = {
  askTeamQuestion: 'whatsapp-support-agent-ask-team-question',
  slackEvents: 'whatsapp-support-agent-slack-events',
};

export const FUNCTION_DESCRIPTIONS = {
  askTeamQuestion:
    'Agent function tool that opens a Slack support thread and stores the pending question in KV.',
  slackEvents:
    'Public Slack webhook that verifies signatures, aggregates thread answers, and resumes the waiting Kapso workflow.',
};

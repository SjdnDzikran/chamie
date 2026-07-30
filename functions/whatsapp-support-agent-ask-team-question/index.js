const SLACK_API_BASE_URL = 'https://slack.com/api';
const QUESTION_PREFIX = 'support-question:';
const THREAD_PREFIX = 'support-thread:';
const OPEN_QUESTION_PREFIX = 'support-open-question:';

function jsonResponse(body, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: {
      'Content-Type': 'application/json',
    },
  });
}

function requireEnv(value, name) {
  if (!value) {
    throw new Error(`Missing required runtime env: ${name}`);
  }
  return value;
}

function asTrimmedString(value) {
  if (typeof value !== 'string') {
    return null;
  }

  const trimmed = value.trim();
  return trimmed.length > 0 ? trimmed : null;
}

function questionKey(questionId) {
  return `${QUESTION_PREFIX}${questionId}`;
}

function threadKey(channelId, threadTs) {
  return `${THREAD_PREFIX}${channelId}:${threadTs}`;
}

function openQuestionKey(workflowExecutionId) {
  return `${OPEN_QUESTION_PREFIX}${workflowExecutionId}`;
}

async function loadQuestion(kv, questionId) {
  const raw = await kv.get(questionKey(questionId));
  return raw ? JSON.parse(raw) : null;
}

async function saveQuestion(kv, question) {
  await kv.put(questionKey(question.id), JSON.stringify(question));
}

async function getOpenQuestionIdByExecution(kv, workflowExecutionId) {
  return kv.get(openQuestionKey(workflowExecutionId));
}

async function setOpenQuestionForExecution(kv, workflowExecutionId, questionId) {
  await kv.put(openQuestionKey(workflowExecutionId), questionId);
}

async function clearOpenQuestionForExecution(kv, workflowExecutionId) {
  await kv.delete(openQuestionKey(workflowExecutionId));
}

async function setThreadQuestionMapping(kv, channelId, threadTs, questionId) {
  await kv.put(threadKey(channelId, threadTs), questionId);
}

async function slackRequest(token, path, init = {}) {
  const response = await fetch(`${SLACK_API_BASE_URL}${path}`, {
    ...init,
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json; charset=utf-8',
      ...(init.headers ?? {}),
    },
  });

  const body = await response.json();
  if (!response.ok || !body.ok) {
    throw new Error(`Slack API request failed for ${path}: ${body.error ?? response.statusText}`);
  }

  return body;
}

async function postQuestionToSlack(token, channelId, messageText) {
  const body = await slackRequest(token, '/chat.postMessage', {
    method: 'POST',
    body: JSON.stringify({
      channel: channelId,
      text: messageText,
    }),
  });

  return {
    channel: body.channel,
    ts: body.ts,
  };
}

function formatSupportQuestionMessage(question) {
  const lines = [`*${question.title}*`, '', question.questionText];

  if (question.summary) {
    lines.push('', `Summary: ${question.summary}`);
  }

  if (question.metadata.customer_phone_number) {
    lines.push('', `Customer phone: ${String(question.metadata.customer_phone_number)}`);
  }

  if (question.metadata.customer_profile_name) {
    lines.push(`Customer profile: ${String(question.metadata.customer_profile_name)}`);
  }

  lines.push('', `Workflow execution: ${question.workflowExecutionId}`);
  lines.push('Reply in this thread and send `done` when the final answer is ready.');

  return lines.join('\n');
}

function resolveWorkflowExecutionId(body) {
  const system = body.execution_context?.system ?? {};
  return asTrimmedString(system.workflow_execution_id) ?? asTrimmedString(system.flow_execution_id);
}

async function handler(request, env) {
  const body = await request.json();
  const input = body.input ?? {};

  const questionText = asTrimmedString(input.question);
  if (!questionText) {
    return jsonResponse({ error: 'question is required' }, 400);
  }

  const workflowExecutionId = resolveWorkflowExecutionId(body);
  if (!workflowExecutionId) {
    return jsonResponse(
      { error: 'workflow execution id is missing from execution_context.system' },
      422,
    );
  }

  const existingQuestionId = await getOpenQuestionIdByExecution(env.KV, workflowExecutionId);
  if (existingQuestionId) {
    const existingQuestion = await loadQuestion(env.KV, existingQuestionId);
    if (existingQuestion?.status === 'pending') {
      return jsonResponse({
        question_id: existingQuestion.id,
        reused_existing_question: true,
        slack_channel_id: existingQuestion.slackChannelId,
        slack_message_ts: existingQuestion.slackMessageTs,
        status: existingQuestion.status,
      });
    }

    await clearOpenQuestionForExecution(env.KV, workflowExecutionId);
  }

  const title = asTrimmedString(input.title) ?? 'Support Question';
  const summary = asTrimmedString(input.summary);
  const now = new Date().toISOString();
  const questionId = crypto.randomUUID();
  const executionContext = body.execution_context ?? {};
  const runtimeContext = executionContext.context ?? {};
  const conversation = body.whatsapp_context?.conversation ?? {};

  const question = {
    answerText: null,
    answeredAt: null,
    conversationId:
      asTrimmedString(conversation.id) ?? asTrimmedString(runtimeContext.conversation_id) ?? null,
    createdAt: now,
    id: questionId,
    metadata: {
      customer_phone_number: asTrimmedString(runtimeContext.phone_number),
      customer_profile_name: asTrimmedString(runtimeContext.contact?.profile_name),
      flow_id: asTrimmedString(executionContext.system?.flow_id),
      flow_name: asTrimmedString(executionContext.system?.flow_name),
      flow_step_id: asTrimmedString(body.flow_info?.step_id),
    },
    questionText,
    slackChannelId: '',
    slackMessageTs: '',
    status: 'pending',
    summary,
    title,
    workflowExecutionId,
  };

  const slackChannelId = requireEnv(env.SLACK_CHANNEL_ID, 'SLACK_CHANNEL_ID');
  const slackBotToken = requireEnv(env.SLACK_BOT_TOKEN, 'SLACK_BOT_TOKEN');
  const slackMessage = await postQuestionToSlack(
    slackBotToken,
    slackChannelId,
    formatSupportQuestionMessage(question),
  );

  question.slackChannelId = slackMessage.channel;
  question.slackMessageTs = slackMessage.ts;

  await saveQuestion(env.KV, question);
  await setThreadQuestionMapping(env.KV, slackMessage.channel, slackMessage.ts, question.id);
  await setOpenQuestionForExecution(env.KV, workflowExecutionId, question.id);

  return jsonResponse({
    question_id: question.id,
    slack_channel_id: question.slackChannelId,
    slack_message_ts: question.slackMessageTs,
    status: question.status,
  });
}

if (globalThis.Bun) {
  globalThis.__supportAgentAskTeamQuestion = {
    clearOpenQuestionForExecution,
    formatSupportQuestionMessage,
    getOpenQuestionIdByExecution,
    handler,
    loadQuestion,
    saveQuestion,
    setOpenQuestionForExecution,
    setThreadQuestionMapping,
  };
}

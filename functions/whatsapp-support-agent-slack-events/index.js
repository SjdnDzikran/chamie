const KAPSO_API_BASE_URL = 'https://api.kapso.ai';
const SLACK_API_BASE_URL = 'https://slack.com/api';
const QUESTION_PREFIX = 'support-question:';
const THREAD_PREFIX = 'support-thread:';
const OPEN_QUESTION_PREFIX = 'support-open-question:';
const MAX_REQUEST_AGE_SECONDS = 60 * 5;
const encoder = new TextEncoder();

class KapsoResumeError extends Error {
  constructor(message, status, details) {
    super(message);
    this.name = 'KapsoResumeError';
    this.status = status;
    this.details = details;
  }
}

function jsonResponse(body, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: {
      'Content-Type': 'application/json',
    },
  });
}

function textResponse(body, status = 200) {
  return new Response(body, {
    status,
    headers: {
      'Content-Type': 'text/plain; charset=utf-8',
    },
  });
}

function requireEnv(value, name) {
  if (!value) {
    throw new Error(`Missing required runtime env: ${name}`);
  }
  return value;
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

async function getQuestionIdByThread(kv, channelId, threadTs) {
  return kv.get(threadKey(channelId, threadTs));
}

async function setThreadQuestionMapping(kv, channelId, threadTs, questionId) {
  await kv.put(threadKey(channelId, threadTs), questionId);
}

async function setOpenQuestionForExecution(kv, workflowExecutionId, questionId) {
  await kv.put(openQuestionKey(workflowExecutionId), questionId);
}

async function clearOpenQuestionForExecution(kv, workflowExecutionId) {
  await kv.delete(openQuestionKey(workflowExecutionId));
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

async function fetchThreadReplies(token, channelId, threadTs) {
  const query = new URLSearchParams({
    channel: channelId,
    ts: threadTs,
  });

  const body = await slackRequest(token, `/conversations.replies?${query.toString()}`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded',
    },
  });

  return body.messages ?? [];
}

function aggregateThreadAnswer(messages, parentTs) {
  const parts = messages
    .filter((message) => {
      if (!message.ts || message.ts === parentTs) {
        return false;
      }
      if (message.bot_id || message.subtype) {
        return false;
      }

      const text = message.text?.trim();
      if (!text) {
        return false;
      }

      return text.toLowerCase() !== 'done';
    })
    .map((message) => message.text.trim());

  return parts.join('\n\n');
}

function bytesToHex(bytes) {
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('');
}

function constantTimeEqual(left, right) {
  if (left.length !== right.length) {
    return false;
  }

  let mismatch = 0;
  for (let index = 0; index < left.length; index += 1) {
    mismatch |= left.charCodeAt(index) ^ right.charCodeAt(index);
  }

  return mismatch === 0;
}

async function createSlackSignature(signingSecret, timestamp, rawBody) {
  const key = await crypto.subtle.importKey(
    'raw',
    encoder.encode(signingSecret),
    { hash: 'SHA-256', name: 'HMAC' },
    false,
    ['sign'],
  );

  const signatureBody = encoder.encode(`v0:${timestamp}:${rawBody}`);
  const digest = await crypto.subtle.sign('HMAC', key, signatureBody);
  return `v0=${bytesToHex(new Uint8Array(digest))}`;
}

async function verifySlackSignature(input) {
  const { signingSecret, signature, timestamp, rawBody } = input;
  if (!signature || !timestamp) {
    return false;
  }

  const numericTimestamp = Number(timestamp);
  if (!Number.isFinite(numericTimestamp)) {
    return false;
  }

  const nowSeconds = Math.floor((input.now ?? Date.now()) / 1000);
  if (Math.abs(nowSeconds - numericTimestamp) > MAX_REQUEST_AGE_SECONDS) {
    return false;
  }

  const expected = await createSlackSignature(signingSecret, timestamp, rawBody);
  return constantTimeEqual(expected, signature);
}

async function verifySignedRequest(request, rawBody, signingSecret) {
  return verifySlackSignature({
    rawBody,
    signature: request.headers.get('x-slack-signature'),
    signingSecret,
    timestamp: request.headers.get('x-slack-request-timestamp'),
  });
}

function isDoneMessage(event, channelId) {
  if (!event || event.type !== 'message') {
    return false;
  }
  if (event.channel !== channelId) {
    return false;
  }
  if (event.bot_id || event.subtype) {
    return false;
  }
  if (!event.thread_ts || !event.ts || event.thread_ts === event.ts) {
    return false;
  }

  return event.text?.trim().toLowerCase() === 'done';
}

async function parseResponseBody(response) {
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

async function resumeWorkflowExecution(apiKey, workflowExecutionId, answer) {
  const response = await fetch(
    `${KAPSO_API_BASE_URL}/platform/v1/workflow_executions/${workflowExecutionId}/resume`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-API-Key': apiKey,
      },
      body: JSON.stringify({
        message: {
          data: answer,
          kind: 'payload',
        },
      }),
    },
  );

  if (!response.ok) {
    const details = await parseResponseBody(response);
    throw new KapsoResumeError(
      `Kapso resume failed with status ${response.status}`,
      response.status,
      details,
    );
  }
}

async function handler(request, env) {
  const rawBody = await request.text();
  const payload = rawBody ? JSON.parse(rawBody) : {};

  if (payload.type === 'url_verification') {
    const signingSecret = env.SLACK_SIGNING_SECRET;
    const hasSlackHeaders =
      request.headers.has('x-slack-signature') && request.headers.has('x-slack-request-timestamp');

    if (signingSecret && hasSlackHeaders) {
      const isValid = await verifySignedRequest(request, rawBody, signingSecret);
      if (!isValid) {
        return textResponse('Unauthorized', 401);
      }
    }

    return jsonResponse({ challenge: payload.challenge ?? '' });
  }

  const signingSecret = requireEnv(env.SLACK_SIGNING_SECRET, 'SLACK_SIGNING_SECRET');
  const isValid = await verifySignedRequest(request, rawBody, signingSecret);
  if (!isValid) {
    return textResponse('Unauthorized', 401);
  }

  if (payload.type !== 'event_callback') {
    return jsonResponse({ ok: true });
  }

  const slackChannelId = requireEnv(env.SLACK_CHANNEL_ID, 'SLACK_CHANNEL_ID');
  if (!isDoneMessage(payload.event, slackChannelId)) {
    return jsonResponse({ ok: true });
  }

  const questionId = await getQuestionIdByThread(env.KV, payload.event.channel, payload.event.thread_ts);
  if (!questionId) {
    return jsonResponse({ ignored: 'unknown_thread', ok: true });
  }

  const question = await loadQuestion(env.KV, questionId);
  if (!question || question.status === 'answered') {
    return jsonResponse({ ignored: 'question_already_answered', ok: true });
  }

  const slackBotToken = requireEnv(env.SLACK_BOT_TOKEN, 'SLACK_BOT_TOKEN');
  const replies = await fetchThreadReplies(slackBotToken, question.slackChannelId, question.slackMessageTs);
  const answer = aggregateThreadAnswer(replies, question.slackMessageTs);

  if (!answer) {
    return jsonResponse({ ignored: 'empty_thread_answer', ok: true });
  }

  const workflowApiKey = requireEnv(env.KAPSO_API_KEY, 'KAPSO_API_KEY');
  try {
    await resumeWorkflowExecution(workflowApiKey, question.workflowExecutionId, answer);
  } catch (error) {
    if (!(error instanceof KapsoResumeError) || ![404, 422].includes(error.status)) {
      throw error;
    }
  }

  question.answerText = answer;
  question.answeredAt = new Date().toISOString();
  question.status = 'answered';

  await saveQuestion(env.KV, question);
  await clearOpenQuestionForExecution(env.KV, question.workflowExecutionId);

  return jsonResponse({ ok: true, resumed: true });
}

if (globalThis.Bun) {
  globalThis.__supportAgentSlackEvents = {
    aggregateThreadAnswer,
    clearOpenQuestionForExecution,
    createSlackSignature,
    getQuestionIdByThread,
    handler,
    loadQuestion,
    saveQuestion,
    setOpenQuestionForExecution,
    setThreadQuestionMapping,
    verifySlackSignature,
  };
}

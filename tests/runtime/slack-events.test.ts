import { afterEach, describe, expect, test } from 'bun:test';

import '../../functions/whatsapp-support-agent-slack-events/index.js';
import { InMemoryKv } from '../support/in-memory-kv.ts';

const originalFetch = globalThis.fetch;
const slackEvents = (globalThis as any).__supportAgentSlackEvents;

afterEach(() => {
  globalThis.fetch = originalFetch;
});

function buildDoneRequest(rawBody: string, timestamp: string): Promise<Request> {
  return slackEvents.createSlackSignature('signing-secret', timestamp, rawBody).then(
    (signature) =>
      new Request('https://example.com', {
        method: 'POST',
        body: rawBody,
        headers: {
          'Content-Type': 'application/json',
          'x-slack-signature': signature,
          'x-slack-request-timestamp': timestamp
        }
      })
  );
}

describe('handleSlackEvents', () => {
  test('aggregates thread replies and resumes the workflow', async () => {
    let resumeCalled = false;
    globalThis.fetch = (async (input: RequestInfo | URL) => {
      const url = input.toString();
      if (url.includes('/conversations.replies')) {
        return new Response(
          JSON.stringify({
            ok: true,
            messages: [
              { ts: '111.222', text: 'parent' },
              { ts: '111.223', text: 'First answer' },
              { ts: '111.224', text: 'done' },
              { ts: '111.225', text: 'Second answer' }
            ]
          }),
          {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          }
        );
      }

      if (url.includes('/workflow_executions/')) {
        resumeCalled = true;
        return new Response(JSON.stringify({ data: { id: 'execution_1' } }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' }
        });
      }

      throw new Error(`Unexpected fetch: ${url}`);
    }) as typeof fetch;

    const kv = new InMemoryKv();
    const question = {
      id: 'question_1',
      status: 'pending',
      title: 'Support Question',
      summary: null,
      questionText: 'What is the policy?',
      workflowExecutionId: 'execution_1',
      conversationId: null,
      slackChannelId: 'C123',
      slackMessageTs: '111.222',
      answerText: null,
      createdAt: '2026-04-17T00:00:00.000Z',
      answeredAt: null,
      metadata: {}
    };

    await slackEvents.saveQuestion(kv, question);
    await slackEvents.setThreadQuestionMapping(kv, question.slackChannelId, question.slackMessageTs, question.id);
    await slackEvents.setOpenQuestionForExecution(kv, question.workflowExecutionId, question.id);

    const env = {
      KV: kv,
      KAPSO_API_KEY: 'kapso-key',
      SLACK_BOT_TOKEN: 'xoxb-test',
      SLACK_CHANNEL_ID: 'C123',
      SLACK_SIGNING_SECRET: 'signing-secret'
    };

    const rawBody = JSON.stringify({
      type: 'event_callback',
      event: {
        type: 'message',
        channel: 'C123',
        ts: '111.224',
        thread_ts: '111.222',
        text: 'done'
      }
    });
    const request = await buildDoneRequest(rawBody, String(Math.floor(Date.now() / 1000)));

    const response = await slackEvents.handler(request, env);
    const stored = await slackEvents.loadQuestion(kv, question.id);

    expect(response.status).toBe(200);
    expect(resumeCalled).toBe(true);
    expect(stored?.status).toBe('answered');
    expect(stored?.answerText).toBe('First answer\n\nSecond answer');
  });

  test('marks the question answered when the workflow is already no longer waiting', async () => {
    globalThis.fetch = (async (input: RequestInfo | URL) => {
      const url = input.toString();
      if (url.includes('/conversations.replies')) {
        return new Response(
          JSON.stringify({
            ok: true,
            messages: [
              { ts: '111.222', text: 'parent' },
              { ts: '111.223', text: 'Final internal answer' }
            ]
          }),
          {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          }
        );
      }

      if (url.includes('/workflow_executions/')) {
        return new Response(JSON.stringify({ error: 'Execution is not waiting' }), {
          status: 422,
          headers: { 'Content-Type': 'application/json' }
        });
      }

      throw new Error(`Unexpected fetch: ${url}`);
    }) as typeof fetch;

    const kv = new InMemoryKv();
    const question = {
      id: 'question_2',
      status: 'pending',
      title: 'Support Question',
      summary: null,
      questionText: 'What is the policy?',
      workflowExecutionId: 'execution_2',
      conversationId: null,
      slackChannelId: 'C123',
      slackMessageTs: '111.222',
      answerText: null,
      createdAt: '2026-04-17T00:00:00.000Z',
      answeredAt: null,
      metadata: {}
    };

    await slackEvents.saveQuestion(kv, question);
    await slackEvents.setThreadQuestionMapping(kv, question.slackChannelId, question.slackMessageTs, question.id);
    await slackEvents.setOpenQuestionForExecution(kv, question.workflowExecutionId, question.id);

    const env = {
      KV: kv,
      KAPSO_API_KEY: 'kapso-key',
      SLACK_BOT_TOKEN: 'xoxb-test',
      SLACK_CHANNEL_ID: 'C123',
      SLACK_SIGNING_SECRET: 'signing-secret'
    };

    const rawBody = JSON.stringify({
      type: 'event_callback',
      event: {
        type: 'message',
        channel: 'C123',
        ts: '111.224',
        thread_ts: '111.222',
        text: 'done'
      }
    });
    const request = await buildDoneRequest(rawBody, String(Math.floor(Date.now() / 1000)));

    await slackEvents.handler(request, env);
    const stored = await slackEvents.loadQuestion(kv, question.id);

    expect(stored?.status).toBe('answered');
    expect(stored?.answerText).toBe('Final internal answer');
  });
});

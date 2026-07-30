import { afterEach, describe, expect, test } from 'bun:test';

import '../../functions/whatsapp-support-agent-ask-team-question/index.js';
import { InMemoryKv } from '../support/in-memory-kv.ts';

const originalFetch = globalThis.fetch;
const askTeamQuestion = (globalThis as any).__supportAgentAskTeamQuestion;

afterEach(() => {
  globalThis.fetch = originalFetch;
});

describe('handleAskTeamQuestion', () => {
  test('creates a Slack thread and reuses it for the same open execution', async () => {
    let postCount = 0;
    globalThis.fetch = (async () => {
      postCount += 1;
      return new Response(
        JSON.stringify({
          ok: true,
          channel: 'C123',
          ts: '111.222'
        }),
        {
          status: 200,
          headers: { 'Content-Type': 'application/json' }
        }
      );
    }) as typeof fetch;

    const kv = new InMemoryKv();
    const env = {
      KV: kv,
      SLACK_BOT_TOKEN: 'xoxb-test',
      SLACK_CHANNEL_ID: 'C123'
    };

    const requestBody = {
      input: {
        question: 'Can we override the normal refund policy?',
        title: 'Refund exception',
        summary: 'VIP customer'
      },
      execution_context: {
        system: {
          workflow_execution_id: 'execution_1'
        },
        context: {
          phone_number: '+15550000000',
          contact: {
            profile_name: 'Alicia'
          }
        }
      },
      whatsapp_context: {
        conversation: {
          id: 'conversation_1'
        }
      }
    };

    const firstResponse = await askTeamQuestion.handler(
      new Request('https://example.com', {
        method: 'POST',
        body: JSON.stringify(requestBody)
      }),
      env
    );
    const secondResponse = await askTeamQuestion.handler(
      new Request('https://example.com', {
        method: 'POST',
        body: JSON.stringify(requestBody)
      }),
      env
    );

    expect(firstResponse.status).toBe(200);
    expect(secondResponse.status).toBe(200);
    expect(postCount).toBe(1);

    const firstBody = (await firstResponse.json()) as { question_id: string };
    const storedQuestion = await askTeamQuestion.loadQuestion(kv, firstBody.question_id);
    expect(storedQuestion?.workflowExecutionId).toBe('execution_1');

    const secondBody = (await secondResponse.json()) as { reused_existing_question?: boolean };
    expect(secondBody.reused_existing_question).toBe(true);
  });
});

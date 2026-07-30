import { describe, expect, test } from 'bun:test';

import '../../functions/whatsapp-support-agent-slack-events/index.js';

const slackEvents = (globalThis as any).__supportAgentSlackEvents;

describe('slack-signature', () => {
  test('verifies a valid Slack signature', async () => {
    const rawBody = JSON.stringify({ type: 'event_callback' });
    const timestamp = String(Math.floor(Date.now() / 1000));
    const signature = await slackEvents.createSlackSignature('secret', timestamp, rawBody);

    const isValid = await slackEvents.verifySlackSignature({
      signingSecret: 'secret',
      signature,
      timestamp,
      rawBody
    });

    expect(isValid).toBe(true);
  });

  test('rejects a stale request', async () => {
    const rawBody = JSON.stringify({ type: 'event_callback' });
    const timestamp = '1000';
    const signature = await slackEvents.createSlackSignature('secret', timestamp, rawBody);

    const isValid = await slackEvents.verifySlackSignature({
      signingSecret: 'secret',
      signature,
      timestamp,
      rawBody,
      now: 1_000_000_000
    });

    expect(isValid).toBe(false);
  });
});

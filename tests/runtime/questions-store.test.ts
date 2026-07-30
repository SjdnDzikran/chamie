import { describe, expect, test } from 'bun:test';

import '../../functions/whatsapp-support-agent-slack-events/index.js';
import { InMemoryKv } from '../support/in-memory-kv.ts';

const slackEvents = (globalThis as any).__supportAgentSlackEvents;

describe('questions-store', () => {
  test('stores and resolves question, thread, and open-execution records', async () => {
    const kv = new InMemoryKv();
    const question = {
      id: 'question_1',
      status: 'pending',
      title: 'Support Question',
      summary: 'Summary',
      questionText: 'How do refunds work?',
      workflowExecutionId: 'execution_1',
      conversationId: 'conversation_1',
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

    expect(await slackEvents.loadQuestion(kv, question.id)).toEqual(question);
    expect(await slackEvents.getQuestionIdByThread(kv, question.slackChannelId, question.slackMessageTs)).toBe(
      question.id
    );

    await slackEvents.clearOpenQuestionForExecution(kv, question.workflowExecutionId);
    expect(await kv.get(`support-open-question:${question.workflowExecutionId}`)).toBeNull();
  });
});

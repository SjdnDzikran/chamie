import { describe, expect, test } from 'bun:test';

import '../../functions/whatsapp-support-agent-ask-team-question/index.js';
import '../../functions/whatsapp-support-agent-slack-events/index.js';

const askTeamQuestion = (globalThis as any).__supportAgentAskTeamQuestion;
const slackEvents = (globalThis as any).__supportAgentSlackEvents;

describe('slack-api helpers', () => {
  test('aggregates only relevant thread replies', () => {
    const answer = slackEvents.aggregateThreadAnswer(
      [
        { ts: '1.0', text: 'parent' },
        { ts: '1.1', text: 'First answer' },
        { ts: '1.2', text: 'done' },
        { ts: '1.3', bot_id: 'B123', text: 'bot reply' },
        { ts: '1.4', subtype: 'message_changed', text: 'edited' },
        { ts: '1.5', text: 'Second answer' }
      ],
      '1.0'
    );

    expect(answer).toBe('First answer\n\nSecond answer');
  });

  test('formats the support question message with core context', () => {
    const question = {
      id: 'question_1',
      status: 'pending',
      title: 'Refund Policy',
      summary: 'VIP customer asking about refunds',
      questionText: 'Can we make an exception for a late refund request?',
      workflowExecutionId: 'execution_1',
      conversationId: 'conversation_1',
      slackChannelId: 'C123',
      slackMessageTs: '111.222',
      answerText: null,
      createdAt: '2026-04-17T00:00:00.000Z',
      answeredAt: null,
      metadata: {
        customer_phone_number: '+15555555555',
        customer_profile_name: 'Alicia'
      }
    };

    const message = askTeamQuestion.formatSupportQuestionMessage(question);

    expect(message).toContain('*Refund Policy*');
    expect(message).toContain('Customer phone: +15555555555');
    expect(message).toContain('Workflow execution: execution_1');
  });
});

import { createMessageStore } from '../useMessageStore'

test('merges consecutive text_delta events with same blockIndex', () => {
  const store = createMessageStore()
  store.getState().handleEvent({
    type: 'text_delta',
    sessionId: 's1',
    time: '2026-06-08T00:00:00Z',
    blockIndex: 0,
    text: 'hel'
  } as any, 's1')
  store.getState().handleEvent({
    type: 'text_delta',
    sessionId: 's1',
    time: '2026-06-08T00:00:01Z',
    blockIndex: 0,
    text: 'lo'
  } as any, 's1')

  const messages = store.getState().messages
  expect(messages).toHaveLength(1)
  expect((messages[0] as any).text).toBe('hello')
})

test('replaces trailing text_delta with text and preserves thinking', () => {
  const store = createMessageStore()
  store.getState().handleEvent({
    type: 'text_delta',
    sessionId: 's1',
    time: '2026-06-08T00:00:00Z',
    blockIndex: 0,
    text: 'hel',
    thinking: 'thinking...'
  } as any, 's1')
  store.getState().handleEvent({
    type: 'text',
    sessionId: 's1',
    time: '2026-06-08T00:00:02Z',
    text: 'hello'
  } as any, 's1')

  const messages = store.getState().messages
  expect(messages).toHaveLength(1)
  expect((messages[0] as any).text).toBe('hello')
  expect((messages[0] as any).thinking).toBe('thinking...')
})

test('deduplicates permission prompts by toolName', () => {
  const store = createMessageStore()
  store.getState().handleEvent({
    type: 'permission_request',
    sessionId: 's1',
    time: '2026-06-08T00:00:00Z',
    toolName: 'Bash',
    message: 'first'
  } as any, 's1')
  store.getState().handleEvent({
    type: 'permission_request',
    sessionId: 's1',
    time: '2026-06-08T00:00:01Z',
    toolName: 'Bash',
    message: 'second'
  } as any, 's1')

  const messages = store.getState().messages
  expect(messages).toHaveLength(1)
  expect((messages[0] as any).message).toBe('second')
})

test('filters internal protocol events from message list', () => {
  const store = createMessageStore()

  store.getState().handleEvent({
    type: 'thinking_start', sessionId: 's1', time: '2026-06-08T00:00:00Z'
  } as any, 's1')
  store.getState().handleEvent({
    type: 'text', sessionId: 's1', time: '2026-06-08T00:00:01Z', text: 'hello'
  } as any, 's1')
  store.getState().handleEvent({
    type: 'turn_end', sessionId: 's1', time: '2026-06-08T00:00:02Z', text: '', message: ''
  } as any, 's1')

  const { messages, thinking, turnActive } = store.getState()
  expect(messages).toHaveLength(1)
  expect((messages[0] as any).text).toBe('hello')
  expect(thinking).toBe(false)
  expect(turnActive).toBe(false)
})

test('ignores hidden events like context_window and plan_mode', () => {
  const store = createMessageStore()

  store.getState().handleEvent({
    type: 'context_window', sessionId: 's1', time: '2026-06-08T00:00:00Z', toolInput: {}
  } as any, 's1')
  store.getState().handleEvent({
    type: 'plan_mode', sessionId: 's1', time: '2026-06-08T00:00:01Z', toolInput: {}
  } as any, 's1')

  expect(store.getState().messages).toHaveLength(0)
})

test('context_window 事件解析到独立状态字段，不进入 messages', () => {
  const store = createMessageStore()

  store.getState().handleEvent({
    type: 'context_window',
    sessionId: 's1',
    time: '2026-06-08T00:00:00Z',
    toolInput: { used_tokens: 87000, total_tokens: 200000 }
  } as any, 's1')

  expect(store.getState().messages).toHaveLength(0)
  expect(store.getState().contextWindow).not.toBeNull()
  expect(store.getState().contextWindow?.used).toBe(87000)
  expect(store.getState().contextWindow?.total).toBe(200000)
})

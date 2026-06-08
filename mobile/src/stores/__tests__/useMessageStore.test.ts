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

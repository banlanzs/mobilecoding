import { SyncService } from '../SyncService'

test('syncSession imports server messages and updates sync state', async () => {
  const sessions = { insert: jest.fn() }
  const messages = { insert: jest.fn() }
  const syncState = {
    findBySessionId: jest.fn().mockReturnValue({ sessionId: 's1', lastSeq: 0, updatedAt: '2026-06-08T00:00:00Z' }),
    upsert: jest.fn()
  }
  const rest = {
    get: jest.fn().mockResolvedValue([
      {
        id: 'm1',
        session_id: 's1',
        type: 'text',
        content: 'Hello',
        seq: 1,
        created_at: '2026-06-08T00:01:00Z'
      }
    ])
  }

  const service = new SyncService(sessions as any, messages as any, syncState as any, rest as any)
  const count = await service.syncSession('s1')

  expect(count).toBe(1)
  expect(messages.insert).toHaveBeenCalledWith({
    id: 'm1',
    sessionId: 's1',
    type: 'text',
    content: 'Hello',
    seq: 1,
    createdAt: '2026-06-08T00:01:00Z'
  })
  expect(syncState.upsert).toHaveBeenCalled()
})

test('handleRealtimeEvent inserts only newer seq events', () => {
  const sessions = { insert: jest.fn() }
  const messages = { insert: jest.fn() }
  const syncState = {
    findBySessionId: jest.fn().mockReturnValue({ sessionId: 's1', lastSeq: 5, updatedAt: '2026-06-08T00:00:00Z' }),
    upsert: jest.fn()
  }
  const rest = { get: jest.fn() }

  const service = new SyncService(sessions as any, messages as any, syncState as any, rest as any)

  service.handleRealtimeEvent({
    type: 'text',
    sessionId: 's1',
    messageId: 'm6',
    text: 'Hello',
    seq: 6,
    time: '2026-06-08T00:06:00Z'
  } as any)

  expect(messages.insert).toHaveBeenCalled()
  expect(syncState.upsert).toHaveBeenCalled()
})

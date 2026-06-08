import { MessageRepository } from '../MessageRepository'
import type { Database } from '../Database'

function createMockDb(): Database {
  const store = new Map<string, Record<string, unknown>[]>()
  return {
    execute(sql: string, params?: unknown[]) {
      if (sql.includes('INSERT OR REPLACE INTO messages')) {
        const [id, session_id, type, content, seq, created_at] = params as unknown[]
        const rows = store.get('messages') || []
        const row = { id, session_id, type, content, seq, created_at }
        const filtered = rows.filter(r => r.id !== id)
        filtered.push(row)
        store.set('messages', filtered)
        return { rows: { _array: [row] } }
      }
      if (sql.includes('SELECT * FROM messages WHERE session_id = ? AND seq > ?')) {
        const [sessionId, seq] = params as unknown[]
        const rows = store.get('messages') || []
        return { rows: { _array: rows.filter(r => r.session_id === sessionId && Number(r.seq) > Number(seq)) } }
      }
      if (sql.includes('SELECT * FROM messages WHERE session_id = ? ORDER BY')) {
        const [sessionId] = params as unknown[]
        const rows = store.get('messages') || []
        return { rows: { _array: rows.filter(r => r.session_id === sessionId) } }
      }
      if (sql.includes('DELETE FROM messages WHERE session_id = ?')) {
        const [sessionId] = params as unknown[]
        const rows = store.get('messages') || []
        const filtered = rows.filter(r => r.session_id !== sessionId)
        store.set('messages', filtered)
        return { rows: { _array: [{ count: rows.length - filtered.length }] } }
      }
      if (sql.includes('SELECT COUNT(*)')) {
        const rows = store.get('messages') || []
        return { rows: { _array: [{ count: rows.length }] } }
      }
      return { rows: { _array: [] } }
    },
    close() {}
  }
}

test('MessageRepository insert and findBySessionId', () => {
  const db = createMockDb()
  const repo = new MessageRepository(db)

  repo.insert({
    id: 'm1',
    sessionId: 's1',
    type: 'text',
    content: 'Hello',
    seq: 1,
    createdAt: '2026-06-08T00:00:00Z'
  })

  const found = repo.findBySessionId('s1')
  expect(found.length).toBe(1)
  expect(found[0].id).toBe('m1')
})

test('MessageRepository findAfterSeq returns newer messages', () => {
  const db = createMockDb()
  const repo = new MessageRepository(db)

  repo.insert({ id: 'm1', sessionId: 's1', type: 'text', content: 'Old', seq: 1, createdAt: '2026-06-08T00:00:00Z' })
  repo.insert({ id: 'm2', sessionId: 's1', type: 'text', content: 'New', seq: 2, createdAt: '2026-06-08T00:01:00Z' })

  const newer = repo.findAfterSeq('s1', 1)
  expect(newer.length).toBe(1)
  expect(newer[0].id).toBe('m2')
})

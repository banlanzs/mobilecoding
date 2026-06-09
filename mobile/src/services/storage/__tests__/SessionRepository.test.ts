import { SessionRepository } from '../SessionRepository'
import type { Database } from '../Database'

function createMockDb(): Database {
  const store = new Map<string, Record<string, unknown>[]>()
  return {
    execute(sql: string, params?: unknown[]) {
      if (sql.includes('INSERT OR REPLACE INTO sessions')) {
        const [id, name, agent, model, cwd, status, created_at, updated_at, message_count] = params as unknown[]
        const row = { id, name, agent, model, cwd, status, created_at, updated_at, message_count }
        store.set('sessions', [row])
        return { rows: { _array: [row] } }
      }
      if (sql.includes('SELECT * FROM sessions WHERE id')) {
        const id = params?.[0] as string
        const rows = store.get('sessions') || []
        return { rows: { _array: rows.filter(r => r.id === id) } }
      }
      if (sql.includes('SELECT * FROM sessions ORDER BY')) {
        return { rows: { _array: store.get('sessions') || [] } }
      }
      if (sql.includes('DELETE FROM sessions WHERE id')) {
        const id = params?.[0] as string
        const rows = store.get('sessions') || []
        const filtered = rows.filter(r => r.id !== id)
        store.set('sessions', filtered)
        return { rows: { _array: [{ count: rows.length - filtered.length }] } }
      }
      if (sql.includes('SELECT COUNT(*)')) {
        const rows = store.get('sessions') || []
        return { rows: { _array: [{ count: rows.length }] } }
      }
      return { rows: { _array: [] } }
    },
    close() {}
  }
}

test('SessionRepository insert and findById', () => {
  const db = createMockDb()
  const repo = new SessionRepository(db)

  repo.insert({
    id: 's1',
    name: 'Test Session',
    agent: 'Claude',
    status: 'active',
    createdAt: '2026-06-08T00:00:00Z',
    updatedAt: '2026-06-08T00:00:00Z',
    messageCount: 0
  })

  const found = repo.findById('s1')
  expect(found).not.toBeNull()
  expect(found?.id).toBe('s1')
  expect(found?.name).toBe('Test Session')
})

test('SessionRepository findAll returns all sessions', () => {
  const db = createMockDb()
  const repo = new SessionRepository(db)

  repo.insert({
    id: 's1',
    name: 'Session 1',
    agent: 'Claude',
    status: 'active',
    createdAt: '2026-06-08T00:00:00Z',
    updatedAt: '2026-06-08T00:00:00Z',
    messageCount: 0
  })

  const all = repo.findAll()
  expect(all.length).toBe(1)
  expect(all[0].id).toBe('s1')
})

test('SessionRepository deleteById removes session', () => {
  const db = createMockDb()
  const repo = new SessionRepository(db)

  repo.insert({
    id: 's1',
    name: 'Session 1',
    agent: 'Claude',
    status: 'active',
    createdAt: '2026-06-08T00:00:00Z',
    updatedAt: '2026-06-08T00:00:00Z',
    messageCount: 0
  })

  const deleted = repo.deleteById('s1')
  expect(deleted).toBe(true)
  expect(repo.findById('s1')).toBeNull()
})

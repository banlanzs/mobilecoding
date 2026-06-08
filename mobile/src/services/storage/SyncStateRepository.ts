import type { Database } from './Database'

export interface SyncState {
  sessionId: string
  lastSeq: number
  updatedAt: string
}

export class SyncStateRepository {
  constructor(private db: Database) {}

  upsert(state: SyncState): void {
    this.db.execute(
      `INSERT OR REPLACE INTO sync_state (session_id, last_seq, updated_at)
       VALUES (?, ?, ?)`,
      [state.sessionId, state.lastSeq, state.updatedAt]
    )
  }

  findBySessionId(sessionId: string): SyncState | null {
    const result = this.db.execute(
      'SELECT * FROM sync_state WHERE session_id = ?',
      [sessionId]
    )
    const rows = result.rows._array
    if (rows.length === 0) return null
    const row = rows[0]
    return {
      sessionId: row.session_id as string,
      lastSeq: row.last_seq as number,
      updatedAt: row.updated_at as string
    }
  }

  deleteBySessionId(sessionId: string): boolean {
    const before = this.db.execute('SELECT COUNT(*) as count FROM sync_state WHERE session_id = ?', [sessionId])
    this.db.execute('DELETE FROM sync_state WHERE session_id = ?', [sessionId])
    return (before.rows._array[0] as { count: number }).count > 0
  }
}

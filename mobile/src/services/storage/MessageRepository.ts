import type { Database } from './Database'

export interface Message {
  id: string
  sessionId: string
  type: string
  content: string
  seq?: number
  createdAt: string
}

export class MessageRepository {
  constructor(private db: Database) {}

  insert(message: Message): void {
    this.db.execute(
      `INSERT OR REPLACE INTO messages (id, session_id, type, content, seq, created_at)
       VALUES (?, ?, ?, ?, ?, ?)`,
      [
        message.id,
        message.sessionId,
        message.type,
        message.content,
        message.seq || null,
        message.createdAt
      ]
    )
  }

  findBySessionId(sessionId: string, limit: number = 100): Message[] {
    const result = this.db.execute(
      'SELECT * FROM messages WHERE session_id = ? ORDER BY created_at ASC LIMIT ?',
      [sessionId, limit]
    )
    return result.rows._array.map(row => this.mapRow(row))
  }

  findAfterSeq(sessionId: string, seq: number): Message[] {
    const result = this.db.execute(
      'SELECT * FROM messages WHERE session_id = ? AND seq > ? ORDER BY seq ASC',
      [sessionId, seq]
    )
    return result.rows._array.map(row => this.mapRow(row))
  }

  deleteBySessionId(sessionId: string): number {
    const before = this.db.execute('SELECT COUNT(*) as count FROM messages WHERE session_id = ?', [sessionId])
    this.db.execute('DELETE FROM messages WHERE session_id = ?', [sessionId])
    return (before.rows._array[0] as { count: number }).count
  }

  private mapRow(row: Record<string, unknown>): Message {
    return {
      id: row.id as string,
      sessionId: row.session_id as string,
      type: row.type as string,
      content: row.content as string,
      seq: row.seq as number | undefined,
      createdAt: row.created_at as string
    }
  }
}

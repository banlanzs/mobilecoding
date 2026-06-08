import type { Database } from './Database'
import type { ServerProfile } from '@/types/server-profile'

export interface Session {
  id: string
  name: string
  agent: string
  model?: string
  cwd?: string
  status: string
  createdAt: string
  updatedAt: string
  messageCount: number
}

export class SessionRepository {
  constructor(private db: Database) {}

  insert(session: Session): void {
    this.db.execute(
      `INSERT OR REPLACE INTO sessions (id, name, agent, model, cwd, status, created_at, updated_at, message_count)
       VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
      [
        session.id,
        session.name,
        session.agent,
        session.model || null,
        session.cwd || null,
        session.status,
        session.createdAt,
        session.updatedAt,
        session.messageCount
      ]
    )
  }

  findById(id: string): Session | null {
    const result = this.db.execute(
      'SELECT * FROM sessions WHERE id = ?',
      [id]
    )
    const rows = result.rows._array
    if (rows.length === 0) return null
    return this.mapRow(rows[0])
  }

  findAll(): Session[] {
    const result = this.db.execute('SELECT * FROM sessions ORDER BY updated_at DESC')
    return result.rows._array.map(row => this.mapRow(row))
  }

  deleteById(id: string): boolean {
    const before = this.db.execute('SELECT COUNT(*) as count FROM sessions WHERE id = ?', [id])
    this.db.execute('DELETE FROM sessions WHERE id = ?', [id])
    return (before.rows._array[0] as { count: number }).count > 0
  }

  private mapRow(row: Record<string, unknown>): Session {
    return {
      id: row.id as string,
      name: row.name as string,
      agent: row.agent as string,
      model: row.model as string | undefined,
      cwd: row.cwd as string | undefined,
      status: row.status as string,
      createdAt: row.created_at as string,
      updatedAt: row.updated_at as string,
      messageCount: row.message_count as number
    }
  }
}

import { open } from '@op-engineering/op-sqlite'

export interface Database {
  execute(sql: string, params?: unknown[]): { rows: { _array: Record<string, unknown>[] } }
  close(): void
}

let db: Database | null = null

export function getDatabase(path: string = 'mobilecoding.db'): Database {
  if (db) return db

  db = open({ name: path }) as unknown as Database

  // Create tables
  db.execute(`
    CREATE TABLE IF NOT EXISTS sessions (
      id TEXT PRIMARY KEY,
      name TEXT NOT NULL,
      agent TEXT NOT NULL,
      model TEXT,
      cwd TEXT,
      status TEXT NOT NULL DEFAULT 'active',
      created_at TEXT NOT NULL,
      updated_at TEXT NOT NULL,
      message_count INTEGER DEFAULT 0
    )
  `)

  db.execute(`
    CREATE TABLE IF NOT EXISTS messages (
      id TEXT PRIMARY KEY,
      session_id TEXT NOT NULL,
      type TEXT NOT NULL,
      content TEXT NOT NULL,
      seq INTEGER,
      created_at TEXT NOT NULL,
      FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
    )
  `)

  db.execute(`
    CREATE TABLE IF NOT EXISTS sync_state (
      session_id TEXT PRIMARY KEY,
      last_seq INTEGER NOT NULL DEFAULT 0,
      updated_at TEXT NOT NULL,
      FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
    )
  `)

  return db
}

export function closeDatabase(): void {
  if (db) {
    db.close()
    db = null
  }
}

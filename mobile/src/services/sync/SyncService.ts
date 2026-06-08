import type { MessageRepository, Message } from '../storage/MessageRepository'
import type { SyncStateRepository } from '../storage/SyncStateRepository'
import type { SessionRepository } from '../storage/SessionRepository'
import type { RestClient } from '../network/RestClient'
import type { AppEvent } from '@/protocol/types'

interface ServerMessage {
  id: string
  session_id: string
  type: string
  content: string
  seq?: number
  created_at: string
}

interface ServerSession {
  id: string
  name: string
  agent: string
  model?: string
  cwd?: string
  status: string
  created_at: string
  updated_at: string
  message_count: number
}

export class SyncService {
  constructor(
    private sessions: SessionRepository,
    private messages: MessageRepository,
    private syncState: SyncStateRepository,
    private rest: RestClient
  ) {}

  async syncSession(sessionId: string): Promise<number> {
    const state = this.syncState.findBySessionId(sessionId)
    const lastSeq = state?.lastSeq || 0

    const serverMessages = await this.rest.get<ServerMessage[]>(
      `/api/v1/sessions/${sessionId}/messages?after_seq=${lastSeq}`
    )

    if (serverMessages.length === 0) return 0

    let maxSeq = lastSeq
    for (const serverMsg of serverMessages) {
      this.messages.insert({
        id: serverMsg.id,
        sessionId: serverMsg.session_id,
        type: serverMsg.type,
        content: serverMsg.content,
        seq: serverMsg.seq,
        createdAt: serverMsg.created_at
      })
      if (serverMsg.seq && serverMsg.seq > maxSeq) {
        maxSeq = serverMsg.seq
      }
    }

    this.syncState.upsert({
      sessionId,
      lastSeq: maxSeq,
      updatedAt: new Date().toISOString()
    })

    return serverMessages.length
  }

  async syncAllSessions(): Promise<number> {
    const serverSessions = await this.rest.get<ServerSession[]>('/api/v1/sessions')

    let totalSynced = 0
    for (const serverSession of serverSessions) {
      this.sessions.insert({
        id: serverSession.id,
        name: serverSession.name,
        agent: serverSession.agent,
        model: serverSession.model,
        cwd: serverSession.cwd,
        status: serverSession.status,
        createdAt: serverSession.created_at,
        updatedAt: serverSession.updated_at,
        messageCount: serverSession.message_count
      })

      const count = await this.syncSession(serverSession.id)
      totalSynced += count
    }

    return totalSynced
  }

  handleRealtimeEvent(event: AppEvent): void {
    if (!event.sessionId || !event.messageId) return

    const state = this.syncState.findBySessionId(event.sessionId)
    const lastSeq = state?.lastSeq || 0

    if (event.seq && event.seq > lastSeq) {
      this.messages.insert({
        id: event.messageId,
        sessionId: event.sessionId,
        type: event.type,
        content: JSON.stringify(event),
        seq: event.seq,
        createdAt: event.time
      })

      this.syncState.upsert({
        sessionId: event.sessionId,
        lastSeq: event.seq,
        updatedAt: new Date().toISOString()
      })
    }
  }
}

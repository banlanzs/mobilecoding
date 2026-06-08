import { create } from 'zustand'
import type { SessionMeta } from '../protocol/types'

interface SessionState {
  sessions: SessionMeta[]
  activeSessionId: string | null
  viewedSessionId: string | null
  readOnly: boolean
  setSessions: (sessions: SessionMeta[]) => void
  setActiveSession: (sessionId: string | null) => void
  viewSession: (sessionId: string, readOnly: boolean) => void
}

export const useSessionStore = create<SessionState>((set) => ({
  sessions: [],
  activeSessionId: null,
  viewedSessionId: null,
  readOnly: false,
  setSessions: (sessions) => set({ sessions }),
  setActiveSession: (activeSessionId) => set({ activeSessionId, viewedSessionId: activeSessionId, readOnly: false }),
  viewSession: (viewedSessionId, readOnly) => set({ viewedSessionId, readOnly })
}))

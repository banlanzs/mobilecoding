import { create } from 'zustand'
import type { ServerProfile } from '../types/server-profile'

interface AuthState {
  activeProfile: ServerProfile | null
  status: 'idle' | 'connecting' | 'connected' | 'reconnecting' | 'closed'
  setActiveProfile: (profile: ServerProfile | null) => void
  setStatus: (status: AuthState['status']) => void
}

export const useAuthStore = create<AuthState>((set) => ({
  activeProfile: null,
  status: 'idle',
  setActiveProfile: (activeProfile) => set({ activeProfile }),
  setStatus: (status) => set({ status })
}))

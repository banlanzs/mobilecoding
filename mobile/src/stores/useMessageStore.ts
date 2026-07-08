import { createStore } from 'zustand/vanilla'
import type { AppEvent, PermissionRequestEvent, TextDeltaEvent } from '../protocol/types'

export interface UserMessage {
  type: 'user'
  sessionId: string
  time: string
  text: string
}

export type DisplayMessage = AppEvent | UserMessage

/** 上下文窗口用量（由 context_window 事件解析，不进入 messages 列表） */
export interface ContextWindowInfo {
  used?: number
  total?: number
  raw?: unknown
}

interface MessageState {
  messages: DisplayMessage[]
  permissionPrompt: PermissionRequestEvent | null
  permissionRequestId: string | null
  thinking: boolean
  turnActive: boolean
  lastSeq: number
  contextWindow: ContextWindowInfo | null
  handleEvent: (event: any, sessionId?: string) => void
  addUserMessage: (text: string, sessionId: string) => void
  clearPermission: () => void
  answerPermission: (allow: boolean) => void
  resetMessages: () => void
}

export function createMessageStore() {
  return createStore<MessageState>((set, get) => ({
    messages: [],
    permissionPrompt: null,
    permissionRequestId: null,
    thinking: false,
    turnActive: false,
    lastSeq: 0,
    contextWindow: null,
    handleEvent: (event: any, sessionId?: string) => {
      const state = get()

      // 内部协议事件：只更新状态，不加入消息列表
      if (event.type === 'thinking_start' || event.type === 'thinking_end' ||
          event.type === 'turn_end' || event.type === 'lifecycle' || event.type === 'agent_state') {
        set({
          thinking: event.type === 'thinking_start' ? true : event.type === 'turn_end' ? false : state.thinking,
          turnActive: event.type === 'turn_end' ? false : true,
          lastSeq: event.seq && event.seq > state.lastSeq ? event.seq : state.lastSeq,
        })
        return
      }

      // context_window：解析用量存入独立状态字段，不进入 messages 列表
      if (event.type === 'context_window') {
        const info = parseContextWindow(event.toolInput)
        set({ contextWindow: info })
        return
      }

      // 隐藏事件：直接忽略
      if (event.type === 'plan_mode' || event.type === 'session') {
        return
      }

      let messages = [...state.messages]

      if (event.type === 'text_delta') {
        const last = messages[messages.length - 1]
        if (last && last.type === 'text_delta' && (last as TextDeltaEvent).blockIndex === event.blockIndex) {
          const merged: TextDeltaEvent = {
            ...(last as TextDeltaEvent),
            text: ((last as TextDeltaEvent).text || '') + event.text,
            thinking: (last as TextDeltaEvent).thinking || event.thinking
          }
          messages = [...messages.slice(0, -1), merged]
        } else {
          messages.push(event)
        }
      } else if (event.type === 'text') {
        const last = messages[messages.length - 1]
        if (last && last.type === 'text_delta') {
          const thinking = (last as TextDeltaEvent).thinking
          messages = [...messages.slice(0, -1), { ...event, thinking: event.thinking || thinking }]
        } else {
          messages.push(event)
        }
      } else if (event.type === 'permission_request' || event.type === 'permission_ask') {
        const last = messages[messages.length - 1] as any
        const duplicate = last && (last.type === 'permission_request' || last.type === 'permission_ask') && last.toolName === (event as any).toolName
        messages = duplicate ? [...messages.slice(0, -1), event] : [...messages, event]
        set({ permissionPrompt: event as PermissionRequestEvent, permissionRequestId: event.messageId || null })
      } else {
        messages.push(event)
      }

      set({
        messages,
        lastSeq: event.seq && event.seq > state.lastSeq ? event.seq : state.lastSeq,
        thinking: event.type === 'thinking_start' ? true : event.type === 'turn_end' ? false : state.thinking,
        turnActive: event.type === 'turn_end' ? false : true
      })
    },
    addUserMessage: (text, sessionId) => {
      const state = get()
      set({
        messages: [...state.messages, { type: 'user', text, sessionId, time: new Date().toISOString() }],
        thinking: true,
        turnActive: true
      })
    },
    clearPermission: () => set({ permissionPrompt: null, permissionRequestId: null }),
    answerPermission: (allow: boolean) => {
      // UI 层调用此方法后，再发送 permission.respond 到服务器
      set({ permissionPrompt: null, permissionRequestId: null })
    },
    resetMessages: () => set({ messages: [], permissionPrompt: null, permissionRequestId: null, thinking: false, turnActive: false, lastSeq: 0, contextWindow: null })
  }))
}

/**
 * 解析 context_window 事件 payload 为用量信息。
 * Claude hook 的 payload 字段名不统一，容错解析多种可能的结构。
 * 解析失败返回 null（UI 不显示，不崩溃）。
 */
function parseContextWindow(data: unknown): ContextWindowInfo | null {
  if (!data || typeof data !== 'object') return null
  const obj = data as Record<string, any>
  // 常见字段名兜底
  const used = obj.used_tokens ?? obj.usedTokens ?? obj.tokens_used ?? obj.used ?? obj.context_used
  const total = obj.total_tokens ?? obj.totalTokens ?? obj.tokens_total ?? obj.total ?? obj.context_total
  if (used == null && total == null) {
    // 没有可识别字段，保留原始数据供调试
    return { raw: data }
  }
  return {
    used: typeof used === 'number' ? used : Number(used) || undefined,
    total: typeof total === 'number' ? total : Number(total) || undefined,
    raw: data,
  }
}

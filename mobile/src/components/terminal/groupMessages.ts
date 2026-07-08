import type { DisplayMessage, UserMessage } from '../../stores/useMessageStore'

/** 渲染分组：把扁平 messages 聚合成「一条 Assistant 消息 = 一个卡片」的单元 */
export type RenderGroup =
  | { kind: 'user'; message: UserMessage }
  | { kind: 'assistant'; items: DisplayMessage[] }
  | { kind: 'standalone'; message: DisplayMessage }

/** 判断是否为需要独立交互卡片的类型（权限请求等） */
function isStandalone(m: DisplayMessage): boolean {
  return m.type === 'permission_request' || m.type === 'permission_ask'
}

/**
 * 把扁平消息流分组成渲染单元：
 * - user 消息独立成组
 * - 权限请求独立成组（需要交互）
 * - 其余 assistant 事件（thinking/text/tool_*）累积进当前 assistant 组
 * - 开头若没有 user 前导的 assistant 事件，自动建一个 assistant 组
 */
export function groupMessages(messages: DisplayMessage[]): RenderGroup[] {
  const groups: RenderGroup[] = []
  let currentAssistant: DisplayMessage[] | null = null

  const flushAssistant = () => {
    if (currentAssistant && currentAssistant.length > 0) {
      groups.push({ kind: 'assistant', items: currentAssistant })
    }
    currentAssistant = null
  }

  for (const m of messages) {
    if (m.type === 'user') {
      flushAssistant()
      groups.push({ kind: 'user', message: m })
      continue
    }
    if (isStandalone(m)) {
      flushAssistant()
      groups.push({ kind: 'standalone', message: m })
      continue
    }
    // assistant 事件：累积到当前组（没有就新建）
    if (!currentAssistant) currentAssistant = []
    currentAssistant.push(m)
  }
  flushAssistant()
  return groups
}

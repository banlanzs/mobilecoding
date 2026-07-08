/** 工具结果是否为错误（Claude tool_result 带 is_error / isError 字段，或输出含明显错误标记） */
export function isToolError(m: any): boolean {
  if (m?.toolResult && typeof m.toolResult === 'object' && (m.toolResult.is_error || m.toolResult.isError)) {
    return true
  }
  const out: string = typeof m?.toolOutput === 'string' ? m.toolOutput : ''
  return /^(error|fatal|traceback|command not found|no such file|permission denied)/i.test(out.trim())
}

/** 工具块配色：默认蓝灰、错误红、成功绿 */
export function toolTone(m: any): { bg: string; border: string; icon: string } {
  if (isToolError(m)) return { bg: '#FFF2F2', border: '#f5c2c2', icon: '✕' }
  const hasOutput = !!(m?.toolOutput || m?.toolResult)
  if (hasOutput) return { bg: '#F0FFF4', border: '#c2e9c9', icon: '✓' }
  return { bg: '#EEF4FF', border: '#c7d8ff', icon: '🔧' }
}

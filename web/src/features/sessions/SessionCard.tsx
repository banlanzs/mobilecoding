import type { SessionMeta } from '../../core/ws/types'

interface SessionCardProps {
  session: SessionMeta
  onClick: () => void
  onDelete: (e: React.MouseEvent) => void
}

export function SessionCard({ session, onClick, onDelete }: SessionCardProps) {
  const formatTime = (timestamp: string) => {
    const date = new Date(timestamp)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffMins = Math.floor(diffMs / 60000)
    const diffHours = Math.floor(diffMins / 60)
    const diffDays = Math.floor(diffHours / 24)

    if (diffMins < 1) return '刚刚'
    if (diffMins < 60) return `${diffMins} 分钟前`
    if (diffHours < 24) return `${diffHours} 小时前`
    if (diffDays < 7) return `${diffDays} 天前`
    return date.toLocaleDateString('zh-CN')
  }

  const getAgentIcon = (agent: string) => {
    switch (agent.toLowerCase()) {
      case 'claude':
        return '🤖'
      case 'codex':
        return '💻'
      case 'opencode':
        return '📝'
      default:
        return '🔧'
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active':
        return '#4ade80'
      case 'inactive':
        return '#94a3b8'
      default:
        return '#64748b'
    }
  }

  return (
    <div className="session-card" onClick={onClick}>
      <div className="session-card-header">
        <div className="session-card-icon">{getAgentIcon(session.agent)}</div>
        <div className="session-card-info">
          <div className="session-card-name">{session.name}</div>
          <div className="session-card-meta">
            {session.agent}
            {session.model && ` · ${session.model}`}
          </div>
        </div>
        <div
          className="session-card-status"
          style={{ backgroundColor: getStatusColor(session.status) }}
        />
      </div>

      <div className="session-card-details">
        <div className="session-card-time">
          最后活跃: {formatTime(session.lastActiveAt)}
        </div>
        <div className="session-card-count">
          {session.messageCount} 条消息
        </div>
      </div>

      <button
        className="session-card-delete"
        onClick={onDelete}
        aria-label="删除会话"
      >
        🗑️
      </button>
    </div>
  )
}

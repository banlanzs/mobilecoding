import { useState } from 'react'
import type { SessionMeta } from '../../core/ws/types'

interface SessionCardProps {
  session: SessionMeta
  onClick: () => void
  onDelete: (e: React.MouseEvent) => void
  onRename: (id: string, name: string) => Promise<void>
  onResume?: (session: SessionMeta) => void
}

export function SessionCard({ session, onClick, onDelete, onRename, onResume }: SessionCardProps) {
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(session.name)
  const [saving, setSaving] = useState(false)

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

  const startEdit = (e: React.MouseEvent) => {
    e.stopPropagation()
    setDraft(session.name)
    setEditing(true)
  }

  const commit = async () => {
    const trimmed = draft.trim()
    if (!trimmed || trimmed === session.name) {
      setEditing(false)
      return
    }
    setSaving(true)
    try {
      await onRename(session.id, trimmed)
      setEditing(false)
    } catch {
      // 错误由调用方提示，此处保持编辑态
    } finally {
      setSaving(false)
    }
  }

  const cancel = () => {
    setDraft(session.name)
    setEditing(false)
  }

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault()
      commit()
    } else if (e.key === 'Escape') {
      e.preventDefault()
      cancel()
    }
  }

  const handleResume = (e: React.MouseEvent) => {
    e.stopPropagation()
    onResume?.(session)
  }

  const canResume = !!onResume && session.status !== 'active' && !!session.resumeSessionId

  return (
    <div className="session-card" onClick={onClick}>
      <div className="session-card-header">
        <div className="session-card-icon">{getAgentIcon(session.agent)}</div>
        <div className="session-card-info">
          {editing ? (
            <input
              className="session-card-name-input"
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={onKeyDown}
              onBlur={commit}
              onClick={(e) => e.stopPropagation()}
              disabled={saving}
              autoFocus
              aria-label="会话名称"
            />
          ) : (
            <div className="session-card-name">{session.name}</div>
          )}
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

      <div className="session-card-actions" onClick={(e) => e.stopPropagation()}>
        {canResume && (
          <button
            className="session-card-resume"
            onClick={handleResume}
            aria-label="继续会话"
            title="继续此会话"
          >
            继续
          </button>
        )}
        <button
          className="session-card-rename"
          onClick={startEdit}
          disabled={editing || saving}
          aria-label="重命名会话"
          title="重命名"
        >
          ✏️
        </button>
        <button
          className="session-card-delete"
          onClick={onDelete}
          aria-label="删除会话"
        >
          🗑️
        </button>
      </div>
    </div>
  )
}

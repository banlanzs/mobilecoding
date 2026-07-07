import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import type { SessionMeta } from '../../core/ws/types'
import { useChat } from '../../core/state/ChatContext'
import { SessionCard } from './SessionCard'
import './sessions.css'

export function SessionListPage() {
  const navigate = useNavigate()
  const { sendStart, sendStop } = useChat()
  const [sessions, setSessions] = useState<SessionMeta[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [resuming, setResuming] = useState<string | null>(null)

  useEffect(() => {
    loadSessions()
  }, [])

  const loadSessions = async () => {
    try {
      setLoading(true)
      setError(null)

      const token = localStorage.getItem('mobilecoding.token')

      if (!token) {
        setError('未找到认证令牌')
        setLoading(false)
        return
      }

      const response = await fetch('/api/v1/sessions', {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })

      if (!response.ok) {
        throw new Error(`加载会话列表失败: ${response.statusText}`)
      }

      const data = await response.json()
      setSessions(data.sessions || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载会话列表失败')
    } finally {
      setLoading(false)
    }
  }

  const handleSessionClick = (sessionId: string) => {
    navigate(`/sessions/${sessionId}`)
  }

  const handleDeleteSession = async (sessionId: string, e: React.MouseEvent) => {
    e.stopPropagation()
    if (!confirm('确认删除此会话吗？')) {
      return
    }

    try {
      const token = localStorage.getItem('mobilecoding.token')
      const response = await fetch(`/api/v1/sessions/${sessionId}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })

      if (!response.ok) {
        throw new Error('删除会话失败')
      }

      loadSessions()
    } catch (err) {
      alert(err instanceof Error ? err.message : '删除会话失败')
    }
  }

  const handleRenameSession = async (sessionId: string, name: string) => {
    const token = localStorage.getItem('mobilecoding.token')
    const response = await fetch(`/api/v1/sessions/${sessionId}`, {
      method: 'PATCH',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ name }),
    })
    if (!response.ok) {
      const msg = await response.text().catch(() => '重命名失败')
      throw new Error(msg)
    }
    // 更新本地列表，避免完整重载
    setSessions((prev) =>
      prev.map((s) => (s.id === sessionId ? { ...s, name } : s))
    )
  }

  // 恢复历史会话：停当前 → 用历史的 command/args/resumeSessionId 启动 → 跳转终端
  const handleResume = async (session: SessionMeta) => {
    if (resuming) return
    setResuming(session.id)
    try {
      // 停止当前活跃会话（若有）
      try {
        await sendStop()
      } catch {
        // 当前无活跃会话或已停止，忽略
      }
      await sendStart({
        command: session.command || session.agent,
        args: session.args,
        cwd: session.cwd,
        resumeSessionId: session.resumeSessionId,
      })
      navigate('/')
    } catch (err) {
      alert(err instanceof Error ? err.message : '恢复会话失败')
    } finally {
      setResuming(null)
    }
  }

  const handleCreateSession = () => {
    navigate('/sessions/new')
  }

  if (loading) {
    return (
      <div className="sessions-page">
        <div className="sessions-loading">加载中...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="sessions-page">
        <div className="sessions-error">
          <p>{error}</p>
          <button onClick={loadSessions}>重试</button>
        </div>
      </div>
    )
  }

  return (
    <div className="sessions-page">
      <header className="sessions-header">
        <h1>会话列表</h1>
        <button className="sessions-create-btn" onClick={handleCreateSession}>
          + 新建会话
        </button>
      </header>

      {sessions.length === 0 ? (
        <div className="sessions-empty">
          <p>暂无会话</p>
          <button onClick={handleCreateSession}>创建第一个会话</button>
        </div>
      ) : (
        <div className="sessions-list">
          {sessions.map(session => (
            <SessionCard
              key={session.id}
              session={session}
              onClick={() => handleSessionClick(session.id)}
              onDelete={(e) => handleDeleteSession(session.id, e)}
              onRename={handleRenameSession}
              onResume={resuming === null ? handleResume : undefined}
            />
          ))}
        </div>
      )}
    </div>
  )
}

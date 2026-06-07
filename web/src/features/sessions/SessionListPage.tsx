import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import type { SessionMeta } from '../../core/ws/types'
import { SessionCard } from './SessionCard'
import './sessions.css'

export function SessionListPage() {
  const navigate = useNavigate()
  const [sessions, setSessions] = useState<SessionMeta[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

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

      // 重新加载列表
      loadSessions()
    } catch (err) {
      alert(err instanceof Error ? err.message : '删除会话失败')
    }
  }

  const handleCreateSession = () => {
    // 当前实现：直接跳转到终端页面（使用默认 agent）
    // 未来可以添加创建会话弹窗
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
            />
          ))}
        </div>
      )}
    </div>
  )
}

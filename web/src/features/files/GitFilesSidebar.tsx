import { useEffect, useState } from 'react'
import { DiffView } from '../terminal/cards/DiffView'
import './files.css'

interface GitFileStatus {
  path: string
  status: string
  staged: boolean
}

interface GitFilesSidebarProps {
  cwd?: string
  onClose: () => void
}

export function GitFilesSidebar({ cwd, onClose }: GitFilesSidebarProps) {
  const [files, setFiles] = useState<GitFileStatus[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedFile, setSelectedFile] = useState<string | null>(null)
  const [diff, setDiff] = useState<string>('')
  const [loadingDiff, setLoadingDiff] = useState(false)

  useEffect(() => {
    loadGitStatus()
  }, [cwd])

  const loadGitStatus = async () => {
    try {
      setLoading(true)
      setError(null)

      const token = localStorage.getItem('mobilecoding.token')
      if (!token) {
        setError('未找到认证令牌')
        return
      }

      const url = cwd
        ? `/api/v1/git/status?cwd=${encodeURIComponent(cwd)}`
        : '/api/v1/git/status'

      const response = await fetch(url, {
        headers: { 'Authorization': `Bearer ${token}` }
      })

      if (!response.ok) {
        throw new Error(`加载 git 状态失败: ${response.statusText}`)
      }

      const data = await response.json()
      setFiles(data.files || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载 git 状态失败')
    } finally {
      setLoading(false)
    }
  }

  const loadDiff = async (filePath: string) => {
    try {
      setLoadingDiff(true)
      setSelectedFile(filePath)

      const token = localStorage.getItem('mobilecoding.token')

      const url = cwd
        ? `/api/v1/git/diff?cwd=${encodeURIComponent(cwd)}&file=${encodeURIComponent(filePath)}`
        : `/api/v1/git/diff?file=${encodeURIComponent(filePath)}`

      const response = await fetch(url, {
        headers: { 'Authorization': `Bearer ${token}` }
      })

      if (!response.ok) {
        throw new Error('加载 diff 失败')
      }

      const data = await response.json()
      setDiff(data.diff || '无变更')
    } catch (err) {
      setDiff(err instanceof Error ? err.message : '加载 diff 失败')
    } finally {
      setLoadingDiff(false)
    }
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'M': return '📝'  // Modified
      case 'A': return '✨'  // Added
      case 'D': return '🗑️'  // Deleted
      case 'R': return '🔄'  // Renamed
      case '??': return '❓' // Untracked
      default: return '📄'
    }
  }

  const getStatusLabel = (status: string) => {
    switch (status) {
      case 'M': return '已修改'
      case 'A': return '新增'
      case 'D': return '已删除'
      case 'R': return '已重命名'
      case '??': return '未跟踪'
      default: return status
    }
  }

  // 按状态分组
  const stagedFiles = files.filter(f => f.staged)
  const unstagedFiles = files.filter(f => !f.staged && f.status !== '??')
  const untrackedFiles = files.filter(f => f.status === '??')

  if (loading) {
    return (
      <div className="git-sidebar">
        <div className="git-sidebar-header">
          <h3>Git 变更</h3>
          <button className="btn-close" onClick={onClose}>✕</button>
        </div>
        <div className="git-sidebar-loading">加载中...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="git-sidebar">
        <div className="git-sidebar-header">
          <h3>Git 变更</h3>
          <button className="btn-close" onClick={onClose}>✕</button>
        </div>
        <div className="git-sidebar-error">
          <p>{error}</p>
          <button onClick={loadGitStatus}>重试</button>
        </div>
      </div>
    )
  }

  return (
    <div className="git-sidebar">
      <div className="git-sidebar-header">
        <h3>Git 变更</h3>
        <button className="btn-refresh" onClick={loadGitStatus} title="刷新">
          🔄
        </button>
        <button className="btn-close" onClick={onClose}>✕</button>
      </div>

      <div className="git-sidebar-content">
        {files.length === 0 ? (
          <div className="git-empty">无变更文件</div>
        ) : (
          <>
            {stagedFiles.length > 0 && (
              <div className="git-group">
                <div className="git-group-title">已暂存 ({stagedFiles.length})</div>
                {stagedFiles.map(file => (
                  <div
                    key={file.path}
                    className={`git-file-item ${selectedFile === file.path ? 'selected' : ''}`}
                    onClick={() => loadDiff(file.path)}
                  >
                    <span className="file-icon">{getStatusIcon(file.status)}</span>
                    <span className="file-path">{file.path}</span>
                    <span className="file-status">{getStatusLabel(file.status)}</span>
                  </div>
                ))}
              </div>
            )}

            {unstagedFiles.length > 0 && (
              <div className="git-group">
                <div className="git-group-title">未暂存 ({unstagedFiles.length})</div>
                {unstagedFiles.map(file => (
                  <div
                    key={file.path}
                    className={`git-file-item ${selectedFile === file.path ? 'selected' : ''}`}
                    onClick={() => loadDiff(file.path)}
                  >
                    <span className="file-icon">{getStatusIcon(file.status)}</span>
                    <span className="file-path">{file.path}</span>
                    <span className="file-status">{getStatusLabel(file.status)}</span>
                  </div>
                ))}
              </div>
            )}

            {untrackedFiles.length > 0 && (
              <div className="git-group">
                <div className="git-group-title">未跟踪 ({untrackedFiles.length})</div>
                {untrackedFiles.map(file => (
                  <div
                    key={file.path}
                    className={`git-file-item ${selectedFile === file.path ? 'selected' : ''}`}
                    onClick={() => loadDiff(file.path)}
                  >
                    <span className="file-icon">{getStatusIcon(file.status)}</span>
                    <span className="file-path">{file.path}</span>
                    <span className="file-status">{getStatusLabel(file.status)}</span>
                  </div>
                ))}
              </div>
            )}
          </>
        )}

        {selectedFile && (
          <div className="git-diff-panel">
            <div className="diff-header">
              <span>{selectedFile}</span>
              <button onClick={() => setSelectedFile(null)}>✕</button>
            </div>
            {loadingDiff ? (
              <div className="diff-loading">加载中...</div>
            ) : (
              <div className="diff-content">
                <DiffView diff={diff} />
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

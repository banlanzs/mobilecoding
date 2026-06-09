import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'

export function RootPage() {
  const navigate = useNavigate()
  const [checking, setChecking] = useState(true)

  useEffect(() => {
    // 1. 检查 localStorage 中是否已有 token
    let token = localStorage.getItem('mobilecoding.token')

    // 2. 如果没有，尝试从 URL 参数获取
    if (!token) {
      const params = new URLSearchParams(window.location.search)
      token = params.get('token')
      if (token) {
        localStorage.setItem('mobilecoding.token', token)
        // 清理 URL 中的 token 参数
        window.history.replaceState({}, '', window.location.pathname)
      }
    }

    // 3. 如果有 token，跳转到会话列表
    if (token) {
      navigate('/sessions', { replace: true })
    } else {
      // 4. 没有 token，停留在当前页显示提示
      setChecking(false)
    }
  }, [navigate])

  if (checking) {
    return (
      <div style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        height: '100vh',
        padding: '20px',
        textAlign: 'center',
        background: 'var(--mc-bg)',
        color: 'var(--mc-fg-2)'
      }}>
        <h2>mobilecoding</h2>
        <p>正在初始化...</p>
      </div>
    )
  }

  return (
    <div style={{
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      height: '100vh',
      padding: '20px',
      textAlign: 'center',
      background: 'var(--mc-bg)',
      color: 'var(--mc-fg-2)'
    }}>
      <h2>mobilecoding</h2>
      <p>
        通过电脑终端中的二维码扫码连接，
        <br />
        或在 URL 后添加 <code>?token=你的令牌</code> 连接。
      </p>
    </div>
  )
}

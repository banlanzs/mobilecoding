import { BrowserRouter, Routes, Route, useLocation, useNavigate } from 'react-router-dom'
import { ChatProvider } from './core/state/ChatContext'
import { TerminalPage } from './features/terminal/TerminalPage'
import { useEffect } from 'react'

// Token 拦截器：全局检查并提取 URL 中的 token
function TokenInterceptor({ children }: { children: React.ReactNode }) {
  const location = useLocation()
  const navigate = useNavigate()

  useEffect(() => {
    // 从 URL 中提取 token
    const params = new URLSearchParams(location.search)
    const urlToken = params.get('token')

    if (urlToken) {
      console.log('[TokenInterceptor] found token in URL, saving to localStorage')
      localStorage.setItem('mobilecoding.token', urlToken)

      // 清理 URL 中的 token 参数
      params.delete('token')
      const newSearch = params.toString()
      const newUrl = location.pathname + (newSearch ? '?' + newSearch : '')
      window.history.replaceState({}, '', newUrl)
    }
  }, [location, navigate])

  return <>{children}</>
}

function App() {
  return (
    <BrowserRouter>
      <ChatProvider>
        <TokenInterceptor>
          <Routes>
            <Route path="/*" element={<TerminalPage />} />
          </Routes>
        </TokenInterceptor>
      </ChatProvider>
    </BrowserRouter>
  )
}

export default App

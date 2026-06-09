import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import './features/terminal/terminal.css'
import App from './App.tsx'

// 初始化主题：渲染前根据 localStorage 设置 data-theme，避免首屏主题闪烁
const savedTheme = localStorage.getItem('mc-theme') || 'dark'
document.documentElement.setAttribute('data-theme', savedTheme)

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
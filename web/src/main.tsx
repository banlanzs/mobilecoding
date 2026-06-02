import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import './features/terminal/terminal.css'
import App from './App.tsx'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
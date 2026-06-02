import { HashRouter, Routes, Route, Link } from 'react-router-dom'
import { ChatProvider } from './core/state/ChatContext'
import { TerminalPage } from './features/terminal/TerminalPage'
import { SkillListPage } from './features/skills/SkillListPage'
import { MemoryListPage } from './features/memory/MemoryListPage'
import './App.css'

function App() {
  return (
    <HashRouter>
      <ChatProvider>
        <nav>
          <Link to="/">Terminal</Link>
          <Link to="/skills" style={{ marginLeft: 12 }}>Skills</Link>
          <Link to="/memory" style={{ marginLeft: 12 }}>Memory</Link>
        </nav>
        <Routes>
          <Route path="/" element={<TerminalPage />} />
          <Route path="/skills" element={<SkillListPage />} />
          <Route path="/memory" element={<MemoryListPage />} />
        </Routes>
      </ChatProvider>
    </HashRouter>
  )
}

export default App
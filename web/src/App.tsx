import { ChatProvider } from './core/state/ChatContext'
import { TerminalPage } from './features/terminal/TerminalPage'

function App() {
  return (
    <ChatProvider>
      <TerminalPage />
    </ChatProvider>
  )
}

export default App

// Agent 状态栏 — 头像 + 名称（按真实 CLI 动态）+ 实时状态徽章
import { useChat } from '../../core/state/ChatContext';

// 头像/名称按当前 CLI 动态显示（项目支持多 CLI，不写死 Claude）
const CLI_INFO: Record<string, { avatar: string; name: string }> = {
  claude: { avatar: 'C', name: 'Claude Code' },
  codex: { avatar: 'X', name: 'Codex' },
  opencode: { avatar: 'O', name: 'OpenCode' },
  qwen: { avatar: 'Q', name: 'Qwen Code' },
  aichat: { avatar: 'A', name: 'Aichat' },
};

const STATE_LABELS: Record<string, string> = {
  thinking: 'Thinking…',
  reading_files: '读取文件',
  editing_files: '编辑文件',
  running_command: '执行命令',
};

export function AgentStatusBar() {
  const { state } = useChat();
  const { agentState } = state;

  const command = state.selectedCommand || state.runtime.defaultCommand || 'claude';
  const cli = CLI_INFO[command] || { avatar: '◆', name: 'AI Agent' };

  const isThinking = agentState.status === 'thinking';
  const isStreaming = state.turnActive && !isThinking;
  const active = isThinking || isStreaming || agentState.status !== 'idle';
  const stateClass = isThinking ? 'thinking' : active ? 'streaming' : 'idle';

  let stateLabel: string;
  if (isThinking) {
    stateLabel = 'Thinking…';
  } else if (agentState.status !== 'idle') {
    stateLabel = STATE_LABELS[agentState.status] || agentState.status;
    if (agentState.toolName) stateLabel += ` · ${agentState.toolName}`;
  } else if (isStreaming) {
    stateLabel = '输出中…';
  } else {
    stateLabel = '空闲';
  }

  const meta = state.sessionId
    ? `session ${state.sessionId.slice(0, 8)}…`
    : state.status === 'connected' ? '已连接' : '等待连接';

  return (
    <div className="agent-status-bar">
      <div className="agent-info">
        <div className="agent-avatar">{cli.avatar}</div>
        <div>
          <div className="agent-name">{cli.name}</div>
          <div className="agent-meta">{meta}</div>
        </div>
      </div>
      <div className={`agent-state ${stateClass}`}>
        <span className="agent-state-dot" />
        <span>{stateLabel}</span>
      </div>
    </div>
  );
}

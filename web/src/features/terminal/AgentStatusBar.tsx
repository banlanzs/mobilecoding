// Agent 状态栏：实时显示 Agent 当前在做什么
import { useChat } from '../../core/state/ChatContext';

const STATUS_CONFIG: Record<string, { icon: string; label: string }> = {
  idle: { icon: '🟢', label: '空闲' },
  thinking: { icon: '🧠', label: '思考中' },
  reading_files: { icon: '📂', label: '读取文件' },
  editing_files: { icon: '✏️', label: '编辑文件' },
  running_command: { icon: '▶️', label: '执行命令' },
};

export function AgentStatusBar() {
  const { state } = useChat();
  const { agentState } = state;

  const cfg = STATUS_CONFIG[agentState.status] || STATUS_CONFIG.idle;
  const toolLabel = agentState.toolName ? ` · ${agentState.toolName}` : '';

  return (
    <div className="agent-status-bar" data-status={agentState.status}>
      <span className="agent-status-icon">{cfg.icon}</span>
      <span className="agent-status-label">{cfg.label}{toolLabel}</span>
    </div>
  );
}
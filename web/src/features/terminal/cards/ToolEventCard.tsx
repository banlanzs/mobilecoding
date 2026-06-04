// 渲染工具生命周期事件 — thinking_start/end, tool_start/end, bash_start/end
import type { AppEvent } from '../../../core/ws/types';

const LABELS: Record<string, { icon: string; text: string }> = {
  thinking_start: { icon: '🧠', text: '思考中…' },
  thinking_end:   { icon: '✅', text: '思考完成' },
  tool_start:     { icon: '🔧', text: '执行工具' },
  tool_end:       { icon: '✅', text: '工具完成' },
  bash_start:     { icon: '▶️', text: '执行命令' },
  bash_end:       { icon: '✅', text: '命令完成' },
  agent_state:    { icon: '📡', text: '状态变更' },
};

export function ToolEventCard({ event }: { event: AppEvent }) {
  const cfg = LABELS[event.type];
  if (!cfg) return null;
  const toolName = (event as any).toolName ? ` · ${(event as any).toolName}` : '';

  return (
    <article className="card card-tool-event">
      <span className="tool-event-icon">{cfg.icon}</span>
      <span className="tool-event-text">{cfg.text}{toolName}</span>
    </article>
  );
}
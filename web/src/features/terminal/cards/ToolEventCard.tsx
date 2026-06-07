// 渲染工具生命周期事件 — thinking_start/end, tool_start/end, bash_start/end
import type { AppEvent } from '../../../core/ws/types';
import { useState } from 'react';

const LABELS: Record<string, { icon: string; text: string; color?: string }> = {
  thinking_start: { icon: '🧠', text: '思考中…', color: 'var(--mc-purple)' },
  thinking_end:   { icon: '✅', text: '思考完成', color: 'var(--mc-success)' },
  tool_start:     { icon: '🔧', text: '执行工具', color: 'var(--mc-accent)' },
  tool_end:       { icon: '✅', text: '工具完成', color: 'var(--mc-success)' },
  bash_start:     { icon: '▶️', text: '执行命令', color: 'var(--mc-accent)' },
  bash_end:       { icon: '✅', text: '命令完成', color: 'var(--mc-success)' },
  agent_state:    { icon: '📡', text: '状态变更', color: 'var(--mc-muted)' },
};

export function ToolEventCard({ event }: { event: AppEvent }) {
  const [expanded, setExpanded] = useState(false);
  const cfg = LABELS[event.type];
  if (!cfg) return null;

  const toolName = (event as any).toolName ? ` · ${(event as any).toolName}` : '';
  const details = (event as any).details;
  const hasDetails = details && (details.command || details.args || details.status);

  // 简单模式：没有详情或不需要展开
  if (!hasDetails) {
    return (
      <article className="card card-tool-event">
        <span className="tool-event-icon">{cfg.icon}</span>
        <span className="tool-event-text" style={{ color: cfg.color }}>
          {cfg.text}{toolName}
        </span>
      </article>
    );
  }

  // 详情模式：可展开查看详细信息
  return (
    <article className="card card-tool-event card-tool-event-expandable">
      <div
        className="tool-event-header"
        onClick={() => setExpanded(!expanded)}
      >
        <span className="tool-event-icon">{cfg.icon}</span>
        <span className="tool-event-text" style={{ color: cfg.color }}>
          {cfg.text}{toolName}
        </span>
        <span className="tool-event-expand">{expanded ? '▼' : '▶'}</span>
      </div>

      {expanded && (
        <div className="tool-event-details">
          {details.command && (
            <div className="tool-detail-item">
              <span className="tool-detail-label">命令:</span>
              <code className="tool-detail-value">{details.command}</code>
            </div>
          )}
          {details.args && details.args.length > 0 && (
            <div className="tool-detail-item">
              <span className="tool-detail-label">参数:</span>
              <code className="tool-detail-value">{details.args.join(' ')}</code>
            </div>
          )}
          {details.status && (
            <div className="tool-detail-item">
              <span className="tool-detail-label">状态:</span>
              <span className="tool-detail-value">{details.status}</span>
            </div>
          )}
          {details.duration && (
            <div className="tool-detail-item">
              <span className="tool-detail-label">耗时:</span>
              <span className="tool-detail-value">{details.duration}</span>
            </div>
          )}
        </div>
      )}
    </article>
  );
}
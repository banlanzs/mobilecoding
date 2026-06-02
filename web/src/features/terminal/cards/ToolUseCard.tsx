// 渲染 tool_use 事件 — 工具调用（可折叠）
import type { ToolUseEvent } from '../../../core/ws/types';
import { useState } from 'react';

export function ToolUseCard({ event }: { event: ToolUseEvent }) {
  const [expanded, setExpanded] = useState(false);

  return (
    <article
      className="card card-tool"
      onClick={() => setExpanded(!expanded)}
      role="button"
      tabIndex={0}
    >
      <header className="card-header">
        <span className="card-badge">tool</span>
        <span className="tool-name">{event.toolName}</span>
        <span style={{ marginLeft: 'auto', color: '#565f89', fontSize: 11 }}>
          {expanded ? '▼' : '▶'}
        </span>
      </header>
      {expanded && (
        <div className="tool-body">
          <pre>{JSON.stringify(event.toolInput, null, 2)}</pre>
        </div>
      )}
    </article>
  );
}
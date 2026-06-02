// 渲染 tool_result 事件 — 工具结果
import type { ToolResultEvent } from '../../../core/ws/types';
import { useState } from 'react';

export function ToolResultCard({ event }: { event: ToolResultEvent }) {
  const [expanded, setExpanded] = useState(false);

  return (
    <article
      className="card card-result"
      onClick={() => setExpanded(!expanded)}
      role="button"
      tabIndex={0}
    >
      <header className="card-header">
        <span className="card-badge">result</span>
        <span className="tool-name">{event.toolName}</span>
        <span style={{ marginLeft: 'auto', color: '#565f89', fontSize: 11 }}>
          {expanded ? '▼' : '▶'}
        </span>
      </header>
      {expanded && (
        <div className="result-body">
          <pre>{formatResult(event.toolResult)}</pre>
        </div>
      )}
    </article>
  );
}

function formatResult(result: unknown): string {
  if (typeof result === 'string') return result;
  try {
    return JSON.stringify(result, null, 2);
  } catch {
    return String(result);
  }
}
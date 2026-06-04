// 渲染 tool_use 事件 — 工具调用（可折叠，带图标）
import type { ToolUseEvent } from '../../../core/ws/types';
import { useState } from 'react';

const TOOL_ICONS: Record<string, string> = {
  Read: '📂', Grep: '🔍', Glob: '🔍', Search: '🔍',
  Edit: '✏️', Write: '✏️', Replace: '✏️', Create: '✏️',
  Bash: '▶️', Run: '▶️',
  Task: '📋', TodoWrite: '📝',
};

function toolIcon(name: string): string {
  for (const [key, icon] of Object.entries(TOOL_ICONS)) {
    if (name.toLowerCase().includes(key.toLowerCase())) return icon;
  }
  return '🔧';
}

export function ToolUseCard({ event }: { event: ToolUseEvent }) {
  const [expanded, setExpanded] = useState(false);
  const icon = toolIcon(event.toolName);
  const summary = formatToolSummary(event.toolName, event.toolInput);

  return (
    <article
      className="card card-tool"
      onClick={() => setExpanded(!expanded)}
      role="button"
      tabIndex={0}
    >
      <header className="card-header">
        <span className="card-badge">{icon} tool</span>
        <span className="tool-name">{event.toolName}</span>
        {summary && <span className="tool-summary">{summary}</span>}
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

function formatToolSummary(_name: string, input: unknown): string {
  if (typeof input !== 'object' || !input) return '';
  const obj = input as Record<string, unknown>;
  if (obj.file_path || obj.path) return String(obj.file_path || obj.path);
  if (obj.command) return String(obj.command).slice(0, 50);
  if (obj.pattern) return String(obj.pattern);
  if (obj.description) return String(obj.description).slice(0, 40);
  return '';
}
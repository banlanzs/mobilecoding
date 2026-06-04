// 渲染 tool_result 事件 — 工具结果（可折叠，带状态指示）
import type { ToolResultEvent } from '../../../core/ws/types';
import { useState } from 'react';
import { DiffView } from './DiffView';

export function ToolResultCard({ event }: { event: ToolResultEvent }) {
  const [expanded, setExpanded] = useState(false);

  const isError = isErrorResult(event.toolResult);
  const resultText = formatResult(event.toolResult);
  const shortResult = resultText.length > 80 ? resultText.slice(0, 80) + '…' : resultText;
  const statusIcon = isError ? '❌' : '✅';

  // 尝试提取 diff
  const diff = extractDiff(event.toolResult);

  return (
    <article
      className={`card card-result ${isError ? 'card-result-error' : ''}`}
      onClick={() => setExpanded(!expanded)}
      role="button"
      tabIndex={0}
    >
      <header className="card-header">
        <span className="card-badge">{statusIcon} result</span>
        <span className="tool-name">{event.toolName}</span>
        <span className="result-summary">{shortResult}</span>
        <span style={{ marginLeft: 'auto', color: '#565f89', fontSize: 11 }}>
          {expanded ? '▼' : '▶'}
        </span>
      </header>
      {expanded && (
        <div className="result-body">
          {diff ? <DiffView oldStr={diff.oldStr} newStr={diff.newStr} /> : <pre>{resultText}</pre>}
        </div>
      )}
    </article>
  );
}

function formatResult(result: unknown): string {
  if (typeof result === 'string') return result;
  try { return JSON.stringify(result, null, 2); } catch { return String(result); }
}

function isErrorResult(result: unknown): boolean {
  const s = typeof result === 'string' ? result : JSON.stringify(result);
  return /error|failed|rejected|denied/i.test(s);
}

function extractDiff(result: unknown): { oldStr: string; newStr: string } | null {
  if (typeof result === 'string') {
    // Claude 的 file edit 结果格式：The file ... has been updated. 通常没有 diff
    // 但有时会包含 diff 格式文本
    const lines = result.split('\n');
    const diffLines = lines.filter(l => l.startsWith('+') || l.startsWith('-'));
    if (diffLines.length > 2) {
      const added = diffLines.filter(l => l.startsWith('+')).map(l => l.slice(1)).join('\n');
      const removed = diffLines.filter(l => l.startsWith('-')).map(l => l.slice(1)).join('\n');
      return { oldStr: removed, newStr: added };
    }
  }
  return null;
}
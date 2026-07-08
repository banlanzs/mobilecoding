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

  // 尝试提取 unified diff 文本
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
        <span style={{ marginLeft: 'auto', color: 'var(--mc-meta)', fontSize: 11 }}>
          {expanded ? '▼' : '▶'}
        </span>
      </header>
      {expanded && (
        <div className="result-body">
          {diff ? <DiffView diff={diff} /> : <pre>{resultText}</pre>}
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

// extractDiff 从工具结果中提取 unified diff 文本。
// 只识别真正的 unified diff：必须含 @@ hunk 头，且其后有 +/- 行。
// 避免把 markdown 列表（- item）或算术表达式误判为 diff。
function extractDiff(result: unknown): string | null {
  if (typeof result !== 'string') return null;
  const lines = result.split('\n');

  // 必须存在 @@ hunk 头才算 unified diff
  const hunkStart = lines.findIndex((l) => /^@@ -\d+/.test(l));
  if (hunkStart < 0) return null;

  // 从第一个 hunk 头开始截取，到 diff 内容结束
  // unified diff 行：以 +/-/空格 开头，或 @@ hunk 头，或 \ No newline
  const diffLines: string[] = [];
  for (let i = hunkStart; i < lines.length; i++) {
    const l = lines[i];
    if (/^@@ -\d+/.test(l) || /^[+\- ]/.test(l) || l === '' || /^\\ No newline/.test(l)) {
      diffLines.push(l);
    } else {
      // 遇到非 diff 行，结束
      break;
    }
  }

  // 至少要有 1 个 +/- 行才算有效 diff
  const hasChange = diffLines.some((l) => l.startsWith('+') || l.startsWith('-'));
  if (!hasChange) return null;

  return diffLines.join('\n');
}

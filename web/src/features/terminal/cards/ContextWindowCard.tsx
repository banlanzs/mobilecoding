// 渲染 context_window 事件 — 上下文窗口用量
import type { ContextWindowEvent } from '../../../core/ws/types';

export function ContextWindowCard({ event }: { event: ContextWindowEvent }) {
  const { used, max } = extractTokens(event.toolInput);
  const pct = max > 0 ? Math.min(100, (used / max) * 100) : 0;

  return (
    <article className="card card-context">
      <header className="card-header">
        <span className="card-badge">context</span>
        <span style={{ color: '#9ece6a' }}>
          {used.toLocaleString()} / {max.toLocaleString()} tokens
        </span>
      </header>
      <div className="ctx-meter">
        <div className="ctx-meter-fill" style={{ width: `${pct}%` }} />
      </div>
    </article>
  );
}

function extractTokens(data: unknown): { used: number; max: number } {
  if (!data || typeof data !== 'object') return { used: 0, max: 0 };
  const obj = data as Record<string, unknown>;
  const used = numberOrZero(obj.usedTokens ?? obj.used ?? obj.tokens);
  const max = numberOrZero(obj.maxTokens ?? obj.max ?? obj.contextWindow);
  return { used, max };
}

function numberOrZero(v: unknown): number {
  if (typeof v === 'number') return v;
  if (typeof v === 'string') {
    const n = parseInt(v, 10);
    return isNaN(n) ? 0 : n;
  }
  return 0;
}
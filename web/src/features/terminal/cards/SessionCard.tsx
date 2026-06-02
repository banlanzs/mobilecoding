// 渲染 session 事件
import type { SessionEvent } from '../../../core/ws/types';

export function SessionCard({ event }: { event: SessionEvent }) {
  return (
    <article className="card card-session">
      <header className="card-header">
        <span className="card-badge">session</span>
      </header>
      <div className="session-msg">{formatSession(event.toolInput)}</div>
    </article>
  );
}

function formatSession(data: unknown): string {
  if (!data) return '(空)';
  if (typeof data === 'string') return data;
  if (typeof data === 'object') {
    const obj = data as Record<string, unknown>;
    if (obj.status && typeof obj.status === 'string') return obj.status;
  }
  try {
    return JSON.stringify(data);
  } catch {
    return String(data);
  }
}
// 事件分发器：按 type 路由到对应 card
import type { DisplayMessage, UserMessage } from '../../core/ws/types';
import { TextCard } from './cards/TextCard';
import { ToolUseCard } from './cards/ToolUseCard';
import { ToolResultCard } from './cards/ToolResultCard';
import { PermissionCard } from './cards/PermissionCard';
import { PlanModeCard } from './cards/PlanModeCard';
import { ContextWindowCard } from './cards/ContextWindowCard';
import { LifecycleCard } from './cards/LifecycleCard';
import { SessionCard } from './cards/SessionCard';

export function EventCard({ event }: { event: DisplayMessage }) {
  // 用户消息（前端合成）
  if ((event as UserMessage).type === 'user') {
    const user = event as UserMessage;
    return (
      <article className="card card-user">
        <header className="card-header">
          <span className="card-badge">you</span>
          <span style={{ color: '#565f89', fontSize: 11 }}>
            {formatTime(user.time)}
          </span>
        </header>
        <pre>{user.text}</pre>
      </article>
    );
  }

  switch (event.type) {
    case 'text':
      return <TextCard event={event} />;
    case 'tool_use':
      return <ToolUseCard event={event} />;
    case 'tool_result':
      return <ToolResultCard event={event} />;
    case 'permission_request':
      return <PermissionCard event={event} />;
    case 'plan_mode':
      return <PlanModeCard event={event} />;
    case 'context_window':
      return <ContextWindowCard event={event} />;
    case 'lifecycle':
      return <LifecycleCard event={event} />;
    case 'session':
      return <SessionCard event={event} />;
    default:
      return null;
  }
}

function formatTime(iso: string): string {
  try {
    const d = new Date(iso);
    return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
  } catch {
    return '';
  }
}
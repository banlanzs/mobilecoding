// 事件分发器：按 type 路由到对应 card
import type { DisplayMessage, UserMessage } from '../../core/ws/types';
import { TextCard } from './cards/TextCard';
import { ToolUseCard } from './cards/ToolUseCard';
import { ToolResultCard } from './cards/ToolResultCard';
import { PlanModeCard } from './cards/PlanModeCard';
import { ContextWindowCard } from './cards/ContextWindowCard';
import { LifecycleCard } from './cards/LifecycleCard';
import { SessionCard } from './cards/SessionCard';
import { ToolEventCard } from './cards/ToolEventCard';

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
    case 'text_delta':
      return <TextCard event={event} />;
    case 'tool_use':
      return <ToolUseCard event={event} />;
    case 'tool_result':
      return <ToolResultCard event={event} />;
    case 'permission_request':
    case 'permission_ask':
      return <PermissionStatus event={event} />;
    case 'plan_mode':
      return <PlanModeCard event={event} />;
    case 'context_window':
      return <ContextWindowCard event={event} />;
    case 'lifecycle':
      return <LifecycleCard event={event} />;
    case 'session':
      return <SessionCard event={event} />;
    case 'thinking_start':
    case 'thinking_end':
    case 'tool_start':
    case 'tool_end':
    case 'bash_start':
    case 'bash_end':
    case 'agent_state':
      return <ToolEventCard event={event} />;
    case 'turn_end':
      return null; // 整轮结束是控制信号，不在前端展示
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

// 内联权限状态：仅显示工具名 + 结果状态，不显示交互按钮（交互由底部 banner 处理）
function PermissionStatus({ event }: { event: DisplayMessage }) {
  const ev = event as any;
  const resolved = ev.resolved as 'allowed' | 'denied' | undefined;
  const badge = resolved === 'allowed'
    ? <span className="permission-status-badge badge-allowed">✓ 已允许</span>
    : resolved === 'denied'
    ? <span className="permission-status-badge badge-denied">✗ 已拒绝</span>
    : <span className="permission-status-badge">等待授权</span>;
  return (
    <article className="card card-permission-status">
      <span className="permission-status-icon">🔐</span>
      <span className="permission-status-tool">{ev.toolName}</span>
      <span className="permission-status-msg">{ev.message || '请求执行工具操作'}</span>
      {badge}
    </article>
  );
}
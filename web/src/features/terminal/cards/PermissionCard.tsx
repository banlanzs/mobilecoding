// 渲染 permission_request 事件 — 权限请求（高亮）
import type { PermissionRequestEvent } from '../../../core/ws/types';

export function PermissionCard({ event }: { event: PermissionRequestEvent }) {
  // 注意：当前后端尚未实现 session.permission.answer，UI 仅展示请求
  return (
    <article className="card card-permission">
      <header className="card-header">
        <span className="card-badge">⚠ 权限请求</span>
        <span className="tool-name" style={{ color: '#f7768e' }}>
          {event.toolName}
        </span>
      </header>
      <div className="permission-msg">{event.message || '请求执行工具操作'}</div>
      <div className="permission-actions">
        <button
          className="btn-allow"
          disabled
          title="后端尚未实现 session.permission.answer 方法"
        >
          Allow
        </button>
        <button
          className="btn-deny"
          disabled
          title="后端尚未实现 session.permission.answer 方法"
        >
          Deny
        </button>
      </div>
    </article>
  );
}
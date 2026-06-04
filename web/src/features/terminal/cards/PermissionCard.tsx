// 渲染 permission_request 事件 — 权限请求（高亮）
import { useState } from 'react';
import type { PermissionRequestEvent, PermissionAskEvent } from '../../../core/ws/types';
import { useChat } from '../../../core/state/ChatContext';

type PermissionEvent = PermissionRequestEvent | PermissionAskEvent;

export function PermissionCard({ event }: { event: PermissionEvent }) {
  const { answerPermission } = useChat();
  const [answered, setAnswered] = useState(false);

  // permission_ask 事件携带 messageId 作为 Claude stdio control_request 的 request_id
  const requestId = event.type === 'permission_ask' ? event.messageId : undefined;

  const handleAnswer = async (allow: boolean) => {
    setAnswered(true);
    try {
      await answerPermission(allow, event.toolName, requestId);
    } catch {
      setAnswered(false);
    }
  };

  return (
    <article className="card card-permission">
      <header className="card-header">
        <span className="card-badge">⚠ 权限请求</span>
        <span className="tool-name" style={{ color: '#f7768e' }}>
          {event.toolName}
        </span>
        {answered && (
          <span style={{ color: '#565f89', fontSize: 11, marginLeft: 'auto' }}>已应答</span>
        )}
      </header>
      <div className="permission-msg">{event.message || '请求执行工具操作'}</div>
      {!answered && (
        <div className="permission-actions">
          <button className="btn-allow" onClick={() => handleAnswer(true)}>
            Allow
          </button>
          <button className="btn-deny" onClick={() => handleAnswer(false)}>
            Deny
          </button>
        </div>
      )}
    </article>
  );
}
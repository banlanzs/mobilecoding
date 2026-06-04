// 渲染 permission_request 事件 — 权限请求（高亮）
import { useState } from 'react';
import type { PermissionRequestEvent } from '../../../core/ws/types';
import { useChat } from '../../../core/state/ChatContext';

export function PermissionCard({ event }: { event: PermissionRequestEvent }) {
  const { answerPermission } = useChat();
  const [answered, setAnswered] = useState(false);

  const handleAnswer = async (allow: boolean) => {
    setAnswered(true);
    try {
      await answerPermission(allow, event.toolName);
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
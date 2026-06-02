// 渲染 text 事件 — assistant 文本消息
import type { TextEvent } from '../../../core/ws/types';
import { useState } from 'react';

export function TextCard({ event }: { event: TextEvent }) {
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(event.text);
      setCopied(true);
      setTimeout(() => setCopied(false), 1200);
    } catch {
      // 忽略剪贴板失败
    }
  };

  return (
    <article className="card card-text">
      <header className="card-header">
        <span className="card-badge">assistant</span>
        <span style={{ color: '#565f89', fontSize: 11 }}>
          {formatTime(event.time)}
        </span>
      </header>
      <pre>{event.text}</pre>
      <div className="card-actions">
        <button onClick={copy} aria-label="copy">
          {copied ? '✓ 已复制' : 'copy'}
        </button>
      </div>
    </article>
  );
}

function formatTime(iso: string): string {
  try {
    const d = new Date(iso);
    return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
  } catch {
    return '';
  }
}
// 渲染 text 事件 — assistant 文本消息，支持简易 Markdown
import type { TextEvent } from '../../../core/ws/types';
import { useState } from 'react';

// 简易 Markdown 渲染
function renderMarkdown(text: string): string {
  if (!text) return '';
  let result = text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');
  // 代码块
  result = result.replace(/```(\w*)\n?([\s\S]*?)```/g, (_, lang, code) =>
    `<pre><code class="${lang || ''}">${code}</code></pre>`);
  // 行内代码
  result = result.replace(/`([^`]+)`/g, '<code>$1</code>');
  // 粗体
  result = result.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
  // 换行
  result = result.replace(/\n/g, '<br>');
  return result;
}

export function TextCard({ event }: { event: TextEvent }) {
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(event.text);
      setCopied(true);
      setTimeout(() => setCopied(false), 1200);
    } catch {
      // ignore
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
      <div className="markdown-body"
           dangerouslySetInnerHTML={{ __html: renderMarkdown(event.text) }} />
      <div className="card-actions">
        <button onClick={copy} aria-label="copy">
          {copied ? '已复制' : 'copy'}
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
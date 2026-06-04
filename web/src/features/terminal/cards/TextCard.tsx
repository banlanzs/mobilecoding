// 渲染 text 事件 — assistant 文本消息，支持完整 Markdown + thinking 折叠
import type { TextEvent, TextDeltaEvent } from '../../../core/ws/types';
import { useState, useMemo } from 'react';
import { marked } from 'marked';

marked.setOptions({
  breaks: true,
  gfm: true,
});

function renderMarkdown(text: string): string {
  if (!text) return '';
  try {
    let html = marked.parse(text) as string;
    // 将 table 包裹在滚动容器中，适配移动端
    html = html.replace(/<table>/g, '<div class="table-wrapper"><table>');
    html = html.replace(/<\/table>/g, '</table></div>');
    return html;
  } catch {
    return text
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/\n/g, '<br>');
  }
}

export function TextCard({ event }: { event: TextEvent | TextDeltaEvent }) {
  const [copied, setCopied] = useState(false);
  const [thinkingOpen, setThinkingOpen] = useState(false);
  const html = useMemo(() => renderMarkdown(event.text), [event.text]);
  const thinkingHtml = useMemo(
    () => (event.thinking ? renderMarkdown(event.thinking) : ''),
    [event.thinking]
  );
  const hasThinking = !!event.thinking;

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
        <button
          className="btn-copy"
          onClick={copy}
          aria-label="复制回复"
        >
          {copied ? '已复制' : '📋'}
        </button>
      </header>

      {hasThinking && (
        <div className="thinking-section">
          <button
            className="thinking-toggle"
            onClick={() => setThinkingOpen(!thinkingOpen)}
          >
            <span className="thinking-icon">{thinkingOpen ? '▼' : '▶'}</span>
            <span className="thinking-label">思考过程</span>
            <span className="thinking-hint">
              {thinkingOpen ? '点击收起' : '点击展开'}
            </span>
          </button>
          {thinkingOpen && (
            <div
              className="thinking-content"
              dangerouslySetInnerHTML={{ __html: thinkingHtml }}
            />
          )}
        </div>
      )}

      <div className="markdown-body"
           dangerouslySetInnerHTML={{ __html: html }} />
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
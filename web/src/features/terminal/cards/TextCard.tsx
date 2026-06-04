// 渲染 text 事件 — assistant 文本消息，支持完整 Markdown + thinking 折叠 + 打字机动画
import type { TextEvent, TextDeltaEvent } from '../../../core/ws/types';
import { useState, useMemo, useEffect, useRef } from 'react';
import { marked } from 'marked';

marked.setOptions({
  breaks: true,
  gfm: true,
});

function renderMarkdown(text: string): string {
  if (!text) return '';
  try {
    let html = marked.parse(text) as string;
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

// 打字机动画：逐字揭示文本
function useTypewriter(text: string, isDelta: boolean): string {
  const [revealed, setRevealed] = useState(isDelta ? text : '');
  const prevFull = useRef('');

  useEffect(() => {
    // text_delta 模式下直接全量显示（已经是增量的）
    if (isDelta) {
      setRevealed(text);
      prevFull.current = text;
      return;
    }

    // 文本未变化，跳过
    if (text === prevFull.current) return;
    prevFull.current = text;

    // 短文本直接显示
    if (text.length < 60) {
      setRevealed(text);
      return;
    }

    // 长文本：打字机动画（~80 步，约 2 秒完成）
    setRevealed(text.slice(0, 3)); // 先显示前几个字，避免瞬间空白
    let i = 1;
    const totalSteps = 80;
    const charsPerStep = Math.max(1, Math.ceil(text.length / totalSteps));
    const timer = setInterval(() => {
      i++;
      const end = Math.min(i * charsPerStep, text.length);
      setRevealed(text.slice(0, end));
      if (end >= text.length) {
        clearInterval(timer);
      }
    }, 25);
    return () => clearInterval(timer);
  }, [text, isDelta]);

  // 确保最终显示完整文本
  if (revealed.length < text.length && !isDelta && text.length < 80) {
    return text;
  }
  return revealed || text;
}

export function TextCard({ event }: { event: TextEvent | TextDeltaEvent }) {
  const [copied, setCopied] = useState(false);
  const [thinkingOpen, setThinkingOpen] = useState(false);
  const isDelta = event.type === 'text_delta';

  const displayText = useTypewriter(event.text, isDelta);
  const html = useMemo(() => renderMarkdown(displayText), [displayText]);
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
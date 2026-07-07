// 渲染 text 事件 — assistant 文本消息，支持完整 Markdown + thinking 折叠 + 代码块复制
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

export function TextCard({ event }: { event: TextEvent | TextDeltaEvent }) {
  const [copied, setCopied] = useState(false);
  const [thinkingOpen, setThinkingOpen] = useState(false);
  const bodyRef = useRef<HTMLDivElement>(null);

  const text = event.text || '';
  const html = useMemo(() => renderMarkdown(text), [text]);
  const thinkingHtml = useMemo(
    () => (event.thinking ? renderMarkdown(event.thinking) : ''),
    [event.thinking]
  );
  const hasThinking = !!event.thinking;

  // 渲染 markdown 后，给每个代码块包一层 header（语言标签 + 复制按钮）。
  // 手动管理 innerHTML：React 不控制 bodyRef 内容，避免 dangerouslySetInnerHTML 与
  // DOM 后处理冲突。流式 delta 时 html 随 text 增长频繁变化，每次重建后重新加 header。
  useEffect(() => {
    const root = bodyRef.current;
    if (!root) return;
    root.innerHTML = html;
    root.querySelectorAll<HTMLPreElement>('pre').forEach((pre) => {
      const code = pre.querySelector('code');
      const langMatch = code?.className.match(/language-([\w-]+)/);
      const lang = langMatch?.[1] || '';

      const wrap = document.createElement('div');
      wrap.className = 'code-block';

      const header = document.createElement('div');
      header.className = 'code-block-header';

      const langLabel = document.createElement('span');
      langLabel.className = 'code-lang';
      langLabel.textContent = lang || 'text';

      const copyBtn = document.createElement('button');
      copyBtn.type = 'button';
      copyBtn.className = 'btn-copy-code';
      copyBtn.textContent = '复制';

      header.appendChild(langLabel);
      header.appendChild(copyBtn);
      pre.parentNode!.insertBefore(wrap, pre);
      wrap.appendChild(header);
      wrap.appendChild(pre);
    });
  }, [html]);

  // 事件委托：复制按钮点击 → 读取同 code-block 下的 code 文本
  const onBodyClick = (e: React.MouseEvent<HTMLDivElement>) => {
    const btn = (e.target as HTMLElement).closest<HTMLButtonElement>('.btn-copy-code');
    if (!btn) return;
    const block = btn.closest('.code-block');
    const codeText = block?.querySelector('code')?.textContent || '';
    navigator.clipboard
      .writeText(codeText)
      .then(() => {
        btn.textContent = '已复制';
        setTimeout(() => {
          btn.textContent = '复制';
        }, 1200);
      })
      .catch(() => {
        // ignore
      });
  };

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(text);
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
        <span style={{ color: 'var(--mc-meta)', fontSize: 11 }}>
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

      <div
        className="markdown-body"
        ref={bodyRef}
        onClick={onBodyClick}
      />
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

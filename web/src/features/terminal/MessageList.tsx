// 消息列表：自动滚动 + 触底检测
import { useRef, useEffect, useState } from 'react';
import { useChat } from '../../core/state/ChatContext';
import { EventCard } from './EventCard';

export function MessageList() {
  const { state } = useChat();
  const listRef = useRef<HTMLDivElement>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);
  const stuckAtBottom = useRef(true);
  const [showScrollBtn, setShowScrollBtn] = useState(false);

  useEffect(() => {
    const el = listRef.current;
    if (!el) return;
    const onScroll = () => {
      const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 80;
      stuckAtBottom.current = nearBottom;
      setShowScrollBtn(!nearBottom);
    };
    el.addEventListener('scroll', onScroll, { passive: true });
    return () => el.removeEventListener('scroll', onScroll);
  }, []);

  useEffect(() => {
    if (stuckAtBottom.current) {
      sentinelRef.current?.scrollIntoView({ block: 'end' });
    } else {
      setShowScrollBtn(true);
    }
  }, [state.messages.length]);

  const scrollToBottom = () => {
    sentinelRef.current?.scrollIntoView({ block: 'end' });
    stuckAtBottom.current = true;
    setShowScrollBtn(false);
  };

  if (state.messages.length === 0) {
    return (
      <div className="empty-state">
        <h2>mobilecoding</h2>
        <p>
          选择一个 AI 引擎，点击 Start 开始会话。
          <br />
          输入消息后按 Enter 发送，Shift+Enter 换行。
        </p>
      </div>
    );
  }

  return (
    <div className="message-list" ref={listRef}>
      {state.messages.map((msg, i) => (
        <EventCard key={i} event={msg} />
      ))}
      <div ref={sentinelRef} />

      {showScrollBtn && (
        <button className="scroll-bottom" onClick={scrollToBottom} aria-label="scroll to bottom">
          ↓
        </button>
      )}
    </div>
  );
}
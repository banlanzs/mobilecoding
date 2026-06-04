// 渲染 lifecycle 事件 — 会话生命周期
import type { LifecycleEvent } from '../../../core/ws/types';

// 内部消息前缀，不在 UI 中展示完整文本
const INTERNAL_PREFIXES = ['thinking:', 'cmd:', 'ready:', 'started:', 'exited'];

function isInternal(msg: string): boolean {
  return INTERNAL_PREFIXES.some((p) => msg.startsWith(p));
}

export function LifecycleCard({ event }: { event: LifecycleEvent }) {
  // 隐藏旧的内部消息格式
  if (isInternal(event.message)) {
    return null;
  }

  return (
    <article className="card card-lifecycle">
      <div className="lifecycle-msg">{event.message}</div>
    </article>
  );
}
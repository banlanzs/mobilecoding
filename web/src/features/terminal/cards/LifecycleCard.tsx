// 渲染 lifecycle 事件 — 会话生命周期
import type { LifecycleEvent } from '../../../core/ws/types';

// 内部消息前缀/内容，不在 UI 中展示
const INTERNAL_PREFIXES = ['thinking:', 'cmd:', 'ready:', 'started:', 'exited', 'turn_end'];
const INTERNAL_EXACT = ['思考中…'];

function isInternal(msg: string): boolean {
  return INTERNAL_PREFIXES.some((p) => msg.startsWith(p))
    || INTERNAL_EXACT.includes(msg);
}

export function LifecycleCard({ event }: { event: LifecycleEvent }) {
  // 隐藏旧的内部消息格式 + 控制信号（turn_end 是控制信号，用于切换按钮状态）
  if (isInternal(event.message)) {
    return null;
  }

  return (
    <article className="card card-lifecycle">
      <div className="lifecycle-msg">{event.message}</div>
    </article>
  );
}
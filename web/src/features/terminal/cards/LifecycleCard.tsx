// 渲染 lifecycle 事件 — 会话生命周期
import type { LifecycleEvent } from '../../../core/ws/types';

export function LifecycleCard({ event }: { event: LifecycleEvent }) {
  return (
    <article className="card card-lifecycle">
      <div className="lifecycle-msg">— {event.message} —</div>
    </article>
  );
}
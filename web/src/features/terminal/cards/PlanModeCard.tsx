// 渲染 plan_mode 事件
import type { PlanModeEvent } from '../../../core/ws/types';

export function PlanModeCard({ event }: { event: PlanModeEvent }) {
  return (
    <article className="card card-plan">
      <header className="card-header">
        <span className="card-badge">plan</span>
        <span style={{ color: '#e0af68' }}>计划模式</span>
      </header>
      <div className="plan-body">
        <pre>{formatPlan(event.toolInput)}</pre>
      </div>
    </article>
  );
}

function formatPlan(data: unknown): string {
  if (!data) return '(空)';
  if (typeof data === 'string') return data;
  try {
    return JSON.stringify(data, null, 2);
  } catch {
    return String(data);
  }
}
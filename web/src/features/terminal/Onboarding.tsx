// 首次使用引导页：显示连接信息、Agent 列表、快速开始
import { useState, useEffect } from 'react';

interface Agent {
  id: string;
  name: string;
  command: string;
  description: string;
}

export function Onboarding({ token }: { token: string }) {
  const [agents, setAgents] = useState<Agent[]>([]);

  useEffect(() => {
    fetch('/api/v1/agents')
      .then(r => r.json())
      .then(data => setAgents(data || []))
      .catch(() => {});
  }, []);

  return (
    <div className="onboarding">
      <div className="onboarding-header">
        <h1>📱 mobilecoding</h1>
        <p className="onboarding-subtitle">手机端 Claude Code / Codex 客户端</p>
      </div>

      <div className="onboarding-card">
        <h2>🚀 快速开始</h2>
        <ol>
          <li>电脑终端运行 <code>mc claude</code> 或 <code>mobilecoding</code></li>
          <li>手机扫码连接（已在 URL 中）</li>
          <li>在终端输入消息，手机端自动同步</li>
        </ol>
      </div>

      <div className="onboarding-card">
        <h2>🤖 支持的 Agent</h2>
        <div className="agent-grid">
          {agents.map(agent => (
            <div key={agent.id} className="agent-item">
              <span className="agent-name">{agent.name}</span>
              <span className="agent-cmd">{agent.command}</span>
            </div>
          ))}
          {agents.length === 0 && (
            <p className="agent-empty">加载中...</p>
          )}
        </div>
      </div>

      <div className="onboarding-card">
        <h2>🔐 连接信息</h2>
        <div className="connect-info">
          <p>Token: <code>{token.slice(0, 12)}...</code></p>
          <p>状态: <span className="status-ok">已连接</span></p>
        </div>
      </div>

      <div className="onboarding-footer">
        <p>终端输入消息后按 Enter 发送，手机端自动同步对话</p>
      </div>
    </div>
  );
}

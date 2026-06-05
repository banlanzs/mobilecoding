// 会话控制栏：command + model 选择 + settings 下拉 + start/stop 按钮
import { useEffect, useState, useCallback } from 'react';
import { useChat } from '../../core/state/ChatContext';

const COMMANDS = [
  { value: 'claude', label: 'Claude' },
  { value: 'codex', label: 'Codex' },
  { value: 'opencode', label: 'OpenCode' },
  { value: 'aichat', label: 'Aichat' },
];

interface ModelOption {
  value: string;
  label: string;
}

interface ClaudeSetting {
  name: string;
  path: string;
}

export function SessionBar() {
  const { state, sendStart, sendStop } = useChat();
  const [command, setCommand] = useState('claude');
  const [model, setModel] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // 模型列表（从服务端拉取）
  const [models, setModels] = useState<ModelOption[]>([{ value: '', label: '默认模型' }]);
  // Claude 配置文件
  const [claudeSettings, setClaudeSettings] = useState<ClaudeSetting[]>([]);
  const [selectedSetting, setSelectedSetting] = useState<string>('');

  useEffect(() => {
    if (state.runtime.defaultCommand) {
      setCommand(state.runtime.defaultCommand);
    }
  }, [state.runtime.defaultCommand]);

  useEffect(() => {
    setError(null);
  }, [command, model]);

  // 拉取模型列表（可指定 settings 路径）
  const fetchModels = useCallback(async (settingsPath?: string) => {
    try {
      const token = localStorage.getItem('mobilecoding.token');
      const url = settingsPath
        ? `/api/v1/models?settings=${encodeURIComponent(settingsPath)}`
        : '/api/v1/models';
      const res = await fetch(url, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        const data: ModelOption[] = await res.json();
        setModels(data);
      }
    } catch {
      // 保持默认列表
    }
  }, []);

  // 连接后拉取 Claude 配置列表
  const fetchClaudeSettings = useCallback(async () => {
    try {
      const token = localStorage.getItem('mobilecoding.token');
      const res = await fetch('/api/v1/claude-settings', {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        const data: ClaudeSetting[] = await res.json();
        setClaudeSettings(data);
        if (data.length > 0) {
          setSelectedSetting(data[0].path);
        }
      }
    } catch {
      // 忽略错误，不显示配置下拉
    }
  }, []);

  useEffect(() => {
    if (state.status === 'connected' && state.connectionMode === 'direct') {
      fetchClaudeSettings();
      fetchModels();
    }
  }, [state.status, state.connectionMode, fetchClaudeSettings, fetchModels]);

  // 切换 settings 时重新拉取对应模型列表
  useEffect(() => {
    if (selectedSetting) {
      fetchModels(selectedSetting);
    }
  }, [selectedSetting, fetchModels]);

  const handleStart = async () => {
    setLoading(true);
    setError(null);
    try {
      let args: string[] = [];

      if (model) {
        args.push('--model', model);
      }

      if (command === 'claude' && selectedSetting) {
        args.push('--settings', selectedSetting);
      } else if (command === state.runtime.defaultCommand) {
        args = [...args, ...state.runtime.defaultArgs];
      }

      await sendStart({ command, args });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start');
      console.error('start session failed:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleStop = async () => {
    setError(null);
    try {
      await sendStop();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to stop');
      console.error('stop session failed:', err);
    }
  };

  // Relay 模式下：只显示停止按钮
  if (state.connectionMode === 'relay') {
    return (
      <div className="session-bar relay-mode">
        {error && <div className="session-error">{error}</div>}
        <div className="relay-indicator">
          <span className="relay-dot" />
          Relay Connected
        </div>
        <button className="btn btn-danger" onClick={handleStop}>
          Disconnect
        </button>
      </div>
    );
  }

  // 有活跃会话时：只显示 Stop 按钮
  if (state.sessionId) {
    return (
      <div className="session-bar">
        {error && <div className="session-error">{error}</div>}
        <span className="session-active">{command}{model ? ` (${model})` : ''} — active</span>
        <button className="btn btn-danger" onClick={handleStop}>
          Stop
        </button>
      </div>
    );
  }

  // 无会话但已连接：遥控器模式（被动监控终端 Claude）
  if (state.status === 'connected') {
    return (
      <div className="session-bar">
        <span className="session-active">🔗 遥控器模式 — 终端 Claude 已连接</span>
      </div>
    );
  }

  // 无会话且未连接：显示完整选择界面
  return (
    <div className="session-bar">
      {error && <div className="session-error">{error}</div>}

      <select
        value={command}
        onChange={(e) => setCommand(e.target.value)}
        disabled={loading}
        className="sel-command"
      >
        {COMMANDS.map((c) => (
          <option key={c.value} value={c.value}>
            {c.label}
          </option>
        ))}
      </select>

      {/* 模型选择 — 仅 Claude 时显示 */}
      {command === 'claude' && (
        <select
          value={model}
          onChange={(e) => setModel(e.target.value)}
          disabled={loading}
          className="sel-model"
          title="选择 AI 模型"
        >
          {models.map((m) => (
            <option key={m.value} value={m.value}>
              {m.label}
            </option>
          ))}
        </select>
      )}

      {/* Claude 配置选择 */}
      {command === 'claude' && claudeSettings.length > 0 && (
        <select
          value={selectedSetting}
          onChange={(e) => setSelectedSetting(e.target.value)}
          disabled={loading}
          title="选择 Claude 配置文件"
          className="sel-settings"
        >
          {claudeSettings.map((s) => (
            <option key={s.path} value={s.path}>
              {s.name}
            </option>
          ))}
        </select>
      )}

      <div className="session-actions">
        <button
          className="btn btn-primary"
          onClick={handleStart}
          disabled={loading}
        >
          {loading ? '启动中…' : 'Start'}
        </button>
      </div>
    </div>
  );
}
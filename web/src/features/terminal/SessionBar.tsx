// 会话控制栏：command 选择 + settings 下拉 + start/stop 按钮
import { useEffect, useState, useCallback } from 'react';
import { useChat } from '../../core/state/ChatContext';

const COMMANDS = [
  { value: 'claude', label: 'Claude' },
  { value: 'codex', label: 'Codex' },
  { value: 'opencode', label: 'OpenCode' },
  { value: 'aichat', label: 'Aichat' },
];

interface ClaudeSetting {
  name: string;
  path: string;
}

export function SessionBar() {
  const { state, sendStart, sendStop } = useChat();
  const [command, setCommand] = useState('claude');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

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
  }, [command]);

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
    }
  }, [state.status, state.connectionMode, fetchClaudeSettings]);

  const handleStart = async () => {
    setLoading(true);
    setError(null);
    try {
      let args: string[] | undefined;

      if (command === 'claude' && selectedSetting) {
        // 使用选中的 Claude 配置文件
        args = ['--settings', selectedSetting];
      } else if (command === state.runtime.defaultCommand) {
        args = state.runtime.defaultArgs;
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
        <span className="session-active">{command} (active)</span>
        <button className="btn btn-danger" onClick={handleStop}>
          Stop
        </button>
      </div>
    );
  }

  // 无会话：显示完整的 CLI 选择界面
  return (
    <div className="session-bar">
      {error && <div className="session-error">{error}</div>}
      <select
        value={command}
        onChange={(e) => setCommand(e.target.value)}
        disabled={loading}
      >
        {COMMANDS.map((c) => (
          <option key={c.value} value={c.value}>
            {c.label}
          </option>
        ))}
      </select>

      {/* Claude 配置选择 */}
      {command === 'claude' && claudeSettings.length > 0 && (
        <select
          value={selectedSetting}
          onChange={(e) => setSelectedSetting(e.target.value)}
          disabled={loading}
          title="选择 Claude 配置文件"
        >
          {claudeSettings.map((s) => (
            <option key={s.path} value={s.path}>
              {s.name}
            </option>
          ))}
        </select>
      )}

      <button
        className="btn btn-primary"
        onClick={handleStart}
        disabled={loading || state.status !== 'connected'}
      >
        {loading ? '启动中…' : 'Start'}
      </button>
    </div>
  );
}

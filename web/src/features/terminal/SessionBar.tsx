// 会话控制栏：command/model/settings 选择 + 上下文进度 + start/stop
import { useEffect, useState, useCallback, useRef, useMemo } from 'react';
import { useChat } from '../../core/state/ChatContext';
import {
  argsWithModel,
  concreteModelOptions,
  modelFromArgs,
  modelSwitchCommand,
  type ModelOption,
} from './sessionControls';

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

interface SessionBarProps {
  onBack?: () => void;
  currentSessionId?: string; // 预留给未来的会话恢复功能
  onToggleFiles?: () => void;
  showFiles?: boolean;
}

export function SessionBar({ onBack, currentSessionId, onToggleFiles, showFiles }: SessionBarProps) {
  const { state, sendStart, sendStop, sendInput, setSelectedCommand } = useChat();
  const [command, setCommand] = useState('claude');
  const [model, setModel] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // 双重确认停止状态
  const [confirmStop, setConfirmStop] = useState(false);
  const confirmTimeoutRef = useRef<number | null>(null);

  // 模型列表（从服务端拉取）
  const [models, setModels] = useState<ModelOption[]>([{ value: '', label: '默认模型' }]);
  // Claude 配置文件
  const [claudeSettings, setClaudeSettings] = useState<ClaudeSetting[]>([]);
  const [selectedSetting, setSelectedSetting] = useState<string>('');

  useEffect(() => {
    const nextCommand = state.selectedCommand || state.runtime.defaultCommand;
    if (nextCommand) {
      setCommand(nextCommand);
    }
  }, [state.runtime.defaultCommand, state.selectedCommand]);

  const selectedModel = model ?? modelFromArgs(state.runtime.defaultArgs || []);
  const hotSwitchModels = useMemo(() => concreteModelOptions(models), [models]);

  useEffect(() => {
    if (state.runtime.launchMode !== 'remote-control') return;
    if (selectedModel || hotSwitchModels.length === 0) return;
    setModel(hotSwitchModels[0].value);
  }, [state.runtime.launchMode, selectedModel, hotSwitchModels]);

  useEffect(() => {
    setError(null);
  }, [command, selectedModel]);

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
      setSelectedCommand(command);
      let args = argsWithModel([], selectedModel);

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

  const handleApplyRemoteModel = async () => {
    if (loading || state.stopping || !selectedModel || hotSwitchModels.length === 0) return;

    setLoading(true);
    setError(null);
    try {
      await sendInput(modelSwitchCommand(selectedModel));
    } catch (err) {
      setError('切换模型失败');
      console.error('apply remote model failed:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleStop = async () => {
    if (state.stopping) return;

    // 双重确认机制
    if (!confirmStop) {
      // 第一次点击：进入确认模式
      setConfirmStop(true);
      // 5 秒后自动取消确认
      if (confirmTimeoutRef.current) {
        clearTimeout(confirmTimeoutRef.current);
      }
      confirmTimeoutRef.current = setTimeout(() => {
        setConfirmStop(false);
      }, 5000);
      return;
    }

    // 第二次点击：真正停止
    setConfirmStop(false);
    if (confirmTimeoutRef.current) {
      clearTimeout(confirmTimeoutRef.current);
      confirmTimeoutRef.current = null;
    }

    setError(null);
    try {
      await sendStop();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to stop');
      console.error('stop session failed:', err);
    }
  };

  // 清理超时定时器
  useEffect(() => {
    return () => {
      if (confirmTimeoutRef.current) {
        clearTimeout(confirmTimeoutRef.current);
      }
    };
  }, []);

  const handleCommandChange = (next: string) => {
    setCommand(next);
    setSelectedCommand(next);
    if (next !== 'claude') {
      setModel('');
    }
  };

  // 上下文进度条：仅在收到真实 context_window 事件后显示（不放假数据）
  const ctx = state.contextWindow;
  const ctxPct = ctx && ctx.max > 0 ? Math.min(100, Math.round((ctx.used / ctx.max) * 100)) : null;
  const contextMeter = ctxPct !== null ? (
    <div className="context-window">
      <span className="context-label">上下文 {ctxPct}%</span>
      <div className="progress-track">
        <div className="progress-bar" style={{ width: `${ctxPct}%` }} />
      </div>
    </div>
  ) : null;

  if (state.readOnly) {
    return (
      <div className="session-bar session-bar-active">
        {onBack && (
          <button className="btn-back" onClick={onBack} title="返回会话列表">
            ←
          </button>
        )}
        {error && <div className="session-error">{error}</div>}
        <span className="session-active" title={currentSessionId || state.viewedSessionId || ''}>
          历史会话，只读
        </span>
      </div>
    );
  }

  // Relay 模式：指示 + 断开
  if (state.connectionMode === 'relay') {
    return (
      <div className="session-bar">
        {onBack && (
          <button className="btn-back" onClick={onBack} title="返回会话列表">
            ←
          </button>
        )}
        {error && <div className="session-error">{error}</div>}
        <div className="relay-indicator">
          <span className="relay-dot" />
          Relay Connected
        </div>
        <div className="session-actions">
          <button
            className={`btn btn-danger${confirmStop ? ' btn-confirm-stop' : ''}`}
            onClick={handleStop}
            disabled={state.stopping}
            title={confirmStop ? '再次点击确认断开' : '断开连接'}
          >
            {state.stopping ? 'Stopping...' : confirmStop ? '⚠️ 确认断开？' : 'Disconnect'}
          </button>
        </div>
      </div>
    );
  }

  // 有活跃会话：会话信息 + 上下文进度 + Stop
  if (state.sessionId) {
    const activeLabel = `${command}${selectedModel ? ` (${selectedModel})` : ''} — active`;
    return (
      <div className="session-bar session-bar-active">
        {onBack && (
          <button className="btn-back" onClick={onBack} title="返回会话列表">
            ←
          </button>
        )}
        {error && <div className="session-error">{error}</div>}
        <span className="session-active" title={activeLabel}>
          {activeLabel}
        </span>
        {contextMeter}
        <div className="session-actions">
          {state.runtime.launchMode === 'remote-control' && command === 'claude' && (
            <>
              <select
                className="sel-model"
                value={selectedModel}
                onChange={(e) => setModel(e.target.value)}
                disabled={loading || state.stopping || hotSwitchModels.length === 0}
                title="选择 Claude 模型"
              >
                {hotSwitchModels.map((m) => (
                  <option key={m.value} value={m.value}>
                    {m.label}
                  </option>
                ))}
              </select>
              <button
                className="btn btn-primary"
                onClick={handleApplyRemoteModel}
                disabled={loading || state.stopping || !selectedModel || hotSwitchModels.length === 0}
                title="通过 Claude Code /model 热切换当前会话模型"
              >
                {loading ? '切换中…' : '切换模型'}
              </button>
            </>
          )}
          {onToggleFiles && (
            <button
              className={`btn-files${showFiles ? ' active' : ''}`}
              onClick={onToggleFiles}
              title="文件变更"
            >
              📁
            </button>
          )}
          <button
            className={`btn btn-danger${confirmStop ? ' btn-confirm-stop' : ''}`}
            onClick={handleStop}
            disabled={state.stopping}
            title={confirmStop ? '再次点击确认停止' : '停止会话'}
          >
            {state.stopping ? 'Stopping...' : confirmStop ? '⚠️ 确认停止？' : 'Stop'}
          </button>
        </div>
      </div>
    );
  }

  // mc claude 启动的 server：无托管 session，只被动监控本地终端 CLI
  if (state.status === 'connected' && state.runtime.launchMode === 'remote-control') {
    return (
      <div className="session-bar">
        {onBack && (
          <button className="btn-back" onClick={onBack} title="返回会话列表">
            ←
          </button>
        )}
        <span className="session-active">🔗 遥控器模式 — 终端 CLI 已连接</span>
      </div>
    );
  }

  // mobilecoding 托管模式：连接后仍显示完整选择界面，由手机端启动 session
  return (
    <div className="session-bar session-bar-setup">
      {onBack && (
        <button className="btn-back" onClick={onBack} title="返回会话列表">
          ←
        </button>
      )}
      {error && <div className="session-error">{error}</div>}

      <select
        className="sel-command"
        value={command}
        onChange={(e) => handleCommandChange(e.target.value)}
        disabled={loading}
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
          className="sel-model"
          value={selectedModel}
          onChange={(e) => setModel(e.target.value)}
          disabled={loading}
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
          className="sel-settings"
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

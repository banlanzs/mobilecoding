// 会话控制栏：command 选择 + start/stop 按钮
import { useEffect, useState } from 'react';
import { useChat } from '../../core/state/ChatContext';

const COMMANDS = [
  { value: 'claude', label: 'Claude' },
  { value: 'codex', label: 'Codex' },
  { value: 'opencode', label: 'OpenCode' },
  { value: 'aichat', label: 'Aichat' },
];

export function SessionBar() {
  const { state, sendStart, sendStop } = useChat();
  const [command, setCommand] = useState('claude');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (state.runtime.defaultCommand) {
      setCommand(state.runtime.defaultCommand);
    }
  }, [state.runtime.defaultCommand]);

  const handleStart = async () => {
    setLoading(true);
    try {
      await sendStart({
        command,
        args: command === state.runtime.defaultCommand ? state.runtime.defaultArgs : undefined,
      });
    } catch (err) {
      console.error('start session failed:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleStop = async () => {
    try {
      await sendStop();
    } catch (err) {
      console.error('stop session failed:', err);
    }
  };

  return (
    <div className="session-bar">
      <select
        value={command}
        onChange={(e) => setCommand(e.target.value)}
        disabled={!!state.sessionId || loading}
      >
        {COMMANDS.map((c) => (
          <option key={c.value} value={c.value}>
            {c.label}
          </option>
        ))}
      </select>

      {!state.sessionId ? (
        <button
          className="btn btn-primary"
          onClick={handleStart}
          disabled={loading || state.status !== 'connected'}
        >
          {loading ? '启动中…' : 'Start'}
        </button>
      ) : (
        <button className="btn btn-danger" onClick={handleStop}>
          Stop
        </button>
      )}
    </div>
  );
}

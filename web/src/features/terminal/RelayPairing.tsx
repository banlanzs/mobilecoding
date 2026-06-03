// Relay 配对组件：输入配对码连接到 CLI
import { useState } from 'react';
import { useChat } from '../../core/state/ChatContext';

interface RelayPairingProps {
  onConnected?: () => void;
}

export function RelayPairing({ onConnected }: RelayPairingProps) {
  const { state, connectRelay } = useChat();
  const [relayUrl, setRelayUrl] = useState('ws://localhost:8443');
  const [sessionId, setSessionId] = useState('');
  const [pairingSecret, setPairingSecret] = useState('');
  const [connecting, setConnecting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleConnect = async () => {
    if (!sessionId || !pairingSecret) {
      setError('Please enter session ID and pairing secret');
      return;
    }

    setConnecting(true);
    setError(null);

    try {
      connectRelay({
        relayUrl,
        sessionId,
        pairingSecret,
      });
      onConnected?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Connection failed');
    } finally {
      setConnecting(false);
    }
  };

  // 如果已连接，显示状态
  if (state.connectionMode === 'relay' && state.status === 'connected') {
    return (
      <div className="relay-pairing connected">
        <div className="relay-status">
          <span className="status-dot connected" />
          Connected to relay (Session: {state.sessionId})
        </div>
      </div>
    );
  }

  return (
    <div className="relay-pairing">
      <h3>Connect via Relay</h3>
      <p className="relay-description">
        Enter the pairing information from your CLI to connect remotely.
      </p>

      {error && <div className="relay-error">{error}</div>}

      <div className="relay-form">
        <label>
          Relay URL
          <input
            type="text"
            value={relayUrl}
            onChange={(e) => setRelayUrl(e.target.value)}
            placeholder="ws://localhost:8443"
            disabled={connecting}
          />
        </label>

        <label>
          Session ID
          <input
            type="text"
            value={sessionId}
            onChange={(e) => setSessionId(e.target.value)}
            placeholder="rs_xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
            disabled={connecting}
          />
        </label>

        <label>
          Pairing Secret
          <input
            type="text"
            value={pairingSecret}
            onChange={(e) => setPairingSecret(e.target.value)}
            placeholder="Enter pairing secret from CLI"
            disabled={connecting}
          />
        </label>

        <button
          className="btn btn-primary"
          onClick={handleConnect}
          disabled={connecting || !sessionId || !pairingSecret}
        >
          {connecting ? 'Connecting...' : 'Connect'}
        </button>
      </div>
    </div>
  );
}

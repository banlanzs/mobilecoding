const status = document.getElementById('conn-status');
const output = document.getElementById('output');
let ws = null;
let reqId = 0;

function setStatus(s) { status.textContent = s; }

function connect() {
  const token = document.getElementById('token').value.trim();
  if (!token) { alert('token required'); return; }
  const proto = location.protocol === 'https:' ? 'wss' : 'ws';
  const url = `${proto}://${location.host}/api/v1/ws?token=${encodeURIComponent(token)}`;
  ws = new WebSocket(url);
  ws.onopen = () => {
    setStatus('online');
    document.getElementById('login').hidden = true;
    document.getElementById('session').hidden = false;
  };
  ws.onclose = () => { setStatus('offline'); };
  ws.onerror = () => { setStatus('error'); };
  ws.onmessage = (ev) => {
    const env = JSON.parse(ev.data);
    if (env.type === 'evt') {
      const e = JSON.parse(env.event || '{}');
      if (e.type === 'text') {
        output.textContent += e.text + '\n';
      } else if (e.type === 'lifecycle') {
        output.textContent += `[${e.message}]\n`;
      }
    } else if (env.type === 'resp' && !env.ok) {
      output.textContent += `[error ${env.error?.code}: ${env.error?.message}]\n`;
    }
  };
}

function send(method, params = {}) {
  if (!ws || ws.readyState !== 1) return;
  reqId += 1;
  ws.send(JSON.stringify({ type: 'req', id: `r${reqId}`, method, params }));
}

document.getElementById('connect').onclick = connect;
document.getElementById('start').onclick = () => {
  const command = document.getElementById('cmd').value.trim();
  const argsRaw = document.getElementById('args').value.trim();
  const args = argsRaw ? argsRaw.split(',').map(s => s.trim()) : [];
  send('session.start', { command, args });
};
document.getElementById('send').onclick = () => {
  const text = document.getElementById('input').value + '\n';
  send('session.input', { text });
};
document.getElementById('stop').onclick = () => send('session.stop');

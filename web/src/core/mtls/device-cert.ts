import { saveDeviceCert, loadDeviceCert } from './cert-storage';

export async function requestDeviceCert(): Promise<{ cert: string; key: string }> {
  const existing = await loadDeviceCert();
  if (existing) {
    return existing;
  }

  const res = await fetch('/api/v1/device-cert', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ device_name: `device-${Date.now()}` }),
  });

  if (!res.ok) {
    throw new Error('Failed to get device cert');
  }

  const { cert, key } = await res.json();
  await saveDeviceCert(cert, key);
  return { cert, key };
}
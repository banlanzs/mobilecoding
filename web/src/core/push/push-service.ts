const PUBLIC_VAPID_KEY = 'BEl62iUYgUivxIkv69yViA6hMFPLKR3MYR2Bj4Sj1L2rFliB'; // placeholder

export async function registerPush(): Promise<PushSubscription | null> {
  if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
    return null;
  }
  try {
    const reg = await navigator.serviceWorker.ready;
    return await reg.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: urlBase64ToUint8Array(PUBLIC_VAPID_KEY),
    });
  } catch (err) {
    console.error('Push registration failed:', err);
    return null;
  }
}

function urlBase64ToUint8Array(base64String: string): ArrayBuffer {
  const padding = '='.repeat((4 - base64String.length % 4) % 4);
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
  const rawData = window.atob(base64);
  const uint8Array = new Uint8Array([...rawData].map(char => char.charCodeAt(0)));
  return uint8Array.buffer as ArrayBuffer;
}

export async function getPushSubscription(): Promise<PushSubscription | null> {
  if (!('serviceWorker' in navigator)) return null;
  const reg = await navigator.serviceWorker.ready;
  return await reg.pushManager.getSubscription();
}
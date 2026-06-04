const DB_NAME = 'mobilecoding-sessions';
const STORE_NAME = 'events';
const MAX_EVENTS = 50;

export async function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, 1);
    request.onerror = () => reject(request.error);
    request.onsuccess = () => resolve(request.result);
    request.onupgradeneeded = (event) => {
      const db = (event.target as IDBOpenDBRequest).result;
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        db.createObjectStore(STORE_NAME, { keyPath: 'id', autoIncrement: true });
      }
    };
  });
}

export async function saveEvent(event: any): Promise<void> {
  const db = await openDB();
  const tx = db.transaction(STORE_NAME, 'readwrite');
  const store = tx.objectStore(STORE_NAME);

  return new Promise((resolve, reject) => {
    const addRequest = store.add({ ...event, timestamp: Date.now() });

    addRequest.onsuccess = () => {
      // Keep only last MAX_EVENTS
      const countRequest = store.count();
      countRequest.onsuccess = () => {
        const count = countRequest.result;
        if (count > MAX_EVENTS) {
          const cursorRequest = store.openCursor();
          cursorRequest.onsuccess = () => {
            const cursor = cursorRequest.result;
            if (cursor) {
              cursor.delete();
            }
          };
        }
      };
    };

    tx.oncomplete = () => resolve();
    tx.onerror = () => reject(tx.error);
  });
}

export async function getRecentEvents(): Promise<any[]> {
  const db = await openDB();
  const tx = db.transaction(STORE_NAME, 'readonly');
  const store = tx.objectStore(STORE_NAME);
  const events: any[] = [];

  return new Promise((resolve, reject) => {
    const request = store.openCursor(null, 'prev');
    request.onsuccess = () => {
      const cursor = request.result;
      if (cursor && events.length < MAX_EVENTS) {
        events.push(cursor.value);
        cursor.continue();
      } else {
        resolve(events.reverse());
      }
    };
    request.onerror = () => reject(request.error);
  });
}

export async function clearEvents(): Promise<void> {
  const db = await openDB();
  const tx = db.transaction(STORE_NAME, 'readwrite');
  await tx.objectStore(STORE_NAME).clear();
}

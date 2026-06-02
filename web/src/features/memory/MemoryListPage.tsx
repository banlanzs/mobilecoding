import { useEffect, useState } from 'react';
import { MemoryCard } from './MemoryCard';
import { MemoryEditor } from './MemoryEditor';

interface Memory {
  name: string;
  content: string;
}

export function MemoryListPage() {
  const [memories, setMemories] = useState<Memory[]>([]);
  const [editing, setEditing] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch('/api/v1/memory')
      .then(res => res.json())
      .then(data => {
        setMemories(data || []);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  const handleSave = async (name: string, content: string) => {
    await fetch(`/api/v1/memory/${name}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content }),
    });
    setMemories(memories.map(m => m.name === name ? { ...m, content } : m));
    setEditing(null);
  };

  if (loading) return <div>Loading...</div>;

  return (
    <div>
      <h2>Memory</h2>
      {memories.map(memory => (
        editing === memory.name ? (
          <MemoryEditor
            key={memory.name}
            memory={memory}
            onSave={handleSave}
            onCancel={() => setEditing(null)}
          />
        ) : (
          <MemoryCard
            key={memory.name}
            memory={memory}
            onEdit={() => setEditing(memory.name)}
          />
        )
      ))}
    </div>
  );
}

import { useState } from 'react';

interface Memory {
  name: string;
  content: string;
}

export function MemoryEditor({ memory, onSave, onCancel }: {
  memory: Memory;
  onSave: (name: string, content: string) => void;
  onCancel: () => void;
}) {
  const [content, setContent] = useState(memory.content);

  return (
    <div className="memory-editor">
      <h3>{memory.name}</h3>
      <textarea
        value={content}
        onChange={e => setContent(e.target.value)}
        rows={10}
      />
      <div>
        <button onClick={() => onSave(memory.name, content)}>Save</button>
        <button onClick={onCancel}>Cancel</button>
      </div>
    </div>
  );
}
interface Memory {
  name: string;
  content: string;
}

export function MemoryCard({ memory, onEdit }: { memory: Memory; onEdit: () => void }) {
  return (
    <div className="memory-card">
      <h3>{memory.name}</h3>
      <pre>{memory.content}</pre>
      <button onClick={onEdit}>Edit</button>
    </div>
  );
}

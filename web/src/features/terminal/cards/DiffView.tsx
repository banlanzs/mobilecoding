// 简易 unified diff 渲染器（移动端适配）
interface DiffViewProps {
  oldStr: string;
  newStr: string;
}

export function DiffView({ oldStr, newStr }: DiffViewProps) {
  const lines = computeDiff(oldStr, newStr);

  return (
    <div className="diff-view">
      {lines.map((line, i) => (
        <div key={i} className={`diff-line diff-${line.type}`}>
          <span className="diff-prefix">{line.prefix}</span>
          <span className="diff-text">{line.text}</span>
        </div>
      ))}
    </div>
  );
}

function computeDiff(oldStr: string, newStr: string): DiffLine[] {
  const oldLines = oldStr.split('\n');
  const newLines = newStr.split('\n');
  const result: DiffLine[] = [];

  // 简易逐行比较
  const maxLen = Math.max(oldLines.length, newLines.length);
  for (let i = 0; i < maxLen; i++) {
    if (i < oldLines.length && i < newLines.length) {
      if (oldLines[i] === newLines[i]) {
        result.push({ type: 'same', prefix: ' ', text: oldLines[i] });
      } else {
        if (oldLines[i]) result.push({ type: 'del', prefix: '-', text: oldLines[i] });
        if (newLines[i]) result.push({ type: 'add', prefix: '+', text: newLines[i] });
      }
    } else if (i < oldLines.length) {
      result.push({ type: 'del', prefix: '-', text: oldLines[i] });
    } else {
      result.push({ type: 'add', prefix: '+', text: newLines[i] });
    }
  }

  return result.slice(0, 200); // 限制行数
}

interface DiffLine {
  type: 'same' | 'add' | 'del';
  prefix: string;
  text: string;
}
// Unified diff 渲染器：解析 git diff 输出，按 hunk 分组渲染 +/ /- 行
interface DiffViewProps {
  diff: string; // unified diff 文本（git diff 输出格式）
  maxLines?: number; // 渲染上限，默认 300
}

interface DiffLine {
  type: 'add' | 'del' | 'context' | 'hunk' | 'meta';
  prefix: string;
  text: string;
}

export function DiffView({ diff, maxLines = 300 }: DiffViewProps) {
  const { lines, truncated, total } = parseUnifiedDiff(diff, maxLines);

  if (total === 0) {
    return <div className="diff-view diff-empty">无变更内容</div>;
  }

  return (
    <div className="diff-view">
      {lines.map((line, i) => (
        <div key={i} className={`diff-line diff-${line.type}`}>
          <span className="diff-prefix">{line.prefix}</span>
          <span className="diff-text">{line.text}</span>
        </div>
      ))}
      {truncated && (
        <div className="diff-truncated">显示前 {maxLines} 行，共 {total} 行</div>
      )}
    </div>
  );
}

// parseUnifiedDiff 解析 unified diff 文本。
// 格式：
//   diff --git a/foo b/foo        <- meta（文件头）
//   index abc..def 100644          <- meta
//   @@ -10,5 +10,7 @@              <- hunk 头
//    context line                  <- context（空格前缀）
//   -removed line                  <- del
//   +added line                    <- add
// 非 diff 行（无前缀）当作 meta 跳过，避免误判。
function parseUnifiedDiff(diff: string, maxLines: number): { lines: DiffLine[]; truncated: boolean; total: number } {
  if (!diff) return { lines: [], truncated: false, total: 0 };

  const rawLines = diff.split('\n');
  const result: DiffLine[] = [];
  let inHunk = false;
  let total = 0;

  for (const raw of rawLines) {
    // 文件头 / index 行等元信息
    if (raw.startsWith('diff --git') || raw.startsWith('index ') || raw.startsWith('--- ') || raw.startsWith('+++ ')) {
      result.push({ type: 'meta', prefix: ' ', text: raw });
      continue;
    }
    // hunk 头：@@ -10,5 +10,7 @@
    if (raw.startsWith('@@')) {
      inHunk = true;
      result.push({ type: 'hunk', prefix: '@', text: raw });
      total++;
      continue;
    }
    if (!inHunk) {
      // hunk 之前的非元信息行（如 "diff --git" 之后的空行）跳过
      continue;
    }
    // hunk 内：按首字符判断
    if (raw.startsWith('+')) {
      result.push({ type: 'add', prefix: '+', text: raw.slice(1) });
      total++;
    } else if (raw.startsWith('-')) {
      result.push({ type: 'del', prefix: '-', text: raw.slice(1) });
      total++;
    } else if (raw.startsWith(' ')) {
      result.push({ type: 'context', prefix: ' ', text: raw.slice(1) });
      total++;
    } else if (raw === '') {
      // 空行当作 context（git diff 中空行无前缀）
      result.push({ type: 'context', prefix: ' ', text: '' });
      total++;
    }
    // 其他行（如 "\ No newline at end of file"）跳过
  }

  const truncated = result.length > maxLines;
  return { lines: truncated ? result.slice(0, maxLines) : result, truncated, total };
}

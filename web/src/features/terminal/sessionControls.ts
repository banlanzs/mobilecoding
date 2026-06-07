export interface ModelOption {
  value: string;
  label: string;
}

export function modelFromArgs(args: string[]): string {
  const idx = args.indexOf('--model');
  return idx >= 0 && idx + 1 < args.length ? args[idx + 1] : '';
}

export function argsWithModel(args: string[], model: string): string[] {
  const next: string[] = [];
  for (let i = 0; i < args.length; i++) {
    if (args[i] === '--model') {
      i++;
      continue;
    }
    next.push(args[i]);
  }
  if (model) {
    next.unshift('--model', model);
  }
  return next;
}

export function concreteModelOptions(models: ModelOption[]): ModelOption[] {
  return models.filter((model) => model.value.trim() !== '');
}

export function modelSwitchCommand(model: string): string {
  const trimmed = model.trim();
  if (!trimmed) {
    throw new Error('请选择具体模型');
  }
  return `/model ${trimmed}`;
}

export function requireActiveSessionId(sessionId: string | null | undefined): string {
  if (!sessionId) {
    throw new Error('桌面 CLI 未就绪，请确认 mc claude 会话仍在运行');
  }
  return sessionId;
}

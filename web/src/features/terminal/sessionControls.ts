export interface ModelOption {
  value: string;
  label: string;
}

type ConnectionMode = 'direct' | 'relay';
type LaunchMode = 'managed' | 'remote-control';
type MaybeLaunchMode = LaunchMode | undefined;
type RuntimeReady<T extends { defaultCommand?: string }> = T & { defaultCommand: string };

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

export function requireActiveSessionId(sessionId: string | null | undefined): string {
  if (!sessionId) {
    throw new Error('桌面 CLI 未就绪，请确认 mc claude 会话仍在运行');
  }
  return sessionId;
}

export function shouldRefreshRemoteControlSession(connectionMode: ConnectionMode, launchMode: LaunchMode): boolean {
  return connectionMode === 'direct' && launchMode === 'remote-control';
}

export function shouldAppendUserMessageAfterSend(connectionMode: ConnectionMode, launchMode: LaunchMode): boolean {
  return connectionMode === 'direct' && launchMode === 'remote-control';
}

export function isRemoteCliNotReady(
  connectionMode: ConnectionMode,
  launchMode: MaybeLaunchMode,
  sessionId: string | null,
): boolean {
  return connectionMode === 'direct' && launchMode === 'remote-control' && !sessionId;
}

export function requireRuntimeReady<T extends { defaultCommand?: string }>(runtime: T): RuntimeReady<T> {
  if (!runtime.defaultCommand) {
    throw new Error('运行时未就绪，请稍后重试');
  }
  return runtime as RuntimeReady<T>;
}

export function sessionIdForDirectSend({
  launchMode,
  currentSessionId,
  refreshedSessionId,
}: {
  launchMode: LaunchMode;
  currentSessionId: string | null | undefined;
  refreshedSessionId?: string | null;
}): string {
  if (launchMode === 'remote-control') {
    return requireActiveSessionId(refreshedSessionId);
  }
  if (!currentSessionId) {
    throw new Error('请先启动会话');
  }
  return currentSessionId;
}

import test from 'node:test';
import assert from 'node:assert/strict';

import {
  argsWithModel,
  isRemoteCliNotReady,
  modelFromArgs,
  requireRuntimeReady,
  shouldAppendUserMessageAfterSend,
  sessionIdForDirectSend,
  shouldRefreshRemoteControlSession,
  requireActiveSessionId,
} from './sessionControls.ts';

test('modelFromArgs reads --model value from args', () => {
  assert.equal(modelFromArgs(['--settings', 'c:/claude/settings.json', '--model', 'claude-sonnet-4-6']), 'claude-sonnet-4-6');
});

test('argsWithModel replaces an existing --model instead of appending another one', () => {
  assert.deepEqual(
    argsWithModel(['--settings', 'c:/claude/settings.json', '--model', 'old-model'], 'new-model'),
    ['--model', 'new-model', '--settings', 'c:/claude/settings.json'],
  );
});

test('argsWithModel removes --model when model is empty', () => {
  assert.deepEqual(
    argsWithModel(['--model', 'old-model', '--settings', 'c:/claude/settings.json'], ''),
    ['--settings', 'c:/claude/settings.json'],
  );
});


test('requireActiveSessionId rejects placeholder-free remote-control input without active session', () => {
  assert.throws(
    () => requireActiveSessionId(null),
    /桌面 CLI 未就绪/,
  );
  assert.throws(
    () => requireActiveSessionId(''),
    /桌面 CLI 未就绪/,
  );
});

test('requireActiveSessionId returns the real active session id', () => {
  assert.equal(requireActiveSessionId('sess_real_123'), 'sess_real_123');
});

test('requireActiveSessionId never fabricates remote_control placeholder session id', () => {
  assert.notEqual(requireActiveSessionId('sess_real_456'), 'remote_control');
  assert.throws(
    () => requireActiveSessionId(undefined),
    /桌面 CLI 未就绪/,
  );
});

test('shouldRefreshRemoteControlSession only refreshes direct remote-control sends', () => {
  assert.equal(shouldRefreshRemoteControlSession('direct', 'remote-control'), true);
  assert.equal(shouldRefreshRemoteControlSession('relay', 'remote-control'), false);
  assert.equal(shouldRefreshRemoteControlSession('direct', 'managed'), false);
});

test('isRemoteCliNotReady only blocks direct remote-control without session', () => {
  assert.equal(isRemoteCliNotReady('direct', 'remote-control', null), true);
  assert.equal(isRemoteCliNotReady('relay', 'remote-control', null), false);
  assert.equal(isRemoteCliNotReady('direct', 'remote-control', 'sess_real_123'), false);
  assert.equal(isRemoteCliNotReady('direct', 'managed', null), false);
  assert.equal(isRemoteCliNotReady('direct', undefined, null), false);
});

test('requireRuntimeReady fails closed when runtime config is still missing', () => {
  assert.equal(requireRuntimeReady({ defaultCommand: 'claude', launchMode: 'remote-control' }).launchMode, 'remote-control');
  assert.throws(
    () => requireRuntimeReady({ defaultCommand: '', launchMode: 'remote-control' }),
    /运行时未就绪/,
  );
});

test('shouldAppendUserMessageAfterSend only delays direct remote-control user message display', () => {
  assert.equal(shouldAppendUserMessageAfterSend('direct', 'remote-control'), true);
  assert.equal(shouldAppendUserMessageAfterSend('direct', 'managed'), false);
  assert.equal(shouldAppendUserMessageAfterSend('relay', 'remote-control'), false);
});

test('sessionIdForDirectSend requires fresh remote-control session instead of stale cached session', () => {
  assert.equal(
    sessionIdForDirectSend({
      launchMode: 'remote-control',
      currentSessionId: 'sess_stale_123',
      refreshedSessionId: 'sess_real_456',
    }),
    'sess_real_456',
  );
  assert.throws(
    () => sessionIdForDirectSend({
      launchMode: 'remote-control',
      currentSessionId: 'sess_stale_123',
      refreshedSessionId: null,
    }),
    /桌面 CLI 未就绪/,
  );
});

test('sessionIdForDirectSend reports managed mode without fabricating session', () => {
  assert.throws(
    () => sessionIdForDirectSend({ launchMode: 'managed', currentSessionId: null }),
    /请先启动会话/,
  );
});

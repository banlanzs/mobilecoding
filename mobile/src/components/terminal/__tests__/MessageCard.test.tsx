import React from 'react'
import { render } from '@testing-library/react-native'
import { MessageCard } from '../MessageCard'

test('renders user message card', () => {
  const screen = render(
    <MessageCard message={{ type: 'user', text: 'Hello', sessionId: 's1', time: '2026-06-08T00:00:00Z' }} />
  )
  expect(screen.getByText('用户')).toBeTruthy()
  expect(screen.getByText('Hello')).toBeTruthy()
})

test('renders assistant text card with thinking', () => {
  const screen = render(
    <MessageCard
      message={{
        type: 'text',
        text: 'Response',
        thinking: 'Analyzing...',
        sessionId: 's1',
        time: '2026-06-08T00:00:00Z'
      } as any}
    />
  )
  expect(screen.getByText('助手')).toBeTruthy()
  expect(screen.getByText('Response')).toBeTruthy()
  expect(screen.getByText('Analyzing...')).toBeTruthy()
})

test('renders tool_use card', () => {
  const screen = render(
    <MessageCard
      message={{
        type: 'tool_use',
        toolName: 'Bash',
        toolInput: { command: 'ls' },
        sessionId: 's1',
        time: '2026-06-08T00:00:00Z'
      } as any}
    />
  )
  expect(screen.getByText('工具调用: Bash')).toBeTruthy()
})

test('renders permission_request card', () => {
  const screen = render(
    <MessageCard
      message={{
        type: 'permission_request',
        toolName: 'Edit',
        message: 'Allow edit?',
        sessionId: 's1',
        time: '2026-06-08T00:00:00Z'
      } as any}
    />
  )
  expect(screen.getByText('权限请求: Edit')).toBeTruthy()
  expect(screen.getByText('Allow edit?')).toBeTruthy()
})

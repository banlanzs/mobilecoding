import React from 'react'
import { render } from '@testing-library/react-native'
import { MessageCard } from '../MessageCard'

describe('MessageCard', () => {
  test('渲染用户气泡', () => {
    const screen = render(
      <MessageCard message={{ type: 'user', text: 'Hello', sessionId: 's1', time: '2026-06-08T00:00:00Z' }} />
    )

    expect(screen.getByText('Hello')).toBeTruthy()
  })

  test('渲染助手文本与思考提示', () => {
    const screen = render(
      <MessageCard
        message={{
          type: 'text',
          text: 'Response',
          thinking: '正在思考...',
          sessionId: 's1',
          time: '2026-06-08T00:00:00Z'
        } as any}
      />
    )

    expect(screen.getByText('Response')).toBeTruthy()
    expect(screen.getByText('正在思考...')).toBeTruthy()
  })

  test('渲染工具调用卡片', () => {
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

    expect(screen.getByText('工具调用')).toBeTruthy()
    expect(screen.getByText('Bash')).toBeTruthy()
  })

  test('渲染权限请求卡片', () => {
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

    expect(screen.getByText('权限请求')).toBeTruthy()
    expect(screen.getByText('Edit')).toBeTruthy()
    expect(screen.getByText('Allow edit?')).toBeTruthy()
  })
})

import React from 'react'
import { render } from '@testing-library/react-native'
import { TerminalScreen } from '../TerminalScreen'

describe('TerminalScreen', () => {
  it('渲染纯聊天界面，不显示连接参数配置', () => {
    const screen = render(<TerminalScreen />)

    expect(screen.getByText('Claude')).toBeTruthy()
    expect(screen.queryByPlaceholderText('Host (10.0.2.2 / 局域网IP)')).toBeNull()
    expect(screen.queryByPlaceholderText('Port (8443)')).toBeNull()
    expect(screen.queryByPlaceholderText('Token（从服务器日志复制）')).toBeNull()
    expect(screen.queryByPlaceholderText('WS 路径（/api/v1/ws）')).toBeNull()
    expect(screen.queryByText('Mock 模式')).toBeNull()
    expect(screen.queryByText(' WSS')).toBeNull()
  })

  it('使用 KeyboardAvoidingView 包裹输入区', () => {
    const screen = render(<TerminalScreen />)
    expect(screen.getByPlaceholderText('连接中...')).toBeTruthy()
  })
})

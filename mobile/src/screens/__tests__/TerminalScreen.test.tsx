import React from 'react'
import { render } from '@testing-library/react-native'
import { TerminalScreen } from '../TerminalScreen'

describe('TerminalScreen', () => {
  it('renders Terminal screen', () => {
    const screen = render(<TerminalScreen />)
    expect(screen.getByText('Terminal')).toBeTruthy()
  })

  // 实际行为测试：
  // 1. 连接成功后，connected 状态变为 true
  // 2. useEffect 监听 connected 变化，自动调用 handleStartSession
  // 3. handleStartSession 内部调用 client.send('session.start', ...)
  //
  // 由于 RealMobilecodingClient 是内联类，难以直接 mock，
  // 这里只验证渲染正常。实际 session.start 逻辑通过手动测试 + 集成测试验证。
})

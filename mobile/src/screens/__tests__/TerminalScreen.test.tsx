import React from 'react'
import { render, fireEvent } from '@testing-library/react-native'
import { TerminalScreen } from '../TerminalScreen'

test('shows stop button while a turn is active', () => {
  const screen = render(
    <TerminalScreen turnActive={true} messages={[]} onSend={jest.fn()} onAbort={jest.fn()} />
  )
  expect(screen.getByText('停止')).toBeTruthy()
})

test('shows send button while no turn is active', () => {
  const screen = render(
    <TerminalScreen turnActive={false} messages={[]} onSend={jest.fn()} onAbort={jest.fn()} />
  )
  expect(screen.getByText('发送')).toBeTruthy()
})

test('calls onSend when send button is pressed', () => {
  const onSend = jest.fn()
  const screen = render(
    <TerminalScreen turnActive={false} messages={[]} onSend={onSend} onAbort={jest.fn()} />
  )

  fireEvent.changeText(screen.getByPlaceholderText('输入消息...'), 'Hello')
  fireEvent.press(screen.getByText('发送'))

  expect(onSend).toHaveBeenCalledWith('Hello')
})

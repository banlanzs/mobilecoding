import React from 'react'
import { render } from '@testing-library/react-native'
import { TerminalScreen } from '../TerminalScreen'

test('renders Terminal screen', () => {
  const screen = render(<TerminalScreen />)
  expect(screen.getByText('Terminal')).toBeTruthy()
})
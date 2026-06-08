import React from 'react'
import { View, Text } from 'react-native'
import type { DisplayMessage } from '../../stores/useMessageStore'

interface MessageCardProps {
  message: DisplayMessage
}

export function MessageCard({ message }: MessageCardProps) {
  if (message.type === 'user') {
    return (
      <View style={{ padding: 12, margin: 8, backgroundColor: '#e3f2fd', borderRadius: 8 }}>
        <Text style={{ fontWeight: '600', marginBottom: 4 }}>用户</Text>
        <Text>{message.text}</Text>
      </View>
    )
  }

  if (message.type === 'text' || message.type === 'text_delta') {
    return (
      <View style={{ padding: 12, margin: 8, backgroundColor: '#f5f5f5', borderRadius: 8 }}>
        <Text style={{ fontWeight: '600', marginBottom: 4 }}>助手</Text>
        <Text>{message.text}</Text>
        {message.thinking && (
          <Text style={{ marginTop: 8, fontStyle: 'italic', color: '#666' }}>{message.thinking}</Text>
        )}
      </View>
    )
  }

  if (message.type === 'tool_use' || message.type === 'tool_start') {
    const m = message as any
    return (
      <View style={{ padding: 12, margin: 8, backgroundColor: '#fff3e0', borderRadius: 8 }}>
        <Text style={{ fontWeight: '600', marginBottom: 4 }}>工具调用: {m.toolName}</Text>
        {m.toolInput && (
          <Text style={{ fontSize: 12, color: '#666' }}>{JSON.stringify(m.toolInput, null, 2)}</Text>
        )}
      </View>
    )
  }

  if (message.type === 'tool_result') {
    const m = message as any
    return (
      <View style={{ padding: 12, margin: 8, backgroundColor: '#e8f5e9', borderRadius: 8 }}>
        <Text style={{ fontWeight: '600', marginBottom: 4 }}>
          工具结果: {m.toolName || ''}
        </Text>
        {m.toolResult && (
          <Text style={{ fontSize: 12, color: '#666' }}>{JSON.stringify(m.toolResult, null, 2)}</Text>
        )}
      </View>
    )
  }

  if (message.type === 'tool_output' || message.type === 'tool_end') {
    const m = message as any
    return (
      <View style={{ padding: 12, margin: 8, backgroundColor: '#e8f5e9', borderRadius: 8 }}>
        <Text style={{ fontWeight: '600', marginBottom: 4 }}>
          工具输出: {m.toolName || ''}
        </Text>
        {m.toolOutput && (
          <Text style={{ fontSize: 12, color: '#666', fontFamily: 'monospace' }}>{m.toolOutput}</Text>
        )}
      </View>
    )
  }

  if (message.type === 'bash_start') {
    return (
      <View style={{ padding: 12, margin: 8, backgroundColor: '#eceff1', borderRadius: 8 }}>
        <Text style={{ fontWeight: '600', marginBottom: 4 }}>命令执行</Text>
        <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>$ {(message as any).toolInput}</Text>
      </View>
    )
  }

  if (message.type === 'bash_output') {
    return (
      <View style={{ padding: 12, margin: 8, backgroundColor: '#eceff1', borderRadius: 8 }}>
        <Text style={{ fontFamily: 'monospace', fontSize: 12, color: '#333' }}>{(message as any).toolOutput}</Text>
      </View>
    )
  }

  if (message.type === 'bash_end') {
    return (
      <View style={{ padding: 12, margin: 8, backgroundColor: '#eceff1', borderRadius: 8 }}>
        <Text style={{ fontSize: 12, color: '#666' }}>命令结束</Text>
      </View>
    )
  }

  if (message.type === 'permission_request' || message.type === 'permission_ask') {
    return (
      <View style={{ padding: 12, margin: 8, backgroundColor: '#fff9c4', borderRadius: 8, borderLeftWidth: 4, borderLeftColor: '#fbc02d' }}>
        <Text style={{ fontWeight: '600', marginBottom: 4 }}>权限请求: {message.toolName}</Text>
        <Text>{message.message}</Text>
      </View>
    )
  }

  // 其他事件类型显示为调试信息
  return (
    <View style={{ padding: 12, margin: 8, backgroundColor: '#fafafa', borderRadius: 8 }}>
      <Text style={{ fontSize: 12, color: '#999' }}>{message.type}</Text>
    </View>
  )
}

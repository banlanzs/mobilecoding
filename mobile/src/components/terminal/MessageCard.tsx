import React from 'react'
import { View, Text } from 'react-native'
import type { DisplayMessage } from '@/stores/useMessageStore'

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

  if (message.type === 'tool_use') {
    return (
      <View style={{ padding: 12, margin: 8, backgroundColor: '#fff3e0', borderRadius: 8 }}>
        <Text style={{ fontWeight: '600', marginBottom: 4 }}>工具调用: {message.toolName}</Text>
        <Text style={{ fontSize: 12, color: '#666' }}>{JSON.stringify(message.toolInput, null, 2)}</Text>
      </View>
    )
  }

  if (message.type === 'tool_result') {
    return (
      <View style={{ padding: 12, margin: 8, backgroundColor: '#e8f5e9', borderRadius: 8 }}>
        <Text style={{ fontWeight: '600', marginBottom: 4 }}>工具结果: {message.toolName}</Text>
        <Text style={{ fontSize: 12, color: '#666' }}>{JSON.stringify(message.toolResult, null, 2)}</Text>
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

  return (
    <View style={{ padding: 12, margin: 8, backgroundColor: '#fafafa', borderRadius: 8 }}>
      <Text style={{ fontSize: 12, color: '#999' }}>{message.type}</Text>
    </View>
  )
}

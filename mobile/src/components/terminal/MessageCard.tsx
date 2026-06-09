import React from 'react'
import { View, Text } from 'react-native'
import type { DisplayMessage } from '../../stores/useMessageStore'

interface MessageCardProps {
  message: DisplayMessage
}

export function MessageCard({ message }: MessageCardProps) {
  if (message.type === 'user') {
    return (
      <View style={{ flexDirection: 'row', justifyContent: 'flex-end', paddingHorizontal: 12, paddingVertical: 4 }}>
        <View style={{ maxWidth: '75%', backgroundColor: '#95ec69', borderRadius: 18, borderBottomRightRadius: 4, paddingHorizontal: 14, paddingVertical: 10 }}>
          <Text style={{ color: '#000' }}>{message.text}</Text>
        </View>
      </View>
    )
  }

  if (message.type === 'text' || message.type === 'text_delta') {
    return (
      <View style={{ flexDirection: 'row', justifyContent: 'flex-start', paddingHorizontal: 12, paddingVertical: 4 }}>
        <View style={{ maxWidth: '75%', backgroundColor: '#ffffff', borderRadius: 18, borderBottomLeftRadius: 4, paddingHorizontal: 14, paddingVertical: 10 }}>
          <Text style={{ color: '#000' }}>{message.text}</Text>
          {message.thinking && (
            <Text style={{ marginTop: 8, fontStyle: 'italic', color: '#666' }}>{message.thinking}</Text>
          )}
        </View>
      </View>
    )
  }

  if (message.type === 'tool_use' || message.type === 'tool_start') {
    const m = message as any
    return (
      <View style={{ paddingHorizontal: 12, paddingVertical: 4 }}>
        <View style={{ alignSelf: 'flex-start', maxWidth: '85%', backgroundColor: '#ffffff', borderRadius: 12, borderWidth: 1, borderColor: '#e5e5e5', paddingHorizontal: 14, paddingVertical: 10 }}>
          <Text style={{ color: '#576b95', fontWeight: '600' }}>工具调用</Text>
          <Text style={{ color: '#000', marginTop: 4 }}>{m.toolName}</Text>
          {m.toolInput && (
            <Text style={{ fontSize: 12, color: '#999', fontFamily: 'monospace', marginTop: 6 }}>{JSON.stringify(m.toolInput, null, 2)}</Text>
          )}
        </View>
      </View>
    )
  }

  if (message.type === 'tool_result') {
    const m = message as any
    return (
      <View style={{ paddingHorizontal: 12, paddingVertical: 4 }}>
        <View style={{ alignSelf: 'flex-start', maxWidth: '85%', backgroundColor: '#ffffff', borderRadius: 12, borderWidth: 1, borderColor: '#e5e5e5', paddingHorizontal: 14, paddingVertical: 10 }}>
          <Text style={{ color: '#576b95', fontWeight: '600' }}>工具结果</Text>
          {m.toolName ? <Text style={{ color: '#000', marginTop: 4 }}>{m.toolName}</Text> : null}
          {m.toolResult && (
            <Text style={{ fontSize: 12, color: '#999', fontFamily: 'monospace', marginTop: 6 }}>{JSON.stringify(m.toolResult, null, 2)}</Text>
          )}
        </View>
      </View>
    )
  }

  if (message.type === 'tool_output' || message.type === 'tool_end') {
    const m = message as any
    return (
      <View style={{ paddingHorizontal: 12, paddingVertical: 4 }}>
        <View style={{ alignSelf: 'flex-start', maxWidth: '85%', backgroundColor: '#ffffff', borderRadius: 12, borderWidth: 1, borderColor: '#e5e5e5', paddingHorizontal: 14, paddingVertical: 10 }}>
          <Text style={{ color: '#576b95', fontWeight: '600' }}>工具输出</Text>
          {m.toolName ? <Text style={{ color: '#000', marginTop: 4 }}>{m.toolName}</Text> : null}
          {m.toolOutput && (
            <Text style={{ fontSize: 12, color: '#666', fontFamily: 'monospace', marginTop: 6 }}>{m.toolOutput}</Text>
          )}
        </View>
      </View>
    )
  }

  if (message.type === 'bash_start') {
    return (
      <View style={{ paddingHorizontal: 12, paddingVertical: 4 }}>
        <View style={{ alignSelf: 'flex-start', maxWidth: '90%', backgroundColor: '#ffffff', borderRadius: 12, borderWidth: 1, borderColor: '#e5e5e5', paddingHorizontal: 14, paddingVertical: 10 }}>
          <Text style={{ color: '#576b95', fontWeight: '600' }}>命令执行</Text>
          <Text style={{ fontFamily: 'monospace', fontSize: 12, color: '#000', marginTop: 6 }}>$ {(message as any).toolInput}</Text>
        </View>
      </View>
    )
  }

  if (message.type === 'bash_output') {
    return (
      <View style={{ paddingHorizontal: 12, paddingVertical: 4 }}>
        <View style={{ alignSelf: 'flex-start', maxWidth: '90%', backgroundColor: '#ffffff', borderRadius: 12, borderWidth: 1, borderColor: '#e5e5e5', paddingHorizontal: 14, paddingVertical: 10 }}>
          <Text style={{ fontFamily: 'monospace', fontSize: 12, color: '#333' }}>{(message as any).toolOutput}</Text>
        </View>
      </View>
    )
  }

  if (message.type === 'bash_end') {
    return (
      <View style={{ paddingHorizontal: 12, paddingVertical: 4 }}>
        <View style={{ alignSelf: 'flex-start', backgroundColor: '#ffffff', borderRadius: 12, borderWidth: 1, borderColor: '#e5e5e5', paddingHorizontal: 14, paddingVertical: 10 }}>
          <Text style={{ fontSize: 12, color: '#666' }}>命令结束</Text>
        </View>
      </View>
    )
  }

  if (message.type === 'permission_request' || message.type === 'permission_ask') {
    return (
      <View style={{ paddingHorizontal: 12, paddingVertical: 4 }}>
        <View style={{ alignSelf: 'flex-start', maxWidth: '90%', backgroundColor: '#fff9c4', borderRadius: 12, borderWidth: 1, borderColor: '#fbc02d', paddingHorizontal: 14, paddingVertical: 10 }}>
          <Text style={{ fontWeight: '600', marginBottom: 6 }}>权限请求</Text>
          <Text style={{ color: '#000', marginBottom: 4 }}>{message.toolName}</Text>
          <Text>{message.message}</Text>
        </View>
      </View>
    )
  }

  return (
    <View style={{ paddingHorizontal: 12, paddingVertical: 4 }}>
      <View style={{ alignSelf: 'flex-start', backgroundColor: '#ffffff', borderRadius: 12, borderWidth: 1, borderColor: '#e5e5e5', paddingHorizontal: 14, paddingVertical: 10 }}>
        <Text style={{ fontSize: 12, color: '#999' }}>{message.type}</Text>
      </View>
    </View>
  )
}

import React, { useState } from 'react'
import { View, Text, Pressable } from 'react-native'
import type { DisplayMessage } from '../../stores/useMessageStore'

interface MessageCardProps {
  message: DisplayMessage
}

/** 工具/命令类消息：默认折叠，点击展开 */
function ToolCard({ icon, label, detail, children }: {
  icon: string
  label: string
  detail?: string
  children?: React.ReactNode
}) {
  const [expanded, setExpanded] = useState(false)
  return (
    <View style={{ paddingHorizontal: 12, paddingVertical: 2 }}>
      <Pressable onPress={() => setExpanded(!expanded)} style={{ alignSelf: 'flex-start', flexDirection: 'row', alignItems: 'center', backgroundColor: '#f5f5f5', borderRadius: 8, paddingHorizontal: 10, paddingVertical: 6, gap: 6 }}>
        <Text style={{ fontSize: 12 }}>{icon}</Text>
        <Text style={{ fontSize: 12, color: '#666' }}>{label}</Text>
        {detail ? <Text style={{ fontSize: 11, color: '#999', maxWidth: 200 }} numberOfLines={1}>{detail}</Text> : null}
        <Text style={{ fontSize: 10, color: '#aaa' }}>{expanded ? '▲' : '▼'}</Text>
      </Pressable>
      {expanded && (
        <View style={{ marginTop: 4, marginLeft: 10, backgroundColor: '#fff', borderRadius: 8, borderWidth: 1, borderColor: '#e5e5e5', padding: 10, maxHeight: 300 }}>
          {children}
        </View>
      )}
    </View>
  )
}

export function MessageCard({ message }: MessageCardProps) {
  // ─── 用户消息：绿色气泡，右对齐 ───
  if (message.type === 'user') {
    return (
      <View style={{ flexDirection: 'row', justifyContent: 'flex-end', paddingHorizontal: 12, paddingVertical: 4 }}>
        <View style={{ flexShrink: 1, maxWidth: '78%', backgroundColor: '#95ec69', borderRadius: 18, borderBottomRightRadius: 4, paddingHorizontal: 14, paddingVertical: 10 }}>
          <Text selectable style={{ color: '#000', lineHeight: 22 }}>{message.text}</Text>
        </View>
      </View>
    )
  }

  // ─── AI 文本回复 ───
  if (message.type === 'text' || message.type === 'text_delta') {
    const hasThinking = message.thinking && message.thinking !== message.text
    return (
      <View style={{ flexDirection: 'row', justifyContent: 'flex-start', paddingHorizontal: 12, paddingVertical: 4 }}>
        <View style={{ flexShrink: 1, maxWidth: '78%', backgroundColor: '#ffffff', borderRadius: 18, borderBottomLeftRadius: 4, paddingHorizontal: 14, paddingVertical: 10 }}>
          <Text selectable style={{ color: '#000', lineHeight: 22 }}>{message.text}</Text>
          {hasThinking && <ThinkingBlock text={message.thinking!} />}
        </View>
      </View>
    )
  }

  // ─── 工具调用（合并 tool_use/tool_start） ───
  if (message.type === 'tool_use' || message.type === 'tool_start') {
    const m = message as any
    const inputStr = m.toolInput ? JSON.stringify(m.toolInput) : ''
    return (
      <ToolCard icon="🔧" label={m.toolName || '工具'} detail={inputStr.substring(0, 50)}>
        {m.toolInput && (
          <Text selectable style={{ fontSize: 12, color: '#333', fontFamily: 'monospace' }}>{JSON.stringify(m.toolInput, null, 2)}</Text>
        )}
      </ToolCard>
    )
  }

  // ─── 工具结果/输出（合并，不重复显示） ───
  if (message.type === 'tool_result' || message.type === 'tool_output' || message.type === 'tool_end') {
    const m = message as any
    const content = m.toolOutput || (m.toolResult ? JSON.stringify(m.toolResult) : '')
    if (!content) return null
    return (
      <ToolCard icon="📋" label={m.toolName || '结果'} detail={content.substring(0, 50)}>
        <Text selectable style={{ fontSize: 12, color: '#333', fontFamily: 'monospace' }}>{content}</Text>
      </ToolCard>
    )
  }

  // ─── 命令执行（合并 bash 系列） ───
  if (message.type === 'bash_start') {
    return (
      <ToolCard icon="💻" label="命令" detail={(message as any).toolInput?.substring(0, 50)}>
        <Text selectable style={{ fontFamily: 'monospace', fontSize: 12, color: '#000' }}>$ {(message as any).toolInput}</Text>
      </ToolCard>
    )
  }

  if (message.type === 'bash_output') {
    const content = (message as any).toolOutput
    if (!content) return null
    return (
      <ToolCard icon="📤" label="输出" detail={content.substring(0, 50)}>
        <Text selectable style={{ fontFamily: 'monospace', fontSize: 12, color: '#333' }}>{content}</Text>
      </ToolCard>
    )
  }

  if (message.type === 'bash_end') {
    return null
  }

  // ─── 权限请求：保持展开 ───
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

  // ─── 其他事件：折叠 ───
  return (
    <ToolCard icon="📌" label={message.type} />
  )
}

/** 思考内容：默认折叠，点击展开，有最大高度限制 */
function ThinkingBlock({ text }: { text: string }) {
  const [expanded, setExpanded] = useState(false)
  if (!text || text.trim().length === 0) return null
  return (
    <Pressable onPress={() => setExpanded(!expanded)} style={{ marginTop: 8 }}>
      <View style={{ flexDirection: 'row', alignItems: 'center', gap: 4 }}>
        <Text style={{ fontSize: 12, color: '#999' }}>💭</Text>
        <Text style={{ fontSize: 12, color: '#aaa' }}>{expanded ? '收起思考' : '思考过程'}</Text>
        <Text style={{ fontSize: 10, color: '#ccc' }}>{expanded ? '▲' : '▼'}</Text>
      </View>
      {expanded && (
        <View style={{ marginTop: 6, backgroundColor: '#f8f8f8', borderRadius: 8, padding: 10, maxHeight: 200 }}>
          <Text selectable style={{ fontSize: 12, color: '#666', fontStyle: 'italic', lineHeight: 20 }}>{text}</Text>
        </View>
      )}
    </Pressable>
  )
}

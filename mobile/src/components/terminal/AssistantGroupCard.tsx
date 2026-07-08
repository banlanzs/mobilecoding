import React from 'react'
import { View, Text, Pressable, Share, Alert } from 'react-native'
import type { DisplayMessage } from '../../stores/useMessageStore'
import { ToolBlock, ThinkingBlock } from './MessageCard'
import { MarkdownText } from './MarkdownText'
import { toolTone } from './toolTone'

interface AssistantGroupCardProps {
  items: DisplayMessage[]
}

/** 把一条 assistant 事件渲染为 group 内的一个块 */
function renderBlock(m: DisplayMessage, idx: number): React.ReactNode {
  if (m.type === 'text' || m.type === 'text_delta') {
    if (!m.text) return null
    return <MarkdownText key={idx} text={m.text} />
  }
  if (m.type === 'tool_use' || m.type === 'tool_start') {
    const e = m as any
    const inputStr = e.toolInput ? (typeof e.toolInput === 'string' ? e.toolInput : JSON.stringify(e.toolInput)) : ''
    return (
      <ToolBlock key={idx} icon="🔧" label={e.toolName || '工具'} detail={inputStr.substring(0, 50)} tone={toolTone(e)}>
        {e.toolInput && (
          <Text selectable style={{ fontSize: 12, color: '#333', fontFamily: 'monospace' }}>
            {typeof e.toolInput === 'string' ? e.toolInput : JSON.stringify(e.toolInput, null, 2)}
          </Text>
        )}
      </ToolBlock>
    )
  }
  if (m.type === 'tool_result' || m.type === 'tool_output' || m.type === 'tool_end') {
    const e = m as any
    const content = e.toolOutput || (e.toolResult ? (typeof e.toolResult === 'string' ? e.toolResult : JSON.stringify(e.toolResult)) : '')
    if (!content) return null
    return (
      <ToolBlock key={idx} icon="📋" label={e.toolName || '结果'} detail={content.substring(0, 50)} tone={toolTone(e)}>
        <Text selectable style={{ fontSize: 12, color: '#333', fontFamily: 'monospace' }}>{content}</Text>
      </ToolBlock>
    )
  }
  if (m.type === 'bash_start') {
    const e = m as any
    return (
      <ToolBlock key={idx} icon="💻" label="Bash" detail={e.toolInput?.substring(0, 50)} tone={toolTone(e)}>
        <Text selectable style={{ fontFamily: 'monospace', fontSize: 12, color: '#000' }}>$ {e.toolInput}</Text>
      </ToolBlock>
    )
  }
  if (m.type === 'bash_output') {
    const e = m as any
    if (!e.toolOutput) return null
    return (
      <ToolBlock key={idx} icon="📤" label="输出" detail={e.toolOutput.substring(0, 50)} tone={toolTone(e)}>
        <Text selectable style={{ fontFamily: 'monospace', fontSize: 12, color: '#333' }}>{e.toolOutput}</Text>
      </ToolBlock>
    )
  }
  if (m.type === 'bash_end') return null
  // 其余未知事件折叠
  return <ToolBlock key={idx} icon="📌" label={m.type} />
}

/** 提取 group 内所有正文文本（用于复制） */
function extractText(items: DisplayMessage[]): string {
  return items
    .filter(m => (m.type === 'text' || m.type === 'text_delta') && m.text)
    .map(m => (m as any).text)
    .join('\n\n')
}

export const AssistantGroupCard = React.memo(function AssistantGroupCard({ items }: AssistantGroupCardProps) {
  const thinkingText = items
    .map(m => (m as any).thinking)
    .filter(Boolean)
    .join('\n')
  const hasThinking = thinkingText && thinkingText.trim().length > 0

  const onLongPress = () => {
    const fullText = extractText(items)
    if (!fullText) {
      Alert.alert('提示', '该消息没有可复制的文本内容')
      return
    }
    Alert.alert(
      '消息操作',
      undefined,
      [
        { text: '复制全部文本', onPress: () => Share.share({ message: fullText }) },
        { text: '复制 Markdown', onPress: () => Share.share({ message: fullText }) },
        { text: '取消', style: 'cancel' },
      ],
      { cancelable: true }
    )
  }

  return (
    <View style={{ flexDirection: 'row', justifyContent: 'flex-start', paddingHorizontal: 12, paddingVertical: 4 }}>
      <Pressable onLongPress={onLongPress} style={{ flexShrink: 1, maxWidth: '85%' }}>
        <View style={{ backgroundColor: '#ffffff', borderRadius: 18, borderBottomLeftRadius: 4, paddingHorizontal: 14, paddingVertical: 10, gap: 6 }}>
          {items.map(renderBlock)}
          {hasThinking && <ThinkingBlock text={thinkingText} />}
        </View>
      </Pressable>
    </View>
  )
})

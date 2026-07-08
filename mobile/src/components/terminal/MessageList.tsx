import React from 'react'
import { FlatList, View, Text } from 'react-native'
import type { DisplayMessage } from '../../stores/useMessageStore'
import { groupMessages, type RenderGroup } from './groupMessages'
import { AssistantGroupCard } from './AssistantGroupCard'

/** 用户气泡（右对齐，绿色） */
const UserBubble = React.memo(function UserBubble({ text }: { text: string }) {
  return (
    <View style={{ flexDirection: 'row', justifyContent: 'flex-end', paddingHorizontal: 12, paddingVertical: 4 }}>
      <View style={{ flexShrink: 1, maxWidth: '80%', backgroundColor: '#95ec69', borderRadius: 18, borderBottomRightRadius: 4, paddingHorizontal: 14, paddingVertical: 10 }}>
        <Text selectable style={{ color: '#000', lineHeight: 22 }}>{text}</Text>
      </View>
    </View>
  )
})

function renderItem({ item, index }: { item: RenderGroup; index: number }) {
  if (item.kind === 'user') {
    return <UserBubble text={item.message.text} />
  }
  if (item.kind === 'assistant') {
    return <AssistantGroupCard items={item.items} />
  }
  // standalone（权限请求等）：TerminalScreen 已有专门的 permissionPrompt 卡片处理交互，
  // 这里仅作为历史回显的兜底渲染
  const m = item.message as any
  return (
    <View style={{ paddingHorizontal: 12, paddingVertical: 4 }}>
      <View style={{ alignSelf: 'flex-start', maxWidth: '90%', backgroundColor: '#fff9c4', borderRadius: 12, borderWidth: 1, borderColor: '#fbc02d', paddingHorizontal: 14, paddingVertical: 10 }}>
        <Text style={{ fontWeight: '600', marginBottom: 6 }}>权限请求</Text>
        <Text style={{ color: '#000', marginBottom: 4 }}>{m.toolName}</Text>
        <Text>{m.message}</Text>
      </View>
    </View>
  )
}

interface MessageListProps {
  messages: DisplayMessage[]
}

export function MessageList({ messages }: MessageListProps) {
  const groups = groupMessages(messages)
  return (
    <FlatList
      data={groups}
      keyExtractor={(item, idx) => {
        if (item.kind === 'user') return `user-${(item.message as any).time || idx}`
        if (item.kind === 'assistant') return `asst-${(item.items[0] as any)?.messageId || (item.items[0] as any)?.seq || idx}`
        return `std-${(item.message as any)?.messageId || idx}`
      }}
      renderItem={renderItem}
      style={{ flex: 1 }}
      contentContainerStyle={{ paddingVertical: 8 }}
      removeClippedSubviews
      maxToRenderPerBatch={10}
    />
  )
}

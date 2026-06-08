import React, { useState } from 'react'
import { SafeAreaView, View, TextInput, Button, Text, FlatList } from 'react-native'

interface TerminalScreenProps {
  turnActive?: boolean
  messages?: Array<any>
  onSend?: (text: string) => void
  onAbort?: () => void
}

export function TerminalScreen({
  turnActive = false,
  messages = [],
  onSend = () => {},
  onAbort = () => {}
}: TerminalScreenProps) {
  const [input, setInput] = useState('')

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: '#f5f5f5' }}>
      <View style={{ padding: 12, backgroundColor: '#e0e0e0' }}>
        <Text>Terminal</Text>
      </View>

      <FlatList
        data={messages}
        keyExtractor={(_, idx) => String(idx)}
        renderItem={({ item }) => (
          <View style={{ padding: 8, margin: 4, backgroundColor: '#fff', borderRadius: 8 }}>
            <Text>{item.text || item.message || item.type}</Text>
          </View>
        )}
        style={{ flex: 1 }}
      />

      <View style={{ padding: 12, flexDirection: 'row', gap: 8 }}>
        <TextInput
          value={input}
          onChangeText={setInput}
          placeholder="输入消息..."
          style={{ flex: 1, borderWidth: 1, borderColor: '#ccc', borderRadius: 8, paddingHorizontal: 12, height: 40 }}
        />
        {turnActive ? (
          <Button title="停止" onPress={onAbort} />
        ) : (
          <Button title="发送" onPress={() => { onSend(input); setInput('') }} />
        )}
      </View>
    </SafeAreaView>
  )
}

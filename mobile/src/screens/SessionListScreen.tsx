import React from 'react'
import { SafeAreaView, FlatList, Text, View, TouchableOpacity } from 'react-native'
import { useSessionStore } from '../stores/useSessionStore'

export function SessionListScreen() {
  const sessions = useSessionStore(state => state.sessions)

  return (
    <SafeAreaView style={{ flex: 1 }}>
      <FlatList
        data={sessions}
        keyExtractor={item => item.id}
        renderItem={({ item }) => (
          <TouchableOpacity style={{ padding: 16, borderBottomWidth: 1, borderBottomColor: '#eee' }}>
            <Text style={{ fontWeight: '600' }}>{item.name}</Text>
            <Text>{item.agent} {item.model ? `· ${item.model}` : ''}</Text>
          </TouchableOpacity>
        )}
        ListEmptyComponent={
          <View style={{ padding: 24, alignItems: 'center' }}>
            <Text>暂无会话</Text>
          </View>
        }
      />
    </SafeAreaView>
  )
}

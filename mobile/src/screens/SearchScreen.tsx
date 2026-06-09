import React, { useState } from 'react'
import { SafeAreaView, TextInput, FlatList, Text, View, TouchableOpacity } from 'react-native'

export function SearchScreen() {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<Array<{ id: string; type: string; text: string }>>([])

  return (
    <SafeAreaView style={{ flex: 1 }}>
      <TextInput
        value={query}
        onChangeText={setQuery}
        placeholder="搜索消息..."
        style={{ margin: 12, padding: 12, borderWidth: 1, borderColor: '#ccc', borderRadius: 8 }}
      />
      <FlatList
        data={results}
        keyExtractor={item => item.id}
        renderItem={({ item }) => (
          <TouchableOpacity style={{ padding: 12, borderBottomWidth: 1, borderBottomColor: '#eee' }}>
            <Text style={{ fontWeight: '600' }}>{item.type}</Text>
            <Text numberOfLines={2}>{item.text}</Text>
          </TouchableOpacity>
        )}
        ListEmptyComponent={
          <View style={{ padding: 24, alignItems: 'center' }}>
            <Text>{query ? '无搜索结果' : '输入关键词搜索'}</Text>
          </View>
        }
      />
    </SafeAreaView>
  )
}

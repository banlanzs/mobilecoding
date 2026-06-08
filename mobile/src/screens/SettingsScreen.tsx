import React from 'react'
import { SafeAreaView, Text, View, Button } from 'react-native'
import { useAuthStore } from '@/stores/useAuthStore'

export function SettingsScreen() {
  const { activeProfile, status } = useAuthStore()

  return (
    <SafeAreaView style={{ flex: 1, padding: 16 }}>
      <Text style={{ fontSize: 20, fontWeight: '600', marginBottom: 16 }}>设置</Text>

      <View style={{ marginBottom: 24 }}>
        <Text style={{ fontWeight: '600', marginBottom: 8 }}>连接状态</Text>
        <Text>{status === 'connected' ? '已连接' : status === 'connecting' ? '连接中...' : '未连接'}</Text>
      </View>

      {activeProfile && (
        <View style={{ marginBottom: 24 }}>
          <Text style={{ fontWeight: '600', marginBottom: 8 }}>服务器</Text>
          <Text>{activeProfile.host}:{activeProfile.port}</Text>
        </View>
      )}

      <View style={{ marginTop: 24 }}>
        <Button title="退出" onPress={() => {}} />
      </View>
    </SafeAreaView>
  )
}

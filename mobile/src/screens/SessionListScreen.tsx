import React, { useEffect, useState } from 'react'
import { SafeAreaView, FlatList, Text, View, TouchableOpacity, Button, Alert } from 'react-native'
import { useSessionStore } from '../stores/useSessionStore'
import { RestClient } from '../services/network/RestClient'
import { AuthService } from '../services/auth/AuthService'

const authService = new AuthService()

export function SessionListScreen({ navigation }: any) {
  const sessions = useSessionStore(state => state.sessions)
  const setSessions = useSessionStore(state => state.setSessions)
  const setActiveSession = useSessionStore(state => state.setActiveSession)
  const [loading, setLoading] = useState(false)
  const [profile, setProfile] = useState<any>(null)

  useEffect(() => {
    const loadProfile = async () => {
      try {
        const saved = await authService.loadProfile('10.0.2.2:8445')
        setProfile(saved)
      } catch {
        setProfile(null)
      }
    }
    loadProfile()
  }, [])

  const handleRefresh = async () => {
    if (!profile) {
      Alert.alert('未找到连接信息', '请先在欢迎页粘贴链接或扫码连接')
      return
    }
    setLoading(true)
    try {
      const protocol = 'http'
      const rest = new RestClient(`${protocol}://${profile.host}:${profile.port}`, async () => profile.token)
      const result = await rest.get<{ sessions: any[] }>('/api/v1/sessions')
      setSessions(result.sessions || [])
    } catch (err) {
      Alert.alert('加载失败', '获取会话列表失败，请确认服务端已启动')
    } finally {
      setLoading(false)
    }
  }

  const handleOpenSession = (item: any) => {
    setActiveSession(item.id)
    navigation.navigate('Terminal', {
      sessionId: item.id,
      sessionName: item.name,
      host: profile?.host,
      port: profile ? String(profile.port) : '8445',
      token: profile?.token || '',
      path: '/api/v1/ws',
      useWss: false,
    })
  }

  return (
    <SafeAreaView style={{ flex: 1 }}>
      <View style={{ padding: 16, borderBottomWidth: 1, borderBottomColor: '#eee' }}>
        <Text style={{ fontSize: 20, fontWeight: '700', marginBottom: 8 }}>会话列表</Text>
        <Button title={loading ? '加载中...' : '刷新会话'} onPress={handleRefresh} disabled={loading} />
      </View>

      <FlatList
        data={sessions}
        keyExtractor={item => item.id}
        renderItem={({ item }) => (
          <TouchableOpacity style={{ padding: 16, borderBottomWidth: 1, borderBottomColor: '#eee' }} onPress={() => handleOpenSession(item)}>
            <Text style={{ fontWeight: '600' }}>{item.name}</Text>
            <Text>{item.agent} {item.model ? `· ${item.model}` : ''}</Text>
            <Text style={{ fontSize: 12, color: '#666', marginTop: 4 }}>{item.id}</Text>
          </TouchableOpacity>
        )}
        ListEmptyComponent={
          <View style={{ padding: 24, alignItems: 'center' }}>
            <Text>{loading ? '正在加载...' : '暂无会话，点击上方按钮刷新'}</Text>
          </View>
        }
      />
    </SafeAreaView>
  )
}

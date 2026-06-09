import React, { useState, useEffect } from 'react'
import { SafeAreaView, Text, Button, TextInput, View, Alert, ActivityIndicator } from 'react-native'
import { Picker } from '@react-native-picker/picker'
import { adaptConnection, looksLikeConnectionUrl } from '../services/network/ConnectionAdapter'
import { AuthService } from '../services/auth/AuthService'

const authService = new AuthService()

interface ModelOption {
  label: string
  value: string
}

export function OnboardingScreen({ navigation }: any) {
  const [connectionUrl, setConnectionUrl] = useState('')
  const [adapted, setAdapted] = useState<any>(null)
  const [models, setModels] = useState<ModelOption[]>([])
  const [selectedModel, setSelectedModel] = useState('')
  const [loadingModels, setLoadingModels] = useState(false)

  const handlePreview = () => {
    if (!looksLikeConnectionUrl(connectionUrl.trim())) {
      Alert.alert('格式不对', '请粘贴扫码得到的完整链接，例如：\nhttps://10.138.77.206:8443/?token=xxx')
      return
    }
    const result = adaptConnection(connectionUrl.trim())
    if (!result.token) {
      Alert.alert('链接里没找到 token', '请确认粘贴的是完整的扫码链接')
      return
    }
    setAdapted(result)
    fetchModels(result)
  }

  const fetchModels = async (connection: any) => {
    setLoadingModels(true)
    try {
      const scheme = connection.useWss ? 'https' : 'http'
      const url = `${scheme}://${connection.host}:${connection.port}/api/v1/models`
      const response = await fetch(url)
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`)
      }
      const data: ModelOption[] = await response.json()
      setModels(data)
      // 默认选中第一个模型（通常是 Sonnet）
      if (data.length > 0) {
        setSelectedModel(data[0].value)
      }
    } catch (err) {
      console.warn('获取模型列表失败:', err)
      // 使用默认模型列表
      const fallback: ModelOption[] = [
        { label: '默认模型', value: '' },
        { label: 'Sonnet 4.6', value: 'claude-sonnet-4-6' },
        { label: 'Opus 4.8', value: 'claude-opus-4-8' },
        { label: 'Haiku 4.5', value: 'claude-haiku-4-5' },
      ]
      setModels(fallback)
      setSelectedModel(fallback[0].value)
    } finally {
      setLoadingModels(false)
    }
  }

  const handleConnect = async () => {
    if (!adapted) {
      handlePreview()
      return
    }

    try {
      await authService.saveProfile({
        id: `${adapted.host}:${adapted.port}`,
        name: `${adapted.host}:${adapted.port}`,
        host: adapted.host,
        port: parseInt(adapted.port, 10),
        token: adapted.token,
        lastConnectedAt: new Date().toISOString(),
        active: true,
      })
      navigation.navigate('Terminal', {
        host: adapted.host,
        port: adapted.port,
        token: adapted.token,
        path: adapted.path,
        useWss: adapted.useWss,
        model: selectedModel,
      })
    } catch (err) {
      Alert.alert('保存失败', '请重试')
    }
  }

  return (
    <SafeAreaView style={{ flex: 1, justifyContent: 'center', padding: 24 }}>
      <Text style={{ marginBottom: 16, fontSize: 22, fontWeight: '700', textAlign: 'center' }}>
        欢迎使用 MobileCoding
      </Text>
      <Text style={{ marginBottom: 16, textAlign: 'center', color: '#666' }}>
        把桌面端二维码对应的连接链接粘贴到这里
      </Text>

      <TextInput
        value={connectionUrl}
        onChangeText={(text) => { setConnectionUrl(text); setAdapted(null) }}
        placeholder="例如：https://10.138.77.206:8443/?token=xxxx"
        multiline
        style={{
          minHeight: 100,
          borderWidth: 1,
          borderColor: '#ccc',
          borderRadius: 8,
          padding: 12,
          marginBottom: 16,
          backgroundColor: '#fff',
          textAlignVertical: 'top',
        }}
      />

      {adapted && (
        <View style={{ padding: 12, backgroundColor: '#e8f5e9', borderRadius: 8, marginBottom: 16, gap: 10 }}>
          <Text style={{ fontWeight: '600' }}>{adapted.label}</Text>
          <Text style={{ fontSize: 12, color: '#333' }}>
            {adapted.useWss ? 'wss' : 'ws'}://{adapted.host}:{adapted.port}{adapted.path}
          </Text>
          {/* 模型选择器 */}
          <View style={{ backgroundColor: '#fff', borderRadius: 8, borderWidth: 1, borderColor: '#ccc', marginTop: 4 }}>
            {loadingModels ? (
              <ActivityIndicator style={{ padding: 10 }} />
            ) : (
              <Picker
                selectedValue={selectedModel}
                onValueChange={(value) => setSelectedModel(value)}
                style={{ height: 48 }}
              >
                {models.map((m) => (
                  <Picker.Item key={m.value} label={m.label} value={m.value} />
                ))}
              </Picker>
            )}
          </View>
        </View>
      )}

      <View style={{ gap: 12 }}>
        <Button title="扫描二维码" onPress={() => navigation.navigate('QRScanner')} />
        {!adapted ? (
          <Button title="粘贴连接链接" onPress={handlePreview} />
        ) : (
          <Button title="连接" onPress={handleConnect} />
        )}
        <Button title="手动进入终端" onPress={() => navigation.navigate('Terminal')} />
      </View>
    </SafeAreaView>
  )
}

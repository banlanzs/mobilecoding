import React from 'react'
import { SafeAreaView, Text, Button } from 'react-native'

export function OnboardingScreen({ navigation }: any) {
  return (
    <SafeAreaView style={{ flex: 1, justifyContent: 'center', alignItems: 'center' }}>
      <Text style={{ marginBottom: 16 }}>欢迎使用 MobileCoding</Text>
      <Text style={{ marginBottom: 24 }}>扫描桌面端二维码连接</Text>
      <Button title="开始扫码" onPress={() => navigation.navigate('Terminal')} />
    </SafeAreaView>
  )
}

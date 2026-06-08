import React from 'react'
import { SafeAreaView, Text, ActivityIndicator } from 'react-native'

export function SplashScreen() {
  return (
    <SafeAreaView style={{ flex: 1, justifyContent: 'center', alignItems: 'center' }}>
      <ActivityIndicator size="large" />
      <Text style={{ marginTop: 12 }}>Loading MobileCoding...</Text>
    </SafeAreaView>
  )
}

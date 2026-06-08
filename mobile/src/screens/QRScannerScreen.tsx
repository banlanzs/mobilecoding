import React, { useCallback, useRef, useState } from 'react'
import { SafeAreaView, View, Text, Button, StyleSheet, Alert } from 'react-native'
import { Camera, useCameraPermission, useCameraDevice, useCodeScanner } from 'react-native-vision-camera'
import { adaptConnection, looksLikeConnectionUrl } from '../services/network/ConnectionAdapter'
import { AuthService } from '../services/auth/AuthService'

const authService = new AuthService()

export function QRScannerScreen({ navigation }: any) {
  const { hasPermission, requestPermission } = useCameraPermission()
  const device = useCameraDevice('back')
  const [scanned, setScanned] = useState(false)
  const [processing, setProcessing] = useState(false)
  const cameraRef = useRef<Camera>(null)

  const codeScanner = useCodeScanner({
    codeTypes: ['qr'],
    onCodeScanned: useCallback((codes: any[]) => {
      if (scanned || processing) return
      const value = codes[0]?.value
      if (!value) return

      if (!looksLikeConnectionUrl(value)) {
        Alert.alert('无效二维码', '这个二维码不是 MobileCoding 连接码')
        return
      }

      setScanned(true)
      setProcessing(true)

      try {
        const adapted = adaptConnection(value)
        if (!adapted.token) {
          Alert.alert('解析失败', '二维码里没有找到 token')
          setScanned(false)
          setProcessing(false)
          return
        }

        authService.saveProfile({
          id: `${adapted.host}:${adapted.port}`,
          name: `${adapted.host}:${adapted.port}`,
          host: adapted.host,
          port: parseInt(adapted.port, 10),
          token: adapted.token,
          lastConnectedAt: new Date().toISOString(),
          active: true,
        })

        navigation.replace('Terminal', {
          host: adapted.host,
          port: adapted.port,
          token: adapted.token,
          path: adapted.path,
          useWss: adapted.useWss,
        })
      } catch (err) {
        Alert.alert('连接失败', '请重试')
        setScanned(false)
        setProcessing(false)
      }
    }, [scanned, processing, navigation]),
  })

  if (!hasPermission) {
    return (
      <SafeAreaView style={styles.container}>
        <Text style={styles.title}>需要相机权限</Text>
        <Text style={styles.subtitle}>扫码功能需要使用相机来扫描二维码</Text>
        <Button title="授权相机" onPress={requestPermission} />
        <View style={{ marginTop: 12 }}>
          <Button title="返回" onPress={() => navigation.goBack()} />
        </View>
      </SafeAreaView>
    )
  }

  if (device == null) {
    return (
      <SafeAreaView style={styles.container}>
        <Text style={styles.title}>未找到可用相机</Text>
        <Text style={styles.subtitle}>请确认模拟器或设备支持相机</Text>
        <Button title="返回" onPress={() => navigation.goBack()} />
      </SafeAreaView>
    )
  }

  return (
    <SafeAreaView style={styles.container}>
      <View style={styles.header}>
        <Text style={styles.headerText}>扫描连接二维码</Text>
        <Button title="取消" onPress={() => navigation.goBack()} />
      </View>

      <View style={styles.cameraContainer}>
        <Camera
          ref={cameraRef}
          style={StyleSheet.absoluteFill}
          device={device}
          isActive={!scanned}
          codeScanner={codeScanner}
        />

        <View style={styles.overlay}>
          <View style={styles.scanFrame} />
          <Text style={styles.hint}>将二维码放入框内</Text>
        </View>

        {processing && (
          <View style={styles.processingOverlay}>
            <Text style={styles.processingText}>正在连接...</Text>
          </View>
        )}
      </View>
    </SafeAreaView>
  )
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: '#000' },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: 16,
    backgroundColor: 'rgba(0,0,0,0.8)',
  },
  headerText: { color: '#fff', fontSize: 18, fontWeight: '600' },
  cameraContainer: { flex: 1, position: 'relative' },
  overlay: {
    ...StyleSheet.absoluteFillObject,
    justifyContent: 'center',
    alignItems: 'center',
  },
  scanFrame: {
    width: 250,
    height: 250,
    borderWidth: 2,
    borderColor: '#fff',
    borderRadius: 12,
  },
  hint: {
    color: '#fff',
    marginTop: 16,
    fontSize: 14,
    textAlign: 'center',
  },
  processingOverlay: {
    ...StyleSheet.absoluteFillObject,
    backgroundColor: 'rgba(0,0,0,0.6)',
    justifyContent: 'center',
    alignItems: 'center',
  },
  processingText: { color: '#fff', fontSize: 18, fontWeight: '600' },
  title: { color: '#fff', fontSize: 22, fontWeight: '700', textAlign: 'center', marginBottom: 12 },
  subtitle: { color: '#aaa', fontSize: 14, textAlign: 'center', marginBottom: 24 },
})

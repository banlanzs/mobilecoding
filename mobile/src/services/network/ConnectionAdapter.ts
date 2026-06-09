import { Platform } from 'react-native'
import { parseConnectionUrl } from '../auth/DeepLinkService'

export interface ConnectionParams {
  host: string
  port: string
  token: string
  path: string
  useWss: boolean
  useMock: boolean
  label: string
}

// Android 模拟器访问宿主机的特殊 IP
const EMULATOR_HOST = '10.0.2.2'

/**
 * 判断当前是否运行在 Android 模拟器上。
 * 通过 Platform.constants 的 Model/Brand/Manufacturer 判断。
 */
function isAndroidEmulator(): boolean {
  if (Platform.OS !== 'android') return false
  const model = (Platform.constants as any)?.Model as string ?? ''
  const brand = (Platform.constants as any)?.Brand as string ?? ''
  const lowerModel = model.toLowerCase()
  const lowerBrand = brand.toLowerCase()
  // 模拟器特征：Model 里含 sdk/generic/emulator，或 Google brand + SDK 类型号
  if (lowerModel.includes('sdk') || lowerModel.includes('emulator') || lowerModel.includes('generic')) return true
  if (lowerBrand === 'google' && (lowerModel.includes('gphone') || lowerModel.includes('sdk'))) return true
  return false
}

/**
 * 适配扫码链接为当前设备环境的真实连接参数
 *
 * 适配逻辑：
 * 1. 解析扫码链接提取 host/port/token
 * 2. Android 模拟器：host → 10.0.2.2，port → 原端口+2（dev WS 端口），ws://
 * 3. Android 真机：保持原 host，port → 原端口+2（dev WS 端口），ws://（跳过自签 TLS）
 * 4. iOS / 其他：保持原 host，port 不变，wss://（TLS 直连）
 */
export function adaptConnection(input: string): ConnectionParams {
  const parsed = parseConnectionUrl(input)

  if (Platform.OS === 'android') {
    const devPort = String(parsed.port + 2)

    if (isAndroidEmulator()) {
      // 模拟器：10.0.2.2 访问宿主机，用 dev WS 端口
      return {
        host: EMULATOR_HOST,
        port: devPort,
        token: parsed.token,
        path: '/api/v1/ws',
        useWss: false,
        useMock: false,
        label: `模拟器 → ${EMULATOR_HOST}:${devPort}`,
      }
    }

    // 真机：直连 LAN IP，用 dev WS 端口（ws://，跳过自签 TLS）
    return {
      host: parsed.host,
      port: devPort,
      token: parsed.token,
      path: '/api/v1/ws',
      useWss: false,
      useMock: false,
      label: `真机 (ws) → ${parsed.host}:${devPort}`,
    }
  }

  // iOS / 其他平台：直连，用 TLS
  return {
    host: parsed.host,
    port: String(parsed.port),
    token: parsed.token,
    path: '/api/v1/ws',
    useWss: true,
    useMock: false,
    label: `直连 (wss) → ${parsed.host}:${parsed.port}`,
  }
}

/**
 * 判断一个字符串是否看起来像连接链接
 */
export function looksLikeConnectionUrl(input: string): boolean {
  return input.includes('token=') && (
    input.startsWith('http://') ||
    input.startsWith('https://') ||
    input.startsWith('mobilecoding://')
  )
}

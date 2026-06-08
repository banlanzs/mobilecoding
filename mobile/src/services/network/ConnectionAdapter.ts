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
 * 适配扫码链接为当前设备环境的真实连接参数
 *
 * 适配逻辑：
 * 1. 解析扫码链接提取 host/port/token
 * 2. 如果是 Android 模拟器：host → 10.0.2.2，port → 原端口+2（开发 WS 端口），ws://
 * 3. 如果是真机/其他：保持原 host，port 不变，wss://
 */
export function adaptConnection(input: string): ConnectionParams {
  const parsed = parseConnectionUrl(input)

  if (Platform.OS === 'android') {
    // Android 模拟器：通过 10.0.2.2 访问宿主机，用开发 WS 端口（原端口+2）
    const devPort = String(parsed.port + 2)
    return {
      host: EMULATOR_HOST,
      port: devPort,
      token: parsed.token,
      path: '/api/v1/ws',
      useWss: false,
      useMock: false,
      label: `模拟器模式 → ${EMULATOR_HOST}:${devPort}`,
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
    label: `直连模式 → ${parsed.host}:${parsed.port}`,
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

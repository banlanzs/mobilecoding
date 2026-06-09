/**
 * ConnectionAdapter 行为验证：
 * - Android 模拟器 → ws://10.0.2.2:{port+2}
 * - Android 真机   → ws://{scanned-host}:{port+2}
 * - iOS            → wss://{scanned-host}:{scanned-port}
 */

// 用 jest.mock 工厂返回可变对象，各测试直接修改即可
jest.mock('react-native', () => {
  const platform = {
    OS: 'android',
    constants: {} as Record<string, string>,
    select: (obj: Record<string, any>) => obj[platform.OS] ?? obj.default,
  }
  return { Platform: platform }
})

import { Platform } from 'react-native'
import { adaptConnection, looksLikeConnectionUrl } from '../ConnectionAdapter'

const sampleInput = 'https://192.168.1.100:8443/?token=abc123'

describe('adaptConnection', () => {
  describe('Android 模拟器', () => {
    beforeEach(() => {
      ;(Platform as any).OS = 'android'
      ;(Platform as any).constants = {
        Brand: 'google',
        Model: 'sdk_gphone64_arm64',
        Manufacturer: 'Google',
      }
    })

    it('host → 10.0.2.2，port → 原端口+2，ws://', () => {
      const result = adaptConnection(sampleInput)
      expect(result.host).toBe('10.0.2.2')
      expect(result.port).toBe('8445') // 8443 + 2
      expect(result.useWss).toBe(false)
      expect(result.token).toBe('abc123')
      expect(result.path).toBe('/api/v1/ws')
    })
  })

  describe('Android 真机', () => {
    beforeEach(() => {
      ;(Platform as any).OS = 'android'
      ;(Platform as any).constants = {
        Brand: 'samsung',
        Model: 'SM-S9280',
        Manufacturer: 'samsung',
      }
    })

    it('host → 扫码 IP，port → 原端口+2，ws://', () => {
      const result = adaptConnection(sampleInput)
      expect(result.host).toBe('192.168.1.100')
      expect(result.port).toBe('8445') // 8443 + 2，dev WS 端口
      expect(result.useWss).toBe(false) // 不用 TLS，跳过自签证书问题
      expect(result.token).toBe('abc123')
      expect(result.path).toBe('/api/v1/ws')
    })
  })

  describe('iOS', () => {
    beforeEach(() => {
      ;(Platform as any).OS = 'ios'
      ;(Platform as any).constants = {}
    })

    it('host → 扫码 IP，port → 原端口，wss://', () => {
      const result = adaptConnection(sampleInput)
      expect(result.host).toBe('192.168.1.100')
      expect(result.port).toBe('8443') // 保持原始端口
      expect(result.useWss).toBe(true)
      expect(result.token).toBe('abc123')
      expect(result.path).toBe('/api/v1/ws')
    })
  })
})

describe('looksLikeConnectionUrl', () => {
  it('正确识别 https 链接', () => {
    expect(looksLikeConnectionUrl(sampleInput)).toBe(true)
  })

  it('正确识别 http 链接', () => {
    expect(looksLikeConnectionUrl('http://192.168.1.1:8443/?token=xyz')).toBe(true)
  })

  it('拒绝无 token 的链接', () => {
    expect(looksLikeConnectionUrl('https://example.com')).toBe(false)
  })

  it('拒绝纯文本', () => {
    expect(looksLikeConnectionUrl('hello world')).toBe(false)
  })
})

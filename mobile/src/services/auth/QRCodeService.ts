import { parseConnectionUrl } from './DeepLinkService'

export class QRCodeService {
  async scanQRCode(): Promise<{ host: string; port: number; token: string } | null> {
    // Stub: real implementation needs react-native-vision-camera
    // For now, return null to signal "user cancelled"
    return null
  }

  parseScannedUrl(rawUrl: string): { host: string; port: number; token: string } {
    return parseConnectionUrl(rawUrl)
  }
}

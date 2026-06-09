export function parseConnectionUrl(input: string): { host: string; port: number; token: string } {
  // Parse mobilecoding:// deep links
  if (input.startsWith('mobilecoding:')) {
    const query = input.split('?')[1] || ''
    const params = new Map<string, string>()
    query.split('&').forEach(pair => {
      const [key, value] = pair.split('=')
      if (key && value) params.set(key, decodeURIComponent(value))
    })

    return {
      host: params.get('host') || '',
      port: Number(params.get('port') || '8443'),
      token: params.get('token') || ''
    }
  }

  // Parse HTTPS QR URLs
  const protocolEnd = input.indexOf('://')
  if (protocolEnd === -1) throw new Error('Invalid URL format')

  const afterProtocol = input.substring(protocolEnd + 3)
  const pathStart = afterProtocol.indexOf('/')
  const hostPort = pathStart === -1 ? afterProtocol : afterProtocol.substring(0, pathStart)

  const colonIndex = hostPort.indexOf(':')
  const host = colonIndex === -1 ? hostPort : hostPort.substring(0, colonIndex)
  const port = colonIndex === -1 ? 8443 : Number(hostPort.substring(colonIndex + 1))

  const queryStart = input.indexOf('?')
  let token = ''
  if (queryStart !== -1) {
    const query = input.substring(queryStart + 1)
    query.split('&').forEach(pair => {
      const [key, value] = pair.split('=')
      if (key === 'token' && value) token = decodeURIComponent(value)
    })
  }

  return { host, port, token }
}

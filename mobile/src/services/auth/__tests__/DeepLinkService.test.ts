import { parseConnectionUrl } from '../DeepLinkService'

test('parses existing desktop QR URL with token query', () => {
  expect(
    parseConnectionUrl('https://10.0.0.5:8443/?token=abc123')
  ).toEqual({
    host: '10.0.0.5',
    port: 8443,
    token: 'abc123'
  })
})

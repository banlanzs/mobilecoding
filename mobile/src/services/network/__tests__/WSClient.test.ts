import { WSClient } from '../WSClient'

test('queues requests before socket opens and flushes them after connect', async () => {
  const sent: string[] = []
  const fakeSocket = {
    readyState: 1,
    send: (payload: string) => sent.push(payload),
    close: jest.fn()
  }

  const client = new WSClient(() => fakeSocket as any)
  const promise = client.send('session.input', { text: 'hello' })
  client.__unsafeHandleOpen()

  expect(sent).toHaveLength(1)
  expect(sent[0]).toContain('session.input')
  await expect(promise).resolves.toBeUndefined()
})

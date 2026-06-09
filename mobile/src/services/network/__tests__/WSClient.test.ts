import { WSClient } from '../WSClient'

test('queues requests before socket opens and resolves them when server responds', async () => {
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

  const envelope = JSON.parse(sent[0])
  client.__unsafeHandleMessage(JSON.stringify({
    type: 'resp',
    id: envelope.id,
    ok: true,
    result: { success: true }
  }))

  await expect(promise).resolves.toEqual({ success: true })
})

test('rejects promise when server responds with error', async () => {
  const sent: string[] = []
  const fakeSocket = {
    readyState: 1,
    send: (payload: string) => sent.push(payload),
    close: jest.fn()
  }

  const client = new WSClient(() => fakeSocket as any)
  const promise = client.send('session.start', {})
  client.__unsafeHandleOpen()

  const envelope = JSON.parse(sent[0])
  client.__unsafeHandleMessage(JSON.stringify({
    type: 'resp',
    id: envelope.id,
    ok: false,
    error: { code: 'ERR_NOT_CONFIGURED', message: 'engine not configured' }
  }))

  await expect(promise).rejects.toThrow('engine not configured')
})

test('times out when server does not respond', async () => {
  const fakeSocket = {
    readyState: 1,
    send: jest.fn(),
    close: jest.fn()
  }

  const client = new WSClient(() => fakeSocket as any)
  const promise = client.send('session.input', { text: 'hello' })
  client.__unsafeHandleOpen()

  await expect(promise).rejects.toThrow('timed out')
}, 35_000)

export class RestClient {
  constructor(private baseUrl: string, private getToken: () => Promise<string>) {}

  async get<T>(path: string): Promise<T> {
    const token = await this.getToken()
    const response = await fetch(`${this.baseUrl}${path}`, {
      headers: {
        Authorization: `Bearer ${token}`,
        Accept: 'application/json'
      }
    })

    if (response.status === 401) {
      throw new Error('token_expired')
    }
    if (!response.ok) {
      throw new Error(`request_failed:${response.status}`)
    }
    return response.json() as Promise<T>
  }
}

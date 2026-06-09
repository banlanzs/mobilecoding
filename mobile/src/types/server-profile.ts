export interface ServerProfile {
  id: string
  name: string
  host: string
  port: number
  token: string
  lastConnectedAt: string | null
  active: boolean
}

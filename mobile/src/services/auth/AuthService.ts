import * as Keychain from 'react-native-keychain'
import type { ServerProfile } from '../../types/server-profile'

export class AuthService {
  async saveProfile(profile: ServerProfile): Promise<void> {
    await Keychain.setGenericPassword(
      profile.id,
      JSON.stringify(profile),
      { service: 'mobilecoding.profile' }
    )
  }

  async loadProfile(id: string): Promise<ServerProfile | null> {
    const result = await Keychain.getGenericPassword({ service: 'mobilecoding.profile' })
    if (!result) return null
    const profile = JSON.parse(result.password) as ServerProfile
    return profile.id === id ? profile : null
  }

  async deleteProfile(id: string): Promise<void> {
    await Keychain.resetGenericPassword({ service: 'mobilecoding.profile' })
  }
}

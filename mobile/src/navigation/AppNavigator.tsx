import React from 'react'
import { NavigationContainer } from '@react-navigation/native'
import { createNativeStackNavigator } from '@react-navigation/native-stack'
import { SplashScreen } from '@/screens/SplashScreen'
import { OnboardingScreen } from '@/screens/OnboardingScreen'
import { SessionListScreen } from '@/screens/SessionListScreen'
import { TerminalScreen } from '@/screens/TerminalScreen'

const Stack = createNativeStackNavigator()

export function AppNavigator() {
  return (
    <NavigationContainer>
      <Stack.Navigator screenOptions={{ headerShown: false }}>
        <Stack.Screen name="Splash" component={SplashScreen} />
        <Stack.Screen name="Onboarding" component={OnboardingScreen} />
        <Stack.Screen name="Sessions" component={SessionListScreen} />
        <Stack.Screen name="Terminal" component={TerminalScreen} />
      </Stack.Navigator>
    </NavigationContainer>
  )
}

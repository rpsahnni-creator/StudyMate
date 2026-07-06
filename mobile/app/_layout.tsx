import "react-native-gesture-handler";
import { useEffect } from "react";
import { LogBox } from "react-native";
import { Stack } from "expo-router";
import { StatusBar } from "expo-status-bar";
import * as SplashScreen from "expo-splash-screen";
import { AuthProvider, AuthGate } from "../hooks/useAuth";

SplashScreen.preventAutoHideAsync().catch(() => undefined);

if (__DEV__) {
  LogBox.ignoreLogs([
    "Cannot connect to Expo CLI",
    "Remote debugger",
  ]);
}

export default function RootLayout() {
  useEffect(() => {
    SplashScreen.hideAsync().catch(() => undefined);
  }, []);

  return (
    <AuthProvider>
      <StatusBar style="dark" />
      <AuthGate>
        <Stack screenOptions={{ headerShown: false }}>
          <Stack.Screen name="(tabs)" />
          <Stack.Screen name="auth" />
          <Stack.Screen name="quiz" />
          <Stack.Screen name="goals" />
        </Stack>
      </AuthGate>
    </AuthProvider>
  );
}

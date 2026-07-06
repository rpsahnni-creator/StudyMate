import { Stack } from "expo-router";
import { colors } from "../../lib/theme";

export default function GoalsLayout() {
  return (
    <Stack
      screenOptions={{
        headerStyle: { backgroundColor: colors.surface },
        headerTitleStyle: { fontWeight: "800", color: colors.text },
        headerTintColor: colors.brand,
        headerShadowVisible: false,
        contentStyle: { backgroundColor: colors.bg },
      }}
    >
      <Stack.Screen name="select" options={{ title: "Choose Goal" }} />
      <Stack.Screen name="practice" options={{ title: "Daily Practice" }} />
      <Stack.Screen name="skills" options={{ title: "Your Skills" }} />
    </Stack>
  );
}

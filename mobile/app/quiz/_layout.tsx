import { Stack } from "expo-router";
import { colors } from "../../lib/theme";

export default function QuizLayout() {
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
      <Stack.Screen name="[quizId]" options={{ title: "Quiz" }} />
      <Stack.Screen name="result" options={{ title: "Result", headerBackVisible: false }} />
      <Stack.Screen name="review" options={{ title: "Review" }} />
    </Stack>
  );
}

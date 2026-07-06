import { Tabs } from "expo-router";
import { Ionicons } from "@expo/vector-icons";
import { useFeatureFlag } from "../../hooks/useFeatureFlags";
import { colors, shadow } from "../../lib/theme";

export default function TabLayout() {
  const careerGoalsEnabled = useFeatureFlag("career_goals_module");

  return (
    <Tabs
      screenOptions={{
        headerShown: false,
        tabBarActiveTintColor: colors.brand,
        tabBarInactiveTintColor: colors.textSubtle,
        tabBarStyle: {
          backgroundColor: colors.surface,
          borderTopColor: colors.border,
          height: 64,
          paddingBottom: 10,
          paddingTop: 8,
          ...shadow.sm,
        },
        tabBarLabelStyle: { fontSize: 11, fontWeight: "700" },
        headerStyle: { backgroundColor: colors.surface },
        headerTitleStyle: { fontWeight: "800", color: colors.text, fontSize: 18 },
        headerShadowVisible: false,
      }}
    >
      <Tabs.Screen
        name="index"
        options={{
          title: "Home",
          tabBarIcon: ({ color, focused }) => (
            <Ionicons name={focused ? "home" : "home-outline"} size={22} color={color} />
          ),
        }}
      />
      <Tabs.Screen
        name="scan"
        options={{
          title: "Scan",
          tabBarIcon: ({ color, focused }) => (
            <Ionicons name={focused ? "scan-circle" : "scan-circle-outline"} size={24} color={color} />
          ),
        }}
      />
      <Tabs.Screen
        name="reports"
        options={{
          title: "Reports",
          tabBarIcon: ({ color, focused }) => (
            <Ionicons name={focused ? "bar-chart" : "bar-chart-outline"} size={22} color={color} />
          ),
        }}
      />
      <Tabs.Screen
        name="profile"
        options={{
          title: "Profile",
          tabBarIcon: ({ color, focused }) => (
            <Ionicons name={focused ? "person" : "person-outline"} size={22} color={color} />
          ),
        }}
      />
      <Tabs.Screen
        name="goals"
        options={{
          title: "Goals",
          href: careerGoalsEnabled ? undefined : null,
          tabBarIcon: ({ color, focused }) => (
            <Ionicons name={focused ? "flag" : "flag-outline"} size={22} color={color} />
          ),
        }}
      />
    </Tabs>
  );
}

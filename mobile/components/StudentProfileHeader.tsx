import { useEffect, useState } from "react";
import { Image, Pressable, StyleSheet, Text, View } from "react-native";
import { useRouter } from "expo-router";
import { Ionicons } from "@expo/vector-icons";
import { useAuth } from "../hooks/useAuth";
import { getMySubscription, planDisplayName, type Entitlements } from "../lib/billing";
import { getProfileImageUri, initialsFromName } from "../lib/profile";
import { colors, radius, shadow, spacing } from "../lib/theme";

const PLAN_TIERS = [
  { id: "free", label: "Free", color: "#475569", bg: "#F1F5F9", border: "#CBD5E1" },
  { id: "basic", label: "Basic", color: "#1D4ED8", bg: "#EFF6FF", border: "#93C5FD" },
  { id: "pro", label: "Pro", color: "#B45309", bg: "#FFFBEB", border: "#FCD34D" },
] as const;

function planBracketColor(plan: string): string {
  const tier = PLAN_TIERS.find((t) => t.id === plan);
  return tier?.color ?? "#475569";
}

export function StudentProfileHeader() {
  const router = useRouter();
  const { user } = useAuth();
  const [entitlements, setEntitlements] = useState<Entitlements | null>(null);
  const [photoUri, setPhotoUri] = useState<string | null>(null);

  const displayName = user?.name?.trim() || "Student";
  const activePlan = entitlements?.plan ?? "free";

  useEffect(() => {
    let mounted = true;
    void getMySubscription().then((sub) => {
      if (mounted) setEntitlements(sub);
    });
    void getProfileImageUri().then((uri) => {
      if (mounted) setPhotoUri(uri);
    });
    return () => {
      mounted = false;
    };
  }, []);

  return (
    <Pressable
      onPress={() => router.push("/(tabs)/profile")}
      style={({ pressed }) => [styles.card, pressed ? styles.cardPressed : null]}
      accessibilityRole="button"
      accessibilityLabel="Open profile"
    >
      <View style={styles.row}>
        <View style={styles.avatarWrap}>
          {photoUri ? (
            <Image source={{ uri: photoUri }} style={styles.avatarImage} />
          ) : (
            <View style={styles.avatarFallback}>
              <Text style={styles.avatarInitials}>{initialsFromName(displayName)}</Text>
            </View>
          )}
        </View>

        <View style={styles.info}>
          <Text style={styles.name} numberOfLines={2}>
            {displayName}{" "}
            <Text style={[styles.planBracket, { color: planBracketColor(activePlan) }]}>
              ({planDisplayName(activePlan)})
            </Text>
          </Text>

          <View style={styles.planRow}>
            {PLAN_TIERS.map((tier) => {
              const isActive = activePlan === tier.id;
              return (
                <View
                  key={tier.id}
                  style={[
                    styles.planPill,
                    {
                      backgroundColor: isActive ? tier.bg : "rgba(255,255,255,0.72)",
                      borderColor: isActive ? tier.border : "rgba(255,255,255,0.9)",
                    },
                    isActive ? styles.planPillActive : null,
                  ]}
                >
                  <Text
                    style={[
                      styles.planPillText,
                      { color: isActive ? tier.color : "#64748B" },
                      isActive ? styles.planPillTextActive : null,
                    ]}
                    numberOfLines={1}
                  >
                    {tier.label}
                  </Text>
                </View>
              );
            })}
          </View>
        </View>

        <Ionicons name="chevron-forward" size={22} color="#94A3B8" style={styles.chevron} />
      </View>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  card: {
    backgroundColor: "rgba(255, 255, 255, 0.88)",
    borderRadius: radius.xl,
    borderWidth: 1,
    borderColor: "rgba(255, 255, 255, 0.95)",
    padding: spacing.lg,
    ...shadow.md,
  },
  cardPressed: {
    opacity: 0.94,
    transform: [{ scale: 0.995 }],
  },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
  },
  avatarWrap: {
    flexShrink: 0,
    overflow: "hidden",
    borderRadius: 42,
  },
  avatarImage: {
    width: 84,
    height: 84,
    borderRadius: 42,
    borderWidth: 3,
    borderColor: colors.white,
    backgroundColor: "#E2E8F0",
  },
  avatarFallback: {
    width: 84,
    height: 84,
    borderRadius: 42,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: "#20B2AA",
    borderWidth: 3,
    borderColor: colors.white,
  },
  avatarInitials: {
    fontSize: 28,
    fontWeight: "800",
    color: colors.white,
    letterSpacing: -0.5,
  },
  info: {
    flex: 1,
    gap: 8,
    minWidth: 0,
  },
  chevron: {
    flexShrink: 0,
  },
  name: {
    fontSize: 20,
    fontWeight: "800",
    color: "#0F172A",
    letterSpacing: -0.3,
    lineHeight: 26,
  },
  planBracket: {
    fontSize: 18,
    fontWeight: "700",
  },
  planRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: 6,
  },
  planPill: {
    flex: 1,
    paddingHorizontal: 6,
    paddingVertical: 6,
    borderRadius: radius.full,
    borderWidth: 1.5,
    alignItems: "center",
    justifyContent: "center",
    minWidth: 0,
  },
  planPillActive: {
    ...shadow.sm,
  },
  planPillText: {
    fontSize: 12,
    fontWeight: "600",
  },
  planPillTextActive: {
    fontWeight: "800",
  },
});

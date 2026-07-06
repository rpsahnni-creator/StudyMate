import { useCallback, useEffect, useState } from "react";
import {
  ActivityIndicator,
  Alert,
  Image,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from "react-native";
import { useRouter } from "expo-router";
import * as ImagePicker from "expo-image-picker";
import { useAuth } from "../../hooks/useAuth";
import { fetchCurrentUser, type AuthUser } from "../../lib/auth";
import {
  getMySubscription,
  planDisplayName,
  scansLabel,
  type Entitlements,
} from "../../lib/billing";
import {
  getProfileImageUri,
  initialsFromName,
  saveProfileImageUri,
} from "../../lib/profile";
import { skyScreen } from "../../lib/skyScreen";
import { SkyBackground } from "../../components/SkyBackground";
import { Card, SecondaryButton } from "../../components/ui";
import { colors, radius, shadow, spacing } from "../../lib/theme";

export default function ProfileScreen() {
  const router = useRouter();
  const { user, logout } = useAuth();
  const [me, setMe] = useState<AuthUser | null>(user);
  const [entitlements, setEntitlements] = useState<Entitlements | null>(null);
  const [photoUri, setPhotoUri] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const displayName = me?.name?.trim() || user?.name?.trim() || "Student";

  const loadAll = useCallback(async () => {
    setLoading(true);
    try {
      const [profile, sub, uri] = await Promise.all([
        fetchCurrentUser(),
        getMySubscription(),
        getProfileImageUri(),
      ]);
      setMe(profile ?? user);
      setEntitlements(sub);
      setPhotoUri(uri);
    } finally {
      setLoading(false);
    }
  }, [user]);

  useEffect(() => {
    void loadAll();
  }, [loadAll]);

  async function pickPhoto() {
    const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!permission.granted) {
      Alert.alert("Permission needed", "Gallery access is required to set your profile photo.");
      return;
    }

    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ["images"],
      allowsEditing: true,
      aspect: [1, 1],
      quality: 0.85,
    });

    if (result.canceled || !result.assets[0]?.uri) return;

    const uri = result.assets[0].uri;
    await saveProfileImageUri(uri);
    setPhotoUri(uri);
  }

  async function handleLogout() {
    await logout();
    router.replace("/auth/login");
  }

  if (loading) {
    return (
      <SkyBackground>
        <View style={skyScreen.center}>
          <ActivityIndicator size="large" color={colors.brand} />
          <Text style={styles.muted}>Loading profile…</Text>
        </View>
      </SkyBackground>
    );
  }

  return (
    <SkyBackground>
      <ScrollView style={styles.screen} contentContainerStyle={skyScreen.content}>
        <Text style={skyScreen.title}>My Profile</Text>

        <Card glass style={styles.identityCard}>
          <View style={styles.avatarBlock}>
            <View style={styles.avatarRing}>
              {photoUri ? (
                <Image source={{ uri: photoUri }} style={styles.avatarImage} />
              ) : (
                <View style={styles.avatarFallback}>
                  <Text style={styles.avatarInitials}>{initialsFromName(displayName)}</Text>
                </View>
              )}
            </View>
            <Pressable onPress={() => void pickPhoto()} style={styles.changePhotoBtn}>
              <Text style={styles.changePhotoText}>Change photo</Text>
            </Pressable>
          </View>

          <Text style={styles.profileName}>
            {displayName}{" "}
            <Text style={styles.planTag}>({planDisplayName(entitlements?.plan ?? "free")})</Text>
          </Text>
          {me?.email ? <Text style={styles.email}>{me.email}</Text> : null}
        </Card>

        <Card glass style={styles.section}>
          <Text style={styles.sectionTitle}>Account</Text>
          <DetailRow label="Name" value={me?.name ?? "—"} />
          <DetailRow label="Email" value={me?.email ?? "—"} />
          <DetailRow label="Role" value={me?.role ?? "student"} />
        </Card>

        <Card glass style={styles.section}>
          <Text style={styles.sectionTitle}>Subscription</Text>
          {entitlements ? (
            <>
              <DetailRow label="Plan" value={planDisplayName(entitlements.plan)} />
              <DetailRow
                label="Status"
                value={entitlements.has_active_sub ? "Active" : "Free tier"}
              />
              <DetailRow label="Daily scans" value={scansLabel(entitlements)} />
              {entitlements.expires_at ? (
                <DetailRow
                  label="Expires"
                  value={`${new Date(entitlements.expires_at).toLocaleDateString()}${
                    entitlements.days_remaining < 7
                      ? ` (${entitlements.days_remaining} days left)`
                      : ""
                  }`}
                />
              ) : null}
            </>
          ) : (
            <Text style={styles.muted}>No subscription data</Text>
          )}
        </Card>

        <SecondaryButton title="Sign out" onPress={() => void handleLogout()} />
      </ScrollView>
    </SkyBackground>
  );
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <View style={styles.detailRow}>
      <Text style={styles.detailLabel}>{label}</Text>
      <Text style={styles.detailValue}>{value}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1 },
  muted: { color: colors.textMuted, fontSize: 14 },
  identityCard: {
    alignItems: "center",
    gap: spacing.md,
    paddingVertical: spacing.xl,
  },
  avatarBlock: {
    alignItems: "center",
    gap: spacing.sm,
  },
  avatarRing: {
    overflow: "hidden",
    borderRadius: 56,
  },
  avatarImage: {
    width: 112,
    height: 112,
    borderRadius: 56,
    borderWidth: 4,
    borderColor: colors.white,
    backgroundColor: "#E2E8F0",
  },
  avatarFallback: {
    width: 112,
    height: 112,
    borderRadius: 56,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: colors.brand,
    borderWidth: 4,
    borderColor: colors.white,
  },
  avatarInitials: {
    fontSize: 40,
    fontWeight: "800",
    color: colors.white,
  },
  changePhotoBtn: {
    paddingVertical: 6,
    paddingHorizontal: 12,
  },
  changePhotoText: {
    color: colors.brandDark,
    fontWeight: "700",
    fontSize: 14,
  },
  profileName: {
    fontSize: 24,
    fontWeight: "800",
    color: colors.text,
    textAlign: "center",
  },
  planTag: {
    color: colors.brandDark,
    fontWeight: "700",
  },
  email: {
    color: colors.textMuted,
    fontSize: 15,
    textAlign: "center",
  },
  section: {
    gap: spacing.sm,
  },
  sectionTitle: {
    fontSize: 17,
    fontWeight: "800",
    color: colors.text,
    marginBottom: spacing.xs,
  },
  detailRow: {
    flexDirection: "row",
    justifyContent: "space-between",
    gap: spacing.md,
    paddingVertical: 8,
    borderBottomWidth: 1,
    borderBottomColor: colors.border,
  },
  detailLabel: {
    color: colors.textMuted,
    fontSize: 14,
    fontWeight: "600",
    flex: 1,
  },
  detailValue: {
    color: colors.text,
    fontSize: 14,
    fontWeight: "700",
    flex: 1.2,
    textAlign: "right",
  },
});

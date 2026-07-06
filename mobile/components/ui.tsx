import React, { useState } from "react";
import {
  ActivityIndicator,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View,
  type TextInputProps,
  type ViewProps,
} from "react-native";
import { Ionicons } from "@expo/vector-icons";
import * as Haptics from "expo-haptics";
import { colors, radius, shadow, spacing } from "../lib/theme";

type IoniconName = keyof typeof Ionicons.glyphMap;

function tapFeedback() {
  void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light).catch(() => undefined);
}

export function Card({ style, children, ...rest }: ViewProps) {
  return (
    <View style={[styles.card, style]} {...rest}>
      {children}
    </View>
  );
}

export function Badge({
  label,
  color = colors.brand,
  bg = colors.brandSoft,
}: {
  label: string;
  color?: string;
  bg?: string;
}) {
  return (
    <View style={[styles.badge, { backgroundColor: bg }]}>
      <Text style={[styles.badgeText, { color }]}>{label}</Text>
    </View>
  );
}

export function PrimaryButton({
  title,
  onPress,
  loading,
  disabled,
  icon,
  style,
}: {
  title: string;
  onPress: () => void;
  loading?: boolean;
  disabled?: boolean;
  icon?: IoniconName;
  style?: object;
}) {
  const isDisabled = disabled || loading;
  return (
    <Pressable
      onPress={() => {
        tapFeedback();
        onPress();
      }}
      disabled={isDisabled}
      style={({ pressed }: { pressed: boolean }) => [
        styles.primaryBtn,
        pressed && !isDisabled ? styles.pressed : null,
        isDisabled ? styles.disabledBtn : null,
        style,
      ]}
    >
      {loading ? (
        <ActivityIndicator color={colors.white} />
      ) : (
        <View style={styles.btnContent}>
          {icon ? <Ionicons name={icon} size={17} color={colors.white} /> : null}
          <Text style={styles.primaryText}>{title}</Text>
        </View>
      )}
    </Pressable>
  );
}

export function SecondaryButton({
  title,
  onPress,
  disabled,
  icon,
  style,
}: {
  title: string;
  onPress: () => void;
  disabled?: boolean;
  icon?: IoniconName;
  style?: object;
}) {
  return (
    <Pressable
      onPress={() => {
        tapFeedback();
        onPress();
      }}
      disabled={disabled}
      style={({ pressed }: { pressed: boolean }) => [
        styles.secondaryBtn,
        pressed && !disabled ? styles.pressed : null,
        disabled ? styles.disabledBtn : null,
        style,
      ]}
    >
      <View style={styles.btnContent}>
        {icon ? <Ionicons name={icon} size={17} color={colors.text} /> : null}
        <Text style={styles.secondaryText}>{title}</Text>
      </View>
    </Pressable>
  );
}

export function Field({
  label,
  style,
  ...rest
}: TextInputProps & { label?: string }) {
  return (
    <View style={styles.field}>
      {label ? <Text style={styles.fieldLabel}>{label}</Text> : null}
      <TextInput
        placeholderTextColor={colors.textSubtle}
        style={[styles.input, style]}
        {...rest}
      />
    </View>
  );
}

export function PasswordField({
  label,
  style,
  ...rest
}: Omit<TextInputProps, "secureTextEntry"> & { label?: string }) {
  const [visible, setVisible] = useState(false);

  return (
    <View style={styles.field}>
      {label ? <Text style={styles.fieldLabel}>{label}</Text> : null}
      <View style={styles.passwordWrap} pointerEvents="box-none">
        <TextInput
          placeholderTextColor={colors.textSubtle}
          style={[styles.input, styles.passwordInput, style]}
          secureTextEntry={!visible}
          {...rest}
        />
        <Pressable
          style={styles.passwordToggle}
          pointerEvents="auto"
          onPress={() => {
            tapFeedback();
            setVisible((current) => !current);
          }}
          hitSlop={8}
          accessibilityRole="button"
          accessibilityLabel={visible ? "Hide password" : "Show password"}
        >
          <Ionicons
            name={visible ? "eye-off-outline" : "eye-outline"}
            size={22}
            color={colors.textMuted}
          />
        </Pressable>
      </View>
    </View>
  );
}

export function Checkbox({
  checked,
  onToggle,
  label,
}: {
  checked: boolean;
  onToggle: () => void;
  label: string;
}) {
  return (
    <Pressable
      style={styles.checkRow}
      onPress={() => {
        tapFeedback();
        onToggle();
      }}
    >
      <View style={[styles.checkbox, checked ? styles.checkboxOn : null]}>
        {checked ? <Text style={styles.checkMark}>✓</Text> : null}
      </View>
      <Text style={styles.checkLabel}>{label}</Text>
    </Pressable>
  );
}

export function LogoMark({ size = 40 }: { size?: number }) {
  return (
    <View
      style={[
        styles.logo,
        { width: size, height: size, borderRadius: size * 0.3 },
      ]}
    >
      <Text style={[styles.logoText, { fontSize: size * 0.5 }]}>S</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    backgroundColor: colors.surface,
    borderRadius: radius.lg,
    borderWidth: 1,
    borderColor: colors.border,
    padding: spacing.lg,
    ...shadow.sm,
  },
  badge: {
    alignSelf: "flex-start",
    paddingHorizontal: 11,
    paddingVertical: 4,
    borderRadius: radius.full,
  },
  badgeText: { fontSize: 12, fontWeight: "700" },
  primaryBtn: {
    backgroundColor: colors.brand,
    borderRadius: radius.md,
    paddingVertical: 14,
    paddingHorizontal: 18,
    alignItems: "center",
    justifyContent: "center",
    ...shadow.brand,
  },
  btnContent: { flexDirection: "row", alignItems: "center", justifyContent: "center", gap: 8 },
  primaryText: { color: colors.white, fontSize: 16, fontWeight: "700" },
  secondaryBtn: {
    backgroundColor: colors.surface,
    borderRadius: radius.md,
    paddingVertical: 13,
    paddingHorizontal: 18,
    alignItems: "center",
    justifyContent: "center",
    borderWidth: 1,
    borderColor: colors.borderStrong,
  },
  secondaryText: { color: colors.text, fontSize: 15, fontWeight: "700" },
  pressed: { opacity: 0.85, transform: [{ scale: 0.99 }] },
  disabledBtn: { opacity: 0.5 },
  field: { gap: 7 },
  fieldLabel: { fontSize: 13, fontWeight: "700", color: colors.text },
  input: {
    borderWidth: 1,
    borderColor: colors.borderStrong,
    borderRadius: radius.md,
    paddingHorizontal: 13,
    paddingVertical: 12,
    fontSize: 16,
    backgroundColor: colors.surface,
    color: colors.text,
  },
  passwordWrap: { position: "relative" },
  passwordInput: { paddingRight: 44 },
  passwordToggle: {
    position: "absolute",
    right: 12,
    top: 0,
    bottom: 0,
    justifyContent: "center",
  },
  checkRow: {
    flexDirection: "row",
    alignItems: "flex-start",
    gap: 12,
    backgroundColor: colors.surfaceAlt,
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radius.md,
    padding: spacing.lg,
  },
  checkbox: {
    width: 22,
    height: 22,
    borderRadius: 6,
    borderWidth: 2,
    borderColor: colors.borderStrong,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: colors.surface,
  },
  checkboxOn: { backgroundColor: colors.brand, borderColor: colors.brand },
  checkMark: { color: colors.white, fontSize: 14, fontWeight: "900" },
  checkLabel: { flex: 1, fontSize: 13.5, color: colors.textMuted, lineHeight: 20 },
  logo: {
    backgroundColor: colors.brand,
    alignItems: "center",
    justifyContent: "center",
    ...shadow.brand,
  },
  logoText: { color: colors.white, fontWeight: "800" },
});

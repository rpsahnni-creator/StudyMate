import React, { useState } from "react";
import {
  ActivityIndicator,
  Image,
  Modal,
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

export function Card({ style, glass, children, ...rest }: ViewProps & { glass?: boolean }) {
  return (
    <View style={[styles.card, glass ? styles.cardGlass : null, style]} {...rest}>
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
  labelColor,
}: {
  title: string;
  onPress: () => void;
  loading?: boolean;
  disabled?: boolean;
  icon?: IoniconName;
  style?: object;
  labelColor?: string;
}) {
  const isDisabled = disabled || loading;
  const textColor = labelColor ?? colors.white;
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
        <ActivityIndicator color={textColor} />
      ) : (
        <View style={styles.btnContent}>
          {icon ? <Ionicons name={icon} size={17} color={textColor} /> : null}
          <Text style={[styles.primaryText, { color: textColor }]}>{title}</Text>
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

export function BoardSelect({
  label,
  value,
  options,
  onChange,
}: {
  label?: string;
  value: string;
  options: { id: string; label: string }[];
  onChange: (id: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const selected = options.find((option) => option.id === value);

  return (
    <View style={styles.field}>
      {label ? <Text style={styles.fieldLabel}>{label}</Text> : null}
      <Pressable
        onPress={() => {
          tapFeedback();
          setOpen(true);
        }}
        style={styles.selectTrigger}
      >
        <Text style={styles.selectValue}>{selected?.label ?? "Select board"}</Text>
        <Ionicons name="chevron-down" size={18} color={colors.textMuted} />
      </Pressable>

      <Modal visible={open} transparent animationType="fade" onRequestClose={() => setOpen(false)}>
        <Pressable style={styles.selectBackdrop} onPress={() => setOpen(false)}>
          <Pressable style={styles.selectSheet} onPress={(e) => e.stopPropagation()}>
            <Text style={styles.selectSheetTitle}>{label ?? "Board"}</Text>
            {options.map((option) => {
              const active = option.id === value;
              return (
                <Pressable
                  key={option.id}
                  onPress={() => {
                    tapFeedback();
                    onChange(option.id);
                    setOpen(false);
                  }}
                  style={[styles.selectOption, active ? styles.selectOptionActive : null]}
                >
                  <Text style={[styles.selectOptionText, active ? styles.selectOptionTextActive : null]}>
                    {option.label}
                  </Text>
                  {active ? <Ionicons name="checkmark" size={18} color={colors.brandDark} /> : null}
                </Pressable>
              );
            })}
          </Pressable>
        </Pressable>
      </Modal>
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

export function LogoMark({
  size = 40,
  width,
  height,
  style,
}: {
  size?: number;
  width?: number;
  height?: number;
  style?: object;
}) {
  const imageWidth = width ?? size;
  const imageHeight = height ?? size;

  return (
    <Image
      source={require("../assets/images/kiji_logo.png")}
      style={[{ width: imageWidth, height: imageHeight }, style]}
      resizeMode="contain"
      accessibilityLabel="Kiji Technology"
    />
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
  cardGlass: {
    backgroundColor: "rgba(255, 255, 255, 0.92)",
    borderColor: "rgba(255, 255, 255, 0.95)",
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
  selectTrigger: {
    borderWidth: 1,
    borderColor: colors.borderStrong,
    borderRadius: radius.md,
    paddingHorizontal: 13,
    paddingVertical: 12,
    backgroundColor: colors.surface,
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
  },
  selectValue: { fontSize: 16, fontWeight: "600", color: colors.text },
  selectBackdrop: {
    flex: 1,
    backgroundColor: "rgba(15, 23, 42, 0.45)",
    justifyContent: "flex-end",
  },
  selectSheet: {
    backgroundColor: colors.surface,
    borderTopLeftRadius: radius.xl,
    borderTopRightRadius: radius.xl,
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.lg,
    paddingBottom: spacing.xl,
    gap: spacing.xs,
    ...shadow.md,
  },
  selectSheetTitle: {
    fontSize: 16,
    fontWeight: "800",
    color: colors.text,
    marginBottom: spacing.sm,
  },
  selectOption: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingVertical: 14,
    paddingHorizontal: spacing.md,
    borderRadius: radius.md,
  },
  selectOptionActive: { backgroundColor: colors.brandSoft },
  selectOptionText: { fontSize: 16, fontWeight: "600", color: colors.text },
  selectOptionTextActive: { color: colors.brandDark, fontWeight: "800" },
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
});

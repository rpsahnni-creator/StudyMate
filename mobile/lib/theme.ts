export const colors = {
  brand: "#20B2AA",
  brandDark: "#1A9A93",
  brandDarker: "#158880",
  accent: "#9ACD32",
  accentDark: "#7BA428",
  brandSoft: "#E6F7F6",
  brandSoftBorder: "#B2E8E4",
  accentSoft: "#EEF8D4",

  bg: "#9ACD32",
  surface: "#ffffff",
  surfaceAlt: "#f8fafc",

  text: "#0f172a",
  textMuted: "#64748b",
  textSubtle: "#94a3b8",

  border: "#e7e9f2",
  borderStrong: "#d7dae6",

  success: "#16a34a",
  successBg: "#ecfdf5",
  warning: "#d97706",
  warningBg: "#fffbeb",
  danger: "#dc2626",
  dangerBg: "#fef2f2",

  white: "#ffffff",
};

export const radius = {
  sm: 8,
  md: 12,
  lg: 16,
  xl: 22,
  full: 999,
};

export const spacing = {
  xs: 4,
  sm: 8,
  md: 12,
  lg: 16,
  xl: 24,
  xxl: 32,
};

export const shadow = {
  sm: {
    shadowColor: "#0f172a",
    shadowOpacity: 0.06,
    shadowRadius: 8,
    shadowOffset: { width: 0, height: 2 },
    elevation: 2,
  },
  md: {
    shadowColor: "#0f172a",
    shadowOpacity: 0.1,
    shadowRadius: 16,
    shadowOffset: { width: 0, height: 6 },
    elevation: 5,
  },
  brand: {
    shadowColor: "#20B2AA",
    shadowOpacity: 0.4,
    shadowRadius: 16,
    shadowOffset: { width: 0, height: 8 },
    elevation: 6,
  },
};

export function scoreColor(score: number): string {
  if (score > 70) return colors.success;
  if (score >= 50) return colors.warning;
  return colors.danger;
}

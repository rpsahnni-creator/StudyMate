export const colors = {
  brand: "#6366f1",
  brandDark: "#4f46e5",
  brandDarker: "#4338ca",
  accent: "#8b5cf6",
  brandSoft: "#eef2ff",
  brandSoftBorder: "#e0e7ff",

  bg: "#f6f7fb",
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
    shadowColor: "#6366f1",
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

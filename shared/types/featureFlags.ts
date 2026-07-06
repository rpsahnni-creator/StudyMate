// Shared between frontend/ and mobile/ — copy or symlink this file so both
// clients agree on flag keys. If you set up a monorepo tool (pnpm workspaces
// + turborepo, as planned), this becomes an actual shared package instead
// of a copy-pasted file.

export type FlagKey = "scan_quiz_module" | "career_goals_module";

// This shape matches exactly what GET /me/features returns from the Go backend.
export type ResolvedFlags = Record<FlagKey, boolean>;

export const DEFAULT_FLAGS: ResolvedFlags = {
  scan_quiz_module: true,
  career_goals_module: false,
};

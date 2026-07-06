import type { ReactNode } from "react";

export function AppShell({ children }: { children: ReactNode }) {
  return (
    <div className="app-shell">
      <div className="app-glow app-glow-cyan" aria-hidden />
      <div className="app-glow app-glow-orange" aria-hidden />
      <div className="app-glow app-glow-violet" aria-hidden />
      <div className="app-shell-inner">{children}</div>
    </div>
  );
}

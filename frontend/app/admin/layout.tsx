"use client";

import { useEffect } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useAuth } from "../../components/AuthProvider";

const NAV = [
  { href: "/admin/features", label: "Feature Flags" },
  { href: "/admin/users", label: "Users" },
  { href: "/admin/jobs", label: "Jobs" },
  { href: "/admin/ai-costs", label: "AI Costs" },
  { href: "/admin/content-flags", label: "Content Flags" },
  { href: "/admin/audit-log", label: "Audit Log" },
];

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  const { user, isLoggedIn, isLoading } = useAuth();
  const router = useRouter();
  const pathname = usePathname();

  const isAdmin = user?.role === "admin";

  useEffect(() => {
    if (isLoading) return;
    if (!isLoggedIn) {
      router.replace(`/auth/login?from=${encodeURIComponent(pathname)}`);
    }
  }, [isLoading, isLoggedIn, router, pathname]);

  if (isLoading) {
    return <div style={styles.centered}>Loading admin panel…</div>;
  }

  if (!isLoggedIn) {
    return <div style={styles.centered}>Redirecting to login…</div>;
  }

  if (!isAdmin) {
    return (
      <div style={styles.centered}>
        <div style={{ textAlign: "center" }}>
          <h2 style={{ margin: "0 0 8px" }}>403 — Admin access required</h2>
          <p style={{ color: "#6b7280" }}>Your account does not have admin privileges.</p>
          <Link href="/" style={styles.homeLink}>
            Back to dashboard
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div style={styles.shell}>
      <aside style={styles.sidebar}>
        <div style={styles.brandRow}>
          <span style={styles.logoMark}>S</span>
          <h1 style={styles.brand}>Admin</h1>
        </div>
        <nav style={styles.nav}>
          {NAV.map((item) => {
            const active = pathname === item.href || pathname.startsWith(`${item.href}/`);
            return (
              <Link
                key={item.href}
                href={item.href}
                style={{
                  ...styles.navLink,
                  ...(active ? styles.navLinkActive : {}),
                }}
              >
                {item.label}
              </Link>
            );
          })}
        </nav>
        <div style={styles.sidebarFooter}>
          <span style={styles.adminEmail}>{user?.email}</span>
          <Link href="/" style={styles.exitLink}>
            ← Exit admin
          </Link>
        </div>
      </aside>
      <main style={styles.content}>{children}</main>
    </div>
  );
}

const styles: Record<string, React.CSSProperties> = {
  shell: {
    display: "flex",
    minHeight: "calc(100vh - 57px)",
    background: "var(--bg)",
  },
  sidebar: {
    width: 236,
    flexShrink: 0,
    background: "linear-gradient(180deg,#111827 0%,#0b1220 100%)",
    color: "#e5e7eb",
    display: "flex",
    flexDirection: "column",
    padding: "20px 14px",
    position: "sticky",
    top: 57,
    alignSelf: "flex-start",
    height: "calc(100vh - 57px)",
  },
  brandRow: {
    display: "flex",
    alignItems: "center",
    gap: 10,
    margin: "0 0 22px",
    padding: "0 6px",
  },
  logoMark: {
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    width: 32,
    height: 32,
    borderRadius: 9,
    background: "var(--brand-gradient)",
    color: "#fff",
    fontWeight: 800,
    fontSize: 17,
    boxShadow: "var(--shadow-brand)",
  },
  brand: {
    fontSize: 18,
    fontWeight: 800,
    margin: 0,
    color: "#fff",
    letterSpacing: "-0.01em",
  },
  nav: {
    display: "flex",
    flexDirection: "column",
    gap: 4,
    flex: 1,
  },
  navLink: {
    padding: "10px 13px",
    borderRadius: "var(--r-md)",
    color: "#cbd5e1",
    textDecoration: "none",
    fontSize: 14,
    fontWeight: 600,
    transition: "background 0.15s ease, color 0.15s ease",
  },
  navLinkActive: {
    background: "var(--brand-gradient)",
    color: "#fff",
    boxShadow: "0 8px 20px -8px rgba(99,102,241,0.6)",
  },
  sidebarFooter: {
    marginTop: 20,
    padding: "14px 8px 0",
    borderTop: "1px solid rgba(255,255,255,0.08)",
    display: "flex",
    flexDirection: "column",
    gap: 8,
  },
  adminEmail: {
    fontSize: 12,
    color: "#94a3b8",
    wordBreak: "break-all",
  },
  exitLink: {
    fontSize: 13,
    color: "#a5b4fc",
    textDecoration: "none",
    fontWeight: 600,
  },
  content: {
    flex: 1,
    padding: 28,
    overflowX: "auto",
  },
  centered: {
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    minHeight: "60vh",
    color: "var(--text)",
  },
  homeLink: {
    display: "inline-block",
    marginTop: 12,
    color: "var(--brand-600)",
    textDecoration: "none",
    fontWeight: 600,
  },
};

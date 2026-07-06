"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { BookOpenCheck, LogOut } from "lucide-react";
import { useAuth } from "./AuthProvider";
import { SubscriptionBadge } from "./SubscriptionBadge";

const NAV_LINKS = [
  { href: "/scan", label: "Scan" },
  { href: "/reports", label: "Reports" },
  { href: "/plans", label: "Plans" },
];

export function AppNav() {
  const { isLoggedIn, logout } = useAuth();
  const pathname = usePathname() ?? "/";

  const isAuthPage = pathname.startsWith("/auth");
  if (isAuthPage) return null;

  return (
    <header style={styles.header}>
      <nav style={styles.nav}>
        <Link href="/" style={styles.brand}>
          <span style={styles.logoMark}>
            <BookOpenCheck size={18} strokeWidth={2.4} />
          </span>
          <span style={styles.brandText}>StudyApp</span>
        </Link>

        <div style={styles.links}>
          {NAV_LINKS.map((link) => {
            const active = pathname === link.href || pathname.startsWith(link.href + "/");
            return (
              <Link
                key={link.href}
                href={link.href}
                style={{ ...styles.link, ...(active ? styles.linkActive : {}) }}
              >
                {link.label}
              </Link>
            );
          })}
          {isLoggedIn ? (
            <Link
              href="/profile"
              style={{
                ...styles.link,
                ...(pathname.startsWith("/profile") ? styles.linkActive : {}),
              }}
            >
              Profile
            </Link>
          ) : null}
        </div>

        <div style={styles.actions}>
          <SubscriptionBadge />
          {isLoggedIn ? (
            <button
              type="button"
              onClick={logout}
              className="btn-reset nav-logout"
              style={styles.logoutBtn}
            >
              <LogOut size={14} strokeWidth={2.4} />
              Log out
            </button>
          ) : (
            <Link href="/auth/login" className="btn btn-primary" style={styles.loginBtn}>
              Log in
            </Link>
          )}
        </div>
      </nav>
    </header>
  );
}

const styles: Record<string, React.CSSProperties> = {
  header: {
    position: "sticky",
    top: 0,
    zIndex: 50,
    borderBottom: "1px solid var(--border)",
    background: "rgba(255, 255, 255, 0.72)",
    backdropFilter: "saturate(180%) blur(14px)",
    WebkitBackdropFilter: "saturate(180%) blur(14px)",
  },
  nav: {
    maxWidth: 1120,
    margin: "0 auto",
    padding: "12px 20px",
    display: "flex",
    alignItems: "center",
    gap: 20,
    flexWrap: "wrap",
  },
  brand: {
    display: "inline-flex",
    alignItems: "center",
    gap: 10,
    textDecoration: "none",
  },
  logoMark: {
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    width: 34,
    height: 34,
    borderRadius: 10,
    background: "var(--brand-gradient)",
    color: "#fff",
    fontWeight: 800,
    fontSize: 18,
    boxShadow: "var(--shadow-brand)",
  },
  brandText: {
    fontWeight: 800,
    fontSize: 18,
    color: "var(--text)",
    letterSpacing: "-0.02em",
  },
  links: {
    display: "flex",
    gap: 4,
    flex: 1,
    flexWrap: "wrap",
  },
  link: {
    padding: "7px 13px",
    borderRadius: 999,
    fontSize: 14,
    fontWeight: 600,
    color: "var(--text-muted)",
    textDecoration: "none",
    transition: "color 0.15s ease, background 0.15s ease",
  },
  linkActive: {
    color: "var(--brand-700)",
    background: "var(--brand-50)",
  },
  actions: {
    display: "flex",
    alignItems: "center",
    gap: 12,
    marginLeft: "auto",
  },
  logoutBtn: {
    display: "inline-flex",
    alignItems: "center",
    gap: 6,
    border: "1px solid var(--border-strong)",
    background: "var(--surface)",
    borderRadius: 10,
    padding: "8px 14px",
    cursor: "pointer",
    fontSize: 13,
    fontWeight: 600,
    color: "var(--text)",
    transition: "border-color 0.15s ease, color 0.15s ease, background 0.15s ease",
  },
  loginBtn: {
    padding: "8px 16px",
    fontSize: 14,
  },
};

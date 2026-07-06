"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { LogOut } from "lucide-react";
import { useAuth } from "./AuthProvider";
import { SubscriptionBadge } from "./SubscriptionBadge";
import { KijiLogo } from "./KijiLogo";

const NAV_LINKS = [
  { href: "/scan", label: "Scan" },
  { href: "/reports", label: "Reports" },
  { href: "/plans", label: "Plans" },
];

export function AppNav() {
  const { isLoggedIn, user, logout, isLoading } = useAuth();
  const pathname = usePathname() ?? "/";

  const isAuthPage = pathname.startsWith("/auth") || pathname === "/reset-password";
  if (isAuthPage) return null;

  return (
    <header className="app-nav">
      <nav className="app-nav-inner" aria-label="Main">
        <Link href="/" className="app-nav-brand" aria-label="Kiji Technology — StudyMate home">
          <KijiLogo width={196} height={70} className="kiji-logo-nav" priority />
          <span className="app-nav-company-name">Kiji Technology</span>
        </Link>

        <div className="app-nav-links">
          {NAV_LINKS.map((link) => {
            const active = pathname === link.href || pathname.startsWith(link.href + "/");
            return (
              <Link
                key={link.href}
                href={link.href}
                className={`app-nav-link${active ? " app-nav-link-active" : ""}`}
              >
                {link.label}
              </Link>
            );
          })}
          {isLoggedIn ? (
            <Link
              href="/profile"
              className={`app-nav-link${pathname.startsWith("/profile") ? " app-nav-link-active" : ""}`}
            >
              Profile
            </Link>
          ) : null}
        </div>

        <div className="app-nav-actions">
          {isLoggedIn ? <SubscriptionBadge /> : null}
          {isLoading ? null : isLoggedIn ? (
            <>
              {user?.name ? <span className="app-nav-user">{user.name}</span> : null}
              <button
                type="button"
                onClick={logout}
                className="btn-reset app-nav-signout"
                aria-label="Sign out"
              >
                <LogOut size={15} strokeWidth={2.2} />
                Sign out
              </button>
            </>
          ) : (
            <Link href="/auth/login" className="btn btn-gold app-nav-login">
              Log in
            </Link>
          )}
        </div>
      </nav>
    </header>
  );
}

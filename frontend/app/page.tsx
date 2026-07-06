"use client";

import Link from "next/link";
import {
  ArrowRight,
  BarChart3,
  ScanLine,
  Sparkles,
  Star,
  Target,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { useAuth } from "../components/AuthProvider";
import { FeatureGate } from "../components/FeatureGate";

export default function DashboardPage() {
  const { isLoggedIn, user } = useAuth();

  return (
    <main style={styles.page}>
      <section style={styles.hero} className="animate-in">
        <span
          className="glow-orb"
          style={{
            width: 420,
            height: 420,
            top: -180,
            left: "50%",
            marginLeft: -340,
            background:
              "radial-gradient(circle, rgba(99,102,241,0.28), rgba(99,102,241,0) 70%)",
            animation: "floatSlow 12s ease-in-out infinite",
          }}
        />
        <span
          className="glow-orb"
          style={{
            width: 360,
            height: 360,
            top: -120,
            left: "50%",
            marginLeft: 60,
            background:
              "radial-gradient(circle, rgba(168,85,247,0.22), rgba(168,85,247,0) 70%)",
            animation: "floatSlower 14s ease-in-out infinite",
          }}
        />

        <div style={styles.heroInner}>
          <span style={styles.eyebrow}>
            <Sparkles size={13} strokeWidth={2.5} />
            AI-powered study companion
          </span>
          <h1 style={styles.heroTitle}>
            {isLoggedIn && user?.name ? (
              <>
                Welcome back,{" "}
                <span style={styles.gradientText}>{user.name.split(" ")[0]}</span>
              </>
            ) : (
              <>
                Scan. Practice.{" "}
                <span style={styles.gradientText}>Improve.</span>
              </>
            )}
          </h1>
          <p style={styles.heroSub}>
            Turn any NCERT or state-board chapter into smart quizzes, track your
            progress with rich analytics, and master your weak topics — all in one place.
          </p>
          <div style={styles.heroCta}>
            <Link href="/scan" className="btn btn-primary" style={styles.ctaPrimary}>
              Start scanning
              <ArrowRight size={17} strokeWidth={2.4} />
            </Link>
            <Link href="/reports" className="btn btn-ghost" style={styles.ctaGhost}>
              <BarChart3 size={17} strokeWidth={2.2} />
              View my reports
            </Link>
          </div>
        </div>
      </section>

      <section style={styles.grid}>
        <ActionCard
          href="/scan"
          icon={ScanLine}
          accent="linear-gradient(135deg,#6366f1,#8b5cf6)"
          title="Scan Chapter"
          desc="Capture questions from your book and generate an instant quiz."
        />
        <ActionCard
          href="/reports"
          icon={BarChart3}
          accent="linear-gradient(135deg,#0ea5e9,#22d3ee)"
          title="My Reports"
          desc="Track scores, streaks, and subject-wise performance over time."
        />
        <ActionCard
          href="/plans"
          icon={Star}
          accent="linear-gradient(135deg,#f59e0b,#f43f5e)"
          title="Plans & Billing"
          desc="Unlock unlimited scans and advanced AI quizzes."
        />
        <FeatureGate flag="career_goals_module">
          <ActionCard
            href="/goals"
            icon={Target}
            accent="linear-gradient(135deg,#10b981,#14b8a6)"
            title="Career Goals"
            desc="Set targets and follow a daily practice path to reach them."
          />
        </FeatureGate>
      </section>

      <section style={styles.stripe} className="card">
        <Sparkles size={140} strokeWidth={1} style={styles.stripeDecor} aria-hidden />
        <div style={styles.stripeContent}>
          <h2 style={styles.stripeTitle}>Ready to level up your prep?</h2>
          <p style={styles.stripeSub}>
            Consistent daily practice is the fastest way to raise your scores.
          </p>
        </div>
        <Link href="/scan" className="btn btn-primary" style={styles.stripeCta}>
          Create a quiz
          <ArrowRight size={17} strokeWidth={2.4} />
        </Link>
      </section>
    </main>
  );
}

function ActionCard({
  href,
  icon: Icon,
  accent,
  title,
  desc,
}: {
  href: string;
  icon: LucideIcon;
  accent: string;
  title: string;
  desc: string;
}) {
  return (
    <Link href={href} className="card card-hover action-card" style={styles.actionCard}>
      <span className="action-icon" style={{ ...styles.actionIcon, background: accent }}>
        <Icon size={24} strokeWidth={2.2} color="#fff" />
      </span>
      <div style={styles.actionBody}>
        <h3 style={styles.actionTitle}>{title}</h3>
        <p style={styles.actionDesc}>{desc}</p>
      </div>
      <span className="action-arrow" style={styles.actionArrow} aria-hidden>
        <ArrowRight size={18} strokeWidth={2.4} />
      </span>
    </Link>
  );
}

const styles: Record<string, React.CSSProperties> = {
  page: {
    maxWidth: 1120,
    margin: "0 auto",
    padding: "40px 20px 64px",
    display: "grid",
    gap: 44,
  },
  hero: {
    position: "relative",
    textAlign: "center",
    padding: "40px 0 16px",
    overflow: "hidden",
  },
  heroInner: {
    position: "relative",
    zIndex: 1,
    display: "grid",
    gap: 18,
    justifyItems: "center",
  },
  eyebrow: {
    display: "inline-flex",
    alignItems: "center",
    gap: 7,
    padding: "6px 14px",
    borderRadius: 999,
    background: "var(--brand-50)",
    color: "var(--brand-700)",
    fontSize: 13,
    fontWeight: 650,
    border: "1px solid var(--brand-100)",
  },
  heroTitle: {
    fontSize: "clamp(34px, 6vw, 58px)",
    fontWeight: 850,
    margin: 0,
    maxWidth: 780,
    letterSpacing: "-0.03em",
  },
  gradientText: {
    background: "var(--brand-gradient)",
    WebkitBackgroundClip: "text",
    backgroundClip: "text",
    WebkitTextFillColor: "transparent",
    color: "transparent",
  },
  heroSub: {
    fontSize: 18,
    color: "var(--text-muted)",
    margin: 0,
    maxWidth: 620,
    lineHeight: 1.6,
  },
  heroCta: {
    display: "flex",
    gap: 12,
    flexWrap: "wrap",
    justifyContent: "center",
    marginTop: 6,
  },
  ctaPrimary: {
    padding: "13px 24px",
    fontSize: 16,
  },
  ctaGhost: {
    padding: "13px 24px",
    fontSize: 16,
  },
  grid: {
    display: "grid",
    gridTemplateColumns: "repeat(auto-fit, minmax(250px, 1fr))",
    gap: 18,
  },
  actionCard: {
    display: "flex",
    alignItems: "center",
    gap: 16,
    padding: 22,
    textDecoration: "none",
    color: "var(--text)",
  },
  actionIcon: {
    position: "relative",
    zIndex: 1,
    flexShrink: 0,
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    width: 52,
    height: 52,
    borderRadius: 14,
    boxShadow: "var(--shadow-md)",
  },
  actionBody: {
    position: "relative",
    zIndex: 1,
  },
  actionTitle: {
    margin: "0 0 4px",
    fontSize: 17,
    fontWeight: 750,
  },
  actionDesc: {
    margin: 0,
    fontSize: 13.5,
    color: "var(--text-muted)",
    lineHeight: 1.5,
  },
  actionArrow: {
    position: "relative",
    zIndex: 1,
    marginLeft: "auto",
    color: "var(--text-subtle)",
    display: "inline-flex",
  },
  stripe: {
    position: "relative",
    overflow: "hidden",
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    gap: 20,
    padding: "30px 32px",
    background: "var(--brand-gradient-soft)",
    flexWrap: "wrap",
  },
  stripeDecor: {
    position: "absolute",
    right: -20,
    top: "50%",
    transform: "translateY(-50%)",
    color: "var(--brand-500)",
    opacity: 0.08,
  },
  stripeContent: {
    position: "relative",
    zIndex: 1,
  },
  stripeTitle: {
    margin: "0 0 4px",
    fontSize: 22,
    fontWeight: 800,
  },
  stripeSub: {
    margin: 0,
    color: "var(--text-muted)",
  },
  stripeCta: {
    position: "relative",
    zIndex: 1,
    padding: "13px 24px",
    fontSize: 16,
  },
};

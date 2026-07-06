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
import { StudyMateWord } from "../components/StudyMateWord";

export default function DashboardPage() {
  const { isLoggedIn, user } = useAuth();

  return (
    <main style={styles.page}>
      <section style={styles.hero} className="animate-in">
        <div style={styles.heroInner}>
          <h1 style={styles.heroTitle}>
            <StudyMateWord size="lg" />
          </h1>
          <p style={styles.tagline}>Learn Smarter Every Day</p>
          <p style={styles.heroSub}>
            {isLoggedIn && user?.name ? (
              <>Welcome back, <strong style={{ color: "#fff" }}>{user.name.split(" ")[0]}</strong>. </>
            ) : null}
            AI quizzes, track progress, and master weak topics.
          </p>
          <div style={styles.heroCta}>
            <Link href="/scan" className="btn btn-gold" style={styles.ctaPrimary}>
              Start scanning
              <ArrowRight size={17} strokeWidth={2.4} />
            </Link>
            <Link href="/reports" className="btn btn-ghost" style={styles.ctaGhost}>
              <BarChart3 size={17} strokeWidth={2.2} />
              View reports
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
          accent="linear-gradient(135deg,#f0b429,#f59e0b)"
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

      <section className="glass-panel" style={styles.stripe}>
        <Sparkles size={120} strokeWidth={1} style={styles.stripeDecor} aria-hidden />
        <div style={styles.stripeContent}>
          <h2 style={styles.stripeTitle}>Ready to level up your prep?</h2>
          <p style={styles.stripeSub}>
            Consistent daily practice is the fastest way to raise your scores.
          </p>
        </div>
        <Link href="/scan" className="btn btn-gold" style={styles.stripeCta}>
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
    padding: "32px 20px 64px",
    display: "grid",
    gap: 40,
  },
  hero: {
    textAlign: "center",
    padding: "12px 0 8px",
  },
  heroInner: {
    display: "grid",
    gap: 14,
    justifyItems: "center",
  },
  heroTitle: {
    margin: 0,
  },
  tagline: {
    margin: 0,
    fontSize: 16,
    color: "rgba(255,255,255,0.72)",
    letterSpacing: "0.02em",
  },
  heroSub: {
    fontSize: 17,
    color: "var(--text-muted)",
    margin: 0,
    maxWidth: 560,
    lineHeight: 1.65,
  },
  heroCta: {
    display: "flex",
    gap: 12,
    flexWrap: "wrap",
    justifyContent: "center",
    marginTop: 8,
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
    color: "#0f172a",
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
    color: "#0f172a",
  },
  actionDesc: {
    margin: 0,
    fontSize: 13.5,
    color: "#64748b",
    lineHeight: 1.5,
  },
  actionArrow: {
    position: "relative",
    zIndex: 1,
    marginLeft: "auto",
    color: "#94a3b8",
    display: "inline-flex",
  },
  stripe: {
    position: "relative",
    overflow: "hidden",
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    gap: 20,
    padding: "28px 30px",
    flexWrap: "wrap",
  },
  stripeDecor: {
    position: "absolute",
    right: -10,
    top: "50%",
    transform: "translateY(-50%)",
    color: "var(--gold)",
    opacity: 0.12,
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

import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import { AuthProvider } from "../components/AuthProvider";
import { AppNav } from "../components/AppNav";
import { FeatureFlagsProvider } from "../lib/featureFlags";

const inter = Inter({
  subsets: ["latin"],
  display: "swap",
  variable: "--font-sans",
});

export const metadata: Metadata = {
  title: "StudyApp — Scan, Practice, Improve",
  description:
    "Scan NCERT & state-board chapters, generate AI quizzes, and track your performance analytics.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={inter.variable}>
      <body>
        <AuthProvider>
          <FeatureFlagsProvider>
            <AppNav />
            {children}
          </FeatureFlagsProvider>
        </AuthProvider>
      </body>
    </html>
  );
}

import type { Metadata } from "next";
import { Outfit } from "next/font/google";
import "./globals.css";
import { AuthProvider } from "../components/AuthProvider";
import { AppNav } from "../components/AppNav";
import { AppShell } from "../components/AppShell";
import { FeatureFlagsProvider } from "../lib/featureFlags";

const outfit = Outfit({
  subsets: ["latin"],
  weight: ["300", "800"],
  display: "swap",
  variable: "--font-display",
});

export const metadata: Metadata = {
  title: "StudyMate — Learn Smarter Every Day",
  description:
    "Scan NCERT & state-board chapters, generate AI quizzes, and track your performance with StudyMate by Kiji Technology.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={outfit.variable}>
      <body>
        <AuthProvider>
          <FeatureFlagsProvider>
            <AppShell>
              <AppNav />
              {children}
            </AppShell>
          </FeatureFlagsProvider>
        </AuthProvider>
      </body>
    </html>
  );
}

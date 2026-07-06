"use client";

import { FeatureGate } from "./FeatureGate";

export function DashboardNav() {
  return (
    <nav>
      <a href="/scan">Scan Chapter / Questions</a>
      <a href="/plans">Plans & Billing</a>
      <a href="/reports">My Reports</a>
      <FeatureGate flag="career_goals_module">
        <a href="/goals">Career Goals</a>
        <a href="/practice/daily">Daily Practice</a>
      </FeatureGate>
    </nav>
  );
}

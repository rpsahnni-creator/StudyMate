"use client";

import { ReactNode } from "react";
import { useFeatureFlag } from "../lib/featureFlags";
import type { FlagKey } from "../types/featureFlags";

interface FeatureGateProps {
  flag: FlagKey;
  children: ReactNode;
  fallback?: ReactNode;
}

export function FeatureGate({ flag, children, fallback = null }: FeatureGateProps) {
  const enabled = useFeatureFlag(flag);
  return <>{enabled ? children : fallback}</>;
}
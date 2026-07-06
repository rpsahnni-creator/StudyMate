"use client";

import { useEffect, useRef, useState } from "react";
import type { FlagKey } from "../../../types/featureFlags";

interface FlagWithStats {
  key: FlagKey;
  enabled: boolean;
  rollout_percentage: number;
  updated_by: string;
  updated_at: string;
  override_count: number;
  ai_call_count_24h: number;
  ai_cache_hit_rate_24h: number;
  ai_estimated_cost_24h_usd: number;
}

import { API_URL, fetchWithAuth } from "../../../lib/auth";
import { tableStyles } from "../../../components/admin/ui";

export default function AdminFeatureFlagsPage() {
  const [flags, setFlags] = useState<FlagWithStats[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [savingKey, setSavingKey] = useState<FlagKey | null>(null);
  const inFlight = useRef<Set<string>>(new Set());

  async function loadFlags() {
    setLoading(true);
    setError(null);
    try {
      const res = await fetchWithAuth(`${API_URL}/admin/features`);
      if (!res.ok) {
        const text = await res.text();
        throw new Error("Backend returned " + res.status + ": " + text);
      }
      const data: FlagWithStats[] = await res.json();
      setFlags(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error loading flags");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadFlags();
  }, []);

  async function updateFlag(key: FlagKey, enabled: boolean, rollout: number) {
    if (inFlight.current.has(key)) return;
    inFlight.current.add(key);
    setSavingKey(key);
    try {
      const res = await fetchWithAuth(`${API_URL}/admin/features`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ key: key, enabled: enabled, rollout_percentage: rollout }),
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error("Backend returned " + res.status + ": " + text);
      }
      await loadFlags();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error updating flag");
    } finally {
      inFlight.current.delete(key);
      setSavingKey(null);
    }
  }

  if (loading) return <p style={{ color: "var(--text-muted)" }}>Loading feature flags…</p>;
  if (error) {
    return (
      <div style={{ maxWidth: 900 }}>
        <h1 style={pageStyles.title}>Feature Flags</h1>
        <div style={pageStyles.errorBox}>
          <strong>Failed to load feature flags:</strong>
          <p style={{ fontFamily: "monospace", margin: "6px 0 12px", wordBreak: "break-all" }}>{error}</p>
          <button onClick={loadFlags}>Retry</button>
        </div>
      </div>
    );
  }

  return (
    <main style={{ maxWidth: 980, margin: "0 auto" }} className="animate-in">
      <h1 style={pageStyles.title}>Feature Flags</h1>
      <p style={pageStyles.lead}>
        Toggle modules on/off instantly — no deploy needed. Watch the cost/usage
        columns before dialing rollout up.
      </p>

      <div style={{ overflowX: "auto", marginTop: 20 }}>
        <table style={tableStyles.table}>
          <thead>
            <tr>
              <th style={tableStyles.th}>Module</th>
              <th style={tableStyles.th}>Status</th>
              <th style={tableStyles.th}>Rollout %</th>
              <th style={tableStyles.th}>Beta overrides</th>
              <th style={tableStyles.th}>AI calls (24h)</th>
              <th style={tableStyles.th}>Cache hit rate</th>
              <th style={tableStyles.th}>Est. cost (24h)</th>
              <th style={tableStyles.th}>Last updated</th>
            </tr>
          </thead>
          <tbody>
            {flags.map((flag) => (
              <FlagRow
                key={flag.key}
                flag={flag}
                saving={savingKey === flag.key}
                onUpdate={updateFlag}
              />
            ))}
          </tbody>
        </table>
      </div>
    </main>
  );
}

const pageStyles: Record<string, React.CSSProperties> = {
  title: { fontSize: 28, fontWeight: 800, margin: "0 0 6px" },
  lead: { color: "var(--text-muted)", margin: 0, fontSize: 15 },
  errorBox: {
    marginTop: 16,
    padding: 16,
    borderRadius: "var(--r-md)",
    background: "var(--danger-bg)",
    color: "var(--danger)",
  },
};

function FlagRow({
  flag,
  saving,
  onUpdate,
}: {
  flag: FlagWithStats;
  saving: boolean;
  onUpdate: (key: FlagKey, enabled: boolean, rollout: number) => void;
}) {
  const [rollout, setRollout] = useState(flag.rollout_percentage);
  const checkboxId = "flag-toggle-" + flag.key;
  const costWarning = flag.ai_estimated_cost_24h_usd > 5;

  return (
    <tr>
      <td style={{ ...tableStyles.td, fontFamily: "monospace", fontWeight: 600 }}>{flag.key}</td>

      <td style={tableStyles.td}>
        <label htmlFor={checkboxId} style={{ display: "inline-flex", alignItems: "center", gap: 8, cursor: "pointer" }}>
          <input
            id={checkboxId}
            type="checkbox"
            checked={flag.enabled}
            disabled={saving}
            onChange={(e) => onUpdate(flag.key, e.target.checked, rollout)}
          />
          <span
            style={{
              padding: "2px 10px",
              borderRadius: 999,
              fontSize: 12,
              fontWeight: 700,
              background: flag.enabled ? "var(--success-bg)" : "var(--surface-2)",
              color: flag.enabled ? "var(--success)" : "var(--text-muted)",
            }}
          >
            {flag.enabled ? "On" : "Off"}
          </span>
        </label>
      </td>

      <td style={tableStyles.td}>
        <input
          type="number"
          min={0}
          max={100}
          value={rollout}
          disabled={saving || !flag.enabled}
          onChange={(e) => setRollout(Number(e.target.value))}
          onBlur={() => onUpdate(flag.key, flag.enabled, rollout)}
          style={{ width: "68px", display: "inline-block" }}
        />
        %
      </td>

      <td style={tableStyles.td}>{flag.override_count}</td>
      <td style={tableStyles.td}>{flag.ai_call_count_24h}</td>
      <td style={tableStyles.td}>
        {(flag.ai_cache_hit_rate_24h * 100).toFixed(0)}%
      </td>
      <td style={{ ...tableStyles.td, color: costWarning ? "var(--danger)" : "var(--text)", fontWeight: costWarning ? 700 : 400 }}>
        ${flag.ai_estimated_cost_24h_usd.toFixed(2)}
        {costWarning && " ⚠"}
      </td>
      <td style={{ ...tableStyles.td, fontSize: "0.85rem", color: "var(--text-muted)" }}>
        {new Date(flag.updated_at).toLocaleString()} by {flag.updated_by}
      </td>
    </tr>
  );
}

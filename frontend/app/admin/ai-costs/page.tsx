"use client";

import { useCallback, useEffect, useState } from "react";
import { getAICosts, type AICostSummary } from "../../../lib/admin";
import { AdminCard, tableStyles } from "../../../components/admin/ui";

const DAILY_COST_WARN = 5; // USD per day threshold

function isoDaysAgo(days: number): string {
  const d = new Date();
  d.setDate(d.getDate() - days);
  return d.toISOString().slice(0, 10);
}

function todayISO(): string {
  return new Date().toISOString().slice(0, 10);
}

export default function AdminAICostsPage() {
  const [from, setFrom] = useState(isoDaysAgo(30));
  const [to, setTo] = useState(todayISO());
  const [data, setData] = useState<AICostSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await getAICosts(from, to);
      setData(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load AI costs");
    } finally {
      setLoading(false);
    }
  }, [from, to]);

  useEffect(() => {
    void load();
  }, [load]);

  // Daily totals for threshold warnings.
  const dailyTotals = new Map<string, number>();
  data?.rows.forEach((row) => {
    dailyTotals.set(row.day, (dailyTotals.get(row.day) ?? 0) + row.total_cost_estimate);
  });

  return (
    <div>
      <h1 style={{ margin: "0 0 16px", fontSize: 24 }}>AI Costs</h1>

      <div style={{ display: "flex", gap: 12, alignItems: "flex-end", marginBottom: 20, flexWrap: "wrap" }}>
        <label style={labelStyle}>
          From
          <input type="date" value={from} max={to} onChange={(e) => setFrom(e.target.value)} style={dateInput} />
        </label>
        <label style={labelStyle}>
          To
          <input type="date" value={to} min={from} onChange={(e) => setTo(e.target.value)} style={dateInput} />
        </label>
        <button type="button" onClick={() => void load()} style={btnPrimary}>
          Apply
        </button>
      </div>

      {error ? <p style={{ color: "#b91c1c" }}>{error}</p> : null}

      {loading ? (
        <p style={{ color: "#6b7280" }}>Loading costs…</p>
      ) : data ? (
        <>
          <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))", gap: 16, marginBottom: 20 }}>
            <SummaryCard label="Total Cost" value={`$${data.total_cost_estimate.toFixed(2)}`} />
            <SummaryCard label="Total Requests" value={data.total_requests.toLocaleString()} />
            <SummaryCard label="Total Tokens" value={data.total_tokens.toLocaleString()} />
            <SummaryCard
              label="Avg Cost / Request"
              value={`$${data.avg_cost_per_request.toFixed(4)}`}
            />
          </div>

          <AdminCard>
            <table style={tableStyles.table}>
              <thead>
                <tr>
                  <th style={tableStyles.th}>Date</th>
                  <th style={tableStyles.th}>Provider</th>
                  <th style={tableStyles.th}>Model</th>
                  <th style={tableStyles.th}>Requests</th>
                  <th style={tableStyles.th}>Tokens</th>
                  <th style={tableStyles.th}>Cost</th>
                </tr>
              </thead>
              <tbody>
                {data.rows.map((row, idx) => {
                  const overThreshold = (dailyTotals.get(row.day) ?? 0) > DAILY_COST_WARN;
                  return (
                    <tr key={`${row.day}-${row.provider}-${row.model}-${idx}`}>
                      <td style={{ ...tableStyles.td, color: overThreshold ? "#b91c1c" : undefined }}>
                        {row.day}
                        {overThreshold ? " ⚠" : ""}
                      </td>
                      <td style={tableStyles.td}>{row.provider}</td>
                      <td style={{ ...tableStyles.td, fontFamily: "monospace", fontSize: 13 }}>{row.model}</td>
                      <td style={tableStyles.td}>{row.request_count.toLocaleString()}</td>
                      <td style={tableStyles.td}>{row.total_tokens.toLocaleString()}</td>
                      <td style={tableStyles.td}>${row.total_cost_estimate.toFixed(4)}</td>
                    </tr>
                  );
                })}
                {data.rows.length === 0 ? (
                  <tr>
                    <td style={tableStyles.td} colSpan={6}>
                      No AI generation logs in this range.
                    </td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </AdminCard>
        </>
      ) : null}
    </div>
  );
}

function SummaryCard({ label, value }: { label: string; value: string }) {
  return (
    <div style={{ background: "#fff", borderRadius: 12, padding: 20, boxShadow: "0 1px 3px rgba(0,0,0,0.06)" }}>
      <div style={{ fontSize: 13, color: "#6b7280" }}>{label}</div>
      <div style={{ fontSize: 24, fontWeight: 700, marginTop: 6 }}>{value}</div>
    </div>
  );
}

const labelStyle: React.CSSProperties = {
  display: "flex",
  flexDirection: "column",
  gap: 4,
  fontSize: 13,
  color: "#6b7280",
};
const dateInput: React.CSSProperties = {
  border: "1px solid #d1d5db",
  borderRadius: 8,
  padding: "8px 12px",
  fontSize: 14,
};
const btnPrimary: React.CSSProperties = {
  background: "#2563eb",
  color: "#fff",
  border: "none",
  borderRadius: 8,
  padding: "9px 18px",
  cursor: "pointer",
};

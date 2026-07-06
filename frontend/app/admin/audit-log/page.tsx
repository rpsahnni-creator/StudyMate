"use client";

import { useCallback, useEffect, useState } from "react";
import { formatIST, getAuditLogs, type AuditLogEntry } from "../../../lib/admin";
import { AdminCard, Pagination, tableStyles } from "../../../components/admin/ui";

const LIMIT = 50;

export default function AdminAuditLogPage() {
  const [logs, setLogs] = useState<AuditLogEntry[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [userId, setUserId] = useState<number | undefined>(undefined);
  const [userIdInput, setUserIdInput] = useState("");
  const [actionFilter, setActionFilter] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getAuditLogs(page, LIMIT, userId);
      setLogs(data.items ?? []);
      setTotal(data.total);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load audit logs");
    } finally {
      setLoading(false);
    }
  }, [page, userId]);

  useEffect(() => {
    void load();
  }, [load]);

  function applyUserFilter(e: React.FormEvent) {
    e.preventDefault();
    const parsed = parseInt(userIdInput.trim(), 10);
    setUserId(Number.isNaN(parsed) ? undefined : parsed);
    setPage(1);
  }

  const actionTypes = Array.from(new Set(logs.map((l) => l.action))).sort();
  const visibleLogs = actionFilter ? logs.filter((l) => l.action === actionFilter) : logs;

  return (
    <div>
      <h1 style={{ margin: "0 0 16px", fontSize: 24 }}>Audit Log</h1>

      <div style={{ display: "flex", gap: 12, marginBottom: 16, flexWrap: "wrap", alignItems: "flex-end" }}>
        <form onSubmit={applyUserFilter} style={{ display: "flex", gap: 8, alignItems: "flex-end" }}>
          <label style={labelStyle}>
            Actor user ID
            <input
              value={userIdInput}
              onChange={(e) => setUserIdInput(e.target.value)}
              placeholder="e.g. 42"
              style={inputStyle}
            />
          </label>
          <button type="submit" style={btnPrimary}>
            Filter
          </button>
          {userId ? (
            <button
              type="button"
              style={btnGhost}
              onClick={() => {
                setUserId(undefined);
                setUserIdInput("");
                setPage(1);
              }}
            >
              Clear
            </button>
          ) : null}
        </form>

        <label style={labelStyle}>
          Action type (this page)
          <select value={actionFilter} onChange={(e) => setActionFilter(e.target.value)} style={inputStyle}>
            <option value="">All actions</option>
            {actionTypes.map((a) => (
              <option key={a} value={a}>
                {a}
              </option>
            ))}
          </select>
        </label>
      </div>

      {error ? <p style={{ color: "#b91c1c" }}>{error}</p> : null}

      <AdminCard>
        {loading ? (
          <p style={{ color: "#6b7280" }}>Loading audit log…</p>
        ) : (
          <table style={tableStyles.table}>
            <thead>
              <tr>
                <th style={tableStyles.th}>Timestamp (IST)</th>
                <th style={tableStyles.th}>User</th>
                <th style={tableStyles.th}>Action</th>
                <th style={tableStyles.th}>Resource</th>
                <th style={tableStyles.th}>Details</th>
              </tr>
            </thead>
            <tbody>
              {visibleLogs.map((log) => (
                <tr key={log.id}>
                  <td style={tableStyles.td}>{formatIST(log.created_at)}</td>
                  <td style={tableStyles.td}>{log.actor_email || (log.actor_user_id ? `#${log.actor_user_id}` : "system")}</td>
                  <td style={{ ...tableStyles.td, fontFamily: "monospace", fontSize: 13 }}>{log.action}</td>
                  <td style={tableStyles.td}>{log.entity_type}</td>
                  <td style={tableStyles.td}>{log.entity_id || "—"}</td>
                </tr>
              ))}
              {visibleLogs.length === 0 ? (
                <tr>
                  <td style={tableStyles.td} colSpan={5}>
                    No audit entries found.
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        )}
        <Pagination page={page} total={total} limit={LIMIT} onChange={setPage} />
      </AdminCard>
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
const inputStyle: React.CSSProperties = {
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
  padding: "9px 16px",
  cursor: "pointer",
};
const btnGhost: React.CSSProperties = {
  background: "#fff",
  color: "#374151",
  border: "1px solid #d1d5db",
  borderRadius: 8,
  padding: "9px 14px",
  cursor: "pointer",
};

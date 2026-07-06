"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import {
  formatDuration,
  formatIST,
  getJobs,
  retryJob,
  type AdminJob,
} from "../../../lib/admin";
import { AdminCard, Pagination, StatusBadge, tableStyles } from "../../../components/admin/ui";

const LIMIT = 20;
const REFRESH_MS = 30000;

const TABS: { label: string; value: string }[] = [
  { label: "All", value: "" },
  { label: "Pending", value: "pending" },
  { label: "Processing", value: "processing" },
  { label: "Failed", value: "failed" },
];

export default function AdminJobsPage() {
  const [jobs, setJobs] = useState<AdminJob[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [retrying, setRetrying] = useState<number | null>(null);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  const load = useCallback(
    async (showSpinner = true) => {
      if (showSpinner) setLoading(true);
      setError(null);
      try {
        const data = await getJobs(status, page, LIMIT);
        setJobs(data.items ?? []);
        setTotal(data.total);
        setLastUpdated(new Date());
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load jobs");
      } finally {
        setLoading(false);
      }
    },
    [status, page]
  );

  useEffect(() => {
    void load(true);
  }, [load]);

  // Auto-refresh every 30s without a spinner flash.
  const loadRef = useRef(load);
  loadRef.current = load;
  useEffect(() => {
    const id = setInterval(() => void loadRef.current(false), REFRESH_MS);
    return () => clearInterval(id);
  }, []);

  async function handleRetry(job: AdminJob) {
    setRetrying(job.id);
    try {
      await retryJob(job.id);
      await load(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to retry job");
    } finally {
      setRetrying(null);
    }
  }

  return (
    <div>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline" }}>
        <h1 style={{ margin: "0 0 16px", fontSize: 24 }}>Scan Jobs</h1>
        <span style={{ fontSize: 12, color: "#9ca3af" }}>
          {lastUpdated ? `Updated ${formatIST(lastUpdated.toISOString())} · auto-refresh 30s` : ""}
        </span>
      </div>

      <div style={{ display: "flex", gap: 6, marginBottom: 16 }}>
        {TABS.map((tab) => (
          <button
            key={tab.value}
            type="button"
            onClick={() => {
              setStatus(tab.value);
              setPage(1);
            }}
            style={{
              ...tabStyle,
              ...(status === tab.value ? tabActive : {}),
            }}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {error ? <p style={{ color: "#b91c1c" }}>{error}</p> : null}

      <AdminCard>
        {loading ? (
          <p style={{ color: "#6b7280" }}>Loading jobs…</p>
        ) : (
          <table style={tableStyles.table}>
            <thead>
              <tr>
                <th style={tableStyles.th}>Job</th>
                <th style={tableStyles.th}>User</th>
                <th style={tableStyles.th}>Status</th>
                <th style={tableStyles.th}>Provider</th>
                <th style={tableStyles.th}>Created</th>
                <th style={tableStyles.th}>Duration</th>
                <th style={tableStyles.th}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {jobs.map((job) => (
                <tr key={job.id}>
                  <td style={{ ...tableStyles.td, fontFamily: "monospace" }}>#{job.id}</td>
                  <td style={tableStyles.td}>{job.user_email}</td>
                  <td style={tableStyles.td}>
                    <StatusBadge status={job.status} />
                    {job.status === "failed" && job.error_message ? (
                      <div style={{ color: "#b91c1c", fontSize: 12, marginTop: 4 }}>
                        {job.error_message}
                      </div>
                    ) : null}
                  </td>
                  <td style={tableStyles.td}>{job.provider || "—"}</td>
                  <td style={tableStyles.td}>{formatIST(job.created_at)}</td>
                  <td style={tableStyles.td}>{formatDuration(job.duration_ms)}</td>
                  <td style={tableStyles.td}>
                    {job.status === "failed" ? (
                      <button
                        type="button"
                        style={btnPrimary}
                        disabled={retrying === job.id}
                        onClick={() => handleRetry(job)}
                      >
                        {retrying === job.id ? "Retrying…" : "Retry"}
                      </button>
                    ) : (
                      "—"
                    )}
                  </td>
                </tr>
              ))}
              {jobs.length === 0 ? (
                <tr>
                  <td style={tableStyles.td} colSpan={7}>
                    No jobs found.
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

const tabStyle: React.CSSProperties = {
  border: "1px solid #d1d5db",
  background: "#fff",
  borderRadius: 999,
  padding: "6px 16px",
  cursor: "pointer",
  fontSize: 13,
  color: "#374151",
};
const tabActive: React.CSSProperties = {
  background: "#111827",
  color: "#fff",
  borderColor: "#111827",
};
const btnPrimary: React.CSSProperties = {
  background: "#2563eb",
  color: "#fff",
  border: "none",
  borderRadius: 8,
  padding: "6px 14px",
  cursor: "pointer",
  fontSize: 13,
};

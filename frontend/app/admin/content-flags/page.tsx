"use client";

import { useCallback, useEffect, useState } from "react";
import {
  formatIST,
  getContentFlags,
  resolveContentFlag,
  type ContentFlag,
} from "../../../lib/admin";
import { AdminCard, Pagination, StatusBadge, tableStyles } from "../../../components/admin/ui";

const LIMIT = 20;

const TABS: { label: string; value: string }[] = [
  { label: "Pending", value: "pending" },
  { label: "Resolved", value: "resolved" },
];

export default function AdminContentFlagsPage() {
  const [flags, setFlags] = useState<ContentFlag[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [tab, setTab] = useState("pending");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [modal, setModal] = useState<{ flag: ContentFlag; action: "approved" | "removed" } | null>(null);
  const [reason, setReason] = useState("");
  const [acting, setActing] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // "resolved" tab shows approved + removed; backend filters exact status,
      // so we request empty status and filter client-side for the resolved view.
      const statusParam = tab === "pending" ? "pending" : "";
      const data = await getContentFlags(statusParam, page, LIMIT);
      const items = tab === "resolved" ? (data.items ?? []).filter((f) => f.status !== "pending") : data.items ?? [];
      setFlags(items);
      setTotal(tab === "resolved" ? items.length : data.total);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load content flags");
    } finally {
      setLoading(false);
    }
  }, [tab, page]);

  useEffect(() => {
    void load();
  }, [load]);

  async function confirmResolve() {
    if (!modal) return;
    setActing(true);
    try {
      await resolveContentFlag(modal.flag.id, modal.action, reason.trim());
      setModal(null);
      setReason("");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to resolve flag");
    } finally {
      setActing(false);
    }
  }

  return (
    <div>
      <h1 style={{ margin: "0 0 16px", fontSize: 24 }}>Content Moderation</h1>

      <div style={{ display: "flex", gap: 6, marginBottom: 16 }}>
        {TABS.map((t) => (
          <button
            key={t.value}
            type="button"
            onClick={() => {
              setTab(t.value);
              setPage(1);
            }}
            style={{ ...tabStyle, ...(tab === t.value ? tabActive : {}) }}
          >
            {t.label}
          </button>
        ))}
      </div>

      {error ? <p style={{ color: "#b91c1c" }}>{error}</p> : null}

      <AdminCard>
        {loading ? (
          <p style={{ color: "#6b7280" }}>Loading flags…</p>
        ) : (
          <table style={tableStyles.table}>
            <thead>
              <tr>
                <th style={tableStyles.th}>Content Hash</th>
                <th style={tableStyles.th}>Reason</th>
                <th style={tableStyles.th}>Reported By</th>
                <th style={tableStyles.th}>Status</th>
                <th style={tableStyles.th}>Date</th>
                {tab === "pending" ? <th style={tableStyles.th}>Actions</th> : null}
              </tr>
            </thead>
            <tbody>
              {flags.map((flag) => (
                <tr key={flag.id}>
                  <td style={{ ...tableStyles.td, fontFamily: "monospace", fontSize: 12 }}>
                    {flag.content_hash ? `${flag.content_hash.slice(0, 12)}…` : `Q#${flag.question_id}`}
                  </td>
                  <td style={tableStyles.td}>{flag.reason}</td>
                  <td style={tableStyles.td}>{flag.reported_by_email || "—"}</td>
                  <td style={tableStyles.td}>
                    <StatusBadge status={flag.status} />
                  </td>
                  <td style={tableStyles.td}>{formatIST(flag.created_at)}</td>
                  {tab === "pending" ? (
                    <td style={tableStyles.td}>
                      <div style={{ display: "flex", gap: 6 }}>
                        <button
                          type="button"
                          style={btnSuccess}
                          onClick={() => {
                            setModal({ flag, action: "approved" });
                            setReason("");
                          }}
                        >
                          Approve
                        </button>
                        <button
                          type="button"
                          style={btnDanger}
                          onClick={() => {
                            setModal({ flag, action: "removed" });
                            setReason("");
                          }}
                        >
                          Remove
                        </button>
                      </div>
                    </td>
                  ) : null}
                </tr>
              ))}
              {flags.length === 0 ? (
                <tr>
                  <td style={tableStyles.td} colSpan={tab === "pending" ? 6 : 5}>
                    No flags to review.
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        )}
        {tab === "pending" ? <Pagination page={page} total={total} limit={LIMIT} onChange={setPage} /> : null}
      </AdminCard>

      {modal ? (
        <div style={modalStyles.overlay} onClick={() => setModal(null)}>
          <div style={modalStyles.box} onClick={(e) => e.stopPropagation()}>
            <h3 style={{ margin: "0 0 8px" }}>
              {modal.action === "removed" ? "Remove content" : "Approve content"}
            </h3>
            <p style={{ color: "#6b7280", marginTop: 0, fontSize: 14 }}>
              {modal.action === "removed"
                ? "Removing invalidates the cached quiz for this content hash. This action is logged."
                : "Approving keeps the content and closes the flag. This action is logged."}
            </p>
            <textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Resolution note (optional)"
              rows={3}
              style={{ ...dateInput, width: "100%", resize: "vertical" }}
            />
            <div style={{ display: "flex", gap: 8, justifyContent: "flex-end", marginTop: 12 }}>
              <button type="button" style={btnGhost} onClick={() => setModal(null)}>
                Cancel
              </button>
              <button
                type="button"
                style={modal.action === "removed" ? btnDanger : btnSuccess}
                disabled={acting}
                onClick={confirmResolve}
              >
                {acting ? "Saving…" : modal.action === "removed" ? "Confirm remove" : "Confirm approve"}
              </button>
            </div>
          </div>
        </div>
      ) : null}
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
const btnSuccess: React.CSSProperties = {
  background: "#16a34a",
  color: "#fff",
  border: "none",
  borderRadius: 8,
  padding: "6px 12px",
  cursor: "pointer",
  fontSize: 13,
};
const btnDanger: React.CSSProperties = {
  background: "#dc2626",
  color: "#fff",
  border: "none",
  borderRadius: 8,
  padding: "6px 12px",
  cursor: "pointer",
  fontSize: 13,
};
const btnGhost: React.CSSProperties = {
  background: "#fff",
  color: "#374151",
  border: "1px solid #d1d5db",
  borderRadius: 8,
  padding: "6px 12px",
  cursor: "pointer",
  fontSize: 13,
};
const dateInput: React.CSSProperties = {
  border: "1px solid #d1d5db",
  borderRadius: 8,
  padding: "8px 12px",
  fontSize: 14,
};

const modalStyles: Record<string, React.CSSProperties> = {
  overlay: {
    position: "fixed",
    inset: 0,
    background: "rgba(0,0,0,0.4)",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    zIndex: 100,
  },
  box: {
    background: "#fff",
    borderRadius: 12,
    padding: 24,
    width: "90%",
    maxWidth: 440,
  },
};

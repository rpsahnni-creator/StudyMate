"use client";

import { useCallback, useEffect, useState } from "react";
import {
  formatIST,
  getUsers,
  suspendUser,
  unsuspendUser,
  type AdminUser,
} from "../../../lib/admin";
import { AdminCard, Pagination, StatusBadge, tableStyles } from "../../../components/admin/ui";

const LIMIT = 20;

export default function AdminUsersPage() {
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [searchInput, setSearchInput] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [modalUser, setModalUser] = useState<AdminUser | null>(null);
  const [reason, setReason] = useState("");
  const [acting, setActing] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getUsers(page, LIMIT, search);
      setUsers(data.items ?? []);
      setTotal(data.total);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load users");
    } finally {
      setLoading(false);
    }
  }, [page, search]);

  useEffect(() => {
    void load();
  }, [load]);

  function submitSearch(e: React.FormEvent) {
    e.preventDefault();
    setPage(1);
    setSearch(searchInput.trim());
  }

  async function confirmSuspend() {
    if (!modalUser || !reason.trim()) return;
    setActing(true);
    try {
      await suspendUser(modalUser.id, reason.trim());
      setModalUser(null);
      setReason("");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to suspend user");
    } finally {
      setActing(false);
    }
  }

  async function handleUnsuspend(user: AdminUser) {
    if (!confirm(`Reactivate ${user.email}?`)) return;
    setActing(true);
    try {
      await unsuspendUser(user.id);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to unsuspend user");
    } finally {
      setActing(false);
    }
  }

  return (
    <div>
      <h1 style={{ margin: "0 0 16px", fontSize: 24 }}>Users</h1>

      <form onSubmit={submitSearch} style={{ display: "flex", gap: 8, marginBottom: 16 }}>
        <input
          value={searchInput}
          onChange={(e) => setSearchInput(e.target.value)}
          placeholder="Search by email or name"
          style={inputStyle}
        />
        <button type="submit" style={btnPrimary}>
          Search
        </button>
      </form>

      {error ? <p style={{ color: "#b91c1c" }}>{error}</p> : null}

      <AdminCard>
        {loading ? (
          <p style={{ color: "#6b7280" }}>Loading users…</p>
        ) : (
          <table style={tableStyles.table}>
            <thead>
              <tr>
                <th style={tableStyles.th}>Name</th>
                <th style={tableStyles.th}>Email</th>
                <th style={tableStyles.th}>Plan</th>
                <th style={tableStyles.th}>Scans today</th>
                <th style={tableStyles.th}>Status</th>
                <th style={tableStyles.th}>Joined</th>
                <th style={tableStyles.th}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {users.map((u) => (
                <tr key={u.id}>
                  <td style={tableStyles.td}>{u.name}</td>
                  <td style={tableStyles.td}>{u.email}</td>
                  <td style={tableStyles.td}>
                    <span style={{ textTransform: "capitalize" }}>{u.plan}</span>
                  </td>
                  <td style={tableStyles.td}>{u.scan_count_today}</td>
                  <td style={tableStyles.td}>
                    <StatusBadge status={u.status} />
                  </td>
                  <td style={tableStyles.td}>{formatIST(u.created_at)}</td>
                  <td style={tableStyles.td}>
                    {u.is_suspended ? (
                      <button
                        type="button"
                        style={btnSuccess}
                        disabled={acting}
                        onClick={() => handleUnsuspend(u)}
                      >
                        Unsuspend
                      </button>
                    ) : (
                      <button
                        type="button"
                        style={btnDanger}
                        disabled={acting || u.role === "admin"}
                        onClick={() => {
                          setModalUser(u);
                          setReason("");
                        }}
                      >
                        Suspend
                      </button>
                    )}
                  </td>
                </tr>
              ))}
              {users.length === 0 ? (
                <tr>
                  <td style={tableStyles.td} colSpan={7}>
                    No users found.
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        )}
        <Pagination page={page} total={total} limit={LIMIT} onChange={setPage} />
      </AdminCard>

      {modalUser ? (
        <div style={modalStyles.overlay} onClick={() => setModalUser(null)}>
          <div style={modalStyles.box} onClick={(e) => e.stopPropagation()}>
            <h3 style={{ margin: "0 0 8px" }}>Suspend {modalUser.email}</h3>
            <p style={{ color: "#6b7280", marginTop: 0, fontSize: 14 }}>
              This blocks the user&apos;s access. A reason is required and will be logged.
            </p>
            <textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Reason for suspension"
              rows={3}
              style={{ ...inputStyle, width: "100%", resize: "vertical" }}
            />
            <div style={{ display: "flex", gap: 8, justifyContent: "flex-end", marginTop: 12 }}>
              <button type="button" style={btnGhost} onClick={() => setModalUser(null)}>
                Cancel
              </button>
              <button
                type="button"
                style={btnDanger}
                disabled={acting || !reason.trim()}
                onClick={confirmSuspend}
              >
                {acting ? "Suspending…" : "Confirm suspend"}
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}

const inputStyle: React.CSSProperties = {
  border: "1px solid #d1d5db",
  borderRadius: 8,
  padding: "8px 12px",
  fontSize: 14,
  flex: 1,
};
const btnPrimary: React.CSSProperties = {
  background: "#2563eb",
  color: "#fff",
  border: "none",
  borderRadius: 8,
  padding: "8px 16px",
  cursor: "pointer",
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
const btnSuccess: React.CSSProperties = {
  background: "#16a34a",
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

"use client";

import type { CSSProperties, ReactNode } from "react";

export function StatusBadge({ status }: { status: string }) {
  const theme = STATUS_THEMES[status] ?? STATUS_THEMES.default;
  return (
    <span
      style={{
        display: "inline-block",
        padding: "2px 10px",
        borderRadius: 999,
        fontSize: 12,
        fontWeight: 600,
        background: theme.bg,
        color: theme.color,
        textTransform: "capitalize",
      }}
    >
      {status.replace(/_/g, " ")}
    </span>
  );
}

const STATUS_THEMES: Record<string, { bg: string; color: string }> = {
  active: { bg: "#dcfce7", color: "#166534" },
  quiz_ready: { bg: "#dcfce7", color: "#166534" },
  approved: { bg: "#dcfce7", color: "#166534" },
  pending: { bg: "#fef9c3", color: "#854d0e" },
  processing: { bg: "#dbeafe", color: "#1d4ed8" },
  ocr_complete: { bg: "#dbeafe", color: "#1d4ed8" },
  uploading: { bg: "#e0e7ff", color: "#4338ca" },
  suspended: { bg: "#fee2e2", color: "#991b1b" },
  failed: { bg: "#fee2e2", color: "#991b1b" },
  removed: { bg: "#fee2e2", color: "#991b1b" },
  default: { bg: "#f3f4f6", color: "#374151" },
};

export function Pagination({
  page,
  total,
  limit,
  onChange,
}: {
  page: number;
  total: number;
  limit: number;
  onChange: (page: number) => void;
}) {
  const totalPages = Math.max(1, Math.ceil(total / limit));
  return (
    <div style={paginationStyles.wrap}>
      <button
        type="button"
        className="btn-reset"
        style={paginationStyles.btn}
        disabled={page <= 1}
        onClick={() => onChange(page - 1)}
      >
        ← Prev
      </button>
      <span style={paginationStyles.info}>
        Page {page} of {totalPages} · {total} total
      </span>
      <button
        type="button"
        className="btn-reset"
        style={paginationStyles.btn}
        disabled={page >= totalPages}
        onClick={() => onChange(page + 1)}
      >
        Next →
      </button>
    </div>
  );
}

export function AdminCard({ title, children }: { title?: string; children: ReactNode }) {
  return (
    <div style={cardStyles.card}>
      {title ? <h3 style={cardStyles.title}>{title}</h3> : null}
      {children}
    </div>
  );
}

export const tableStyles: Record<string, CSSProperties> = {
  table: {
    width: "100%",
    borderCollapse: "collapse",
    background: "var(--surface)",
    borderRadius: "var(--r-lg)",
    overflow: "hidden",
    boxShadow: "var(--shadow-sm)",
    border: "1px solid var(--border)",
  },
  th: {
    textAlign: "left",
    padding: "13px 14px",
    fontSize: 12,
    textTransform: "uppercase",
    letterSpacing: 0.5,
    fontWeight: 700,
    color: "var(--text-muted)",
    borderBottom: "1px solid var(--border)",
    background: "var(--surface-2)",
  },
  td: {
    padding: "13px 14px",
    fontSize: 14,
    borderBottom: "1px solid var(--surface-2)",
    color: "var(--text)",
  },
};

const paginationStyles: Record<string, CSSProperties> = {
  wrap: {
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    gap: 12,
    marginTop: 16,
  },
  btn: {
    border: "1px solid var(--border-strong)",
    background: "var(--surface)",
    borderRadius: "var(--r-sm)",
    padding: "8px 14px",
    cursor: "pointer",
    fontSize: 13,
    fontWeight: 600,
    color: "var(--text)",
  },
  info: {
    fontSize: 13,
    color: "var(--text-muted)",
  },
};

const cardStyles: Record<string, CSSProperties> = {
  card: {
    background: "var(--surface)",
    borderRadius: "var(--r-lg)",
    padding: 20,
    boxShadow: "var(--shadow-sm)",
    border: "1px solid var(--border)",
  },
  title: {
    margin: "0 0 12px",
    fontSize: 13,
    color: "var(--text-muted)",
    fontWeight: 700,
    textTransform: "uppercase",
    letterSpacing: "0.03em",
  },
};

"use client";

import dynamic from "next/dynamic";
import { notFound } from "next/navigation";
import { useMemo, type ComponentType } from "react";
import "swagger-ui-react/swagger-ui.css";

const SwaggerUI = dynamic(() => import("swagger-ui-react"), { ssr: false }) as ComponentType<{
  url: string;
  docExpansion?: string;
  defaultModelsExpandDepth?: number;
}>;

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export default function ApiDocsPage() {
  const isProduction = process.env.NEXT_PUBLIC_ENV === "production";

  const specUrl = useMemo(() => `${API_URL}/openapi.yaml`, []);

  if (isProduction) {
    notFound();
  }

  return (
    <main style={{ minHeight: "100vh", background: "#fff" }}>
      <div style={{ padding: "1rem 1.5rem", borderBottom: "1px solid #e2e8f0" }}>
        <h1 style={{ margin: 0, fontSize: "1.25rem" }}>StudyApp API Docs</h1>
        <p style={{ margin: "0.25rem 0 0", color: "#64748b", fontSize: "0.875rem" }}>
          Development only — spec loaded from {specUrl}
        </p>
      </div>
      <SwaggerUI url={specUrl} docExpansion="list" defaultModelsExpandDepth={1} />
    </main>
  );
}

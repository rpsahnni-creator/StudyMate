import { apiCall } from "./auth";

export type ScanJobStatusValue =
  | "uploading"
  | "pending"
  | "queued"
  | "processing"
  | "ocr_complete"
  | "needs_strategy"
  | "review_ready"
  | "quiz_ready"
  | "completed"
  | "failed";

export type GenerationStrategy = "extract_questions" | "generate_from_chapter";

export interface ScanUploadStatus {
  status: ScanJobStatusValue;
  progress: number;
  retry_count: number;
  last_error?: string;
  can_retry: boolean;
  message?: string;
  quiz_id?: number;
  detected_page_type?: string;
  needs_strategy?: boolean;
  needs_review?: boolean;
  chapter_summary?: string;
}

const READY_STATUSES = new Set<ScanJobStatusValue>(["quiz_ready", "completed"]);
const TERMINAL_STATUSES = new Set<ScanJobStatusValue>([
  "quiz_ready",
  "completed",
  "review_ready",
  "failed",
]);

export function isJobReady(status: ScanJobStatusValue): boolean {
  return READY_STATUSES.has(status);
}

export function isJobTerminal(status: ScanJobStatusValue): boolean {
  return TERMINAL_STATUSES.has(status);
}

export function isJobAwaitingStrategy(status: ScanUploadStatus): boolean {
  return status.status === "needs_strategy" || Boolean(status.needs_strategy);
}

// Question-scan jobs finish in review_ready: the quiz is drafted and the user
// must review/edit/answer and publish before students can take it.
export function isJobNeedsReview(status: ScanUploadStatus): boolean {
  return status.status === "review_ready" || Boolean(status.needs_review);
}

export function jobStageLabel(status: ScanJobStatusValue): string {
  switch (status) {
    case "uploading":
      return "Uploading image…";
    case "pending":
    case "queued":
      return "Queued for OCR…";
    case "processing":
      return "Reading text from your page…";
    case "ocr_complete":
      return "Generating quiz questions…";
    case "needs_strategy":
      return "Choose how to build your quiz…";
    case "review_ready":
      return "Review questions & answers…";
    case "quiz_ready":
    case "completed":
      return "Quiz ready!";
    case "failed":
      return "Scan failed";
    default:
      return "Processing…";
  }
}

export async function fetchScanJobStatus(jobId: number): Promise<ScanUploadStatus> {
  const res = await apiCall(`/scan/uploads/${jobId}/status`);
  if (!res.ok) {
    throw new Error(`Failed to fetch job status (${res.status})`);
  }
  return (await res.json()) as ScanUploadStatus;
}

export async function setScanJobStrategy(jobId: number, strategy: GenerationStrategy): Promise<void> {
  const res = await apiCall(`/scan/jobs/${jobId}/strategy`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ generation_strategy: strategy }),
  });
  if (!res.ok) {
    const data = (await res.json().catch(() => ({}))) as { message?: string; error?: string };
    throw new Error(data.message ?? data.error ?? `Failed to set strategy (${res.status})`);
  }
}

export interface PollOptions {
  intervalMs?: number;
  timeoutMs?: number;
  onUpdate?: (status: ScanUploadStatus) => void;
  onNeedsStrategy?: (status: ScanUploadStatus) => void | Promise<void>;
}

/**
 * Polls the scan job status until it reaches a terminal state (quiz_ready,
 * completed, or failed) or the timeout elapses.
 */
export function pollScanJobStatus(jobId: number, options: PollOptions = {}): { cancel: () => void; promise: Promise<ScanUploadStatus> } {
  const intervalMs = options.intervalMs ?? 3000;
  const timeoutMs = options.timeoutMs ?? 5 * 60 * 1000;
  let cancelled = false;
  let timeoutHandle: ReturnType<typeof setTimeout> | null = null;

  const promise = new Promise<ScanUploadStatus>((resolve, reject) => {
    const startedAt = Date.now();

    async function tick() {
      if (cancelled) return;
      try {
        const status = await fetchScanJobStatus(jobId);
        if (cancelled) return;
        options.onUpdate?.(status);

        if (isJobAwaitingStrategy(status)) {
          await options.onNeedsStrategy?.(status);
          timeoutHandle = setTimeout(() => void tick(), intervalMs);
          return;
        }

        if (isJobTerminal(status.status)) {
          resolve(status);
          return;
        }
        if (Date.now() - startedAt >= timeoutMs) {
          reject(new Error("Timed out waiting for scan to finish. Check the Reports tab shortly."));
          return;
        }
        timeoutHandle = setTimeout(() => void tick(), intervalMs);
      } catch (error) {
        if (cancelled) return;
        if (Date.now() - startedAt >= timeoutMs) {
          reject(error instanceof Error ? error : new Error("Polling failed"));
          return;
        }
        timeoutHandle = setTimeout(() => void tick(), intervalMs);
      }
    }

    void tick();
  });

  return {
    cancel: () => {
      cancelled = true;
      if (timeoutHandle) clearTimeout(timeoutHandle);
    },
    promise,
  };
}

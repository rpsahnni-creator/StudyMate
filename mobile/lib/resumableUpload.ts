import * as FileSystem from "expo-file-system/legacy";
import { Platform } from "react-native";
import { formatApiErrorBody } from "./auth";
import { API_URL } from "./config";
import { formatNetworkError, isRetryableNetworkError } from "./network";

export const MAX_PAGES_PER_SCAN = 10;

export interface UploadProgress {
  uploadedBytes: number;
  totalBytes: number;
  percent: number;
}

const PAGE_UPLOAD_RETRIES = 3;
const RETRY_BASE_MS = 1500;

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function normalizeFileUri(uri: string): string {
  if (Platform.OS === "ios" && !uri.startsWith("file://")) {
    return `file://${uri}`;
  }
  return uri;
}

function parseUploadError(body: string, status: number): string {
  try {
    const data = JSON.parse(body) as {
      message?: string;
      error?: string;
      details?: Record<string, string>;
    };
    return formatApiErrorBody(data) || `Upload failed (${status})`;
  } catch {
    const trimmed = body.trim();
    return trimmed || `Upload failed (${status})`;
  }
}

async function uploadPageImageOnce(
  uri: string,
  jobId: string,
  pageNumber: number,
  token: string,
  onProgress?: (progress: UploadProgress) => void
): Promise<void> {
  onProgress?.({ uploadedBytes: 0, totalBytes: 100, percent: 0 });

  const result = await FileSystem.uploadAsync(
    `${API_URL}/scan/jobs/${jobId}/pages/${pageNumber}/image`,
    normalizeFileUri(uri),
    {
      uploadType: FileSystem.FileSystemUploadType.MULTIPART,
      fieldName: "file",
      mimeType: "image/jpeg",
      headers: {
        Authorization: `Bearer ${token}`,
      },
    }
  );

  if (result.status < 200 || result.status >= 300) {
    throw new Error(parseUploadError(result.body, result.status));
  }

  onProgress?.({ uploadedBytes: 100, totalBytes: 100, percent: 100 });
}

/** Direct multipart upload with per-page retries for flaky Wi‑Fi. */
export async function uploadPageImage(
  uri: string,
  jobId: string,
  pageNumber: number,
  token: string,
  onProgress?: (progress: UploadProgress) => void
): Promise<void> {
  let lastError: unknown;
  for (let attempt = 0; attempt < PAGE_UPLOAD_RETRIES; attempt++) {
    try {
      await uploadPageImageOnce(uri, jobId, pageNumber, token, onProgress);
      return;
    } catch (error) {
      lastError = error;
      const retryable = isRetryableNetworkError(error);
      if (!retryable || attempt === PAGE_UPLOAD_RETRIES - 1) {
        break;
      }
      await sleep(RETRY_BASE_MS * 2 ** attempt);
    }
  }
  throw new Error(formatNetworkError(lastError));
}

export async function uploadInChunks(
  uri: string,
  jobId: string,
  pageNumber: number,
  token: string,
  onProgress?: (progress: UploadProgress) => void
): Promise<void> {
  return uploadPageImage(uri, jobId, pageNumber, token, onProgress);
}

export interface MultiPageUploadOptions {
  startFromPageIndex?: number;
  onPageComplete?: (pageIndex: number, totalPages: number) => void;
}

export async function uploadMultiplePages(
  uris: string[],
  jobId: string,
  token: string,
  onProgress?: (percent: number, pageIndex: number, totalPages: number) => void,
  options: MultiPageUploadOptions = {}
): Promise<number> {
  if (uris.length === 0) {
    throw new Error("No pages to upload");
  }
  if (uris.length > MAX_PAGES_PER_SCAN) {
    throw new Error(`Maximum ${MAX_PAGES_PER_SCAN} pages per scan`);
  }

  const totalPages = uris.length;
  const startIndex = Math.max(0, Math.min(options.startFromPageIndex ?? 0, totalPages - 1));

  for (let i = startIndex; i < totalPages; i++) {
    await uploadPageImage(uris[i], jobId, i + 1, token, (pageProgress) => {
      const completedWeight = i / totalPages;
      const currentWeight = pageProgress.percent / 100 / totalPages;
      const overall = Math.round((completedWeight + currentWeight) * 100);
      onProgress?.(overall, i, totalPages);
    });
    options.onPageComplete?.(i, totalPages);
    onProgress?.(Math.round(((i + 1) / totalPages) * 100), i, totalPages);
  }

  return totalPages;
}
